package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"carryless/internal/database"
	"carryless/internal/models"
)

func TestHandleCreateSubItem(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	cat, _ := database.CreateCategory(db, user.ID, "Gear")
	item, _ := database.CreateItem(db, user.ID, models.Item{CategoryID: cat.ID, Name: "Sleeping Bag", WeightGrams: 800})

	form := url.Values{"name": {"Stuff Sack"}}
	w := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/api/items/%d/sub-items", item.ID), form.Encode(), cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var sub map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&sub); err != nil {
		t.Fatal("failed to decode JSON:", err)
	}
	if sub["name"] != "Stuff Sack" {
		t.Errorf("expected name 'Stuff Sack', got %v", sub["name"])
	}
}

func TestHandleCreateSubItem_EmptyName(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	cat, _ := database.CreateCategory(db, user.ID, "Gear")
	item, _ := database.CreateItem(db, user.ID, models.Item{CategoryID: cat.ID, Name: "Item", WeightGrams: 100})

	form := url.Values{"name": {""}}
	w := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/api/items/%d/sub-items", item.ID), form.Encode(), cookie)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w.Code)
	}
}

func TestHandleUpdateSubItem(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	cat, _ := database.CreateCategory(db, user.ID, "Gear")
	item, _ := database.CreateItem(db, user.ID, models.Item{CategoryID: cat.ID, Name: "Tent", WeightGrams: 1200})
	sub, _ := database.CreateSubItem(db, item.ID, "Rainfly")

	form := url.Values{"name": {"Updated Rainfly"}}
	w := doJSONRequest(router, http.MethodPut,
		fmt.Sprintf("/api/items/%d/sub-items/%d", item.ID, sub.ID), form.Encode(), cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleDeleteSubItem(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	cat, _ := database.CreateCategory(db, user.ID, "Gear")
	item, _ := database.CreateItem(db, user.ID, models.Item{CategoryID: cat.ID, Name: "Pack", WeightGrams: 600})
	sub, _ := database.CreateSubItem(db, item.ID, "Hip Belt")

	w := doJSONRequest(router, http.MethodDelete,
		fmt.Sprintf("/api/items/%d/sub-items/%d", item.ID, sub.ID), "", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM item_sub_items WHERE id = ?", sub.ID).Scan(&count)
	if count != 0 {
		t.Errorf("sub-item still exists after delete")
	}
}

func TestHandleToggleSubItemCheck(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	cat, _ := database.CreateCategory(db, user.ID, "Gear")
	item, _ := database.CreateItem(db, user.ID, models.Item{CategoryID: cat.ID, Name: "Headlamp", WeightGrams: 150})
	sub, _ := database.CreateSubItem(db, item.ID, "Batteries")

	// Create pack
	packForm := url.Values{"name": {"Test Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", cookie)
	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", user.ID).Scan(&packID)

	// Add item to pack
	addForm := url.Values{"item_id": {fmt.Sprintf("%d", item.ID)}}
	doJSONRequest(router, http.MethodPost, "/packs/"+packID+"/items", addForm.Encode(), cookie)

	// Toggle sub-item check
	toggleForm := url.Values{"is_checked": {"true"}}
	w := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/packs/%s/items/%d/sub-items/%d/toggle", packID, item.ID, sub.ID),
		toggleForm.Encode(), cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode JSON:", err)
	}
	if resp["is_checked"] != true {
		t.Errorf("expected is_checked=true, got %v", resp["is_checked"])
	}
}
