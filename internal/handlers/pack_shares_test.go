package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"carryless/internal/database"
)

func TestHandleCreatePackShare(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	owner := testActivatedUser(t, db, "shareowner", "shareowner@example.com")
	target := testActivatedUser(t, db, "sharetarget", "sharetarget@example.com")

	pack, _ := database.CreatePack(db, owner.ID, "My Pack")

	r := newRouter(db, owner)
	r.POST("/packs/:id/shares", handleCreatePackShare)

	// Share with existing user by email
	w := req(t, r, "POST", "/packs/"+pack.ID+"/shares", url.Values{
		"username_or_email": {target.Email},
		"permission":        {"view"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create share: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	shares, _ := database.GetPackShares(db, pack.ID)
	if len(shares) != 1 {
		t.Fatalf("expected 1 share, got %d", len(shares))
	}
	if shares[0].SharedWithUserID != target.ID {
		t.Errorf("share target mismatch")
	}
	if shares[0].Permission != "view" {
		t.Errorf("expected 'view' permission, got %q", shares[0].Permission)
	}

	// Share with nonexistent user → error
	w = req(t, r, "POST", "/packs/"+pack.ID+"/shares", url.Values{
		"username_or_email": {"nobody@nowhere.com"},
		"permission":        {"view"},
	})
	if w.Code == http.StatusOK {
		t.Error("sharing with nonexistent user should fail")
	}

	// Share self → should be prevented
	w = req(t, r, "POST", "/packs/"+pack.ID+"/shares", url.Values{
		"username_or_email": {owner.Email},
		"permission":        {"view"},
	})
	if w.Code == http.StatusOK {
		t.Error("owner sharing pack with themselves should fail")
	}

	// Non-owner cannot share
	r2 := newRouter(db, target)
	r2.POST("/packs/:id/shares", handleCreatePackShare)
	w = req(t, r2, "POST", "/packs/"+pack.ID+"/shares", url.Values{
		"username_or_email": {owner.Email},
		"permission":        {"view"},
	})
	if w.Code != http.StatusForbidden && w.Code != http.StatusNotFound {
		t.Errorf("non-owner share: expected 403/404, got %d", w.Code)
	}
}

func TestHandleUpdatePackShare(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	owner := testActivatedUser(t, db, "updshareowner", "updshareowner@example.com")
	target := testActivatedUser(t, db, "updsharetarget", "updsharetarget@example.com")

	pack, _ := database.CreatePack(db, owner.ID, "Share Pack")
	database.CreatePackShare(db, pack.ID, owner.ID, target.ID, "view")

	r := newRouter(db, owner)
	r.POST("/packs/:id/shares/:user_id", handleUpdatePackShare)

	w := req(t, r, "POST", "/packs/"+pack.ID+"/shares/"+istr(target.ID),
		url.Values{"permission": {"edit"}})
	if w.Code != http.StatusOK {
		t.Fatalf("update share: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	shares, _ := database.GetPackShares(db, pack.ID)
	if len(shares) != 1 || shares[0].Permission != "edit" {
		t.Errorf("expected 'edit' permission after update, got %v", shares)
	}

	// Invalid permission value
	w = req(t, r, "POST", "/packs/"+pack.ID+"/shares/"+istr(target.ID),
		url.Values{"permission": {"superadmin"}})
	if w.Code == http.StatusOK {
		t.Error("invalid permission should be rejected")
	}
}

func TestHandleRevokePackShare(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	owner := testActivatedUser(t, db, "revokeowner", "revokeowner@example.com")
	target := testActivatedUser(t, db, "revoketarget", "revoketarget@example.com")

	pack, _ := database.CreatePack(db, owner.ID, "Revoke Pack")
	database.CreatePackShare(db, pack.ID, owner.ID, target.ID, "view")

	r := newRouter(db, owner)
	r.DELETE("/packs/:id/shares/:user_id", handleRevokePackShare)

	w := req(t, r, "DELETE", "/packs/"+pack.ID+"/shares/"+istr(target.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("revoke share: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	shares, _ := database.GetPackShares(db, pack.ID)
	if len(shares) != 0 {
		t.Errorf("expected 0 shares after revoke, got %d", len(shares))
	}

	// Revoking nonexistent share
	w = req(t, r, "DELETE", "/packs/"+pack.ID+"/shares/99999", nil)
	if w.Code == http.StatusOK {
		t.Error("revoking nonexistent share should fail")
	}
}

func TestHandleCreateInviteLink(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	owner := testActivatedUser(t, db, "inviteowner", "inviteowner@example.com")
	pack, _ := database.CreatePack(db, owner.ID, "Invite Pack")

	r := newRouter(db, owner)
	r.POST("/packs/:id/invites", handleCreateInviteLink)

	w := req(t, r, "POST", "/packs/"+pack.ID+"/invites",
		url.Values{"permission": {"view"}})
	if w.Code != http.StatusOK {
		t.Fatalf("create invite: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["token"] == nil {
		t.Errorf("invite response missing token, got: %s", w.Body.String())
	}
	if resp["permission"] == nil {
		t.Error("invite response missing permission")
	}

	invites, _ := database.GetPackInvites(db, pack.ID)
	if len(invites) != 1 {
		t.Errorf("expected 1 invite, got %d", len(invites))
	}
}

func TestHandleDeleteInviteLink(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	owner := testActivatedUser(t, db, "delinvowner", "delinvowner@example.com")
	pack, _ := database.CreatePack(db, owner.ID, "Del Invite Pack")

	invite, _ := database.CreatePackInvite(db, pack.ID, owner.ID, "view")

	r := newRouter(db, owner)
	r.DELETE("/packs/:id/invites/:invite_id", handleDeleteInviteLink)

	w := req(t, r, "DELETE", "/packs/"+pack.ID+"/invites/"+istr(invite.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("delete invite: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	invites, _ := database.GetPackInvites(db, pack.ID)
	if len(invites) != 0 {
		t.Errorf("expected 0 invites after delete, got %d", len(invites))
	}
}

func TestHandleUserSearch(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	searcher := testActivatedUser(t, db, "searcher", "searcher@example.com")
	testActivatedUser(t, db, "alice", "alice@example.com")
	testActivatedUser(t, db, "bob", "bob@example.com")

	r := newRouter(db, searcher)
	r.GET("/api/users/search", handleUserSearch)

	// Search by username prefix
	w := req(t, r, "GET", "/api/users/search?q=ali", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("user search: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "alice") {
		t.Errorf("expected 'alice' in search results, got: %s", body)
	}
	if strings.Contains(body, "bob") {
		t.Errorf("'bob' should not appear in 'ali' search, got: %s", body)
	}

	// Empty query → empty results or all users
	w = req(t, r, "GET", "/api/users/search?q=", nil)
	if w.Code != http.StatusOK {
		t.Errorf("empty search: expected 200, got %d", w.Code)
	}
}

func TestHandleRedeemInvite(t *testing.T) {
	db := setupHandlerTestDB(t)
	defer db.Close()

	owner := testActivatedUser(t, db, "redeemowner", "redeemowner@example.com")
	redeemer := testActivatedUser(t, db, "redeemer", "redeemer@example.com")

	pack, _ := database.CreatePack(db, owner.ID, "Redeem Pack")
	invite, _ := database.CreatePackInvite(db, pack.ID, owner.ID, "view")

	r := newRouter(db, redeemer)
	r.GET("/invite/:token", handleRedeemInvite)

	w := req(t, r, "GET", "/invite/"+invite.Token, nil)
	// Should redirect to the pack after redeeming
	if w.Code != http.StatusFound {
		t.Fatalf("redeem invite: expected 302, got %d: %s", w.Code, w.Body.String())
	}

	// Verify share was created
	shares, _ := database.GetPackShares(db, pack.ID)
	if len(shares) != 1 {
		t.Errorf("expected 1 share after redeem, got %d", len(shares))
	}
	if shares[0].SharedWithUserID != redeemer.ID {
		t.Error("share should be for redeemer")
	}

	// Redeem same token again → should fail or be idempotent
	w = req(t, r, "GET", "/invite/"+invite.Token, nil)
	if w.Code == http.StatusInternalServerError {
		t.Error("re-redeeming invite should not 500")
	}

	// Invalid token
	w = req(t, r, "GET", "/invite/invalid-token-xyz", nil)
	if w.Code == http.StatusFound {
		t.Error("invalid token should not redirect successfully")
	}
}
