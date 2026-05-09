package database

import (
	"database/sql"
	"os"
	"testing"
	"time"

	"carryless/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Use a named shared in-memory database unique to this test so that all
	// connections in the pool see the same data. Plain ":memory:" gives each
	// connection its own database, which causes "no such table" errors when
	// production code mixes transactions and bare db.Query calls.
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_foreign_keys=on"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal("Failed to open test database:", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatal("Failed to run migrations:", err)
	}

	return db
}

func TestUserCreationAndAuthentication(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", user.Username)
	}

	if user.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", user.Email)
	}

	authUser, err := AuthenticateUser(db, "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to authenticate user:", err)
	}

	if authUser.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, authUser.ID)
	}

	_, err = AuthenticateUser(db, "test@example.com", "wrongpassword")
	if err == nil {
		t.Error("Expected authentication to fail with wrong password")
	}
}

func TestSessionManagement(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	sessionDuration := 7 * 24 * time.Hour

	session, err := CreateSession(db, user.ID, sessionDuration)
	if err != nil {
		t.Fatal("Failed to create session:", err)
	}

	if len(session.ID) == 0 {
		t.Error("Session ID should not be empty")
	}

	validatedUser, err := ValidateSession(db, session.ID, sessionDuration)
	if err != nil {
		t.Fatal("Failed to validate session:", err)
	}

	if validatedUser.ID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, validatedUser.ID)
	}

	err = DeleteSession(db, session.ID)
	if err != nil {
		t.Fatal("Failed to delete session:", err)
	}

	_, err = ValidateSession(db, session.ID, sessionDuration)
	if err == nil {
		t.Error("Expected session validation to fail after deletion")
	}
}

func TestCategoryOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	category, err := CreateCategory(db, user.ID, "Sleeping")
	if err != nil {
		t.Fatal("Failed to create category:", err)
	}

	if category.Name != "Sleeping" {
		t.Errorf("Expected category name 'Sleeping', got %s", category.Name)
	}

	categories, err := GetCategories(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get categories:", err)
	}

	if len(categories) != 1 {
		t.Errorf("Expected 1 category, got %d", len(categories))
	}

	err = UpdateCategory(db, user.ID, category.ID, "Sleep System")
	if err != nil {
		t.Fatal("Failed to update category:", err)
	}

	updatedCategory, err := GetCategory(db, user.ID, category.ID)
	if err != nil {
		t.Fatal("Failed to get updated category:", err)
	}

	if updatedCategory.Name != "Sleep System" {
		t.Errorf("Expected category name 'Sleep System', got %s", updatedCategory.Name)
	}

	err = DeleteCategory(db, user.ID, category.ID)
	if err != nil {
		t.Fatal("Failed to delete category:", err)
	}

	_, err = GetCategory(db, user.ID, category.ID)
	if err == nil {
		t.Error("Expected category retrieval to fail after deletion")
	}
}

func TestItemOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	category, err := CreateCategory(db, user.ID, "Sleeping")
	if err != nil {
		t.Fatal("Failed to create category:", err)
	}

	item := models.Item{
		CategoryID:   category.ID,
		Name:         "Sleeping Bag",
		Note:         "Down sleeping bag",
		WeightGrams:  800,
		Price:        299.99,
	}

	createdItem, err := CreateItem(db, user.ID, item)
	if err != nil {
		t.Fatal("Failed to create item:", err)
	}

	if createdItem.Name != "Sleeping Bag" {
		t.Errorf("Expected item name 'Sleeping Bag', got %s", createdItem.Name)
	}

	items, err := GetItems(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get items:", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	updatedItem := models.Item{
		CategoryID:   category.ID,
		Name:         "Down Sleeping Bag",
		Note:         "Lightweight down sleeping bag",
		WeightGrams:  750,
		Price:        299.99,
	}

	err = UpdateItem(db, user.ID, createdItem.ID, updatedItem)
	if err != nil {
		t.Fatal("Failed to update item:", err)
	}

	retrievedItem, err := GetItem(db, user.ID, createdItem.ID)
	if err != nil {
		t.Fatal("Failed to get updated item:", err)
	}

	if retrievedItem.Name != "Down Sleeping Bag" {
		t.Errorf("Expected item name 'Down Sleeping Bag', got %s", retrievedItem.Name)
	}

	if retrievedItem.WeightGrams != 750 {
		t.Errorf("Expected weight 750g, got %dg", retrievedItem.WeightGrams)
	}

	err = DeleteItem(db, user.ID, createdItem.ID)
	if err != nil {
		t.Fatal("Failed to delete item:", err)
	}

	_, err = GetItem(db, user.ID, createdItem.ID)
	if err == nil {
		t.Error("Expected item retrieval to fail after deletion")
	}
}

func TestPackOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	pack, err := CreatePack(db, user.ID, "Weekend Trip")
	if err != nil {
		t.Fatal("Failed to create pack:", err)
	}

	if pack.Name != "Weekend Trip" {
		t.Errorf("Expected pack name 'Weekend Trip', got %s", pack.Name)
	}

	if len(pack.ID) == 0 {
		t.Error("Pack ID should not be empty")
	}

	packs, err := GetPacks(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get packs:", err)
	}

	if len(packs) != 1 {
		t.Errorf("Expected 1 pack, got %d", len(packs))
	}

	err = UpdatePack(db, user.ID, pack.ID, "Extended Weekend Trip", true)
	if err != nil {
		t.Fatal("Failed to update pack:", err)
	}

	updatedPack, err := GetPack(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get updated pack:", err)
	}

	if updatedPack.Name != "Extended Weekend Trip" {
		t.Errorf("Expected pack name 'Extended Weekend Trip', got %s", updatedPack.Name)
	}

	if !updatedPack.IsPublic {
		t.Error("Expected pack to be public")
	}

	err = DeletePack(db, user.ID, pack.ID)
	if err != nil {
		t.Fatal("Failed to delete pack:", err)
	}

	_, err = GetPack(db, pack.ID)
	if err == nil {
		t.Error("Expected pack retrieval to fail after deletion")
	}
}

func TestAuthEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	_, err := CreateUser(db, "alice", "alice@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create first user:", err)
	}

	// Duplicate username
	_, err = CreateUser(db, "alice", "other@example.com", "password123")
	if err == nil {
		t.Error("Expected error on duplicate username, got nil")
	}

	// Duplicate email
	_, err = CreateUser(db, "bob", "alice@example.com", "password123")
	if err == nil {
		t.Error("Expected error on duplicate email, got nil")
	}

	// Session expiry: create session then backdate expires_at
	user, _ := CreateUser(db, "carol", "carol@example.com", "password123")
	session, err := CreateSession(db, user.ID, 7*24*time.Hour)
	if err != nil {
		t.Fatal("Failed to create session:", err)
	}

	_, err = db.Exec("UPDATE sessions SET expires_at = datetime('now', '-1 hour') WHERE id = ?", session.ID)
	if err != nil {
		t.Fatal("Failed to backdate session:", err)
	}

	_, err = ValidateSession(db, session.ID, 7*24*time.Hour)
	if err == nil {
		t.Error("Expected expired session to fail validation")
	}
}

func TestPackAdvancedOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	category, err := CreateCategory(db, user.ID, "Shelter")
	if err != nil {
		t.Fatal("Failed to create category:", err)
	}

	item, err := CreateItem(db, user.ID, models.Item{
		CategoryID:  category.ID,
		Name:        "Tent",
		WeightGrams: 1200,
	})
	if err != nil {
		t.Fatal("Failed to create item:", err)
	}

	pack, err := CreatePack(db, user.ID, "Summer Trip")
	if err != nil {
		t.Fatal("Failed to create pack:", err)
	}

	// Add item to pack
	err = AddItemToPack(db, pack.ID, item.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to add item to pack:", err)
	}

	withItems, err := GetPackWithItems(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get pack with items:", err)
	}
	if len(withItems.Items) != 1 {
		t.Errorf("Expected 1 item in pack, got %d", len(withItems.Items))
	}

	// Remove item from pack
	err = RemoveItemFromPack(db, pack.ID, item.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to remove item from pack:", err)
	}

	withItems, err = GetPackWithItems(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get pack with items after remove:", err)
	}
	if len(withItems.Items) != 0 {
		t.Errorf("Expected 0 items after removal, got %d", len(withItems.Items))
	}

	// Toggle pack lock
	err = TogglePackLock(db, user.ID, pack.ID, true)
	if err != nil {
		t.Fatal("Failed to lock pack:", err)
	}
	locked, err := GetPack(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get pack:", err)
	}
	if !locked.IsLocked {
		t.Error("Expected pack to be locked")
	}

	err = TogglePackLock(db, user.ID, pack.ID, false)
	if err != nil {
		t.Fatal("Failed to unlock pack:", err)
	}

	// Make pack public — UpdatePack generates a short_id
	err = UpdatePack(db, user.ID, pack.ID, pack.Name, true)
	if err != nil {
		t.Fatal("Failed to make pack public:", err)
	}
	publicPack, err := GetPack(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get public pack:", err)
	}
	if !publicPack.IsPublic {
		t.Error("Expected pack to be public")
	}
	if publicPack.ShortID == "" {
		t.Error("Expected short_id to be generated for public pack")
	}

	// Look up by short ID
	byShortID, err := GetPackByShortID(db, publicPack.ShortID)
	if err != nil {
		t.Fatal("Failed to get pack by short ID:", err)
	}
	if byShortID.ID != pack.ID {
		t.Errorf("Short ID lookup returned wrong pack: got %s, want %s", byShortID.ID, pack.ID)
	}

	// Duplicate pack
	dupPack, err := DuplicatePack(db, user.ID, pack.ID)
	if err != nil {
		t.Fatal("Failed to duplicate pack:", err)
	}
	if dupPack.Name != pack.Name+" Copy" {
		t.Errorf("Expected duplicate name %q, got %q", pack.Name+" Copy", dupPack.Name)
	}
	if dupPack.ID == pack.ID {
		t.Error("Duplicate pack should have a different ID")
	}
}

func TestItemAdvancedOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	category, err := CreateCategory(db, user.ID, "Clothing")
	if err != nil {
		t.Fatal("Failed to create category:", err)
	}

	item1, err := CreateItem(db, user.ID, models.Item{CategoryID: category.ID, Name: "Rain Jacket", WeightGrams: 300})
	if err != nil {
		t.Fatal("Failed to create item1:", err)
	}
	item2, err := CreateItem(db, user.ID, models.Item{CategoryID: category.ID, Name: "Fleece", WeightGrams: 400})
	if err != nil {
		t.Fatal("Failed to create item2:", err)
	}
	item3, err := CreateItem(db, user.ID, models.Item{CategoryID: category.ID, Name: "Base Layer", WeightGrams: 150})
	if err != nil {
		t.Fatal("Failed to create item3:", err)
	}

	// Duplicate item
	dup, err := DuplicateItem(db, user.ID, item1.ID)
	if err != nil {
		t.Fatal("Failed to duplicate item:", err)
	}
	if dup.ID == item1.ID {
		t.Error("Duplicate item should have different ID")
	}
	if dup.WeightGrams != item1.WeightGrams {
		t.Errorf("Duplicate item weight mismatch: got %d, want %d", dup.WeightGrams, item1.WeightGrams)
	}

	// Bulk update — change weight of item2 and item3
	err = BulkUpdateItems(db, user.ID, []int{item2.ID, item3.ID}, map[string]interface{}{
		"weight_grams": 500,
	})
	if err != nil {
		t.Fatal("Failed to bulk update items:", err)
	}
	updated2, err := GetItem(db, user.ID, item2.ID)
	if err != nil {
		t.Fatal("Failed to get item2 after bulk update:", err)
	}
	if updated2.WeightGrams != 500 {
		t.Errorf("Expected weight 500 after bulk update, got %d", updated2.WeightGrams)
	}

	// Bulk delete
	deleted, err := BulkDeleteItems(db, user.ID, []int{item2.ID, item3.ID})
	if err != nil {
		t.Fatal("Failed to bulk delete:", err)
	}
	if deleted != 2 {
		t.Errorf("Expected 2 items deleted, got %d", deleted)
	}

	_, err = GetItem(db, user.ID, item2.ID)
	if err == nil {
		t.Error("Expected item2 to be deleted")
	}

	// item1 and its duplicate should still exist
	items, err := GetItems(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get items:", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 items remaining, got %d", len(items))
	}
}

func TestCategoryEdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	_, err = CreateCategory(db, user.ID, "Electronics")
	if err != nil {
		t.Fatal("Failed to create category:", err)
	}

	// Duplicate name for same user should fail
	_, err = CreateCategory(db, user.ID, "Electronics")
	if err == nil {
		t.Error("Expected error on duplicate category name for same user, got nil")
	}

	// Different user can have the same category name
	user2, _ := CreateUser(db, "other", "other@example.com", "password123")
	_, err = CreateCategory(db, user2.ID, "Electronics")
	if err != nil {
		t.Errorf("Different user should be able to create same category name, got: %v", err)
	}
}

func TestPackLabelOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	pack, err := CreatePack(db, user.ID, "Label Test Pack")
	if err != nil {
		t.Fatal("Failed to create pack:", err)
	}

	// Create pack label
	label, err := CreatePackLabel(db, pack.ID, "Worn", "#ff0000", user.ID)
	if err != nil {
		t.Fatal("Failed to create pack label:", err)
	}
	if label.Name != "Worn" {
		t.Errorf("Expected label name 'Worn', got %s", label.Name)
	}

	// Get pack labels
	labels, err := GetPackLabels(db, pack.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to get pack labels:", err)
	}
	if len(labels) != 1 {
		t.Errorf("Expected 1 label, got %d", len(labels))
	}

	// Update pack label
	err = UpdatePackLabel(db, label.ID, "Consumable", "#00ff00", user.ID)
	if err != nil {
		t.Fatal("Failed to update pack label:", err)
	}
	updatedLabels, err := GetPackLabels(db, pack.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to get updated labels:", err)
	}
	if updatedLabels[0].Name != "Consumable" {
		t.Errorf("Expected updated label name 'Consumable', got %s", updatedLabels[0].Name)
	}

	// Delete pack label
	err = DeletePackLabel(db, label.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to delete pack label:", err)
	}
	labels, err = GetPackLabels(db, pack.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to get labels after delete:", err)
	}
	if len(labels) != 0 {
		t.Errorf("Expected 0 labels after delete, got %d", len(labels))
	}
}

func TestUserPackLabelOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	pack, err := CreatePack(db, user.ID, "UPL Test Pack")
	if err != nil {
		t.Fatal("Failed to create pack:", err)
	}

	// Create user pack label
	upl, err := CreateUserPackLabel(db, user.ID, "Ultralight", "#0000ff")
	if err != nil {
		t.Fatal("Failed to create user pack label:", err)
	}
	if upl.Name != "Ultralight" {
		t.Errorf("Expected label name 'Ultralight', got %s", upl.Name)
	}

	// Get user pack labels
	labels, err := GetUserPackLabels(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get user pack labels:", err)
	}
	if len(labels) != 1 {
		t.Errorf("Expected 1 user pack label, got %d", len(labels))
	}

	// Update user pack label
	err = UpdateUserPackLabel(db, upl.ID, "Gram Weenie", "#ff00ff", user.ID)
	if err != nil {
		t.Fatal("Failed to update user pack label:", err)
	}

	// Assign label to pack
	err = AssignLabelToPack(db, pack.ID, upl.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to assign label to pack:", err)
	}

	packLabels, err := GetPackLevelLabels(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get pack level labels:", err)
	}
	if len(packLabels) != 1 {
		t.Errorf("Expected 1 pack level label, got %d", len(packLabels))
	}

	// Remove label from pack
	err = RemoveLabelFromPack(db, pack.ID, upl.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to remove label from pack:", err)
	}

	packLabels, err = GetPackLevelLabels(db, pack.ID)
	if err != nil {
		t.Fatal("Failed to get pack level labels after remove:", err)
	}
	if len(packLabels) != 0 {
		t.Errorf("Expected 0 pack level labels after removal, got %d", len(packLabels))
	}

	// Delete user pack label
	err = DeleteUserPackLabel(db, upl.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to delete user pack label:", err)
	}
	labels, err = GetUserPackLabels(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get user pack labels after delete:", err)
	}
	if len(labels) != 0 {
		t.Errorf("Expected 0 user pack labels after delete, got %d", len(labels))
	}
}

func TestItemLinkOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	category, err := CreateCategory(db, user.ID, "Cooking")
	if err != nil {
		t.Fatal("Failed to create category:", err)
	}

	cookset, err := CreateItem(db, user.ID, models.Item{CategoryID: category.ID, Name: "Cookset", WeightGrams: 300})
	if err != nil {
		t.Fatal("Failed to create cookset:", err)
	}
	stove, err := CreateItem(db, user.ID, models.Item{CategoryID: category.ID, Name: "Stove", WeightGrams: 100})
	if err != nil {
		t.Fatal("Failed to create stove:", err)
	}
	pot, err := CreateItem(db, user.ID, models.Item{CategoryID: category.ID, Name: "Pot", WeightGrams: 180})
	if err != nil {
		t.Fatal("Failed to create pot:", err)
	}

	// Create item links
	err = CreateItemLink(db, user.ID, cookset.ID, stove.ID)
	if err != nil {
		t.Fatal("Failed to create stove link:", err)
	}
	err = CreateItemLink(db, user.ID, cookset.ID, pot.ID)
	if err != nil {
		t.Fatal("Failed to create pot link:", err)
	}

	// Get linked items
	links, err := GetLinkedItems(db, cookset.ID)
	if err != nil {
		t.Fatal("Failed to get linked items:", err)
	}
	if len(links) != 2 {
		t.Errorf("Expected 2 linked items, got %d", len(links))
	}

	// Delete one link
	err = DeleteItemLink(db, user.ID, cookset.ID, stove.ID)
	if err != nil {
		t.Fatal("Failed to delete item link:", err)
	}
	links, err = GetLinkedItems(db, cookset.ID)
	if err != nil {
		t.Fatal("Failed to get links after delete:", err)
	}
	if len(links) != 1 {
		t.Errorf("Expected 1 linked item after deletion, got %d", len(links))
	}

	// Self-link should fail
	err = CreateItemLink(db, user.ID, cookset.ID, cookset.ID)
	if err == nil {
		t.Error("Expected self-link to fail, got nil error")
	}
}

func TestTripOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	pack, err := CreatePack(db, user.ID, "Trip Pack")
	if err != nil {
		t.Fatal("Failed to create pack:", err)
	}

	// Create trip
	desc := "A great adventure"
	loc := "Rocky Mountains"
	trip, err := CreateTrip(db, user.ID, "PCT Section", &desc, &loc, nil, nil, false)
	if err != nil {
		t.Fatal("Failed to create trip:", err)
	}
	if trip.Name != "PCT Section" {
		t.Errorf("Expected trip name 'PCT Section', got %s", trip.Name)
	}

	// Get trips
	trips, err := GetTrips(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get trips:", err)
	}
	if len(trips) != 1 {
		t.Errorf("Expected 1 trip, got %d", len(trips))
	}

	// Get single trip
	fetched, err := GetTrip(db, trip.ID)
	if err != nil {
		t.Fatal("Failed to get trip:", err)
	}
	if fetched.ID != trip.ID {
		t.Errorf("Fetched wrong trip: got %s, want %s", fetched.ID, trip.ID)
	}

	// Update trip
	newDesc := "Updated description"
	err = UpdateTrip(db, user.ID, trip.ID, "PCT Section J1", &newDesc, nil, nil, nil, false)
	if err != nil {
		t.Fatal("Failed to update trip:", err)
	}
	updated, err := GetTrip(db, trip.ID)
	if err != nil {
		t.Fatal("Failed to get updated trip:", err)
	}
	if updated.Name != "PCT Section J1" {
		t.Errorf("Expected updated trip name 'PCT Section J1', got %s", updated.Name)
	}

	// Add pack to trip
	err = AddPackToTrip(db, trip.ID, pack.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to add pack to trip:", err)
	}
	tripPacks, err := GetTripPacks(db, trip.ID)
	if err != nil {
		t.Fatal("Failed to get trip packs:", err)
	}
	if len(tripPacks) != 1 {
		t.Errorf("Expected 1 pack in trip, got %d", len(tripPacks))
	}

	// Remove pack from trip
	err = RemovePackFromTrip(db, trip.ID, pack.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to remove pack from trip:", err)
	}
	tripPacks, err = GetTripPacks(db, trip.ID)
	if err != nil {
		t.Fatal("Failed to get trip packs after removal:", err)
	}
	if len(tripPacks) != 0 {
		t.Errorf("Expected 0 packs after removal, got %d", len(tripPacks))
	}

	// Checklist CRUD
	checkItem, err := AddChecklistItem(db, trip.ID, "Pack food", user.ID)
	if err != nil {
		t.Fatal("Failed to add checklist item:", err)
	}
	if checkItem.Content != "Pack food" {
		t.Errorf("Expected checklist content 'Pack food', got %s", checkItem.Content)
	}

	err = ToggleChecklistItem(db, checkItem.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to toggle checklist item:", err)
	}

	checkItems, err := GetChecklistItems(db, trip.ID)
	if err != nil {
		t.Fatal("Failed to get checklist items:", err)
	}
	if len(checkItems) != 1 || !checkItems[0].IsChecked {
		t.Error("Expected checklist item to be checked after toggle")
	}

	err = DeleteChecklistItem(db, checkItem.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to delete checklist item:", err)
	}
	checkItems, err = GetChecklistItems(db, trip.ID)
	if err != nil {
		t.Fatal("Failed to get checklist items after delete:", err)
	}
	if len(checkItems) != 0 {
		t.Errorf("Expected 0 checklist items after delete, got %d", len(checkItems))
	}

	// Transport step CRUD
	depTime := time.Now().Add(24 * time.Hour)
	step, err := AddTransportStep(db, trip.ID, "outbound", "Trailhead", &depTime, nil, nil, nil, nil, nil, user.ID)
	if err != nil {
		t.Fatal("Failed to add transport step:", err)
	}
	if step.DeparturePlace != "Trailhead" {
		t.Errorf("Expected departure place 'Trailhead', got %s", step.DeparturePlace)
	}

	steps, err := GetTransportSteps(db, trip.ID)
	if err != nil {
		t.Fatal("Failed to get transport steps:", err)
	}
	if len(steps) != 1 {
		t.Errorf("Expected 1 transport step, got %d", len(steps))
	}

	err = DeleteTransportStep(db, step.ID, user.ID)
	if err != nil {
		t.Fatal("Failed to delete transport step:", err)
	}
	steps, err = GetTransportSteps(db, trip.ID)
	if err != nil {
		t.Fatal("Failed to get transport steps after delete:", err)
	}
	if len(steps) != 0 {
		t.Errorf("Expected 0 transport steps after delete, got %d", len(steps))
	}

	// Delete trip
	err = DeleteTrip(db, user.ID, trip.ID)
	if err != nil {
		t.Fatal("Failed to delete trip:", err)
	}
	trips, err = GetTrips(db, user.ID)
	if err != nil {
		t.Fatal("Failed to get trips after delete:", err)
	}
	if len(trips) != 0 {
		t.Errorf("Expected 0 trips after delete, got %d", len(trips))
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}