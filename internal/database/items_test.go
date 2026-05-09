package database

import (
	"database/sql"
	"testing"

	"carryless/internal/models"
)

func ensureCat(t *testing.T, db *sql.DB, userID int) int {
	t.Helper()
	cat, err := GetOrCreateCategory(db, userID, "Gear")
	if err != nil {
		t.Fatalf("GetOrCreateCategory: %v", err)
	}
	return cat.ID
}

func item(name string, weight, catID int) models.Item {
	return models.Item{
		Name:        name,
		CategoryID:  catID,
		WeightGrams: weight,
	}
}

func TestItemCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "itemuser", "item@example.com", "pass")
	catID := ensureCat(t, db, user.ID)

	// Create
	created, err := CreateItem(db, user.ID, item("Tent", 1200, catID))
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected non-zero ID after create")
	}
	if created.Name != "Tent" {
		t.Errorf("expected 'Tent', got %q", created.Name)
	}

	// GetItem
	got, err := GetItem(db, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}
	if got.WeightGrams != 1200 {
		t.Errorf("expected weight 1200, got %d", got.WeightGrams)
	}

	// GetItem with wrong user should fail
	_, err = GetItem(db, 99999, created.ID)
	if err == nil {
		t.Error("expected error for wrong user ID")
	}

	// UpdateItem
	if err := UpdateItem(db, user.ID, created.ID, item("Tent Pro", 900, catID)); err != nil {
		t.Fatalf("UpdateItem failed: %v", err)
	}
	got2, _ := GetItem(db, user.ID, created.ID)
	if got2.Name != "Tent Pro" {
		t.Errorf("expected 'Tent Pro', got %q", got2.Name)
	}
	if got2.WeightGrams != 900 {
		t.Errorf("expected weight 900, got %d", got2.WeightGrams)
	}

	// DeleteItem
	if err := DeleteItem(db, user.ID, created.ID); err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
	_, err = GetItem(db, user.ID, created.ID)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestGetItems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "listuser", "list@example.com", "pass")
	other, _ := CreateUser(db, "otheruser", "other@example.com", "pass")
	catID := ensureCat(t, db, user.ID)
	otherCatID := ensureCat(t, db, other.ID)

	CreateItem(db, user.ID, item("ItemA", 100, catID))
	CreateItem(db, user.ID, item("ItemB", 200, catID))
	CreateItem(db, other.ID, item("OtherItem", 300, otherCatID))

	items, err := GetItems(db, user.ID)
	if err != nil {
		t.Fatalf("GetItems failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items for user, got %d", len(items))
	}
	for _, it := range items {
		if it.UserID != user.ID {
			t.Errorf("item %d belongs to wrong user %d", it.ID, it.UserID)
		}
	}
}

func TestPatchItem(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "patchuser", "patch@example.com", "pass")
	catID := ensureCat(t, db, user.ID)
	created, _ := CreateItem(db, user.ID, item("Sleeping Bag", 800, catID))

	updated, err := PatchItem(db, user.ID, created.ID, map[string]interface{}{
		"name":         "Sleeping Bag UL",
		"weight_grams": 600,
	})
	if err != nil {
		t.Fatalf("PatchItem failed: %v", err)
	}
	if updated.Name != "Sleeping Bag UL" {
		t.Errorf("expected 'Sleeping Bag UL', got %q", updated.Name)
	}
	if updated.WeightGrams != 600 {
		t.Errorf("expected 600, got %d", updated.WeightGrams)
	}

	// Invalid column name should be rejected
	_, err = PatchItem(db, user.ID, created.ID, map[string]interface{}{
		"drop table items --": "hack",
	})
	if err == nil {
		t.Error("expected error for invalid column name")
	}

	// Wrong user should fail
	_, err = PatchItem(db, 99999, created.ID, map[string]interface{}{"name": "x"})
	if err == nil {
		t.Error("expected error for wrong user")
	}
}

func TestBulkDeleteItems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "bulkdeluser", "bulkdel@example.com", "pass")
	other, _ := CreateUser(db, "otherowner", "otherowner@example.com", "pass")
	catID := ensureCat(t, db, user.ID)
	otherCatID := ensureCat(t, db, other.ID)

	a, _ := CreateItem(db, user.ID, item("A", 100, catID))
	b, _ := CreateItem(db, user.ID, item("B", 200, catID))
	c, _ := CreateItem(db, other.ID, item("C", 300, otherCatID))

	// Delete two owned items
	n, err := BulkDeleteItems(db, user.ID, []int{a.ID, b.ID})
	if err != nil {
		t.Fatalf("BulkDeleteItems failed: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 deleted, got %d", n)
	}

	// Trying to delete another user's item should fail
	_, err = BulkDeleteItems(db, user.ID, []int{c.ID})
	if err == nil {
		t.Error("expected error deleting another user's item")
	}

	// Empty list should fail
	_, err = BulkDeleteItems(db, user.ID, []int{})
	if err == nil {
		t.Error("expected error for empty ID list")
	}
}

func TestBulkUpdateItems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "bulkupduser", "bulkupd@example.com", "pass")
	catID := ensureCat(t, db, user.ID)
	a, _ := CreateItem(db, user.ID, item("A", 100, catID))
	b, _ := CreateItem(db, user.ID, item("B", 200, catID))

	err := BulkUpdateItems(db, user.ID, []int{a.ID, b.ID}, map[string]interface{}{
		"note": "bulk updated",
	})
	if err != nil {
		t.Fatalf("BulkUpdateItems failed: %v", err)
	}

	gotA, _ := GetItem(db, user.ID, a.ID)
	if gotA.Note != "bulk updated" {
		t.Errorf("expected 'bulk updated', got %q", gotA.Note)
	}

	// Empty inputs should fail
	err = BulkUpdateItems(db, user.ID, []int{}, map[string]interface{}{"note": "x"})
	if err == nil {
		t.Error("expected error for empty item IDs")
	}

	// Invalid column
	err = BulkUpdateItems(db, user.ID, []int{a.ID}, map[string]interface{}{"badcol": "x"})
	if err == nil {
		t.Error("expected error for invalid column")
	}
}

func TestDuplicateItem(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "dupuser", "dup@example.com", "pass")
	catID := ensureCat(t, db, user.ID)
	created, _ := CreateItem(db, user.ID, item("Original", 500, catID))

	dupe, err := DuplicateItem(db, user.ID, created.ID)
	if err != nil {
		t.Fatalf("DuplicateItem failed: %v", err)
	}
	if dupe.ID == created.ID {
		t.Error("duplicate should have different ID")
	}
	if dupe.WeightGrams != created.WeightGrams {
		t.Errorf("expected weight %d, got %d", created.WeightGrams, dupe.WeightGrams)
	}
}

func TestGetPacksUsingItem(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "packsuser", "packs@example.com", "pass")
	db.Exec("UPDATE users SET is_activated = 1 WHERE id = ?", user.ID)
	catID := ensureCat(t, db, user.ID)

	created, _ := CreateItem(db, user.ID, item("Shared Item", 100, catID))
	pack1, _ := CreatePack(db, user.ID, "Pack One")
	pack2, _ := CreatePack(db, user.ID, "Pack Two")

	AddItemToPack(db, pack1.ID, created.ID, user.ID)
	AddItemToPack(db, pack2.ID, created.ID, user.ID)

	packNames, err := GetPacksUsingItem(db, user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetPacksUsingItem failed: %v", err)
	}
	if len(packNames) != 2 {
		t.Errorf("expected 2 packs, got %d: %v", len(packNames), packNames)
	}
}

func TestDeleteAllItems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "deletealluser", "deleteall@example.com", "pass")
	other, _ := CreateUser(db, "keepmeuser", "keepme@example.com", "pass")
	catID := ensureCat(t, db, user.ID)
	otherCatID := ensureCat(t, db, other.ID)

	CreateItem(db, user.ID, item("ToDelete1", 100, catID))
	CreateItem(db, user.ID, item("ToDelete2", 200, catID))
	CreateItem(db, other.ID, item("KeepMe", 300, otherCatID))

	if err := DeleteAllItems(db, user.ID); err != nil {
		t.Fatalf("DeleteAllItems failed: %v", err)
	}

	items, _ := GetItems(db, user.ID)
	if len(items) != 0 {
		t.Errorf("expected 0 items after DeleteAllItems, got %d", len(items))
	}

	otherItems, _ := GetItems(db, other.ID)
	if len(otherItems) != 1 {
		t.Errorf("other user should still have 1 item, got %d", len(otherItems))
	}
}

func TestGetItemsToVerify(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "verifyitems", "verify@example.com", "pass")
	catID := ensureCat(t, db, user.ID)

	i1 := item("NeedsVerify", 100, catID)
	i1.WeightToVerify = true
	i2 := item("Verified", 200, catID)
	i2.WeightToVerify = false

	CreateItem(db, user.ID, i1)
	CreateItem(db, user.ID, i2)

	toVerify, err := GetItemsToVerify(db, user.ID)
	if err != nil {
		t.Fatalf("GetItemsToVerify failed: %v", err)
	}
	if len(toVerify) != 1 {
		t.Errorf("expected 1 item to verify, got %d", len(toVerify))
	}
	if !toVerify[0].WeightToVerify {
		t.Error("returned item should have WeightToVerify=true")
	}
}

func TestGetItemsWithFilters(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, _ := CreateUser(db, "filteruser", "filter@example.com", "pass")
	catID := ensureCat(t, db, user.ID)

	brand := "TestBrand"
	model := "TestModel"

	i1 := item("WithBrand", 100, catID)
	i1.Brand = &brand
	i1.Model = &model

	i2 := item("NoBrand", 200, catID)
	i2.WeightToVerify = true

	CreateItem(db, user.ID, i1)
	CreateItem(db, user.ID, i2)

	// Filter: weight to verify only
	filtered, err := GetItemsWithFilters(db, user.ID, true, false, false)
	if err != nil {
		t.Fatalf("GetItemsWithFilters failed: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Name != "NoBrand" {
		t.Errorf("expected 1 item with verify filter, got %d", len(filtered))
	}

	// Filter: empty brand
	filtered2, err := GetItemsWithFilters(db, user.ID, false, true, false)
	if err != nil {
		t.Fatalf("GetItemsWithFilters (emptyBrand) failed: %v", err)
	}
	if len(filtered2) != 1 || filtered2[0].Name != "NoBrand" {
		t.Errorf("expected 1 item with empty brand, got %d", len(filtered2))
	}
}
