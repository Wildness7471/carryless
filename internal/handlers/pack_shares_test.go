package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestHandlePackSharesPage(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	owner := createAdminUser(t, db)
	cookie := loginAndGetCookie(t, router, owner.Email, "password123")

	// Create a pack
	packForm := url.Values{"name": {"Shared Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", cookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", owner.ID).Scan(&packID)

	w := doRequest(router, http.MethodGet, "/packs/"+packID+"/shares", "", "", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleCreatePackShare(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	owner := createAdminUser(t, db)
	sharee := createRegularUser(t, db)
	ownerCookie := loginAndGetCookie(t, router, owner.Email, "password123")

	// Owner creates pack
	packForm := url.Values{"name": {"Owner Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", ownerCookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", owner.ID).Scan(&packID)

	// Share with sharee (handler expects username_or_email, not user_id)
	shareForm := url.Values{
		"username_or_email": {sharee.Email},
		"permission":        {"view"},
	}
	w := doJSONRequest(router, http.MethodPost, "/packs/"+packID+"/shares",
		shareForm.Encode(), ownerCookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 creating share, got %d: %s", w.Code, w.Body.String())
	}

	// Verify share exists
	var count int
	db.QueryRow("SELECT COUNT(*) FROM pack_shares WHERE pack_id = ? AND shared_with_user_id = ?",
		packID, sharee.ID).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 share, got %d", count)
	}
}

func TestHandleUpdatePackShare(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	owner := createAdminUser(t, db)
	sharee := createRegularUser(t, db)
	ownerCookie := loginAndGetCookie(t, router, owner.Email, "password123")

	packForm := url.Values{"name": {"Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", ownerCookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", owner.ID).Scan(&packID)

	// Share with view permission
	shareForm := url.Values{"username_or_email": {sharee.Email}, "permission": {"view"}}
	doJSONRequest(router, http.MethodPost, "/packs/"+packID+"/shares", shareForm.Encode(), ownerCookie)

	// Update to edit permission
	updateForm := url.Values{"permission": {"edit"}}
	w := doJSONRequest(router, http.MethodPost,
		fmt.Sprintf("/packs/%s/shares/%d", packID, sharee.ID),
		updateForm.Encode(), ownerCookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 updating share, got %d: %s", w.Code, w.Body.String())
	}

	var perm string
	db.QueryRow("SELECT permission FROM pack_shares WHERE pack_id = ? AND shared_with_user_id = ?",
		packID, sharee.ID).Scan(&perm)
	if perm != "edit" {
		t.Errorf("expected permission 'edit', got %q", perm)
	}
}

func TestHandleRevokePackShare(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	owner := createAdminUser(t, db)
	sharee := createRegularUser(t, db)
	ownerCookie := loginAndGetCookie(t, router, owner.Email, "password123")

	packForm := url.Values{"name": {"Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", ownerCookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", owner.ID).Scan(&packID)

	shareForm := url.Values{"username_or_email": {sharee.Email}, "permission": {"view"}}
	doJSONRequest(router, http.MethodPost, "/packs/"+packID+"/shares", shareForm.Encode(), ownerCookie)

	// Revoke
	w := doJSONRequest(router, http.MethodDelete,
		fmt.Sprintf("/packs/%s/shares/%d", packID, sharee.ID), "", ownerCookie)

	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("expected success revoking share, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM pack_shares WHERE pack_id = ? AND shared_with_user_id = ?",
		packID, sharee.ID).Scan(&count)
	if count != 0 {
		t.Errorf("share still exists after revoke")
	}
}

func TestHandleCreateInviteLink(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	owner := createAdminUser(t, db)
	cookie := loginAndGetCookie(t, router, owner.Email, "password123")

	packForm := url.Values{"name": {"Pack"}}
	doRequest(router, http.MethodPost, "/packs", packForm.Encode(), "application/x-www-form-urlencoded", cookie)

	var packID string
	db.QueryRow("SELECT id FROM packs WHERE user_id = ?", owner.ID).Scan(&packID)

	inviteForm := url.Values{"permission": {"view"}}
	w := doJSONRequest(router, http.MethodPost, "/packs/"+packID+"/invites",
		inviteForm.Encode(), cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 creating invite, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode JSON:", err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty invite token")
	}
}

func TestHandleUserSearch(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	createAdminUser(t, db)
	createNamedUser(t, db, "hiker123", "hiker123@example.com", "password123")
	cookie := loginAndGetCookie(t, router, "admin@example.com", "password123")

	w := doJSONRequest(router, http.MethodGet, "/api/users/search?q=hiker", "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode JSON:", err)
	}
	users, ok := resp["users"].([]interface{})
	if !ok {
		t.Fatalf("expected 'users' array in response, got %T: %v", resp["users"], resp)
	}
	if len(users) == 0 {
		t.Error("expected at least 1 user matching 'hiker'")
	}
	// Verify hiker123 is in results
	found := false
	for _, u := range users {
		if userMap, ok := u.(map[string]interface{}); ok {
			if strings.Contains(fmt.Sprintf("%v", userMap["username"]), "hiker") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("hiker123 not found in search results: %v", users)
	}
}
