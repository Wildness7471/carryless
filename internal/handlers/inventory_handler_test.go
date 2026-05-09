package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"carryless/internal/database"
)

func TestHandleCreateItem(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "createitem", "createitem@example.com")

	r := newRouter(db, user)
	r.POST("/inventory/items", handleCreateItem)

	w := req(t, r, "POST", "/inventory/items", url.Values{
		"name":          {"Tent"},
		"category_name": {"Shelter"},
		"weight_grams":  {"1200"},
	})
	if w.Code != http.StatusFound {
		t.Errorf("create item: expected 302, got %d: %s", w.Code, w.Body.String())
	}

	items, _ := database.GetItems(db, user.ID)
	if len(items) != 1 || items[0].Name != "Tent" {
		t.Errorf("item should exist after create, got %v", items)
	}
}

func TestHandleCreateItem_MissingName(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "createitemval", "createitemval@example.com")

	r := newRouter(db, user)
	r.POST("/inventory/items", handleCreateItem)

	// Missing name → validation fails → HTML render (panics in test, caught as 500, which is != 302)
	w := req(t, r, "POST", "/inventory/items", url.Values{
		"category_name": {"Shelter"},
		"weight_grams":  {"100"},
	})
	if w.Code == http.StatusFound {
		t.Error("create item with empty name should not redirect to success")
	}

	items, _ := database.GetItems(db, user.ID)
	if len(items) != 0 {
		t.Errorf("no items should be created for invalid request")
	}
}

func TestHandleDeleteItem(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "deleteitem", "deleteitem@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("ToDelete", 100, catID))

	r := newRouter(db, user)
	r.POST("/inventory/items/:id/delete", handleDeleteItem)

	w := req(t, r, "POST", "/inventory/items/"+istr(item.ID)+"/delete", nil)
	if w.Code != http.StatusFound {
		t.Fatalf("delete item: expected 302, got %d", w.Code)
	}

	items, _ := database.GetItems(db, user.ID)
	if len(items) != 0 {
		t.Errorf("item should be gone after delete, got %d items", len(items))
	}
}

func TestHandleDeleteItem_InPack(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "deleteinpack", "deleteinpack@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("InPack", 100, catID))
	pack, _ := database.CreatePack(db, user.ID, "Pack")
	database.AddItemToPack(db, pack.ID, item.ID, user.ID)

	r := newRouter(db, user)
	r.POST("/inventory/items/:id/delete", handleDeleteItem)

	// Regular delete should redirect with error (item is in a pack)
	w := req(t, r, "POST", "/inventory/items/"+istr(item.ID)+"/delete", nil)
	if w.Code != http.StatusFound {
		t.Fatalf("expected redirect, got %d", w.Code)
	}
	if !strings.Contains(w.Header().Get("Location"), "error") {
		t.Errorf("expected error redirect, got: %s", w.Header().Get("Location"))
	}

	// Force delete should succeed
	w = req(t, r, "POST", "/inventory/items/"+istr(item.ID)+"/delete",
		url.Values{"force": {"true"}})
	if w.Code != http.StatusFound {
		t.Fatalf("force delete: expected 302, got %d", w.Code)
	}

	items, _ := database.GetItems(db, user.ID)
	if len(items) != 0 {
		t.Errorf("item should be gone after force delete, got %d items", len(items))
	}
}

func TestHandleDuplicateItem(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "dupitem", "dupitem@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Original", 500, catID))

	r := newRouter(db, user)
	r.POST("/inventory/items/:id/duplicate", handleDuplicateItem)

	w := req(t, r, "POST", "/inventory/items/"+istr(item.ID)+"/duplicate", nil)
	if w.Code != http.StatusFound {
		t.Fatalf("duplicate item: expected 302, got %d", w.Code)
	}

	items, _ := database.GetItems(db, user.ID)
	if len(items) != 2 {
		t.Errorf("expected 2 items after duplicate, got %d", len(items))
	}
}

func TestHandleCheckItemPacks(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "checkpacks", "checkpacks@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("MultiPack", 200, catID))
	pack1, _ := database.CreatePack(db, user.ID, "Alpha")
	pack2, _ := database.CreatePack(db, user.ID, "Beta")
	database.AddItemToPack(db, pack1.ID, item.ID, user.ID)
	database.AddItemToPack(db, pack2.ID, item.ID, user.ID)

	r := newRouter(db, user)
	r.GET("/inventory/items/:id/packs", handleCheckItemPacks)

	w := req(t, r, "GET", "/inventory/items/"+istr(item.ID)+"/packs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("check item packs: expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	packs, ok := resp["packs"].([]interface{})
	if !ok || len(packs) != 2 {
		t.Errorf("expected 2 packs in response, got: %v", resp)
	}
}

func TestHandlePatchItem(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "patchitem", "patchitem@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Old Name", 300, catID))

	r := newRouter(db, user)
	r.PATCH("/api/items/:id", handlePatchItem)

	body := `{"name": "New Name", "weight_grams": 250}`
	httpReq := httptest.NewRequest("PATCH", "/api/items/"+istr(item.ID),
		bytes.NewBufferString(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("patch item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	got, _ := database.GetItem(db, user.ID, item.ID)
	if got.Name != "New Name" {
		t.Errorf("expected 'New Name', got %q", got.Name)
	}
	if got.WeightGrams != 250 {
		t.Errorf("expected weight 250, got %d", got.WeightGrams)
	}
}

func TestHandleBulkDeleteItems(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "bulkdel", "bulkdel@example.com")

	catID := ensureCategory(t, db, user.ID)
	a, _ := database.CreateItem(db, user.ID, newItemWithCat("A", 100, catID))
	b, _ := database.CreateItem(db, user.ID, newItemWithCat("B", 200, catID))
	_, _ = database.CreateItem(db, user.ID, newItemWithCat("C", 300, catID))

	r := newRouter(db, user)
	r.POST("/inventory/items/bulk-delete", handleBulkDeleteItems)

	// Bulk delete redirects on success
	w := req(t, r, "POST", "/inventory/items/bulk-delete", url.Values{
		"item_ids": {istr(a.ID) + "," + istr(b.ID)},
	})
	if w.Code != http.StatusFound {
		t.Fatalf("bulk delete: expected 302 redirect, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Header().Get("Location"), "error") {
		t.Errorf("bulk delete should succeed, got: %s", w.Header().Get("Location"))
	}

	items, _ := database.GetItems(db, user.ID)
	if len(items) != 1 {
		t.Errorf("expected 1 remaining item, got %d", len(items))
	}
}

func TestHandleGetItemLinks(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "linkuser", "link@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Main", 100, catID))

	r := newRouter(db, user)
	r.GET("/api/items/:id/links", handleGetItemLinks)

	w := req(t, r, "GET", "/api/items/"+istr(item.ID)+"/links", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get item links: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreateDeleteItemLink(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "linkcreatedelete", "linkcd@example.com")

	catID := ensureCategory(t, db, user.ID)
	a, _ := database.CreateItem(db, user.ID, newItemWithCat("A", 100, catID))
	b, _ := database.CreateItem(db, user.ID, newItemWithCat("B", 200, catID))

	r := newRouter(db, user)
	r.POST("/api/items/:id/links", handleCreateItemLink)
	r.DELETE("/api/items/:id/links/:linked_id", handleDeleteItemLink)

	// Create link — handler expects JSON body
	jsonBody := `{"linked_item_id": ` + istr(b.ID) + `}`
	httpReq := httptest.NewRequest("POST", "/api/items/"+istr(a.ID)+"/links",
		bytes.NewBufferString(jsonBody))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusCreated {
		t.Fatalf("create item link: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Delete link
	w2 := req(t, r, "DELETE", "/api/items/"+istr(a.ID)+"/links/"+istr(b.ID), nil)
	if w2.Code != http.StatusOK {
		t.Fatalf("delete item link: expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestHandleImportInventory(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "importuser", "import@example.com")

	csvContent := "Name,Category,Weight (grams),Weight To Verify,Price,Notes,Brand,Model,Purchased,Capacity,Capacity Unit,Link\n" +
		"Tent,Shelter,1200,false,150.00,,,,,,,\n" +
		"Sleeping Bag,Sleep,800,false,200.00,,,,,,,\n"

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("csvFile", "inventory.csv")
	part.Write([]byte(csvContent))
	writer.Close()

	httpReq := httptest.NewRequest("POST", "/inventory/import", &body)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	r := newRouter(db, user)
	r.POST("/inventory/import", handleImportInventory)
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusFound {
		t.Fatalf("import: expected 302 redirect, got %d: %s", w.Code, w.Body.String())
	}

	items, _ := database.GetItems(db, user.ID)
	if len(items) != 2 {
		t.Errorf("expected 2 imported items, got %d", len(items))
	}
}

func TestHandleExportInventory(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "exportuser", "export@example.com")

	catID := ensureCategory(t, db, user.ID)
	database.CreateItem(db, user.ID, newItemWithCat("Tent", 1200, catID))
	database.CreateItem(db, user.ID, newItemWithCat("Sleeping Bag", 800, catID))

	r := newRouter(db, user)
	r.GET("/inventory/export", handleExportInventory)

	httpReq := httptest.NewRequest("GET", "/inventory/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httpReq)

	if w.Code != http.StatusOK {
		t.Fatalf("export: expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("expected text/csv content type, got %q", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Name,Category") {
		t.Error("CSV missing header row")
	}
	if !strings.Contains(body, "Tent") || !strings.Contains(body, "Sleeping Bag") {
		t.Error("CSV missing expected item names")
	}
}
