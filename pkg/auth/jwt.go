package auth

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
)

// JWT configuration constants
const (
	// Default values - actual values are loaded from configuration
	defaultJWTSecret       = "fallback_secret_change_in_production"
	defaultTokenExpiration = 24 * time.Hour
)

// getJWTSecret retrieves the JWT secret from environment variable or configuration
func getJWTSecret() string {
	// First try environment variable
	if envSecret := os.Getenv("JWT_SECRET_KEY"); envSecret != "" {
		return envSecret
	}

	// Fallback to configuration file
	secret := configuration.GetString("JWT", "secret_key", defaultJWTSecret)
	if secret == defaultJWTSecret || secret == "ENVIRONMENT_VARIABLE_NOT_SET_FALLBACK" {
		logger.SecurityWarn("Using fallback JWT secret - set JWT_SECRET_KEY environment variable for production!")
	}
	return secret
}

// getTokenExpiration retrieves the token expiration duration from configuration
func getTokenExpiration() time.Duration {
	hours := configuration.GetInt("JWT", "token_expiration_hours", 24)
	return time.Duration(hours) * time.Hour
}

// getTemporaryTokenExpiration returns a short expiration time for temporary users
func getTemporaryTokenExpiration() time.Duration {
	// 15 minutes expiration for temporary users like "dyson"
	return 15 * time.Minute
}

// GuestClaims definiert die Ansprüche für einen Gast-JWT-Token
type GuestClaims struct {
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// UserClaims definiert die Ansprüche für einen angemeldeten Benutzer-JWT-Token
type UserClaims struct {
	SessionID   string `json:"sid"`
	Username    string `json:"username"`
	IsTempToken bool   `json:"is_temp_token,omitempty"`
	jwt.RegisteredClaims
}

// GenerateGuestToken generates a JWT token for a guest session
func GenerateGuestToken(sessionID string) (string, error) {
	// Get configuration values
	secretKey := getJWTSecret()
	tokenExpiration := getTokenExpiration()

	// Token creation time
	now := time.Now()

	// Define claims for the token
	claims := GuestClaims{
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "tinyos",
			Subject:   "guest",
			ID:        sessionID,
		},
	}
	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("token konnte nicht signiert werden: %v", err)
	}
	logger.AuthInfo("Gasttoken generiert für Session ID: %s", sessionID)
	return signedToken, nil
}

// GenerateUserToken generates a JWT token for a logged-in user session
func GenerateUserToken(sessionID, username string) (string, error) {
	// Get configuration values
	secretKey := getJWTSecret()
	tokenExpiration := getTokenExpiration()

	// Token creation time
	now := time.Now()

	// Define claims for the token
	claims := UserClaims{
		SessionID: sessionID,
		Username:  username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "tinyos",
			Subject:   username,
			ID:        sessionID,
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("token konnte nicht signiert werden: %v", err)
	}

	logger.AuthInfo("Benutzertoken generiert für Session ID: %s, Username: %s", sessionID, username)
	return signedToken, nil
}

// GenerateTemporaryUserToken generates a temporary JWT token for fictional users
// These tokens have a very short lifespan and are not persisted in the database
func GenerateTemporaryUserToken(username, sessionID string) (string, error) {
	// Get configuration values
	secretKey := getJWTSecret()
	tokenExpiration := getTemporaryTokenExpiration()

	// Token creation time
	now := time.Now()

	// Define claims for the token
	claims := UserClaims{
		SessionID:   sessionID,
		Username:    username,
		IsTempToken: true, // Mark this as a temporary token
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "tinyos",
			Subject:   username,
			ID:        sessionID,
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("token could not be signed: %v", err)
	}

	logger.AuthInfo("Temporary user token generated for session ID: %s, username: %s (expires in %v)", sessionID, username, tokenExpiration)
	return signedToken, nil
}

// IsTemporaryUser checks if a username should receive temporary tokens
func IsTemporaryUser(username string) bool {
	temporaryUsers := []string{"dyson"}
	for _, tmpUser := range temporaryUsers {
		if username == tmpUser {
			return true
		}
	}
	return false
}

// ValidateGuestToken validates a JWT token for a guest session
func ValidateGuestToken(tokenString string) (*GuestClaims, error) {
	// Get secret key from configuration
	secretKey := getJWTSecret()

	// Parse and validate token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&GuestClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Check signing algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing algorithm: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %v", err)
	}

	// Check if token is valid
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims from token
	claims, ok := token.Claims.(*GuestClaims)
	if !ok {
		return nil, fmt.Errorf("could not extract token claims")
	}

	// Check if token is expired
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}
	return claims, nil
}

// ValidateUserToken validates a JWT token for a logged-in user session
func ValidateUserToken(tokenString string) (*UserClaims, error) {
	// Get secret key from configuration
	secretKey := getJWTSecret()

	// Parse and validate token
	token, err := jwt.ParseWithClaims(
		tokenString,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			// Check signing algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing algorithm: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %v", err)
	}

	// Check if token is valid
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims from token
	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, fmt.Errorf("could not extract token claims")
	}
	// Check if token is expired
	if claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}

// ValidateToken validates a JWT token and returns either UserClaims or GuestClaims
// This function automatically detects the token type based on the subject field
// and implements special handling for temporary users like "dyson"
func ValidateToken(tokenString string) (interface{}, bool, error) {
	// Get secret key from configuration
	secretKey := getJWTSecret()

	// First, parse token to check the subject field to determine token type
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, false, fmt.Errorf("token parsing failed: %v", err)
	}

	// Extract the subject from the token to determine type
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		subject, exists := claims["sub"].(string)
		if !exists {
			return nil, false, fmt.Errorf("no subject found in token")
		}
		// If subject is "guest", it's a guest token
		if subject == "guest" {
			guestClaims, err := ValidateGuestToken(tokenString)
			return guestClaims, false, err // false = not a user token
		} else {
			// Otherwise it's a user token
			userClaims, err := ValidateUserToken(tokenString)
			return userClaims, true, err // true = is a user token
		}
	}

	return nil, false, fmt.Errorf("could not extract claims from token")
}

// ExtractTokenFromRequest extracts the JWT token from the HTTP request
// The token can be passed in the Authorization header (Bearer Token) or as a cookie
func ExtractTokenFromRequest(r *http.Request) (string, error) {
	// First try from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" { // Format: "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1], nil
		}
		return "", fmt.Errorf("invalid authorization header format")
	}

	// Next try from cookie
	cookie, err := r.Cookie("guest_token")
	if err == nil {
		return cookie.Value, nil
	}

	// Finally try from URL query parameter
	token := r.URL.Query().Get("token")
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no token found in request")
}

// RequireGuestToken ist ein Middleware für HTTP-Handler, die einen gültigen Gast-Token erfordert
func RequireGuestToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// OPTIONS-Anfrage für CORS-Preflight erlauben ohne Token-Überprüfung
		if r.Method == "OPTIONS" {
			next(w, r)
			return
		}
		// Token aus dem Request extrahieren
		tokenString, err := ExtractTokenFromRequest(r)
		if err != nil {
			logger.AuthWarn("Kein Token im Request gefunden: %v", err)
			http.Error(w, "Unbefugt: Token fehlt", http.StatusUnauthorized)
			return
		}

		// Token validieren
		claims, err := ValidateGuestToken(tokenString)
		if err != nil {
			logger.AuthWarn("Ungültiger Token: %v", err)
			http.Error(w, "Unbefugt: Ungültiger Token", http.StatusUnauthorized)
			return
		}

		// Token ist gültig, füge Claims dem Request-Kontext hinzu
		r = r.WithContext(AddClaimsToContext(r.Context(), claims))

		// An den nächsten Handler weiterleiten
		next(w, r)
	}
}
