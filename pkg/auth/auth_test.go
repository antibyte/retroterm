package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestGenerateSessionID tests session ID generation
func TestGenerateSessionID(t *testing.T) {
	sessionID1 := generateSessionID()
	sessionID2 := generateSessionID()

	// Session IDs should not be empty
	if sessionID1 == "" {
		t.Error("Session ID should not be empty")
	}

	// Session IDs should be unique
	if sessionID1 == sessionID2 {
		t.Error("Session IDs should be unique")
	}
	// Session IDs should have reasonable length (UUID format)
	if len(sessionID1) < 30 {
		t.Errorf("Session ID length should be at least 30 characters, got %d", len(sessionID1))
	}
}

// TestJWTTokenGeneration tests JWT token creation and validation
func TestJWTTokenGeneration(t *testing.T) {
	sessionID := "test-session-123"

	// Generate token
	token, err := GenerateGuestToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("Generated token should not be empty")
	}

	// Validate token
	claims, err := ValidateGuestToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, claims.SessionID)
	}
}

// TestJWTTokenExpiration tests token expiration
// Note: Since guestTokenExpiration is a constant, we test with expired tokens
func TestJWTTokenExpiration(t *testing.T) {
	sessionID := "test-session-expire"

	// Generate a valid token first
	token, err := GenerateGuestToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Token should be valid immediately
	_, err = ValidateGuestToken(token)
	if err != nil {
		t.Errorf("Token should be valid immediately: %v", err)
	}

	// Test with manually crafted expired token
	expiredClaims := GuestClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "tinyos",
			Subject:   "guest",
			ID:        sessionID,
		},
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, err := expiredToken.SignedString([]byte(getJWTSecret()))
	if err != nil {
		t.Fatalf("Failed to create expired token: %v", err)
	}

	// Expired token should be rejected
	_, err = ValidateGuestToken(expiredTokenString)
	if err == nil {
		t.Error("Expired token should be rejected")
	}
}

// TestInvalidToken tests validation of invalid tokens
func TestInvalidToken(t *testing.T) {
	testCases := []string{
		"",                                     // Empty token
		"invalid.token.here",                   // Invalid format
		"eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9", // Incomplete token
	}

	for _, token := range testCases {
		_, err := ValidateGuestToken(token)
		if err == nil {
			t.Errorf("Token %s should be invalid", token)
		}
	}
}

// TestSessionCreationHandler tests the session creation endpoint
func TestSessionCreationHandler(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/auth/session", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	HandleCreateSession(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success   bool   `json:"success"`
		SessionID string `json:"sessionId"`
		Message   string `json:"message"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	if response.SessionID == "" {
		t.Error("Session ID should not be empty")
	}
}

// TestLoginHandler tests the login endpoint
func TestLoginHandler(t *testing.T) {
	sessionID := "test-session-login"

	// Create request body
	reqBody := map[string]string{
		"sessionId": sessionID,
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	HandleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success   bool   `json:"success"`
		Token     string `json:"token"`
		SessionID string `json:"sessionId"`
		Message   string `json:"message"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	if response.Token == "" {
		t.Error("Token should not be empty")
	}

	if response.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, response.SessionID)
	}
	// Verify token is valid
	claims, err := ValidateGuestToken(response.Token)
	if err != nil {
		t.Errorf("Generated token should be valid: %v", err)
	}

	if claims.SessionID != sessionID {
		t.Errorf("Token should contain session ID %s, got %s", sessionID, claims.SessionID)
	}
}

// TestLoginHandlerInvalidRequest tests login with invalid requests
func TestLoginHandlerInvalidRequest(t *testing.T) {
	testCases := []struct {
		name         string
		requestBody  string
		expectedCode int
	}{
		{
			name:         "Empty request body",
			requestBody:  "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Invalid JSON",
			requestBody:  "invalid json",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "Missing sessionId",
			requestBody:  "{}",
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			HandleLogin(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, w.Code)
			}
		})
	}
}

// TestTokenValidationHandler tests the token validation endpoint
func TestTokenValidationHandler(t *testing.T) {
	sessionID := "test-session-validate"

	// Generate a valid token
	token, err := GenerateGuestToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Test with Authorization header
	req := httptest.NewRequest("GET", "/api/auth/validate", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	w := httptest.NewRecorder()
	HandleTokenValidation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success   bool   `json:"success"`
		SessionID string `json:"sessionId"`
		Message   string `json:"message"`
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	if response.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, response.SessionID)
	}
}

// TestTokenValidationHandlerWithCookie tests token validation with cookie
func TestTokenValidationHandlerWithCookie(t *testing.T) {
	sessionID := "test-session-cookie"

	// Generate a valid token
	token, err := GenerateGuestToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Test with cookie - note the cookie name should match what's expected
	req := httptest.NewRequest("GET", "/api/auth/validate", nil)
	req.AddCookie(&http.Cookie{
		Name:  "guest_token", // Match the cookie name used in ExtractTokenFromRequest
		Value: token,
	})

	w := httptest.NewRecorder()
	HandleTokenValidation(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success   bool   `json:"success"`
		SessionID string `json:"sessionId"`
		Message   string `json:"message"`
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	}

	if response.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, response.SessionID)
	}
}

// TestTokenValidationHandlerInvalid tests validation with invalid tokens
func TestTokenValidationHandlerInvalid(t *testing.T) {
	testCases := []struct {
		name         string
		token        string
		expectedCode int
	}{
		{
			name:         "No token",
			token:        "",
			expectedCode: http.StatusUnauthorized,
		},
		{
			name:         "Invalid token",
			token:        "invalid.token.here",
			expectedCode: http.StatusUnauthorized,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/auth/validate", nil)
			if tc.token != "" {
				req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tc.token))
			}

			w := httptest.NewRecorder()
			HandleTokenValidation(w, req)

			if w.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, w.Code)
			}
		})
	}
}

// TestLogoutHandler tests the logout endpoint
func TestLogoutHandler(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/auth/logout", nil)

	w := httptest.NewRecorder()
	HandleLogout(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got %v", response.Success)
	} // Check that guest_token cookie is cleared
	cookies := w.Header()["Set-Cookie"]
	found := false

	for _, cookie := range cookies {
		if bytes.Contains([]byte(cookie), []byte("guest_token")) &&
			(bytes.Contains([]byte(cookie), []byte("Max-Age=-1")) || bytes.Contains([]byte(cookie), []byte("Max-Age=0"))) {
			found = true
			break
		}
	}
	if !found {
		t.Error("Logout should clear guest_token cookie")
	}
}

// TestExtractTokenFromRequest tests token extraction from different sources
func TestExtractTokenFromRequest(t *testing.T) {
	sessionID := "test-session-extract"
	token, err := GenerateGuestToken(sessionID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Test Authorization header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	extractedToken, err := ExtractTokenFromRequest(req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if extractedToken != token {
		t.Errorf("Expected token %s, got %s", token, extractedToken)
	}

	// Test cookie
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.AddCookie(&http.Cookie{
		Name:  "guest_token",
		Value: token,
	})

	extractedToken2, err2 := ExtractTokenFromRequest(req2)
	if err2 != nil {
		t.Errorf("Expected no error, got %v", err2)
	}
	if extractedToken2 != token {
		t.Errorf("Expected token %s, got %s", token, extractedToken2)
	}

	// Test no token
	req3 := httptest.NewRequest("GET", "/test", nil)
	extractedToken3, err3 := ExtractTokenFromRequest(req3)
	if err3 == nil {
		t.Error("Expected error when no token present")
	}
	if extractedToken3 != "" {
		t.Errorf("Expected empty token, got %s", extractedToken3)
	}
}

// BenchmarkTokenGeneration benchmarks token generation performance
func BenchmarkTokenGeneration(b *testing.B) {
	sessionID := "benchmark-session"

	for i := 0; i < b.N; i++ {
		_, err := GenerateGuestToken(sessionID)
		if err != nil {
			b.Fatalf("Failed to generate token: %v", err)
		}
	}
}

// BenchmarkTokenValidation benchmarks token validation performance
func BenchmarkTokenValidation(b *testing.B) {
	sessionID := "benchmark-session"
	token, err := GenerateGuestToken(sessionID)
	if err != nil {
		b.Fatalf("Failed to generate token: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ValidateGuestToken(token)
		if err != nil {
			b.Fatalf("Failed to validate token: %v", err)
		}
	}
}
