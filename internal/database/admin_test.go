package database

import (
	"testing"
)

func TestGetAdminStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	stats, err := GetAdminStats(db)
	if err != nil {
		t.Fatalf("GetAdminStats failed: %v", err)
	}
	if stats.TotalUsers != 0 {
		t.Errorf("expected 0 users initially, got %d", stats.TotalUsers)
	}
	if stats.TotalPacks != 0 {
		t.Errorf("expected 0 packs initially, got %d", stats.TotalPacks)
	}

	// Add users and packs
	u1, _ := CreateUser(db, "admin1", "admin1@example.com", "pass")
	CreateUser(db, "admin2", "admin2@example.com", "pass")
	CreatePack(db, u1.ID, "Pack")

	stats, _ = GetAdminStats(db)
	if stats.TotalUsers != 2 {
		t.Errorf("expected 2 users, got %d", stats.TotalUsers)
	}
	if stats.TotalPacks != 1 {
		t.Errorf("expected 1 pack, got %d", stats.TotalPacks)
	}
}

func TestGetAllUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	CreateUser(db, "ua", "ua@example.com", "pass")
	CreateUser(db, "ub", "ub@example.com", "pass")

	users, err := GetAllUsers(db)
	if err != nil {
		t.Fatalf("GetAllUsers failed: %v", err)
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestGetAllUsersWithStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	u, _ := CreateUser(db, "statsuser", "stats@example.com", "pass")
	CreatePack(db, u.ID, "Pack1")
	CreatePack(db, u.ID, "Pack2")

	users, err := GetAllUsersWithStats(db)
	if err != nil {
		t.Fatalf("GetAllUsersWithStats failed: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].PackCount != 2 {
		t.Errorf("expected pack_count=2, got %d", users[0].PackCount)
	}
}

func TestToggleUserAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First user is auto-admin, second is not
	_, _ = CreateUser(db, "firstadm", "firstadm@example.com", "pass")
	second, _ := CreateUser(db, "nonadm", "nonadm@example.com", "pass")

	if second.IsAdmin {
		t.Fatal("second user should not be admin")
	}

	// Toggle to admin
	if err := ToggleUserAdmin(db, second.ID); err != nil {
		t.Fatalf("ToggleUserAdmin failed: %v", err)
	}
	got, _ := GetUserByID(db, second.ID)
	if !got.IsAdmin {
		t.Error("user should be admin after toggle")
	}

	// Toggle back
	if err := ToggleUserAdmin(db, second.ID); err != nil {
		t.Fatalf("ToggleUserAdmin second time failed: %v", err)
	}
	got, _ = GetUserByID(db, second.ID)
	if got.IsAdmin {
		t.Error("user should not be admin after second toggle")
	}
}

func TestBanUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	admin, _ := CreateUser(db, "adminbanner", "adminbanner@example.com", "pass")
	target, _ := CreateUser(db, "bantarget", "bantarget@example.com", "pass")

	// Create some data for target
	CreatePack(db, target.ID, "Target Pack")
	cat, _ := GetOrCreateCategory(db, target.ID, "Gear")
	CreateItem(db, target.ID, item("Target Item", 100, cat.ID))
	CreateSession(db, target.ID, 24*3600*1000000000) // 24 hours

	_ = admin // used above
	if err := BanUser(db, target.ID); err != nil {
		t.Fatalf("BanUser failed: %v", err)
	}

	// User should be deleted
	_, err := GetUserByID(db, target.ID)
	if err == nil {
		t.Error("banned user should not exist")
	}

	// User's packs should be deleted
	packs, _ := GetPacks(db, target.ID)
	if len(packs) != 0 {
		t.Errorf("banned user's packs should be deleted, got %d", len(packs))
	}
}

func TestIsRegistrationEnabled(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Default should be enabled (no setting → true)
	enabled, err := IsRegistrationEnabled(db)
	if err != nil {
		t.Fatalf("IsRegistrationEnabled failed: %v", err)
	}
	if !enabled {
		t.Error("registration should be enabled by default")
	}
}

func TestToggleRegistration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Insert the setting row first (migrations should have done this)
	db.Exec(`INSERT OR IGNORE INTO system_settings (key, value) VALUES ('registration_enabled', 'true')`)

	if err := ToggleRegistration(db); err != nil {
		t.Fatalf("ToggleRegistration failed: %v", err)
	}

	enabled, _ := IsRegistrationEnabled(db)
	if enabled {
		t.Error("registration should be disabled after toggle")
	}

	// Toggle back
	ToggleRegistration(db)
	enabled, _ = IsRegistrationEnabled(db)
	if !enabled {
		t.Error("registration should be re-enabled after second toggle")
	}
}

func TestToggleUserActivation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "activtoggle", "activtoggle@example.com", "pass")

	// Activate
	if err := ToggleUserActivation(db, user.ID); err != nil {
		t.Fatalf("ToggleUserActivation failed: %v", err)
	}
	got, _ := GetUserByID(db, user.ID)
	if !got.IsActivated {
		t.Error("user should be activated after toggle")
	}

	// Deactivate
	if err := ToggleUserActivation(db, user.ID); err != nil {
		t.Fatalf("ToggleUserActivation (deactivate) failed: %v", err)
	}
	got, _ = GetUserByID(db, user.ID)
	if got.IsActivated {
		t.Error("user should be deactivated after second toggle")
	}

	// Nonexistent user
	if err := ToggleUserActivation(db, 99999); err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestGetAllAdmins(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// First user is auto-admin
	first, _ := CreateUser(db, "firstadmin2", "fa2@example.com", "pass")
	CreateUser(db, "notadmin2", "notadmin2@example.com", "pass")

	admins, err := GetAllAdmins(db)
	if err != nil {
		t.Fatalf("GetAllAdmins failed: %v", err)
	}
	if len(admins) != 1 {
		t.Errorf("expected 1 admin, got %d", len(admins))
	}
	if admins[0].ID != first.ID {
		t.Errorf("expected first user to be admin")
	}
}
