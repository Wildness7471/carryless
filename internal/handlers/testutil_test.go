package handlers

import (
	"database/sql"
	"html/template"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"carryless/internal/config"
	"carryless/internal/database"
	"carryless/internal/email"
	"carryless/internal/models"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates a named shared in-memory SQLite database for a test.
// Named shared-cache mode ensures all connections in the pool see the same data,
// which avoids "no such table" errors when handlers mix transactions and bare queries.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := "file:" + sanitizeName(t.Name()) + "?mode=memory&cache=shared&_foreign_keys=on"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatal("Failed to open test database:", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal("Failed to run migrations:", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// sanitizeName makes t.Name() safe for use as a SQLite file URI component.
func sanitizeName(name string) string {
	r := strings.NewReplacer("/", "_", " ", "_", "=", "_")
	return r.Replace(name)
}

// setupHandlerTest creates a full test router (same template+funcmap as production)
// backed by a fresh in-memory SQLite database. cfg.Environment is "development" so
// CSRF tokens and rate limiting are automatically bypassed.
func setupHandlerTest(t *testing.T) (*gin.Engine, *sql.DB, *config.Config) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := setupTestDB(t)

	cfg := &config.Config{
		Environment:     "development",
		SessionDuration: 24 * time.Hour,
	}

	r := gin.New()
	r.SetFuncMap(testFuncMap())

	// Load templates — TestMain has already chdir'd to the repo root.
	files, _ := filepath.Glob("templates/*.html")
	partials, _ := filepath.Glob("templates/partials/*.html")
	r.LoadHTMLFiles(append(files, partials...)...)

	emailSvc := email.NewService(cfg) // disabled (no Mailgun credentials in test config)
	SetupRoutes(r, db, emailSvc, cfg)

	return r, db, cfg
}

// testFuncMap returns a template.FuncMap that mirrors main.go so that templates
// parse without errors. Because most tests use Accept:application/json, templates
// are rarely actually rendered during tests.
func testFuncMap() template.FuncMap {
	return template.FuncMap{
		"jsonify": func(v interface{}) template.JS {
			return template.JS(fmt.Sprintf("%v", v))
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"toUpper": func(s string) string { return strings.ToUpper(s) },
		"groupByCategory": func(items []models.PackItem) map[string][]models.PackItem {
			groups := make(map[string][]models.PackItem)
			for _, item := range items {
				cat := item.Item.Category.Name
				groups[cat] = append(groups[cat], item)
			}
			return groups
		},
		"groupItemsByCategory": func(items []models.Item) map[string][]models.Item {
			groups := make(map[string][]models.Item)
			for _, item := range items {
				cat := item.Category.Name
				groups[cat] = append(groups[cat], item)
			}
			return groups
		},
		"redactEmail": func(email string) string {
			parts := strings.Split(email, "@")
			if len(parts) != 2 || len(parts[0]) <= 2 {
				return email
			}
			p := parts[0]
			return string(p[0]) + "***" + string(p[len(p)-1]) + "@" + parts[1]
		},
		"sequence": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		"getLabelForItem": func(labels []models.ItemLabel, idx int) *models.ItemLabel {
			cur := 0
			for _, l := range labels {
				if cur <= idx && idx < cur+l.Count {
					return &l
				}
				cur += l.Count
			}
			return nil
		},
		"timeAgo": func(t time.Time) string {
			d := time.Since(t)
			switch {
			case d.Minutes() < 1:
				return "Just now"
			case d.Hours() < 1:
				return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
			case d.Hours() < 24:
				return fmt.Sprintf("%d hours ago", int(d.Hours()))
			default:
				return t.Format("Jan 2")
			}
		},
		"deref": func(s *string) string {
			if s == nil {
				return ""
			}
			return *s
		},
	}
}

// createAdminUser creates the first user in an empty DB (auto-promoted to admin)
// and activates them. Returns the user.
func createAdminUser(t *testing.T, db *sql.DB) *models.User {
	t.Helper()
	user, err := database.CreateUser(db, "admin", "admin@example.com", "password123")
	if err != nil {
		t.Fatal("createAdminUser:", err)
	}
	activateUser(t, db, user.ID)
	user.IsActivated = true
	user.IsAdmin = true
	return user
}

// createRegularUser creates a non-admin activated user.
// Requires at least one user to already exist so this user is not auto-promoted.
func createRegularUser(t *testing.T, db *sql.DB) *models.User {
	t.Helper()
	// Ensure the admin user (first user) exists so this one is not auto-promoted.
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		createAdminUser(t, db)
	}
	user, err := database.CreateUser(db, "user1", "user1@example.com", "password123")
	if err != nil {
		t.Fatal("createRegularUser:", err)
	}
	activateUser(t, db, user.ID)
	user.IsActivated = true
	return user
}

// createNamedUser creates an activated non-admin user with the given credentials.
func createNamedUser(t *testing.T, db *sql.DB, username, email, password string) *models.User {
	t.Helper()
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if count == 0 {
		createAdminUser(t, db)
	}
	user, err := database.CreateUser(db, username, email, password)
	if err != nil {
		t.Fatalf("createNamedUser(%s): %v", username, err)
	}
	activateUser(t, db, user.ID)
	user.IsActivated = true
	return user
}

// activateUser sets is_activated=true for the given user ID.
func activateUser(t *testing.T, db *sql.DB, userID int) {
	t.Helper()
	if _, err := db.Exec("UPDATE users SET is_activated = true WHERE id = ?", userID); err != nil {
		t.Fatal("activateUser:", err)
	}
}

// loginAndGetCookie POSTs /login and returns the "session_id=<value>" cookie string.
func loginAndGetCookie(t *testing.T, router *gin.Engine, emailAddr, password string) string {
	t.Helper()
	form := url.Values{"email": {emailAddr}, "password": {password}}
	req := httptest.NewRequest(http.MethodPost, "/login",
		strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("login failed (status %d): %s", w.Code, w.Body.String())
	}
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "session_id" {
			return "session_id=" + cookie.Value
		}
	}
	t.Fatal("loginAndGetCookie: no session_id cookie in response")
	return ""
}

// doRequest is a shorthand for making a test HTTP request through the router.
func doRequest(router *gin.Engine, method, path, body, contentType, cookie string) *httptest.ResponseRecorder {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// doJSONRequest makes a request with Accept: application/json set.
func doJSONRequest(router *gin.Engine, method, path, body, cookie string) *httptest.ResponseRecorder {
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Accept", "application/json")
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}
