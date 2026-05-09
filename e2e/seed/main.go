// e2e/seed/main.go seeds the E2E test database with pre-activated users.
// Run: go run ./e2e/seed/main.go
package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"carryless/internal/database"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "/tmp/e2e-test.db"
	}

	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		log.Fatal("open db:", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatal("migrate:", err)
	}

	users := []struct{ username, email, password string }{
		{"e2etester", "e2e@example.com", "password123"},
		{"e2eadmin", "admin@example.com", "adminpass123"},
	}

	for _, u := range users {
		user, err := database.CreateUser(db, u.username, u.email, u.password)
		if err != nil {
			fmt.Printf("user %s already exists (skipping): %v\n", u.email, err)
			continue
		}
		if _, err := db.Exec("UPDATE users SET is_activated = TRUE WHERE id = ?", user.ID); err != nil {
			log.Fatalf("activate user %s: %v", u.email, err)
		}
		fmt.Printf("created and activated user: %s\n", u.email)
	}

	// Make e2eadmin an admin
	var adminID int
	db.QueryRow("SELECT id FROM users WHERE email = 'admin@example.com'").Scan(&adminID)
	if adminID > 0 {
		db.Exec("UPDATE users SET is_admin = TRUE WHERE id = ?", adminID)
		fmt.Println("granted admin to admin@example.com")
	}
}
