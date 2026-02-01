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
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
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
	db, _ := setupTestDB(t)
	defer db.Close()

	expectedPass := "securepass123"
	os.Setenv("INITIAL_ADMIN_PASSWORD", expectedPass)
	defer os.Unsetenv("INITIAL_ADMIN_PASSWORD")

	if err := CreateDefaultUsers(db); err != nil {
		t.Fatalf("CreateDefaultUsers failed: %v", err)
	}

	var storedHash string
	err := db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&storedHash)
	if err != nil {
		t.Fatalf("Failed to query dyson user: %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(expectedPass)); err != nil {
		t.Errorf("Password mismatch for env var case: %v", err)
	}
}

func TestCreateDefaultUsers_WithoutEnvVar(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	os.Unsetenv("INITIAL_ADMIN_PASSWORD")

	if err := CreateDefaultUsers(db); err != nil {
		t.Fatalf("CreateDefaultUsers failed: %v", err)
	}

	var storedHash string
	err := db.QueryRow("SELECT password FROM users WHERE username = ?", "dyson").Scan(&storedHash)
	if err != nil {
		t.Fatalf("Failed to query dyson user: %v", err)
	}

	// This should fail if the password is still "daniel"
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("daniel")); err == nil {
		t.Errorf("Password should NOT be 'daniel' when env var is unset")
	}
}
