package auth

import (
	"context"
)

// Schlüsselkonstante für die Session-ID im Kontext
type contextKey string

const (
	sessionIDKey contextKey = "session_id"
	claimsKey    contextKey = "jwt_claims"
)

// NewContextWithSessionID erstellt einen neuen Kontext, der die Session-ID enthält
func NewContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDKey, sessionID)
}

// GetSessionIDFromContext extrahiert die Session-ID aus dem Kontext
// Gibt die Session-ID und einen Bool zurück, der angibt, ob die ID gefunden wurde
func GetSessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionIDKey).(string)
	return sessionID, ok
}

// SessionIDFromContext extrahiert die Session-ID aus dem Kontext
// Im Gegensatz zu GetSessionIDFromContext gibt diese Funktion nur die ID zurück
// und einen leeren String, wenn keine ID vorhanden ist
func SessionIDFromContext(ctx context.Context) string {
	sessionID, ok := GetSessionIDFromContext(ctx)
	if !ok || sessionID == "" {
		return "guest" // Fallback für ungültige Sessions
	}
	return sessionID
}

// AddClaimsToContext fügt JWT-Claims zum Kontext hinzu
func AddClaimsToContext(ctx context.Context, claims *GuestClaims) context.Context {
	// Fügt die Claims zum Context hinzu
	ctx = context.WithValue(ctx, claimsKey, claims)

	// Extrahiert auch die SessionID aus den Claims und fügt sie separat hinzu
	// für einfacheren Zugriff über SessionIDFromContext
	if claims != nil {
		ctx = NewContextWithSessionID(ctx, claims.SessionID)
	}

	return ctx
}

// GetClaimsFromContext extrahiert die JWT-Claims aus dem Kontext
func GetClaimsFromContext(ctx context.Context) (*GuestClaims, bool) {
	claims, ok := ctx.Value(claimsKey).(*GuestClaims)
	return claims, ok
}
