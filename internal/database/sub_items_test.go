package database

import (
	"testing"

	"carryless/internal/models"
)

func TestSubItemCRUD(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner, _ := CreateUser(db, "siuser", "siuser@example.com", "password")
	cat, _ := CreateCategory(db, owner.ID, "Clothing")
	item, _ := CreateItem(db, owner.ID, models.Item{
		CategoryID:  cat.ID,
		Name:        "Rain Jacket",
		WeightGrams: 500,
	})

	// Create sub-items
	s1, err := CreateSubItem(db, item.ID, "Stuff Sack")
	if err != nil {
		t.Fatalf("CreateSubItem: %v", err)
	}
	s2, err := CreateSubItem(db, item.ID, "Hood")
	if err != nil {
		t.Fatalf("CreateSubItem second: %v", err)
	}
	if s1.SortOrder >= s2.SortOrder {
		t.Error("expected s2 to have higher sort order than s1")
	}

	// GetSubItems
	subs, err := GetSubItems(db, item.ID)
	if err != nil {
		t.Fatalf("GetSubItems: %v", err)
	}
	if len(subs) != 2 {
		t.Fatalf("expected 2 sub-items, got %d", len(subs))
	}

	// UpdateSubItem
	if err := UpdateSubItem(db, s1.ID, owner.ID, "Compression Sack"); err != nil {
		t.Fatalf("UpdateSubItem: %v", err)
	}
	subs2, _ := GetSubItems(db, item.ID)
	if subs2[0].Name != "Compression Sack" {
		t.Errorf("expected updated name, got %q", subs2[0].Name)
	}

	// UpdateSubItem — wrong owner
	other, _ := CreateUser(db, "other_si", "other_si@example.com", "password")
	if err := UpdateSubItem(db, s1.ID, other.ID, "Hacked"); err == nil {
		t.Error("expected error when updating sub-item owned by different user")
	}

	// DeleteSubItem — wrong owner
	if err := DeleteSubItem(db, s1.ID, other.ID); err == nil {
		t.Error("expected error when deleting sub-item owned by different user")
	}

	// DeleteSubItem
	if err := DeleteSubItem(db, s1.ID, owner.ID); err != nil {
		t.Fatalf("DeleteSubItem: %v", err)
	}
	subs3, _ := GetSubItems(db, item.ID)
	if len(subs3) != 1 || subs3[0].ID != s2.ID {
		t.Errorf("expected 1 sub-item remaining (s2), got %d", len(subs3))
	}
}

func TestSubItemPackChecks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	owner, _ := CreateUser(db, "checksi", "checksi@example.com", "password")
	cat, _ := CreateCategory(db, owner.ID, "Sleep")
	item, _ := CreateItem(db, owner.ID, models.Item{
		CategoryID:  cat.ID,
		Name:        "Sleeping Bag",
		WeightGrams: 800,
	})

	sub, err := CreateSubItem(db, item.ID, "Compression Sack")
	if err != nil {
		t.Fatalf("CreateSubItem: %v", err)
	}

	pack, _ := CreatePack(db, owner.ID, "Summer Pack")
	if err := AddItemToPack(db, pack.ID, item.ID, owner.ID); err != nil {
		t.Fatalf("AddItemToPack: %v", err)
	}

	// GetSubItemsForPack — initially unchecked
	subs, err := GetSubItemsForPack(db, item.ID, pack.ID)
	if err != nil {
		t.Fatalf("GetSubItemsForPack: %v", err)
	}
	if len(subs) != 1 {
		t.Fatalf("expected 1 sub-item, got %d", len(subs))
	}
	if subs[0].IsChecked {
		t.Error("expected sub-item to start unchecked")
	}

	// Toggle to checked
	if err := ToggleSubItemCheck(db, pack.ID, sub.ID, true); err != nil {
		t.Fatalf("ToggleSubItemCheck: %v", err)
	}
	subs, _ = GetSubItemsForPack(db, item.ID, pack.ID)
	if !subs[0].IsChecked {
		t.Error("expected sub-item to be checked after toggle")
	}

	// Toggle back to unchecked (idempotent upsert)
	if err := ToggleSubItemCheck(db, pack.ID, sub.ID, false); err != nil {
		t.Fatalf("ToggleSubItemCheck uncheck: %v", err)
	}
	subs, _ = GetSubItemsForPack(db, item.ID, pack.ID)
	if subs[0].IsChecked {
		t.Error("expected sub-item to be unchecked after second toggle")
	}

	// Cascade: force-deleting the item cascades sub-items and check rows
	if err := DeleteItemWithForce(db, owner.ID, item.ID, true); err != nil {
		t.Fatalf("DeleteItemWithForce: %v", err)
	}
	subs, _ = GetSubItems(db, item.ID)
	if len(subs) != 0 {
		t.Errorf("expected 0 sub-items after item delete, got %d", len(subs))
	}
}
