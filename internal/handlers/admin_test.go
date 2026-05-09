package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

func TestHandleAdminPanel_AsAdmin(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	admin := createAdminUser(t, db)
	cookie := loginAndGetCookie(t, router, admin.Email, "password123")

	w := doRequest(router, http.MethodGet, "/admin/", "", "", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for admin panel, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleAdminPanel_AsRegularUser(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	createAdminUser(t, db)
	regular := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, regular.Email, "password123")

	w := doRequest(router, http.MethodGet, "/admin/", "", "", cookie)

	if w.Code != http.StatusFound && w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Errorf("expected redirect or 403/401 for regular user accessing admin, got %d", w.Code)
	}
}

func TestHandleToggleUserAdmin(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	admin := createAdminUser(t, db)
	target := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, admin.Email, "password123")

	w := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/admin/users/%d/toggle-admin", target.ID), "", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode JSON:", err)
	}
}

func TestHandleToggleUserActivation(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	admin := createAdminUser(t, db)
	target := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, admin.Email, "password123")

	w := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/admin/users/%d/toggle-activation", target.ID), "", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleBanUser(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	admin := createAdminUser(t, db)
	target := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, admin.Email, "password123")

	w := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/admin/users/%d/ban", target.ID), "", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 after ban, got %d: %s", w.Code, w.Body.String())
	}

	// Verify user is deleted
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", target.ID).Scan(&count)
	if count != 0 {
		t.Errorf("expected user to be deleted after ban, but still exists")
	}
}

func TestHandleToggleRegistration(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	admin := createAdminUser(t, db)
	cookie := loginAndGetCookie(t, router, admin.Email, "password123")

	// Toggle registration (should flip from true to false)
	w := doJSONRequest(router, http.MethodPost, "/admin/toggle-registration", "", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode JSON:", err)
	}
}
