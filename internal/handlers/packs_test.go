package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"carryless/internal/database"
	"carryless/internal/models"
)


func TestHandlePacks(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	w := doJSONRequest(router, http.MethodGet, "/packs", "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON content-type, got %s", ct)
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode JSON:", err)
	}
}

func TestHandlePacks_Unauthenticated(t *testing.T) {
	router, _, _ := setupHandlerTest(t)

	w := doJSONRequest(router, http.MethodGet, "/packs", "", "")

	if w.Code != http.StatusFound && w.Code != http.StatusUnauthorized {
		t.Errorf("expected redirect or 401, got %d", w.Code)
	}
}

func TestHandleCreatePack(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	form := url.Values{
		"name": {"My Test Pack"},
		"note": {"A test note"},
	}
	w := doRequest(router, http.MethodPost, "/packs",
		form.Encode(), "application/x-www-form-urlencoded", cookie)

	if w.Code != http.StatusFound {
		t.Errorf("expected redirect after create, got %d: %s", w.Code, w.Body.String())
	}

	// Verify pack was created
	var count int
	db.QueryRow("SELECT COUNT(*) FROM packs WHERE user_id = ? AND name = 'My Test Pack'", user.ID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 pack, got %d", count)
	}
}

func TestHandlePackDetail(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	// Create pack via POST
	form := url.Values{"name": {"Detail Pack"}}
	doRequest(router, http.MethodPost, "/packs",
		form.Encode(), "application/x-www-form-urlencoded", cookie)

	// Get the pack ID
	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ? AND name = 'Detail Pack'", user.ID).Scan(&packID)

	w := doJSONRequest(router, http.MethodGet, "/packs/"+packID, "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode JSON:", err)
	}
}

func TestHandlePackDetail_NotFound(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	w := doJSONRequest(router, http.MethodGet, "/packs/nonexistent-pack-id", "", cookie)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent pack, got %d", w.Code)
	}
}

func TestHandlePackDetail_OtherUserPack(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	adminUser := createAdminUser(t, db)
	regularUser := createRegularUser(t, db)

	adminCookie := loginAndGetCookie(t, router, adminUser.Email, "password123")
	regularCookie := loginAndGetCookie(t, router, regularUser.Email, "password123")

	// Admin creates a private pack
	form := url.Values{"name": {"Admin Private Pack"}}
	doRequest(router, http.MethodPost, "/packs",
		form.Encode(), "application/x-www-form-urlencoded", adminCookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ? AND name = 'Admin Private Pack'", adminUser.ID).Scan(&packID)

	// Regular user tries to access admin's pack
	w := doJSONRequest(router, http.MethodGet, "/packs/"+packID, "", regularCookie)

	if w.Code != http.StatusNotFound && w.Code != http.StatusForbidden {
		t.Errorf("expected 404 or 403 for another user's pack, got %d", w.Code)
	}
}

func TestHandleUpdatePack(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	// Create pack
	form := url.Values{"name": {"Original Name"}}
	doRequest(router, http.MethodPost, "/packs",
		form.Encode(), "application/x-www-form-urlencoded", cookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ? AND name = 'Original Name'", user.ID).Scan(&packID)

	// Update pack
	updateForm := url.Values{"name": {"Updated Name"}, "note": {"Updated note"}}
	w := doRequest(router, http.MethodPost, "/packs/"+packID,
		updateForm.Encode(), "application/x-www-form-urlencoded", cookie)

	if w.Code != http.StatusFound {
		t.Errorf("expected redirect after update, got %d: %s", w.Code, w.Body.String())
	}

	var name string
	db.QueryRow("SELECT name FROM packs WHERE id = ?", packID).Scan(&name)
	if name != "Updated Name" {
		t.Errorf("pack name not updated, got %q", name)
	}
}

func TestHandleDeletePack(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	form := url.Values{"name": {"ToDelete"}}
	doRequest(router, http.MethodPost, "/packs",
		form.Encode(), "application/x-www-form-urlencoded", cookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ? AND name = 'ToDelete'", user.ID).Scan(&packID)

	w := doRequest(router, http.MethodPost, "/packs/"+packID+"/delete",
		"", "application/x-www-form-urlencoded", cookie)

	if w.Code != http.StatusFound {
		t.Errorf("expected redirect after delete, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM packs WHERE id = ?", packID).Scan(&count)
	if count != 0 {
		t.Errorf("pack still exists after delete")
	}
}

func TestHandleDuplicatePack(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	form := url.Values{"name": {"Original Pack"}}
	doRequest(router, http.MethodPost, "/packs",
		form.Encode(), "application/x-www-form-urlencoded", cookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ? AND name = 'Original Pack'", user.ID).Scan(&packID)

	w := doRequest(router, http.MethodPost, "/packs/"+packID+"/duplicate",
		"", "application/x-www-form-urlencoded", cookie)

	if w.Code != http.StatusFound {
		t.Errorf("expected redirect after duplicate, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM packs WHERE user_id = ?", user.ID).Scan(&count)
	if count < 2 {
		t.Errorf("expected at least 2 packs after duplicate, got %d", count)
	}
}

func TestHandleAddItemToPack(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	// Create category + item + pack in DB directly
	cat, _ := database.CreateCategory(db, user.ID, "Test Category")
	item, _ := database.CreateItem(db, user.ID, models.Item{CategoryID: cat.ID, Name: "Test Item", WeightGrams: 500})

	packForm := url.Values{"name": {"Test Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", cookie)
	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", user.ID).Scan(&packID)

	// Add item to pack
	form := url.Values{"item_id": {fmt.Sprintf("%d", item.ID)}}
	w := doJSONRequest(router, http.MethodPost, "/packs/"+packID+"/items",
		form.Encode(), cookie)

	if w.Code != http.StatusOK && w.Code != http.StatusCreated && w.Code != http.StatusFound {
		t.Errorf("expected success adding item to pack, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleRemoveItemFromPack(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	cat, _ := database.CreateCategory(db, user.ID, "Category")
	item, _ := database.CreateItem(db, user.ID, models.Item{CategoryID: cat.ID, Name: "Item", WeightGrams: 100})

	packForm := url.Values{"name": {"Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", cookie)
	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", user.ID).Scan(&packID)

	// Add item
	addForm := url.Values{"item_id": {fmt.Sprintf("%d", item.ID)}}
	doJSONRequest(router, http.MethodPost, "/packs/"+packID+"/items", addForm.Encode(), cookie)

	// Remove item
	w := doJSONRequest(router, http.MethodDelete,
		fmt.Sprintf("/packs/%s/items/%d", packID, item.ID), "", cookie)

	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("expected success removing item, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleTogglePackLock(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	packForm := url.Values{"name": {"Lockable Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", cookie)
	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", user.ID).Scan(&packID)

	w := doJSONRequest(router, http.MethodPost, "/packs/"+packID+"/lock", "", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 after toggle lock, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandlePackLabels(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	packForm := url.Values{"name": {"Labeled Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", cookie)
	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", user.ID).Scan(&packID)

	// Create label
	labelForm := url.Values{"name": {"Worn"}, "color": {"#ff0000"}}
	wCreate := doJSONRequest(router, http.MethodPost,
		"/packs/"+packID+"/labels", labelForm.Encode(), cookie)
	if wCreate.Code != http.StatusOK && wCreate.Code != http.StatusCreated {
		t.Fatalf("expected success creating label, got %d: %s", wCreate.Code, wCreate.Body.String())
	}

	var labelID int
	db.QueryRow("SELECT id FROM pack_labels WHERE pack_id = ?", packID).Scan(&labelID)
	if labelID == 0 {
		t.Fatal("no label created")
	}

	// Update label
	updateForm := url.Values{"name": {"Worn+Updated"}, "color": {"#00ff00"}}
	wUpdate := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/packs/%s/labels/%d", packID, labelID), updateForm.Encode(), cookie)
	if wUpdate.Code != http.StatusOK {
		t.Errorf("expected 200 updating label, got %d: %s", wUpdate.Code, wUpdate.Body.String())
	}

	// Delete label
	wDelete := doJSONRequest(router, http.MethodDelete,
		fmt.Sprintf("/packs/%s/labels/%d", packID, labelID), "", cookie)
	if wDelete.Code != http.StatusOK && wDelete.Code != http.StatusNoContent {
		t.Errorf("expected success deleting label, got %d: %s", wDelete.Code, wDelete.Body.String())
	}
}
