package tests

import (
	"testing"
	"time"

	"github.com/antibyte/retroterm/pkg/auth"
	"github.com/antibyte/retroterm/pkg/tinyos"
)

// TestTemporaryUserSessionManagement tests the core functionality for temporary users
func TestTemporaryUserSessionManagement(t *testing.T) {
	// Test that dyson is identified as a temporary user
	if !auth.IsTemporaryUser("dyson") {
		t.Errorf("dyson should be identified as a temporary user")
	}

	// Test that regular users are not identified as temporary users
	if auth.IsTemporaryUser("alice") {
		t.Errorf("alice should not be identified as a temporary user")
	}

	if auth.IsTemporaryUser("bob") {
		t.Errorf("bob should not be identified as a temporary user")
	}
}

// TestTemporaryTokenGeneration tests JWT token generation for temporary users
func TestTemporaryTokenGeneration(t *testing.T) {
	// Test temporary token generation
	tempToken, err := auth.GenerateTemporaryUserToken("dyson", "temp_session_123")
	if err != nil {
		t.Fatalf("Failed to generate temporary token: %v", err)
	}

	if tempToken == "" {
		t.Error("Temporary token should not be empty")
	}

	// Validate the temporary token
	claims, isUserToken, err := auth.ValidateToken(tempToken)
	if err != nil {
		t.Fatalf("Failed to validate temporary token: %v", err)
	}

	if !isUserToken {
		t.Error("Temporary token should be identified as user token")
	}

	userClaims, ok := claims.(*auth.UserClaims)
	if !ok {
		t.Error("Claims should be UserClaims")
	}

	if userClaims.Username != "dyson" {
		t.Errorf("Expected username 'dyson', got '%s'", userClaims.Username)
	}

	if userClaims.SessionID != "temp_session_123" {
		t.Errorf("Expected session ID 'temp_session_123', got '%s'", userClaims.SessionID)
	}

	if !userClaims.IsTempToken {
		t.Error("Token should be marked as temporary")
	}

	// Check that token expires quickly (should be less than 1 hour)
	if userClaims.ExpiresAt.Time.Sub(time.Now()) > time.Hour {
		t.Error("Temporary token should expire within 1 hour")
	}
}

// TestRegularTokenGeneration tests JWT token generation for regular users
func TestRegularTokenGeneration(t *testing.T) {
	// Test regular token generation
	regularToken, err := auth.GenerateUserToken("user_session_456", "alice")
	if err != nil {
		t.Fatalf("Failed to generate regular token: %v", err)
	}

	if regularToken == "" {
		t.Error("Regular token should not be empty")
	}

	// Validate the regular token
	claims, isUserToken, err := auth.ValidateToken(regularToken)
	if err != nil {
		t.Fatalf("Failed to validate regular token: %v", err)
	}

	if !isUserToken {
		t.Error("Regular token should be identified as user token")
	}

	userClaims, ok := claims.(*auth.UserClaims)
	if !ok {
		t.Error("Claims should be UserClaims")
	}

	if userClaims.Username != "alice" {
		t.Errorf("Expected username 'alice', got '%s'", userClaims.Username)
	}

	if userClaims.SessionID != "user_session_456" {
		t.Errorf("Expected session ID 'user_session_456', got '%s'", userClaims.SessionID)
	}

	if userClaims.IsTempToken {
		t.Error("Regular token should not be marked as temporary")
	}

	// Check that token expires in 24 hours
	expectedExpiry := time.Now().Add(24 * time.Hour)
	actualExpiry := userClaims.ExpiresAt.Time
	if actualExpiry.Sub(expectedExpiry) > time.Minute || expectedExpiry.Sub(actualExpiry) > time.Minute {
		t.Error("Regular token should expire in approximately 24 hours")
	}
}

// TestLoginHandlerForTemporaryUser tests the core login logic for temporary users
func TestLoginHandlerForTemporaryUser(t *testing.T) {
	// Test the core logic components that would be used in login handler

	// Test that dyson generates a temporary token
	tempToken, err := auth.GenerateTemporaryUserToken("dyson", "temp_session_123")
	if err != nil {
		t.Fatalf("Failed to generate temporary token: %v", err)
	}

	// Validate the token has correct properties
	claims, isUserToken, err := auth.ValidateToken(tempToken)
	if err != nil {
		t.Fatalf("Failed to validate temporary token: %v", err)
	}

	if !isUserToken {
		t.Error("Token should be identified as user token")
	}

	userClaims, ok := claims.(*auth.UserClaims)
	if !ok {
		t.Error("Claims should be UserClaims")
	}

	if !userClaims.IsTempToken {
		t.Error("Token for dyson should be marked as temporary")
	}

	if userClaims.Username != "dyson" {
		t.Errorf("Expected username 'dyson', got '%s'", userClaims.Username)
	}

	// Test that the token expires quickly (temporary token should be short-lived)
	if userClaims.ExpiresAt.Time.Sub(time.Now()) > time.Hour {
		t.Error("Temporary token should expire within 1 hour")
	}
}

// TestSessionRestorationBlocking tests that temporary user sessions cannot be restored
func TestSessionRestorationBlocking(t *testing.T) {
	// This test would require integration with the WebSocket handler
	// For now, we'll test the core logic

	// Create a mock session for dyson
	dysonSession := &tinyos.Session{
		ID:        "dyson_session_123",
		Username:  "dyson",
		IPAddress: "127.0.0.1",
		// ... other session fields
	}

	// Test that IsTemporaryUser correctly identifies dyson session
	if !auth.IsTemporaryUser(dysonSession.Username) {
		t.Error("Dyson session should be identified as temporary")
	}

	// Create a mock session for regular user
	aliceSession := &tinyos.Session{
		ID:        "alice_session_456",
		Username:  "alice",
		IPAddress: "127.0.0.1",
		// ... other session fields
	}

	// Test that regular user session is not identified as temporary
	if auth.IsTemporaryUser(aliceSession.Username) {
		t.Error("Alice session should not be identified as temporary")
	}
}

// TestCookieSettingsForTemporaryUser tests the logic for temporary user cookie settings
func TestCookieSettingsForTemporaryUser(t *testing.T) {
	// Test that temporary tokens expire quickly
	tempToken, err := auth.GenerateTemporaryUserToken("dyson", "temp_session")
	if err != nil {
		t.Fatalf("Failed to generate temporary token: %v", err)
	}

	// Validate the token and check expiry
	claims, isUserToken, err := auth.ValidateToken(tempToken)
	if err != nil {
		t.Fatalf("Failed to validate temporary token: %v", err)
	}

	if !isUserToken {
		t.Error("Temporary token should be identified as user token")
	}

	userClaims, ok := claims.(*auth.UserClaims)
	if !ok {
		t.Error("Claims should be UserClaims")
	}

	// Check that temporary tokens have short expiry (15 minutes or less)
	maxExpiry := time.Now().Add(16 * time.Minute) // Allow 1 minute tolerance
	if userClaims.ExpiresAt.Time.After(maxExpiry) {
		t.Error("Temporary token should expire within 15 minutes")
	}

	// Verify it's marked as temporary
	if !userClaims.IsTempToken {
		t.Error("Token should be marked as temporary")
	}
}

// TestSecurityBoundaries tests various security boundaries
func TestSecurityBoundaries(t *testing.T) {
	tests := []struct {
		username     string
		shouldBeTemp bool
	}{
		{"dyson", true},
		{"DYSON", false},  // Case sensitive
		{"dyson ", false}, // Whitespace
		{" dyson", false}, // Leading whitespace
		{"alice", false},
		{"bob", false},
		{"admin", false},
		{"guest", false},
		{"", false},
	}

	for _, test := range tests {
		result := auth.IsTemporaryUser(test.username)
		if result != test.shouldBeTemp {
			t.Errorf("IsTemporaryUser('%s') = %v, expected %v", test.username, result, test.shouldBeTemp)
		}
	}
}

// BenchmarkTokenGeneration benchmarks token generation performance
func BenchmarkTokenGeneration(b *testing.B) {
	b.Run("RegularToken", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := auth.GenerateUserToken("testuser", "test_session")
			if err != nil {
				b.Fatalf("Token generation failed: %v", err)
			}
		}
	})

	b.Run("TemporaryToken", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := auth.GenerateTemporaryUserToken("dyson", "temp_session")
			if err != nil {
				b.Fatalf("Temporary token generation failed: %v", err)
			}
		}
	})
}

// BenchmarkTokenValidation benchmarks token validation performance
func BenchmarkTokenValidation(b *testing.B) {
	// Generate tokens for benchmarking
	regularToken, _ := auth.GenerateUserToken("testuser", "test_session")
	tempToken, _ := auth.GenerateTemporaryUserToken("dyson", "temp_session")

	b.Run("RegularTokenValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err := auth.ValidateToken(regularToken)
			if err != nil {
				b.Fatalf("Token validation failed: %v", err)
			}
		}
	})

	b.Run("TemporaryTokenValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _, err := auth.ValidateToken(tempToken)
			if err != nil {
				b.Fatalf("Temporary token validation failed: %v", err)
			}
		}
	})
}

// TestMain sets up and tears down test environment
func TestMain(m *testing.M) {
	// Setup test environment - no HTTP routes needed for unit tests

	// Run tests
	code := m.Run()

	// Cleanup - tests completed
	if code != 0 {
		// Some tests failed
	}
}
