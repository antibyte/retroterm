package tinyos

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// Database is a wrapper around the SQLite database connection
type Database struct {
	conn *sql.DB
}

// InitDB initializes the SQLite database connection and returns the connection object.
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ensure the database is accessible
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

// CreateTables ensures all required tables exist in the database.
func CreateTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password TEXT NOT NULL,
			last_login INTEGER,
			login_attempts INTEGER DEFAULT 0,
			is_admin INTEGER DEFAULT 0,
			is_active INTEGER DEFAULT 1,
			is_logged_in INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			ip_address TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS banned_users (
			identifier TEXT PRIMARY KEY,
			expiry INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_usage (
			username TEXT NOT NULL,
			date TEXT NOT NULL,
			time_used INTEGER DEFAULT 0,
			last_session_start INTEGER,
			PRIMARY KEY (username, date)
		)`,
		`CREATE TABLE IF NOT EXISTS registration_attempts (
			ip_address TEXT NOT NULL,
			timestamp INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS virtual_files (
			username TEXT NOT NULL,
			path TEXT NOT NULL,
			content BLOB,
			is_dir INTEGER DEFAULT 0,
			mod_time INTEGER NOT NULL,
			PRIMARY KEY (username, path)
		)`,
		`CREATE TABLE IF NOT EXISTS env_vars (
			name TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// CSRFToken represents a CSRF token for a user and a session
type CSRFToken struct {
	ID           string
	UserID       int
	Token        string
	SessionToken string
	IP           string
	UserAgent    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
}

// InitCSRFTokenTable creates the CSRF token table if it doesn't exist
func (db *Database) InitCSRFTokenTable() error {
	_, err := db.conn.Exec(`
		CREATE TABLE IF NOT EXISTS csrf_tokens (
			id TEXT PRIMARY KEY,
			user_id INTEGER,
			token TEXT NOT NULL,
			session_token TEXT NOT NULL,
			ip TEXT,
			user_agent TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(rowid)
		)
	`)
	if err != nil {
		return fmt.Errorf("error creating CSRF token table: %v", err)
	}

	// Index for faster lookups
	_, err = db.conn.Exec(`
		CREATE INDEX IF NOT EXISTS idx_csrf_tokens_token ON csrf_tokens(token);
		CREATE INDEX IF NOT EXISTS idx_csrf_tokens_session ON csrf_tokens(session_token);
		CREATE INDEX IF NOT EXISTS idx_csrf_tokens_expires ON csrf_tokens(expires_at);
	`)
	if err != nil {
		return fmt.Errorf("error creating CSRF token indexes: %v", err)
	}

	return nil
}

// CreateCSRFToken creates a new CSRF token for a user and a session
func (db *Database) CreateCSRFToken(userID int, sessionToken, ip, userAgent string) (string, error) {
	// Generate a random token
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("error generating CSRF token: %v", err)
	}

	token := hex.EncodeToString(tokenBytes)
	id := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour) // Token expires after 24 hours

	_, err = db.conn.Exec(`
		INSERT INTO csrf_tokens (
			id, user_id, token, session_token, ip, user_agent, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, userID, token, sessionToken, ip, userAgent, expiresAt)

	if err != nil {
		return "", fmt.Errorf("error saving CSRF token: %v", err)
	}

	return token, nil
}

// ValidateCSRFToken checks the validity of a CSRF token for a session
func (db *Database) ValidateCSRFToken(token, sessionToken, ip string) bool {
	if token == "" || sessionToken == "" {
		return false
	}

	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM csrf_tokens 
		WHERE token = ? AND session_token = ? AND expires_at > CURRENT_TIMESTAMP
	`, token, sessionToken).Scan(&count)

	if err != nil || count == 0 {
		return false
	}

	// Update the last usage of the token
	_, _ = db.conn.Exec(`
		UPDATE csrf_tokens SET created_at = CURRENT_TIMESTAMP 
		WHERE token = ? AND session_token = ?
	`, token, sessionToken)

	return true
}

// RotateCSRFToken rotates a CSRF token for increased security
func (db *Database) RotateCSRFToken(oldToken, sessionToken, ip, userAgent string) (string, error) {
	// Check the old token and get the user ID
	var userID int
	err := db.conn.QueryRow(`
		SELECT user_id FROM csrf_tokens 
		WHERE token = ? AND session_token = ? AND expires_at > CURRENT_TIMESTAMP
	`, oldToken, sessionToken).Scan(&userID)

	if err != nil {
		return "", fmt.Errorf("invalid CSRF token: %v", err)
	}

	// Generate a new token
	tokenBytes := make([]byte, 32)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		return "", fmt.Errorf("error generating new CSRF token: %v", err)
	}

	newToken := hex.EncodeToString(tokenBytes)
	id := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour)

	// Begin a transaction to invalidate the old token and create a new one
	tx, err := db.conn.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	// Invalidate the old token
	_, err = tx.Exec(`
		UPDATE csrf_tokens SET expires_at = CURRENT_TIMESTAMP
		WHERE token = ? AND session_token = ?
	`, oldToken, sessionToken)

	if err != nil {
		return "", err
	}

	// Create a new token
	_, err = tx.Exec(`
		INSERT INTO csrf_tokens (
			id, user_id, token, session_token, ip, user_agent, expires_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, userID, newToken, sessionToken, ip, userAgent, expiresAt)

	if err != nil {
		return "", err
	}

	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return newToken, nil
}

// CleanupExpiredTokens removes expired tokens from the database
func (db *Database) CleanupExpiredTokens() error {
	_, err := db.conn.Exec(`DELETE FROM csrf_tokens WHERE expires_at < CURRENT_TIMESTAMP`)
	return err
}

// GetUserIDBySessionToken returns the user ID for a valid session token
func (db *Database) GetUserIDBySessionToken(sessionToken string) (int, error) {
	var userID int
	err := db.conn.QueryRow(`
		SELECT user_id FROM csrf_tokens 
		WHERE session_token = ? AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC LIMIT 1
	`, sessionToken).Scan(&userID)

	if err != nil {
		return 0, err
	}

	return userID, nil
}

// ShouldRotateToken checks if a CSRF token should be rotated based on its age
func (db *Database) ShouldRotateToken(token, sessionToken string) bool {
	if token == "" || sessionToken == "" {
		return false
	}

	var createdTime time.Time
	err := db.conn.QueryRow(`
		SELECT created_at FROM csrf_tokens 
		WHERE token = ? AND session_token = ? AND expires_at > CURRENT_TIMESTAMP
	`, token, sessionToken).Scan(&createdTime)

	if err != nil {
		return false
	}

	// Rotate if the token is older than 4 hours
	return time.Since(createdTime) > 4*time.Hour
}

// GetUserID returns the ID of a user by their username
func (db *Database) GetUserID(username string) (int, error) {
	var id int
	err := db.conn.QueryRow(`
		SELECT rowid FROM users WHERE username = ?
	`, username).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

// CreateDefaultUsers creates default system users if they don't exist
func CreateDefaultUsers(db *sql.DB) error {
	// Check if dyson user already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", "dyson").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check for dyson user: %w", err)
	}
	// Create dyson user if it doesn't exist
	if count == 0 {
		// Password is "daniel" (his son's name) - needs to be hashed
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("daniel"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash dyson password: %w", err)
		}

		_, err = db.Exec(`
			INSERT INTO users (username, password, last_login, login_attempts, is_admin, is_active, is_logged_in, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, "dyson", string(hashedPassword), 0, 0, 1, 1, 0, time.Now().Unix())

		if err != nil {
			return fmt.Errorf("failed to create dyson user: %w", err)
		}

		log.Printf("[INIT] Created default user: dyson (password: daniel)")
	}

	return nil
}
