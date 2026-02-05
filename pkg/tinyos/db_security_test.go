package tinyos

import (
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestSecurity_DefaultUserPassword(t *testing.T) {
	// Case 1: INITIAL_ADMIN_PASSWORD is set
	t.Run("WithEnvironmentVariable", func(t *testing.T) {
		expectedPassword := "securePassword123"
		os.Setenv("INITIAL_ADMIN_PASSWORD", expectedPassword)
		defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")

		// Initialize in-memory database
		db, err := InitDB(":memory:")
		if err != nil {
			t.Fatalf("Failed to init DB: %v", err)
		}
		defer db.Close()

		if err := CreateTables(db); err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}

		if err := CreateDefaultUsers(db); err != nil {
			t.Fatalf("Failed to create default users: %v", err)
		}

		// Verify password matches env var
		var hashedPassword string
		err = db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&hashedPassword)
		if err != nil {
			t.Fatalf("Failed to find dyson user: %v", err)
		}

		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(expectedPassword))
		if err != nil {
			t.Errorf("Password mismatch. Expected '%s', got hash mismatch: %v", expectedPassword, err)
		}
	})

	// Case 2: INITIAL_ADMIN_PASSWORD is unset (random generation)
	t.Run("WithoutEnvironmentVariable", func(t *testing.T) {
		os.Unsetenv("INITIAL_ADMIN_PASSWORD")

		// Initialize in-memory database
		db, err := InitDB(":memory:")
		if err != nil {
			t.Fatalf("Failed to init DB: %v", err)
		}
		defer db.Close()

		if err := CreateTables(db); err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}

		if err := CreateDefaultUsers(db); err != nil {
			t.Fatalf("Failed to create default users: %v", err)
		}

		// Verify dyson user exists
		var hashedPassword string
		err = db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&hashedPassword)
		if err != nil {
			t.Fatalf("Failed to find dyson user: %v", err)
		}

		// Verify it is NOT "daniel"
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte("daniel"))
		if err == nil {
			t.Errorf("Security check failed: Password is still 'daniel'!")
		}

		// Verify it is NOT empty
		if len(hashedPassword) == 0 {
			t.Errorf("Password hash is empty")
		}
	})
}
