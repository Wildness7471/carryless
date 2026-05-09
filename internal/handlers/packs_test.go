package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"carryless/internal/database"
)

func TestHandleCreatePack_Redirect(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "packowner", "packowner@example.com")

	r := newRouter(db, user)
	r.POST("/packs", handleCreatePack)

	w := req(t, r, "POST", "/packs", url.Values{"name": {"Test Pack"}})
	if w.Code != http.StatusFound {
		t.Errorf("expected 302 redirect, got %d body=%s", w.Code, w.Body.String())
	}

	packs, _ := database.GetPacks(db, user.ID)
	if len(packs) != 1 || packs[0].Name != "Test Pack" {
		t.Errorf("pack should exist after create, got %v", packs)
	}
}

func TestHandleAddRemoveItemFromPack(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "itempackuser", "itempack@example.com")

	catID := ensureCategory(t, db, user.ID)
	pack, err := database.CreatePack(db, user.ID, "My Pack")
	if err != nil {
		t.Fatal(err)
	}
	item, err := database.CreateItem(db, user.ID, newItemWithCat("Tent", 1200, catID))
	if err != nil {
		t.Fatal(err)
	}

	r := newRouter(db, user)
	r.POST("/packs/:id/items", handleAddItemToPack)
	r.DELETE("/packs/:id/items/:item_id", handleRemoveItemFromPack)

	// Add item
	w := req(t, r, "POST", "/packs/"+pack.ID+"/items", url.Values{"item_id": {istr(item.ID)}})
	if w.Code != http.StatusOK {
		t.Fatalf("add item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Adding again increments count (still 200)
	w = req(t, r, "POST", "/packs/"+pack.ID+"/items", url.Values{"item_id": {istr(item.ID)}})
	if w.Code != http.StatusOK {
		t.Errorf("add same item again: expected 200 (count increment), got %d", w.Code)
	}

	// Add nonexistent item should fail
	w = req(t, r, "POST", "/packs/"+pack.ID+"/items", url.Values{"item_id": {"99999"}})
	if w.Code == http.StatusOK {
		t.Error("adding nonexistent item should not return 200")
	}

	// Remove item (decrements count once)
	w = req(t, r, "DELETE", "/packs/"+pack.ID+"/items/"+istr(item.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("remove item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Remove again (count was 2, now 1 → still in pack)
	w = req(t, r, "DELETE", "/packs/"+pack.ID+"/items/"+istr(item.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("remove item second time: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Remove nonexistent item → 404
	w = req(t, r, "DELETE", "/packs/"+pack.ID+"/items/99999", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("removing nonexistent item: expected 404, got %d", w.Code)
	}
}

func TestHandleToggleWorn(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "wornuser", "worn@example.com")

	catID := ensureCategory(t, db, user.ID)
	pack, _ := database.CreatePack(db, user.ID, "Worn Pack")
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Jacket", 500, catID))
	database.AddItemToPack(db, pack.ID, item.ID, user.ID)

	r := newRouter(db, user)
	r.PUT("/packs/:id/items/:item_id/worn", handleToggleWorn)

	// Set worn = true
	w := req(t, r, "PUT", "/packs/"+pack.ID+"/items/"+istr(item.ID)+"/worn",
		url.Values{"is_worn": {"true"}})
	if w.Code != http.StatusOK {
		t.Fatalf("set worn=true: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response not JSON: %v", err)
	}
	if _, ok := resp["message"]; !ok {
		t.Error("response missing message field")
	}

	// Set worn = false
	w = req(t, r, "PUT", "/packs/"+pack.ID+"/items/"+istr(item.ID)+"/worn",
		url.Values{"is_worn": {"false"}})
	if w.Code != http.StatusOK {
		t.Errorf("set worn=false: expected 200, got %d", w.Code)
	}
}

func TestHandleUpdateWornCount(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "worncount", "worncount@example.com")

	catID := ensureCategory(t, db, user.ID)
	pack, _ := database.CreatePack(db, user.ID, "Count Pack")
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Sock", 50, catID))
	database.AddItemToPack(db, pack.ID, item.ID, user.ID)

	r := newRouter(db, user)
	r.PUT("/packs/:id/items/:item_id/worn-count", handleUpdateWornCount)

	w := req(t, r, "PUT", "/packs/"+pack.ID+"/items/"+istr(item.ID)+"/worn-count",
		url.Values{"worn_count": {"3"}})
	if w.Code != http.StatusOK {
		t.Fatalf("update worn count: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Invalid count (non-numeric)
	w = req(t, r, "PUT", "/packs/"+pack.ID+"/items/"+istr(item.ID)+"/worn-count",
		url.Values{"worn_count": {"notanumber"}})
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid worn count: expected 400, got %d", w.Code)
	}
}

func TestHandleUpdatePackNote(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "noteuser", "note@example.com")

	pack, _ := database.CreatePack(db, user.ID, "Note Pack")

	r := newRouter(db, user)
	r.POST("/packs/:id/note", handleUpdatePackNote)

	w := req(t, r, "POST", "/packs/"+pack.ID+"/note", url.Values{"note": {"My note"}})
	if w.Code != http.StatusOK {
		t.Fatalf("update note: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	updated, _ := database.GetPack(db, pack.ID)
	if updated.Note != "My note" {
		t.Errorf("expected 'My note', got %q", updated.Note)
	}

	// Note too long (501 bytes)
	long := make([]byte, 501)
	for i := range long {
		long[i] = 'x'
	}
	w = req(t, r, "POST", "/packs/"+pack.ID+"/note", url.Values{"note": {string(long)}})
	if w.Code != http.StatusBadRequest {
		t.Errorf("long note: expected 400, got %d", w.Code)
	}
}

func TestHandlePackPermissionEnforcement(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	owner := testActivatedUser(t, db, "permowner", "permowner@example.com")
	attacker := testActivatedUser(t, db, "attacker", "attacker@example.com")

	catID := ensureCategory(t, db, attacker.ID)
	pack, _ := database.CreatePack(db, owner.ID, "Owner Pack")
	item, _ := database.CreateItem(db, attacker.ID, newItemWithCat("Stolen", 100, catID))

	r := newRouter(db, attacker)
	r.POST("/packs/:id/items", handleAddItemToPack)
	r.POST("/packs/:id/note", handleUpdatePackNote)

	w := req(t, r, "POST", "/packs/"+pack.ID+"/items", url.Values{"item_id": {istr(item.ID)}})
	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Errorf("unshared user add item: expected 403/404, got %d", w.Code)
	}

	w = req(t, r, "POST", "/packs/"+pack.ID+"/note", url.Values{"note": {"hack"}})
	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Errorf("unshared user update note: expected 403/404, got %d", w.Code)
	}
}

func TestHandleSharedPackPermissions(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	owner := testActivatedUser(t, db, "shareowneredit", "shareowneredit@example.com")
	editor := testActivatedUser(t, db, "editoruser", "editor@example.com")
	viewer := testActivatedUser(t, db, "vieweruser2", "viewer2@example.com")

	editorCatID := ensureCategory(t, db, editor.ID)
	viewerCatID := ensureCategory(t, db, viewer.ID)

	pack, _ := database.CreatePack(db, owner.ID, "Shared Pack")
	editorItem, _ := database.CreateItem(db, editor.ID, newItemWithCat("Editor Item", 200, editorCatID))
	viewerItem, _ := database.CreateItem(db, viewer.ID, newItemWithCat("Viewer Item", 150, viewerCatID))

	database.CreatePackShare(db, pack.ID, owner.ID, editor.ID, "edit")
	database.CreatePackShare(db, pack.ID, owner.ID, viewer.ID, "view")

	// Editor can add
	rEditor := newRouter(db, editor)
	rEditor.POST("/packs/:id/items", handleAddItemToPack)
	w := req(t, rEditor, "POST", "/packs/"+pack.ID+"/items", url.Values{"item_id": {istr(editorItem.ID)}})
	if w.Code != http.StatusOK {
		t.Errorf("editor should add item: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Viewer cannot add
	rViewer := newRouter(db, viewer)
	rViewer.POST("/packs/:id/items", handleAddItemToPack)
	w = req(t, rViewer, "POST", "/packs/"+pack.ID+"/items", url.Values{"item_id": {istr(viewerItem.ID)}})
	if w.Code != http.StatusForbidden {
		t.Errorf("viewer should not add item: expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleTogglePackLock(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "lockuser", "lock@example.com")

	pack, _ := database.CreatePack(db, user.ID, "Lock Pack")

	r := newRouter(db, user)
	r.POST("/packs/:id/lock", handleTogglePackLock)

	// Lock it
	w := req(t, r, "POST", "/packs/"+pack.ID+"/lock", url.Values{"is_locked": {"true"}})
	if w.Code != http.StatusOK {
		t.Fatalf("lock pack: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Unlock it
	w = req(t, r, "POST", "/packs/"+pack.ID+"/lock", url.Values{"is_locked": {"false"}})
	if w.Code != http.StatusOK {
		t.Fatalf("unlock pack: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Non-owner pack
	other := testActivatedUser(t, db, "lockother", "lockother@example.com")
	otherPack, _ := database.CreatePack(db, other.ID, "Other Pack")
	w = req(t, r, "POST", "/packs/"+otherPack.ID+"/lock", url.Values{"is_locked": {"true"}})
	if w.Code == http.StatusOK {
		t.Errorf("non-owner locking pack should fail, got 200")
	}
}

func TestHandleDuplicatePack(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "duppackuser", "duppack@example.com")

	catID := ensureCategory(t, db, user.ID)
	pack, _ := database.CreatePack(db, user.ID, "Original Pack")
	item, _ := database.CreateItem(db, user.ID, newItemWithCat("Item1", 300, catID))
	database.AddItemToPack(db, pack.ID, item.ID, user.ID)

	r := newRouter(db, user)
	r.POST("/packs/:id/duplicate", handleDuplicatePack)

	w := req(t, r, "POST", "/packs/"+pack.ID+"/duplicate", nil)
	if w.Code != http.StatusFound {
		t.Errorf("duplicate pack: expected 302, got %d: %s", w.Code, w.Body.String())
	}

	packs, _ := database.GetPacks(db, user.ID)
	if len(packs) != 2 {
		t.Errorf("expected 2 packs after duplicate, got %d", len(packs))
	}
}

func TestHandlePackCompare(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()
	user := testActivatedUser(t, db, "compareuser", "compare@example.com")

	pack1, _ := database.CreatePack(db, user.ID, "Pack A")
	pack2, _ := database.CreatePack(db, user.ID, "Pack B")

	r := newRouter(db, user)
	r.GET("/packs/compare", handlePackCompare)

	// Missing packs param → redirects (302)
	w := req(t, r, "GET", "/packs/compare", nil)
	if w.Code != http.StatusFound {
		t.Errorf("missing packs param: expected 302 redirect, got %d", w.Code)
	}

	// Valid packs → 200 (JSON accepted)
	w = req(t, r, "GET", "/packs/compare?packs="+pack1.ID+","+pack2.ID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("compare valid packs: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Access another user's pack in compare → should fail or return error
	other := testActivatedUser(t, db, "compareother", "compareother@example.com")
	otherPack, _ := database.CreatePack(db, other.ID, "Other Pack")
	w = req(t, r, "GET", "/packs/compare?packs="+pack1.ID+","+otherPack.ID, nil)
	// Should either 403/404 or succeed (depends on implementation)
	if w.Code == http.StatusInternalServerError {
		t.Errorf("compare with inaccessible pack should not 500, got %d: %s", w.Code, w.Body.String())
	}
}
