package database

import (
	"testing"
	"time"
)

func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "findme", "findme@example.com", "pass123")
	if err != nil {
		t.Fatal(err)
	}

	found, err := GetUserByID(db, user.ID)
	if err != nil {
		t.Fatalf("GetUserByID failed: %v", err)
	}
	if found.Username != "findme" {
		t.Errorf("expected 'findme', got %q", found.Username)
	}

	_, err = GetUserByID(db, 99999)
	if err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestVerifyPassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "veryuser", "verify@example.com", "correcthorsebattery")
	if err != nil {
		t.Fatal(err)
	}

	if err := VerifyPassword(db, user.ID, "correcthorsebattery"); err != nil {
		t.Errorf("VerifyPassword with correct password failed: %v", err)
	}
	if err := VerifyPassword(db, user.ID, "wrongpassword"); err == nil {
		t.Error("expected error for wrong password")
	}
	if err := VerifyPassword(db, 99999, "anything"); err == nil {
		t.Error("expected error for nonexistent user")
	}
}

func TestUpdatePassword(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "pwuser", "pw@example.com", "oldpassword")
	if err != nil {
		t.Fatal(err)
	}

	if err := UpdatePassword(db, user.ID, "newpassword"); err != nil {
		t.Fatalf("UpdatePassword failed: %v", err)
	}

	if err := VerifyPassword(db, user.ID, "newpassword"); err != nil {
		t.Error("new password should verify correctly")
	}
	if err := VerifyPassword(db, user.ID, "oldpassword"); err == nil {
		t.Error("old password should no longer work")
	}
}

func TestUpdateUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "oldname", "rename@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}

	if err := UpdateUsername(db, user.ID, "newname"); err != nil {
		t.Fatalf("UpdateUsername failed: %v", err)
	}

	found, err := GetUserByID(db, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Username != "newname" {
		t.Errorf("expected 'newname', got %q", found.Username)
	}
}

func TestUpdateUserCurrency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "curruser", "curr@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}

	if err := UpdateUserCurrency(db, user.ID, "€"); err != nil {
		t.Fatalf("UpdateUserCurrency failed: %v", err)
	}

	found, err := GetUserByID(db, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.Currency != "€" {
		t.Errorf("expected '€', got %q", found.Currency)
	}
}

func TestCSRFTokenLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "csrfuser", "csrf@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}

	token, err := CreateCSRFToken(db, user.ID)
	if err != nil {
		t.Fatalf("CreateCSRFToken failed: %v", err)
	}
	if token.Token == "" {
		t.Error("token should not be empty")
	}
	if token.UserID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, token.UserID)
	}
	if token.ExpiresAt.Before(time.Now()) {
		t.Error("token should not be expired on creation")
	}

	// Validate consumes the token
	if err := ValidateCSRFToken(db, token.Token, user.ID); err != nil {
		t.Fatalf("ValidateCSRFToken failed: %v", err)
	}

	// Token should be consumed — second validation must fail
	if err := ValidateCSRFToken(db, token.Token, user.ID); err == nil {
		t.Error("second validation should fail — token was consumed")
	}

	// Wrong user ID should fail
	token2, _ := CreateCSRFToken(db, user.ID)
	if err := ValidateCSRFToken(db, token2.Token, 99999); err == nil {
		t.Error("validation with wrong user should fail")
	}
}

func TestCleanupExpiredCSRFTokens(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "csrfclean", "csrfclean@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}

	// Insert an already-expired token directly
	_, err = db.Exec(
		`INSERT INTO csrf_tokens (token, user_id, expires_at) VALUES (?, ?, ?)`,
		"expiredtoken", user.ID, time.Now().Add(-1*time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := CleanupExpiredCSRFTokens(db); err != nil {
		t.Fatalf("CleanupExpiredCSRFTokens failed: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM csrf_tokens WHERE token = 'expiredtoken'").Scan(&count)
	if count != 0 {
		t.Error("expired token should have been cleaned up")
	}
}

func TestBearerTokenLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "beareruser", "bearer@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}
	// Activate the user so ValidateSession returns a valid activated user
	_, err = db.Exec("UPDATE users SET is_activated = 1 WHERE id = ?", user.ID)
	if err != nil {
		t.Fatal(err)
	}

	token, expiresAt, err := CreateBearerToken(db, user.ID)
	if err != nil {
		t.Fatalf("CreateBearerToken failed: %v", err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
	if expiresAt.Before(time.Now()) {
		t.Error("token should not be expired on creation")
	}

	validated, err := ValidateBearerToken(db, token)
	if err != nil {
		t.Fatalf("ValidateBearerToken failed: %v", err)
	}
	if validated.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, validated.ID)
	}

	// Invalid token
	_, err = ValidateBearerToken(db, "notarealtoken")
	if err == nil {
		t.Error("expected error for invalid bearer token")
	}
}

func TestActivationTokenLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "activateuser", "activate@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}

	token, err := CreateActivationToken(db, user.ID)
	if err != nil {
		t.Fatalf("CreateActivationToken failed: %v", err)
	}
	if token.Token == "" {
		t.Error("activation token should not be empty")
	}

	// Validate returns user
	foundUser, err := ValidateActivationToken(db, token.Token)
	if err != nil {
		t.Fatalf("ValidateActivationToken failed: %v", err)
	}
	if foundUser.ID != user.ID {
		t.Errorf("expected user ID %d, got %d", user.ID, foundUser.ID)
	}

	// Invalid token
	_, err = ValidateActivationToken(db, "badtoken")
	if err == nil {
		t.Error("expected error for invalid activation token")
	}
}

func TestActivateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "notactivated", "notact@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if user.IsActivated {
		t.Error("new user should not be activated")
	}

	token, err := CreateActivationToken(db, user.ID)
	if err != nil {
		t.Fatal(err)
	}

	if err := ActivateUser(db, user.ID, token.Token); err != nil {
		t.Fatalf("ActivateUser failed: %v", err)
	}

	found, err := GetUserByID(db, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !found.IsActivated {
		t.Error("user should now be activated")
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	user, err := CreateUser(db, "sessionclean", "sessionclean@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}

	// Insert expired session directly
	_, err = db.Exec(
		`INSERT INTO sessions (id, user_id, expires_at) VALUES (?, ?, ?)`,
		"expiredsession", user.ID, time.Now().Add(-1*time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := CleanupExpiredSessions(db); err != nil {
		t.Fatalf("CleanupExpiredSessions failed: %v", err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = 'expiredsession'").Scan(&count)
	if count != 0 {
		t.Error("expired session should have been cleaned up")
	}
}

func TestFirstUserBecomesAdmin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	first, err := CreateUser(db, "firstadmin", "admin@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if !first.IsAdmin {
		t.Error("first user should be admin")
	}

	second, err := CreateUser(db, "notadmin", "notadmin@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if second.IsAdmin {
		t.Error("second user should not be admin")
	}
}
