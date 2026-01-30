package tinyos

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, string) {
	// Create a temporary file for the database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_tinyos.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}

	if err := CreateTables(db); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return db, dbPath
}

func TestCreateDefaultUsers_WithEnvVar(t *testing.T) {
	// Set environment variable
	testPass := "secure_test_password_123"
	os.Setenv("INITIAL_ADMIN_PASSWORD", testPass)
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")

	db, _ := setupTestDB(t)
	defer db.Close()

	// Call CreateDefaultUsers
	if err := CreateDefaultUsers(db); err != nil {
		t.Fatalf("CreateDefaultUsers failed: %v", err)
	}

	// Verify user dyson exists
	var hashedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&hashedPassword)
	if err != nil {
		t.Fatalf("Failed to query dyson user: %v", err)
	}

	// Verify password matches env var
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(testPass))
	if err != nil {
		t.Errorf("Password mismatch. Expected to match '%s', but failed: %v", testPass, err)
	}
}

func TestCreateDefaultUsers_NoEnvVar(t *testing.T) {
	// Ensure environment variable is unset
	os.Unsetenv("INITIAL_ADMIN_PASSWORD")

	db, _ := setupTestDB(t)
	defer db.Close()

	// Call CreateDefaultUsers
	if err := CreateDefaultUsers(db); err != nil {
		t.Fatalf("CreateDefaultUsers failed: %v", err)
	}

	// Verify user dyson exists
	var hashedPassword string
	err := db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&hashedPassword)
	if err != nil {
		t.Fatalf("Failed to query dyson user: %v", err)
	}

	// Verify password is NOT "daniel" (the hardcoded one)
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte("daniel"))
	if err == nil {
		t.Error("Security vulnerability: Default password is still 'daniel'")
	}
}
