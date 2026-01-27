package tinyos

import (
	"database/sql"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func TestCreateDefaultUsers_WithEnvVar(t *testing.T) {
	// Setup in-memory DB
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize tables
	if err := CreateTables(db); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Set env var
	expectedPassword := "securetestpassword"
	os.Setenv("INITIAL_ADMIN_PASSWORD", expectedPassword)
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")

	// Call function under test
	if err := CreateDefaultUsers(db); err != nil {
		t.Fatalf("CreateDefaultUsers failed: %v", err)
	}

	// Verify user exists and password is correct
	var hashedPassword string
	err = db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&hashedPassword)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	// Verify hash matches expected password
	// Note: Currently this test will FAIL because code still uses "daniel"
	// But once I fix the code, it should PASS.
	// To verify the failure first, I can run it.

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(expectedPassword))
	if err != nil {
		t.Errorf("Password mismatch. Hash validation failed: %v", err)
	}
}

func TestCreateDefaultUsers_NoEnvVar(t *testing.T) {
	// Setup in-memory DB
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Initialize tables
	if err := CreateTables(db); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Ensure env var is unset
	os.Unsetenv("INITIAL_ADMIN_PASSWORD")

	// Call function under test
	if err := CreateDefaultUsers(db); err != nil {
		t.Fatalf("CreateDefaultUsers failed: %v", err)
	}

	// Verify user exists
	var hashedPassword string
	err = db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&hashedPassword)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}

	// In the current buggy code, this will be hash of "daniel".
	// In the fixed code, it will be a random password.
	// We can't easily verify the random password itself without capturing logs,
	// but we can verify it is NOT "daniel" (unless random chance, unlikely).

	// Check if it is "daniel" (should be false after fix)
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte("daniel"))
	if err == nil {
		t.Error("Password should NOT be 'daniel' when INITIAL_ADMIN_PASSWORD is not set")
	}
}
