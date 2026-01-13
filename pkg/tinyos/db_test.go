package tinyos

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/antibyte/retroterm/pkg/configuration"
	_ "modernc.org/sqlite"
)

func TestCreateDefaultUsers(t *testing.T) {
	// Setup temp dir for config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_settings.cfg")

	// Initialize configuration
	// This will create a default config file in the temp dir
	configuration.Initialize(configPath)

	// Case 1: Configured User
	configuration.SetString("DefaultUser", "username", "testadmin")
	configuration.SetString("DefaultUser", "password", "testpass")

	// Setup in-memory DB
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Create tables
	err = CreateTables(db)
	if err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	// Test CreateDefaultUsers
	err = CreateDefaultUsers(db)
	if err != nil {
		t.Fatalf("CreateDefaultUsers failed: %v", err)
	}

	// Verify user exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'testadmin'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query user: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 user 'testadmin', found %d", count)
	}

	// Verify password is hashed (not plain text)
	var password string
	err = db.QueryRow("SELECT password FROM users WHERE username = 'testadmin'").Scan(&password)
	if err != nil {
		t.Fatalf("Failed to get password: %v", err)
	}
	if password == "testpass" {
		t.Errorf("Password stored in plain text!")
	}

	// Case 2: No Configured User
	configuration.SetString("DefaultUser", "username", "")
	configuration.SetString("DefaultUser", "password", "")

	// Reset DB
	db2, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open DB2: %v", err)
	}
	defer db2.Close()
	CreateTables(db2)

	err = CreateDefaultUsers(db2)
	if err != nil {
		t.Fatalf("CreateDefaultUsers failed (empty config): %v", err)
	}

	err = db2.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query user count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 users when config is empty, found %d", count)
	}
}
