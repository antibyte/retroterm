package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
)

// TinyOSInterface defines the interface for TinyOS operations needed by auth handlers
type TinyOSInterface interface {
	GetSessionUsername(sessionID string) string
}

// Global reference to TinyOS instance
var tinyOSInstance TinyOSInterface

// SetTinyOSInstance sets the TinyOS instance for use in auth handlers
func SetTinyOSInstance(instance TinyOSInterface) {
	tinyOSInstance = instance
}

// LoginRequest definiert die Struktur für Login-Anfragen
type LoginRequest struct {
	SessionID string `json:"sessionId"`
}

// LoginResponse definiert die Struktur für Login-Antworten
type LoginResponse struct {
	Success   bool   `json:"success"`
	Token     string `json:"token,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Message   string `json:"message"`
}

// SessionRequest definiert die Struktur für Session-Anfragen
type SessionRequest struct {
	RequestID string `json:"requestId,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

// SessionResponse definiert die Struktur für Session-Antworten
type SessionResponse struct {
	Success   bool   `json:"success"`
	SessionID string `json:"sessionId,omitempty"`
	Message   string `json:"message"`
}

// HandleLogin verarbeitet Login-Anfragen und generiert JWT-Tokens
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	// Setze CORS-Header
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	// Handle OPTIONS (Preflight) request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Nur POST-Anfragen akzeptieren
	if r.Method != "POST" {
		logger.AuthWarn("Invalid method for login: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Request Body parsen
	var loginReq LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		logger.AuthWarn("Invalid JSON in login request: %v", err)
		respondWithError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Session-ID validieren
	if loginReq.SessionID == "" {
		logger.AuthWarn("Missing session ID in login request")
		respondWithError(w, "Session ID required", http.StatusBadRequest)
		return
	}
	// Check if this session belongs to an authenticated user
	var token string
	var err error
	// Get session information from TinyOS to determine if it's a user or guest session
	if tinyOSInstance != nil {
		username := tinyOSInstance.GetSessionUsername(loginReq.SessionID)
		if username != "" && username != "guest" { // Check if this is a temporary user (like dyson) that should get a short-lived token
			if IsTemporaryUser(username) {
				// Generate temporary token for fictional users
				token, err = GenerateTemporaryUserToken(loginReq.SessionID, username)
				if err != nil {
					logger.AuthError("Failed to generate temporary JWT token for session %s (user: %s): %v", loginReq.SessionID, username, err)
					respondWithError(w, "Failed to generate token", http.StatusInternalServerError)
					return
				}
				logger.AuthInfo("Generated temporary token for fictional user: %s (session: %s)", username, loginReq.SessionID)
			} else {
				// Generate regular user token for authenticated user
				token, err = GenerateUserToken(loginReq.SessionID, username)
				if err != nil {
					logger.AuthError("Failed to generate user JWT token for session %s (user: %s): %v", loginReq.SessionID, username, err)
					respondWithError(w, "Failed to generate token", http.StatusInternalServerError)
					return
				}
				logger.AuthInfo("Generated user token for authenticated user: %s (session: %s)", username, loginReq.SessionID)
			}
		} else {
			// Generate guest token for guest session
			token, err = GenerateGuestToken(loginReq.SessionID)
			if err != nil {
				logger.AuthError("Failed to generate guest JWT token for session %s: %v", loginReq.SessionID, err)
				respondWithError(w, "Failed to generate token", http.StatusInternalServerError)
				return
			}
			logger.AuthInfo("Generated guest token for session: %s", loginReq.SessionID)
		}
	} else {
		// Fallback to guest token if TinyOS is not available
		token, err = GenerateGuestToken(loginReq.SessionID)
		if err != nil {
			logger.AuthError("Failed to generate fallback JWT token for session %s: %v", loginReq.SessionID, err)
			respondWithError(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}
		logger.AuthWarn("Generated fallback guest token for session %s (TinyOS not available)", loginReq.SessionID)
	} // Determine cookie expiration based on user type
	var cookieMaxAge int
	if tinyOSInstance != nil {
		username := tinyOSInstance.GetSessionUsername(loginReq.SessionID)
		if IsTemporaryUser(username) {
			// Temporary users get short-lived cookies
			cookieMaxAge = int(getTemporaryTokenExpiration().Seconds())
		} else {
			// Regular users get standard expiration
			cookieMaxAge = int(getTokenExpiration().Seconds())
		}
	} else {
		// Fallback to standard expiration
		cookieMaxAge = int(getTokenExpiration().Seconds())
	}

	// Cookie setzen für automatische Übertragung
	cookie := &http.Cookie{
		Name:     "guest_token",
		Value:    token,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,  // XSS-Schutz
		Secure:   false, // In Produktion auf true setzen bei HTTPS
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	// Erfolgreiche Antwort
	response := LoginResponse{
		Success:   true,
		Token:     token,
		SessionID: loginReq.SessionID,
		Message:   "Login successful",
	}

	logger.AuthInfo("JWT token generated for session: %s", loginReq.SessionID)
	json.NewEncoder(w).Encode(response)
}

// HandleTokenValidation validiert ein JWT-Token
func HandleTokenValidation(w http.ResponseWriter, r *http.Request) {
	// Setze CORS-Header
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	// Handle OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Token aus Request extrahieren
	tokenString, err := ExtractTokenFromRequest(r)
	if err != nil {
		logger.AuthWarn("No token found in validation request: %v", err)
		respondWithError(w, "Token not found", http.StatusUnauthorized)
		return
	} // Token validieren using the new ValidateToken function that handles both guest and user tokens
	claims, isUserToken, err := ValidateToken(tokenString)
	if err != nil {
		logger.AuthWarn("Token validation failed: %v", err)
		respondWithError(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Extract SessionID based on token type
	var sessionID string
	if isUserToken {
		if userClaims, ok := claims.(*UserClaims); ok {
			sessionID = userClaims.SessionID
		} else {
			logger.AuthError("Failed to cast user claims")
			respondWithError(w, "Invalid token format", http.StatusInternalServerError)
			return
		}
	} else {
		if guestClaims, ok := claims.(*GuestClaims); ok {
			sessionID = guestClaims.SessionID
		} else {
			logger.AuthError("Failed to cast guest claims")
			respondWithError(w, "Invalid token format", http.StatusInternalServerError)
			return
		}
	}

	// Erfolgreiche Validierung
	response := LoginResponse{
		Success:   true,
		SessionID: sessionID,
		Message:   "Token valid",
	}

	logger.AuthInfo("Token validated for session: %s", sessionID)
	json.NewEncoder(w).Encode(response)
}

// HandleLogout löscht das JWT-Token Cookie
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	// Setze CORS-Header
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	// Handle OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Cookie löschen
	cookie := &http.Cookie{
		Name:     "guest_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Sofort löschen
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)

	// Erfolgreiche Antwort
	response := LoginResponse{
		Success: true,
		Message: "Logout successful",
	}

	logger.AuthInfo("User logged out, token cookie cleared")
	json.NewEncoder(w).Encode(response)
}

// HandleCreateSession creates a new guest session and returns the session ID
func HandleCreateSession(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Content-Type", "application/json")

	// Handle OPTIONS request
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Only accept POST requests
	if r.Method != "POST" {
		logger.AuthWarn("Invalid method for session creation: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get client IP address
	clientIP := getClientIP(r)
	// Generate new session ID
	sessionID := generateSessionID()

	// Return session ID to client
	response := SessionResponse{
		Success:   true,
		SessionID: sessionID,
		Message:   "Session created successfully",
	}

	logger.AuthInfo("New guest session created: %s for IP: %s", sessionID, clientIP)
	json.NewEncoder(w).Encode(response)
}

// generateSessionID creates a unique session ID
func generateSessionID() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("guest_%d", time.Now().UnixNano())
	}
	return "guest_" + hex.EncodeToString(bytes)
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for load balancers/proxies)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// respondWithError sendet eine Fehlerantwort als JSON
func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	response := LoginResponse{
		Success: false,
		Message: message,
	}
	json.NewEncoder(w).Encode(response)
}
