package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"carryless/internal/database"
)

func TestHandleCreateSubItem(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "subuser", "sub@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Tent", 1200, catID))

	r := newRouter(db, user)
	r.POST("/api/items/:id/sub-items", handleCreateSubItem)

	// Create sub-item
	w := req(t, r, "POST", "/api/items/"+istr(item.ID)+"/sub-items",
		url.Values{"name": {"Footprint"}})
	if w.Code != http.StatusOK {
		t.Fatalf("create sub-item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if resp["name"] != "Footprint" {
		t.Errorf("expected name 'Footprint', got %v", resp["name"])
	}
	if resp["id"] == nil {
		t.Error("response missing id")
	}

	// Empty name → 400
	w = req(t, r, "POST", "/api/items/"+istr(item.ID)+"/sub-items",
		url.Values{"name": {""}})
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty name: expected 400, got %d", w.Code)
	}

	// Other user's item → 404
	other := testActivatedUser(t, db, "subother", "subother@example.com")
	otherCatID := ensureCategory(t, db, other.ID)
	otherItem, _ := database.CreateItem(db, other.ID, newItemWithCat("Other", 100, otherCatID))
	w = req(t, r, "POST", "/api/items/"+istr(otherItem.ID)+"/sub-items",
		url.Values{"name": {"Hack"}})
	if w.Code != http.StatusNotFound {
		t.Errorf("other user's item: expected 404, got %d", w.Code)
	}
}

func TestHandleUpdateSubItem(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "subupduser", "subupd@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Pack", 800, catID))
	sub, _ := database.CreateSubItem(db, item.ID, "Hip Belt")

	r := newRouter(db, user)
	r.PUT("/api/items/:id/sub-items/:sub_id", handleUpdateSubItem)

	w := req(t, r, "PUT", "/api/items/"+istr(item.ID)+"/sub-items/"+istr(sub.ID),
		url.Values{"name": {"Hip Belt Pocket"}})
	if w.Code != http.StatusOK {
		t.Fatalf("update sub-item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] == nil {
		t.Errorf("expected message in response, got %v", resp)
	}

	// Verify the update was actually persisted
	subs, _ := database.GetSubItems(db, item.ID)
	if len(subs) == 0 || subs[0].Name != "Hip Belt Pocket" {
		t.Errorf("expected sub-item name 'Hip Belt Pocket' in DB, got %v", subs)
	}

	// Empty name → 400
	w = req(t, r, "PUT", "/api/items/"+istr(item.ID)+"/sub-items/"+istr(sub.ID),
		url.Values{"name": {""}})
	if w.Code != http.StatusBadRequest {
		t.Errorf("empty name: expected 400, got %d", w.Code)
	}

	// Nonexistent sub-item → error
	w = req(t, r, "PUT", "/api/items/"+istr(item.ID)+"/sub-items/99999",
		url.Values{"name": {"x"}})
	if w.Code == http.StatusOK {
		t.Error("nonexistent sub-item: should not return 200")
	}
}

func TestHandleDeleteSubItem(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "subdeluser", "subdel@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Jacket", 300, catID))
	sub, _ := database.CreateSubItem(db, item.ID, "Hood")

	r := newRouter(db, user)
	r.DELETE("/api/items/:id/sub-items/:sub_id", handleDeleteSubItem)

	w := req(t, r, "DELETE", "/api/items/"+istr(item.ID)+"/sub-items/"+istr(sub.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete sub-item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	subs, _ := database.GetSubItems(db, item.ID)
	if len(subs) != 0 {
		t.Errorf("expected 0 sub-items after delete, got %d", len(subs))
	}

	// Delete other user's sub-item → error
	other := testActivatedUser(t, db, "subdelother", "subdelother@example.com")
	otherCatID := ensureCategory(t, db, other.ID)
	otherItem, _ := database.CreateItem(db, other.ID, newItemWithCat("Other", 100, otherCatID))
	otherSub, _ := database.CreateSubItem(db, otherItem.ID, "Other Sub")

	w = req(t, r, "DELETE", "/api/items/"+istr(otherItem.ID)+"/sub-items/"+istr(otherSub.ID), nil)
	if w.Code == http.StatusOK {
		t.Error("deleting another user's sub-item should fail")
	}
}

func TestHandleToggleSubItemCheck(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "subtoggleuser", "subtoggle@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Sleeping Bag", 900, catID))
	sub, _ := database.CreateSubItem(db, item.ID, "Compression Sack")
	pack, _ := database.CreatePack(db, user.ID, "Trip Pack")
	database.AddItemToPack(db, pack.ID, item.ID, user.ID)

	r := newRouter(db, user)
	r.POST("/packs/:id/items/:item_id/sub-items/:sub_id/toggle", handleToggleSubItemCheck)

	// Toggle on
	w := req(t, r, "POST",
		"/packs/"+pack.ID+"/items/"+istr(item.ID)+"/sub-items/"+istr(sub.ID)+"/toggle",
		url.Values{"is_checked": {"true"}})
	if w.Code != http.StatusOK {
		t.Fatalf("toggle sub-item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["is_checked"] != true {
		t.Errorf("expected is_checked=true, got %v", resp["is_checked"])
	}

	// Toggle off
	w = req(t, r, "POST",
		"/packs/"+pack.ID+"/items/"+istr(item.ID)+"/sub-items/"+istr(sub.ID)+"/toggle",
		url.Values{"is_checked": {"false"}})
	if w.Code != http.StatusOK {
		t.Fatalf("toggle off: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["is_checked"] != false {
		t.Errorf("expected is_checked=false, got %v", resp["is_checked"])
	}

	// View-only user CAN toggle (checklist state is shared; view perm is sufficient)
	viewer := testActivatedUser(t, db, "subviewer", "subviewer@example.com")
	database.CreatePackShare(db, pack.ID, user.ID, viewer.ID, "view")

	rViewer := newRouter(db, viewer)
	rViewer.POST("/packs/:id/items/:item_id/sub-items/:sub_id/toggle", handleToggleSubItemCheck)
	w = req(t, rViewer, "POST",
		"/packs/"+pack.ID+"/items/"+istr(item.ID)+"/sub-items/"+istr(sub.ID)+"/toggle",
		url.Values{"is_checked": {"true"}})
	if w.Code != http.StatusOK {
		t.Errorf("view-only user toggle: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// User with no access cannot toggle (perm == "none")
	noAccess := testActivatedUser(t, db, "subnoaccess", "subnoaccess@example.com")
	rNoAccess := newRouter(db, noAccess)
	rNoAccess.POST("/packs/:id/items/:item_id/sub-items/:sub_id/toggle", handleToggleSubItemCheck)
	w = req(t, rNoAccess, "POST",
		"/packs/"+pack.ID+"/items/"+istr(item.ID)+"/sub-items/"+istr(sub.ID)+"/toggle",
		url.Values{"is_checked": {"true"}})
	if w.Code != http.StatusForbidden {
		t.Errorf("no-access user toggle: expected 403, got %d: %s", w.Code, w.Body.String())
	}

	// Pack that doesn't exist → 403 (pack == nil triggers forbidden)
	w = req(t, r, "POST",
		"/packs/nonexistent/items/"+istr(item.ID)+"/sub-items/"+istr(sub.ID)+"/toggle",
		url.Values{"is_checked": {"true"}})
	if w.Code != http.StatusForbidden {
		t.Errorf("nonexistent pack: expected 403, got %d", w.Code)
	}
}

func TestSubItemCRUDOwnershipBoundary(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	user := testActivatedUser(t, db, "subboundary", "subboundary@example.com")
	other := testActivatedUser(t, db, "subboundaryother", "subboundaryother@example.com")

	catID := ensureCategory(t, db, user.ID)
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Item", 100, catID))

	r := newRouter(db, other)
	r.POST("/api/items/:id/sub-items", handleCreateSubItem)
	r.DELETE("/api/items/:id/sub-items/:sub_id", handleDeleteSubItem)

	// Other user cannot create sub-item on user's item
	w := req(t, r, "POST", "/api/items/"+istr(item.ID)+"/sub-items",
		url.Values{"name": {"Attack"}})
	if w.Code != http.StatusNotFound {
		t.Errorf("other user create sub-item: expected 404, got %d", w.Code)
	}
}
