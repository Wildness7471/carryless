package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"carryless/internal/database"
)

func TestHandleRegister(t *testing.T) {
	tests := []struct {
		name       string
		form       url.Values
		wantStatus int
		wantInBody string
	}{
		{
			name: "valid registration",
			form: url.Values{
				"username":         {"newuser"},
				"email":            {"newuser@example.com"},
				"password":         {"password123"},
				"confirm_password": {"password123"},
			},
			wantStatus: http.StatusOK,
			wantInBody: "Registration",
		},
		{
			name: "username too short",
			form: url.Values{
				"username":         {"ab"},
				"email":            {"ab@example.com"},
				"password":         {"password123"},
				"confirm_password": {"password123"},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "password too short",
			form: url.Values{
				"username":         {"validuser"},
				"email":            {"valid@example.com"},
				"password":         {"short"},
				"confirm_password": {"short"},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "passwords do not match",
			form: url.Values{
				"username":         {"validuser"},
				"email":            {"valid@example.com"},
				"password":         {"password123"},
				"confirm_password": {"different123"},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid email",
			form: url.Values{
				"username":         {"validuser"},
				"email":            {"notanemail"},
				"password":         {"password123"},
				"confirm_password": {"password123"},
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, _, _ := setupHandlerTest(t)

			w := doRequest(router, http.MethodPost, "/register",
				tc.form.Encode(), "application/x-www-form-urlencoded", "")

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			if tc.wantInBody != "" && !strings.Contains(w.Body.String(), tc.wantInBody) {
				t.Errorf("body does not contain %q; got: %s", tc.wantInBody, w.Body.String())
			}
		})
	}
}

func TestHandleRegister_DuplicateUser(t *testing.T) {
	router, db, _ := setupHandlerTest(t)

	// Create an existing user
	database.CreateUser(db, "existing", "existing@example.com", "password123")

	form := url.Values{
		"username":         {"existing"},
		"email":            {"existing@example.com"},
		"password":         {"password123"},
		"confirm_password": {"password123"},
	}
	w := doRequest(router, http.MethodPost, "/register",
		form.Encode(), "application/x-www-form-urlencoded", "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for duplicate user, got %d", w.Code)
	}
}

func TestHandleLogin(t *testing.T) {
	tests := []struct {
		name       string
		email      string
		password   string
		wantStatus int
		wantCookie bool
	}{
		{
			name:       "valid login",
			email:      "user@example.com",
			password:   "password123",
			wantStatus: http.StatusFound,
			wantCookie: true,
		},
		{
			name:       "wrong password",
			email:      "user@example.com",
			password:   "wrongpassword",
			wantStatus: http.StatusBadRequest,
			wantCookie: false,
		},
		{
			name:       "nonexistent email",
			email:      "nobody@example.com",
			password:   "password123",
			wantStatus: http.StatusBadRequest,
			wantCookie: false,
		},
		{
			name:       "empty credentials",
			email:      "",
			password:   "",
			wantStatus: http.StatusBadRequest,
			wantCookie: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router, db, _ := setupHandlerTest(t)
			createAdminUser(t, db) // creates "admin@example.com"
			// Create the target user
			database.CreateUser(db, "user", "user@example.com", "password123")

			form := url.Values{"email": {tc.email}, "password": {tc.password}}
			w := doRequest(router, http.MethodPost, "/login",
				form.Encode(), "application/x-www-form-urlencoded", "")

			if w.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d; body: %s", w.Code, tc.wantStatus, w.Body.String())
			}
			hasCookie := false
			for _, c := range w.Result().Cookies() {
				if c.Name == "session_id" && c.Value != "" {
					hasCookie = true
				}
			}
			if hasCookie != tc.wantCookie {
				t.Errorf("hasCookie = %v, want %v", hasCookie, tc.wantCookie)
			}
		})
	}
}

func TestHandleLogout(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createAdminUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	w := doRequest(router, http.MethodPost, "/logout", "", "", cookie)

	if w.Code != http.StatusFound {
		t.Errorf("expected redirect after logout, got %d", w.Code)
	}
	// Cookie should be cleared (MaxAge < 0 or empty value)
	for _, c := range w.Result().Cookies() {
		if c.Name == "session_id" && c.MaxAge > 0 {
			t.Error("expected session_id cookie to be cleared after logout")
		}
	}
}

func TestHandleCSRFToken(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createAdminUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	w := doJSONRequest(router, http.MethodGet, "/api/csrf-token", "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode response:", err)
	}
	if resp["token"] == "" {
		t.Error("expected non-empty CSRF token in response")
	}
}

func TestHandleCSRFToken_Unauthenticated(t *testing.T) {
	router, _, _ := setupHandlerTest(t)

	w := doJSONRequest(router, http.MethodGet, "/api/csrf-token", "", "")

	// Should redirect to login or return 401
	if w.Code != http.StatusFound && w.Code != http.StatusUnauthorized {
		t.Errorf("expected redirect or 401, got %d", w.Code)
	}
}

func TestHandleCreateBearerToken(t *testing.T) {
	router, db, _ := setupHandlerTest(t)
	user := createAdminUser(t, db)
	cookie := loginAndGetCookie(t, router, user.Email, "password123")

	w := doJSONRequest(router, http.MethodPost, "/api/auth/token", "", cookie)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal("failed to decode response:", err)
	}
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty bearer token in response")
	}
}
