package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestHandleAccountPage(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	// Account page currently returns HTML (not JSON-capable yet)
	w := doRequest(router, http.MethodGet, "/account", "", "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleChangePassword(t *testing.T) {
	tests := []struct {
		name            string
		currentPassword string
		newPassword     string
		confirmPassword string
		wantStatus      int
	}{
		{
			name:            "valid password change",
			currentPassword: "password123",
			newPassword:     "newpassword456",
			confirmPassword: "newpassword456",
			wantStatus:      http.StatusOK,
		},
		{
			name:            "wrong current password",
			currentPassword: "wrongpassword",
			newPassword:     "newpassword456",
			confirmPassword: "newpassword456",
			wantStatus:      http.StatusBadRequest,
		},
		{
			name:            "passwords do not match",
			currentPassword: "password123",
			newPassword:     "newpassword456",
			confirmPassword: "different789",
			wantStatus:      http.StatusBadRequest,
		},
		{
			name:            "new password too short",
			currentPassword: "password123",
			newPassword:     "short",
			confirmPassword: "short",
			wantStatus:      http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, db, _ := setupHandlerTest(t)
			user := createRegularUser(t, db)
			cookie := loginAndGetCookie(t, router, user.Email, "password123")

			form := url.Values{
				"current_password": {tc.currentPassword},
				"new_password":     {tc.newPassword},
				"confirm_password": {tc.confirmPassword},
			}
			w := doRequest(router, http.MethodPost, "/account/password",
				form.Encode(), "application/x-www-form-urlencoded", cookie)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

func TestHandleChangeCurrency(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createRegularUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	form := url.Values{"currency": {"€"}}
	w := doRequest(router, http.MethodPost, "/account/currency",
		form.Encode(), "application/x-www-form-urlencoded", cookie)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 after currency change, got %d: %s", w.Code, w.Body.String())
	}

	var currency string
	db.QueryRow("SELECT COALESCE(currency, '$') FROM users WHERE id = ?", user.ID).Scan(&currency)
	if currency != "€" {
		t.Errorf("expected currency '€', got %q", currency)
	}
}

func TestHandleChangeUsername(t *testing.T) {
	tests := []struct {
		name        string
		newUsername string
		wantStatus  int
	}{
		{
			name:        "valid username change",
			newUsername: "newusername",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "username too short",
			newUsername: "a",
			wantStatus:  http.StatusBadRequest,
		},
		{
			name:        "username too long",
			newUsername: strings.Repeat("a", 51),
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, db, _ := setupHandlerTest(t)
			user := createRegularUser(t, db)
			cookie := loginAndGetCookie(t, router, user.Email, "password123")

			form := url.Values{"username": {tc.newUsername}}
			w := doRequest(router, http.MethodPost, "/account/username",
				form.Encode(), "application/x-www-form-urlencoded", cookie)

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}
