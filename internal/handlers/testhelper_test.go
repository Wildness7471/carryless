package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"carryless/internal/config"
	"carryless/internal/database"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// setupHandlerTestDB creates a named in-memory SQLite DB with migrations applied.
func setupHandlerTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_foreign_keys=on"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal("Failed to open test database:", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal("Failed to run migrations:", err)
	}
	return db
}

// testActivatedUser creates and activates a user.
func testActivatedUser(t *testing.T, db *sql.DB, username, email string) *models.User {
	t.Helper()
	user, err := database.CreateUser(db, username, email, "testpass123")
	if err != nil {
		t.Fatalf("create user %s: %v", username, err)
	}
	if _, err := db.Exec("UPDATE users SET is_activated = 1 WHERE id = ?", user.ID); err != nil {
		t.Fatalf("activate user %s: %v", username, err)
	}
	user.IsActivated = true
	return user
}

// testConfig returns a development config (CSRF + rate limiting disabled).
func testConfig() *config.Config {
	return &config.Config{Environment: "development"}
}

// authInject injects db + user into gin context, bypassing real session auth.
func authInject(db *sql.DB, user *models.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Set("user", user)
		c.Set("user_id", user.ID)
		c.Next()
	}
}

// newRouter creates a test gin engine with the given user pre-authenticated.
func newRouter(db *sql.DB, user *models.User) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(authInject(db, user))
	return r
}

// req performs an HTTP request accepting JSON, returning the response recorder.
func req(t *testing.T, r *gin.Engine, method, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	var httpReq *http.Request
	if form != nil {
		httpReq = httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
		httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		httpReq = httptest.NewRequest(method, path, nil)
	}
	httpReq.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httpReq)
	return w
}

// ensureCategory returns the ID of a "Gear" category for the given user, creating it if needed.
func ensureCategory(t *testing.T, db *sql.DB, userID int) int {
	t.Helper()
	cat, err := database.GetOrCreateCategory(db, userID, "Gear")
	if err != nil {
		t.Fatalf("ensureCategory: %v", err)
	}
	return cat.ID
}

// newItem builds a minimal models.Item for test use. Uses categoryID from ensureCategory.
func newItemWithCat(name string, weight, catID int) models.Item {
	return models.Item{
		Name:        name,
		CategoryID:  catID,
		WeightGrams: weight,
	}
}

// newItem builds a minimal models.Item with a dummy category ID (use when FK not enforced).
func newItem(name string, weight int) models.Item {
	return models.Item{
		Name:        name,
		WeightGrams: weight,
	}
}

// istr converts int to string without importing fmt in every test file.
func istr(n int) string {
	return fmt.Sprintf("%d", n)
}
