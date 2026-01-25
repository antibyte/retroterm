package tinyos

import (
	"database/sql"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestCreateDefaultUsers_Secure(t *testing.T) {
	// 1. Test with INITIAL_ADMIN_PASSWORD set
	t.Run("WithEnvironmentVariable", func(t *testing.T) {
		tempDB := "test_db_env.sqlite"
		os.Remove(tempDB)
		defer os.Remove(tempDB)

		db, err := InitDB(tempDB)
		if err != nil {
			t.Fatalf("Failed to init db: %v", err)
		}
		defer db.Close()

		if err := CreateTables(db); err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}

		expectedPass := "SecurePass123!"
		os.Setenv("INITIAL_ADMIN_PASSWORD", expectedPass)
		defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")

		if err := CreateDefaultUsers(db); err != nil {
			t.Fatalf("CreateDefaultUsers failed: %v", err)
		}

		// Verify password
		var hash string
		err = db.QueryRow("SELECT password FROM users WHERE username = 'dyson'").Scan(&hash)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}

		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(expectedPass)); err != nil {
			t.Errorf("Password hash mismatch. Expected password '%s' to work.", expectedPass)
		}
	})

	// 2. Test without INITIAL_ADMIN_PASSWORD (should be random, NOT "daniel")
	t.Run("RandomGeneration", func(t *testing.T) {
		tempDB := "test_db_random.sqlite"
		os.Remove(tempDB)
		defer os.Remove(tempDB)

		db, err := InitDB(tempDB)
		if err != nil {
			t.Fatalf("Failed to init db: %v", err)
		}
		defer db.Close()

		if err := CreateTables(db); err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}

		os.Unsetenv("INITIAL_ADMIN_PASSWORD")

		if err := CreateDefaultUsers(db); err != nil {
			t.Fatalf("CreateDefaultUsers failed: %v", err)
		}

		// Verify password
		var hash string
		err = db.QueryRow("SELECT password FROM users WHERE username = 'dyson'").Scan(&hash)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}

		// Should NOT be "daniel"
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("daniel")); err == nil {
			t.Error("Security Vulnerability: Default password is still 'daniel'")
		}
	})
}

// Helper to check if table exists
func tableExists(db *sql.DB, tableName string) bool {
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&name)
	return err == nil
}
