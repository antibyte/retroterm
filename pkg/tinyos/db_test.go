package tinyos

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestCreateDefaultUsers_Secure(t *testing.T) {
	// Case 1: Random password (no env var)
	t.Run("RandomPassword", func(t *testing.T) {
		// Ensure env var is unset
		os.Unsetenv("INITIAL_ADMIN_PASSWORD")

		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test_secure_random.db")

		db, err := InitDB(dbPath)
		if err != nil {
			t.Fatalf("Failed to init DB: %v", err)
		}
		defer db.Close()

		err = CreateTables(db)
		if err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}

		err = CreateDefaultUsers(db)
		if err != nil {
			t.Fatalf("Failed to create default users: %v", err)
		}

		var passwordHash string
		err = db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&passwordHash)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}

		// Verify it is NOT "daniel"
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte("daniel"))
		if err == nil {
			t.Fatal("SECURITY FAILURE: Default password is still 'daniel' when env var is unset!")
		}
	})

	// Case 2: Explicit env var
	t.Run("EnvVarPassword", func(t *testing.T) {
		expectedPass := "SuperSecret123!"
		os.Setenv("INITIAL_ADMIN_PASSWORD", expectedPass)
		defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")

		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test_secure_env.db")

		db, err := InitDB(dbPath)
		if err != nil {
			t.Fatalf("Failed to init DB: %v", err)
		}
		defer db.Close()

		err = CreateTables(db)
		if err != nil {
			t.Fatalf("Failed to create tables: %v", err)
		}

		err = CreateDefaultUsers(db)
		if err != nil {
			t.Fatalf("Failed to create default users: %v", err)
		}

		var passwordHash string
		err = db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&passwordHash)
		if err != nil {
			t.Fatalf("Failed to query user: %v", err)
		}

		// Verify it MATCHES expectedPass
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(expectedPass))
		if err != nil {
			t.Fatalf("Functionality check failed: Password does not match environment variable: %v", err)
		}
	})
}
