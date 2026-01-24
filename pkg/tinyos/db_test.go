package tinyos

import (
	"database/sql"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestCreateDefaultUsers_Security(t *testing.T) {
	// Setup temporary database
	tempDB := "test_security.db"

	// Helper to init db
	initTestDB := func() *sql.DB {
		os.Remove(tempDB) // ensure clean state for each init
		db, err := InitDB(tempDB)
		if err != nil {
			t.Fatalf("Failed to init db: %v", err)
		}
		if err := CreateTables(db); err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}
		return db
	}

	defer os.Remove(tempDB)

	// Case 1: Use Environment Variable
	t.Run("With_Env_Var", func(t *testing.T) {
		db := initTestDB()
		defer db.Close()

		expectedPass := "secure_admin_123"
		os.Setenv("INITIAL_ADMIN_PASSWORD", expectedPass)
		defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")

		if err := CreateDefaultUsers(db); err != nil {
			t.Fatalf("CreateDefaultUsers failed: %v", err)
		}

		// Verify password
		var hash string
		err := db.QueryRow("SELECT password FROM users WHERE username = 'dyson'").Scan(&hash)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(expectedPass)); err != nil {
			t.Errorf("Password hash does not match expected password '%s'", expectedPass)
		}
	})

	// Case 2: No Environment Variable (Should be random, NOT "daniel")
	t.Run("Without_Env_Var", func(t *testing.T) {
		db := initTestDB()
		defer db.Close()

		os.Unsetenv("INITIAL_ADMIN_PASSWORD")

		if err := CreateDefaultUsers(db); err != nil {
			t.Fatalf("CreateDefaultUsers failed: %v", err)
		}

		// Verify password
		var hash string
		err := db.QueryRow("SELECT password FROM users WHERE username = 'dyson'").Scan(&hash)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}

		// Should NOT be "daniel"
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("daniel")); err == nil {
			t.Error("Security Vulnerability: Default password is still 'daniel'")
		}
	})
}
