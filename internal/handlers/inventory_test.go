package handlers

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"

	"carryless/internal/database"

	_ "github.com/mattn/go-sqlite3"
)

// memFile wraps a bytes.Reader to satisfy multipart.File
type memFile struct{ *bytes.Reader }

func (m *memFile) Close() error                              { return nil }
func (m *memFile) ReadAt(p []byte, off int64) (int, error)  { return m.Reader.ReadAt(p, off) }
func (m *memFile) Seek(offset int64, whence int) (int64, error) {
	return m.Reader.Seek(offset, whence)
}

func newMemFile(content string) multipart.File {
	return &memFile{bytes.NewReader([]byte(content))}
}

func newMemHeader(filename string, size int64) *multipart.FileHeader {
	return &multipart.FileHeader{Filename: filename, Size: size}
}

// TestValidateCSVFile_SmallFile verifies Bug 2 fix: small CSV files (< 512 bytes)
// no longer return an error due to io.EOF from file.Read.
func TestValidateCSVFile_SmallFile(t *testing.T) {
	content := "Name,Category,Weight (grams),Weight To Verify,Price,Notes,Brand,Model,Purchased,Capacity,Capacity Unit,Link\nTent,Shelter,1200,false,150.00,,,,,,,\n"
	f := newMemFile(content)
	h := newMemHeader("inventory.csv", int64(len(content)))

	if err := validateCSVFile(f, h); err != nil {
		t.Errorf("validateCSVFile returned error for small CSV: %v", err)
	}
}

// TestValidateCSVFile_LargeFile verifies validation still works for files >= 512 bytes.
func TestValidateCSVFile_LargeFile(t *testing.T) {
	header := "Name,Category,Weight (grams),Weight To Verify,Price,Notes,Brand,Model,Purchased,Capacity,Capacity Unit,Link\n"
	row := strings.Repeat("Tent,Shelter,1200,false,150.00,,,,,,,\n", 20)
	content := header + row
	f := newMemFile(content)
	h := newMemHeader("inventory.csv", int64(len(content)))

	if err := validateCSVFile(f, h); err != nil {
		t.Errorf("validateCSVFile returned error for large CSV: %v", err)
	}
}

// TestValidateCSVFile_WrongExtension verifies non-.csv files are rejected.
func TestValidateCSVFile_WrongExtension(t *testing.T) {
	content := "Name,Category\nTent,Shelter\n"
	f := newMemFile(content)
	h := newMemHeader("inventory.txt", int64(len(content)))

	if err := validateCSVFile(f, h); err == nil {
		t.Error("validateCSVFile should have rejected .txt extension")
	}
}

// TestParseCSVFile_RoundTrip verifies Bug 3 fix: all 12 export columns import correctly.
func TestParseCSVFile_RoundTrip(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := database.CreateUser(db, "testuser", "test@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	// Simulate a 12-column export row (all fields populated)
	csv12 := "Name,Category,Weight (grams),Weight To Verify,Price,Notes,Brand,Model,Purchased,Capacity,Capacity Unit,Link\n" +
		"Tent,Shelter,1200,true,149.99,Lightweight tent,Nemo,Hornet 2P,2023-06-15,2.00,L,https://example.com/tent\n"

	f := newMemFile(csv12)
	items, err := parseCSVFile(f, db, user.ID)
	if err != nil {
		t.Fatalf("parseCSVFile failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}

	imp := items[0]
	item := imp.Item
	if item.Name != "Tent" {
		t.Errorf("Name: expected 'Tent', got '%s'", item.Name)
	}
	if item.WeightGrams != 1200 {
		t.Errorf("WeightGrams: expected 1200, got %d", item.WeightGrams)
	}
	if !item.WeightToVerify {
		t.Error("WeightToVerify: expected true")
	}
	if item.Price != 149.99 {
		t.Errorf("Price: expected 149.99, got %f", item.Price)
	}
	if item.Note != "Lightweight tent" {
		t.Errorf("Note: expected 'Lightweight tent', got '%s'", item.Note)
	}
	if item.Brand == nil || *item.Brand != "Nemo" {
		t.Errorf("Brand: expected 'Nemo'")
	}
	if item.Model == nil || *item.Model != "Hornet 2P" {
		t.Errorf("Model: expected 'Hornet 2P'")
	}
	if item.PurchaseDate == nil {
		t.Error("PurchaseDate: expected non-nil")
	}
	if item.Capacity == nil || *item.Capacity != 2.0 {
		t.Errorf("Capacity: expected 2.0")
	}
	if item.CapacityUnit == nil || *item.CapacityUnit != "L" {
		t.Errorf("CapacityUnit: expected 'L'")
	}
	if item.Link == nil || *item.Link != "https://example.com/tent" {
		t.Errorf("Link: expected 'https://example.com/tent'")
	}
	if len(imp.SubItemNames) != 0 {
		t.Errorf("SubItemNames: expected none for 12-field CSV, got %v", imp.SubItemNames)
	}
}

// TestParseCSVFile_SmallFile verifies Bug 2+3 together: a small single-item CSV
// (which used to EOF-fail before the fix) now parses all 12 fields.
func TestParseCSVFile_SmallFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := database.CreateUser(db, "testuser2", "test2@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	// This content is < 512 bytes — the case that triggered Bug 2
	csvContent := "Name,Category,Weight (grams),Weight To Verify,Price,Notes,Brand,Model,Purchased,Capacity,Capacity Unit,Link\nTent,Shelter,800,false,0.00,,,,,,,\n"
	if len(csvContent) >= 512 {
		t.Fatalf("test assumption broken: content is %d bytes, not < 512", len(csvContent))
	}

	f := newMemFile(csvContent)
	items, err := parseCSVFile(f, db, user.ID)
	if err != nil {
		t.Fatalf("parseCSVFile failed on small file: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(items))
	}
	if items[0].Item.Name != "Tent" {
		t.Errorf("Expected name 'Tent', got '%s'", items[0].Item.Name)
	}
}

// TestParseCSVFile_SubItems verifies the 13-field format with sub-items parses correctly.
func TestParseCSVFile_SubItems(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := database.CreateUser(db, "testuser3", "test3@example.com", "password123")
	if err != nil {
		t.Fatal("Failed to create user:", err)
	}

	csv13 := "Name,Category,Weight (grams),Weight To Verify,Price,Notes,Brand,Model,Purchased,Capacity,Capacity Unit,Link,Sub-items\n" +
		"Sleeping Bag,Sleep,800,false,200.00,Warm bag,,,,,,, Compression Sack ; Hood\n" +
		"Tent,Shelter,1200,false,300.00,,,,,,,,\n"

	f := newMemFile(csv13)
	items, err := parseCSVFile(f, db, user.ID)
	if err != nil {
		t.Fatalf("parseCSVFile failed for 13-field format: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	bag := items[0]
	if bag.Item.Name != "Sleeping Bag" {
		t.Errorf("Name: expected 'Sleeping Bag', got '%s'", bag.Item.Name)
	}
	if len(bag.SubItemNames) != 2 {
		t.Fatalf("Expected 2 sub-items, got %d: %v", len(bag.SubItemNames), bag.SubItemNames)
	}
	if bag.SubItemNames[0] != "Compression Sack" {
		t.Errorf("SubItem[0]: expected 'Compression Sack', got '%s'", bag.SubItemNames[0])
	}
	if bag.SubItemNames[1] != "Hood" {
		t.Errorf("SubItem[1]: expected 'Hood', got '%s'", bag.SubItemNames[1])
	}

	tent := items[1]
	if len(tent.SubItemNames) != 0 {
		t.Errorf("Expected no sub-items for Tent, got %v", tent.SubItemNames)
	}
}
