package database

import (
	"testing"
	"time"
)

func TestPackShareCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner, _ := CreateUser(db, "owner", "owner@example.com", "password")
	guest, _ := CreateUser(db, "guest", "guest@example.com", "password")
	pack, _ := CreatePack(db, owner.ID, "Shared Pack")

	// Owner has "owner" permission
	perm := GetUserSharePermission(db, pack.ID, owner.ID)
	if perm != "owner" {
		t.Errorf("Expected 'owner', got %q", perm)
	}

	// Unknown user has "none"
	perm = GetUserSharePermission(db, pack.ID, guest.ID)
	if perm != "none" {
		t.Errorf("Expected 'none' before share, got %q", perm)
	}

	// Create share
	err := CreatePackShare(db, pack.ID, owner.ID, guest.ID, "view")
	if err != nil {
		t.Fatal("Failed to create share:", err)
	}

	perm = GetUserSharePermission(db, pack.ID, guest.ID)
	if perm != "view" {
		t.Errorf("Expected 'view', got %q", perm)
	}

	// Upsert to higher permission
	err = CreatePackShare(db, pack.ID, owner.ID, guest.ID, "edit")
	if err != nil {
		t.Fatal("Failed to upsert share:", err)
	}
	perm = GetUserSharePermission(db, pack.ID, guest.ID)
	if perm != "edit" {
		t.Errorf("Expected 'edit' after upsert, got %q", perm)
	}

	// GetPackShares returns the guest
	shares, err := GetPackShares(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get pack shares:", err)
	}
	if len(shares) != 1 {
		t.Fatalf("Expected 1 share, got %d", len(shares))
	}
	if shares[0].SharedWithUserID != guest.ID {
		t.Errorf("Share has wrong user: got %d, want %d", shares[0].SharedWithUserID, guest.ID)
	}
	if shares[0].SharedWithUser == nil || shares[0].SharedWithUser.Username != "guest" {
		t.Error("Expected SharedWithUser to be populated")
	}

	// UpdatePackShare
	err = UpdatePackShare(db, pack.ID, owner.ID, guest.ID, "add")
	if err != nil {
		t.Fatal("Failed to update share:", err)
	}
	perm = GetUserSharePermission(db, pack.ID, guest.ID)
	if perm != "add" {
		t.Errorf("Expected 'add' after update, got %q", perm)
	}

	// GetPacksSharedWithUser
	sharedPacks, err := GetPacksSharedWithUser(db, guest.ID)
	if err != nil {
		t.Fatal("Failed to get shared packs:", err)
	}
	if len(sharedPacks) != 1 || sharedPacks[0].ID != pack.ID {
		t.Errorf("Expected 1 shared pack %s, got %d packs", pack.ID, len(sharedPacks))
	}
	if sharedPacks[0].UserPermission != "add" {
		t.Errorf("Expected UserPermission 'add', got %q", sharedPacks[0].UserPermission)
	}

	// DeletePackShare
	err = DeletePackShare(db, pack.ID, owner.ID, guest.ID)
	if err != nil {
		t.Fatal("Failed to delete share:", err)
	}
	perm = GetUserSharePermission(db, pack.ID, guest.ID)
	if perm != "none" {
		t.Errorf("Expected 'none' after delete, got %q", perm)
	}
}

func TestPackShareCascadeOnPackDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner, _ := CreateUser(db, "owner", "owner@example.com", "password")
	guest, _ := CreateUser(db, "guest", "guest@example.com", "password")
	pack, _ := CreatePack(db, owner.ID, "Pack")
	_ = CreatePackShare(db, pack.ID, owner.ID, guest.ID, "view")

	err := DeletePack(db, owner.ID, pack.ID)
	if err != nil {
		t.Fatal("Failed to delete pack:", err)
	}

	// Share should be gone due to ON DELETE CASCADE
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM pack_shares WHERE pack_id = ?`, pack.ID).Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 shares after pack delete, got %d", count)
	}
}

func TestGetUserByUsernameOrEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "alice", "alice@example.com", "password")

	byUsername, err := GetUserByUsernameOrEmail(db, "alice")
	if err != nil {
		t.Fatal("Failed by username:", err)
	}
	if byUsername.ID != user.ID {
		t.Errorf("Wrong user by username: got %d, want %d", byUsername.ID, user.ID)
	}

	byEmail, err := GetUserByUsernameOrEmail(db, "alice@example.com")
	if err != nil {
		t.Fatal("Failed by email:", err)
	}
	if byEmail.ID != user.ID {
		t.Errorf("Wrong user by email: got %d, want %d", byEmail.ID, user.ID)
	}

	_, err = GetUserByUsernameOrEmail(db, "nobody")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestSearchUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Need activated users for search
	owner, _ := CreateUser(db, "owner", "owner@example.com", "password")
	_, _ = CreateUser(db, "alice_hiker", "alice@trail.com", "password")
	_, _ = CreateUser(db, "bob_hiker", "bob@trail.com", "password")

	// Activate them so they appear in search
	db.Exec(`UPDATE users SET is_activated = TRUE WHERE username IN ('alice_hiker','bob_hiker')`)

	results, err := SearchUsers(db, "hiker", owner.ID)
	if err != nil {
		t.Fatal("Search failed:", err)
	}
	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'hiker', got %d", len(results))
	}

	// Should not return the calling user
	results, err = SearchUsers(db, "owner", owner.ID)
	if err != nil {
		t.Fatal("Search failed:", err)
	}
	if len(results) != 0 {
		t.Errorf("Search should exclude the calling user, got %d results", len(results))
	}
}

func TestPackInviteCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner, _ := CreateUser(db, "owner", "owner@example.com", "password")
	guest, _ := CreateUser(db, "guest", "guest@example.com", "password")
	db.Exec(`UPDATE users SET is_activated = TRUE WHERE id = ?`, guest.ID)
	pack, _ := CreatePack(db, owner.ID, "Pack")

	// Create invite
	invite, err := CreatePackInvite(db, pack.ID, owner.ID, "view")
	if err != nil {
		t.Fatal("Failed to create invite:", err)
	}
	if invite.Token == "" {
		t.Error("Expected non-empty invite token")
	}
	if invite.ExpiresAt.Before(time.Now()) {
		t.Error("Invite should expire in the future")
	}

	// Get invites
	invites, err := GetPackInvites(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get invites:", err)
	}
	if len(invites) != 1 {
		t.Fatalf("Expected 1 invite, got %d", len(invites))
	}

	// Redeem invite
	err = RedeemPackInvite(db, invite.Token, guest.ID)
	if err != nil {
		t.Fatal("Failed to redeem invite:", err)
	}
	perm := GetUserSharePermission(db, pack.ID, guest.ID)
	if perm != "view" {
		t.Errorf("Expected 'view' after redeem, got %q", perm)
	}

	// Owner cannot redeem their own invite
	err = RedeemPackInvite(db, invite.Token, owner.ID)
	if err == nil {
		t.Error("Expected error when owner redeems own invite")
	}

	// Delete invite
	err = DeletePackInvite(db, invite.ID, owner.ID)
	if err != nil {
		t.Fatal("Failed to delete invite:", err)
	}
	invites, _ = GetPackInvites(db, pack.ID)
	if len(invites) != 0 {
		t.Errorf("Expected 0 invites after delete, got %d", len(invites))
	}
}

func TestPermissionHelpers(t *testing.T) {
	tests := []struct {
		perm    string
		write   bool
		edit    bool
		admin   bool
	}{
		{"owner", true, true, true},
		{"admin", true, true, true},
		{"edit", true, true, false},
		{"add", true, false, false},
		{"view", false, false, false},
		{"none", false, false, false},
	}
	for _, tc := range tests {
		if CanWrite(tc.perm) != tc.write {
			t.Errorf("CanWrite(%q) = %v, want %v", tc.perm, CanWrite(tc.perm), tc.write)
		}
		if CanEdit(tc.perm) != tc.edit {
			t.Errorf("CanEdit(%q) = %v, want %v", tc.perm, CanEdit(tc.perm), tc.edit)
		}
		if CanAdmin(tc.perm) != tc.admin {
			t.Errorf("CanAdmin(%q) = %v, want %v", tc.perm, CanAdmin(tc.perm), tc.admin)
		}
	}
}
