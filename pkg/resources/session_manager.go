package resources

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/antibyte/retroterm/pkg/configuration"
)

// SessionResourceManager verwaltet die Ressourcen für WebSocket-Sessions
type SessionResourceManager struct {
	*TinyBASICResourceManager
	sessions      map[string]*SessionResource // SessionID -> SessionResource
	sessionsMutex sync.RWMutex
}

// SessionResource verwaltet die Ressourcen einer einzelnen Session
type SessionResource struct {
	SessionID       string
	Username        string
	IPAddress       string
	CreatedAt       time.Time
	LastActivity    time.Time
	MessageCount    int64 // Anzahl verarbeiteter Nachrichten
	MaxMessages     int64 // Maximum Nachrichten pro Minute
	ConnectionCount int   // Anzahl WebSocket-Verbindungen
	MaxConnections  int   // Maximum gleichzeitige Verbindungen
	BandwidthUsed   int64 // Verwendete Bandbreite in Bytes
	MaxBandwidth    int64 // Maximum Bandbreite pro Minute in Bytes
	RateLimitResets int   // Anzahl Rate-Limit-Zurücksetzungen
}

// NewSessionResourceManager erstellt einen neuen Session-Ressourcenmanager
func NewSessionResourceManager() *SessionResourceManager {
	return &SessionResourceManager{
		TinyBASICResourceManager: NewTinyBASICResourceManager(),
		sessions:                 make(map[string]*SessionResource),
	}
}

// RegisterSession registriert eine neue Session im Ressourcenmanager
func (srm *SessionResourceManager) RegisterSession(sessionID, username, ipAddress string) error {
	srm.sessionsMutex.Lock()
	defer srm.sessionsMutex.Unlock() // Check if session already exists - if so, just update last activity
	if existingSession, exists := srm.sessions[sessionID]; exists {
		existingSession.LastActivity = time.Now()
		log.Printf("[SESSION-RESOURCE] Session activity updated: %s (User: %s, IP: %s)",
			sessionID, username, ipAddress)
		return nil
	}

	// Prüfe maximale Sessions pro Benutzer
	userSessionCount := 0
	for _, session := range srm.sessions {
		if session.Username == username {
			userSessionCount++
		}
	}
	maxSessionsPerUser := configuration.GetInt("Security", "max_sessions_per_user", 3)
	if userSessionCount >= maxSessionsPerUser {
		return fmt.Errorf("maximum sessions per user reached for %s: %d", username, userSessionCount)
	}

	// Prüfe maximale Sessions pro IP
	ipSessionCount := 0
	for _, session := range srm.sessions {
		if session.IPAddress == ipAddress {
			ipSessionCount++
		}
	}

	maxSessionsPerIP := configuration.GetInt("Security", "max_sessions_per_ip", 5)
	if ipSessionCount >= maxSessionsPerIP {
		return fmt.Errorf("maximum sessions per IP reached for %s: %d", ipAddress, ipSessionCount)
	}
	sessionResource := &SessionResource{
		SessionID:      sessionID,
		Username:       username,
		IPAddress:      ipAddress,
		CreatedAt:      time.Now(),
		LastActivity:   time.Now(),
		MaxMessages:    int64(configuration.GetInt("Security", "rate_limit_messages", 60)),
		MaxConnections: 2,
		MaxBandwidth:   int64(configuration.GetInt("Security", "rate_limit_bandwidth", 10240)),
	}

	srm.sessions[sessionID] = sessionResource

	// Registriere Benutzer im übergeordneten Ressourcenmanager
	err := srm.RegisterUser(username)
	if err != nil {
		delete(srm.sessions, sessionID)
		return fmt.Errorf("failed to register user in resource manager: %v", err)
	}

	log.Printf("[SESSION-RESOURCE] Session registered: %s (User: %s, IP: %s)",
		sessionID, username, ipAddress)

	return nil
}

// UnregisterSession entfernt eine Session aus dem Ressourcenmanager
func (srm *SessionResourceManager) UnregisterSession(sessionID string) error {
	srm.sessionsMutex.Lock()
	defer srm.sessionsMutex.Unlock()

	session, exists := srm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	username := session.Username

	// Stoppe laufende BASIC-Programme
	srm.StopBasicExecution(username)

	// Entferne Session
	delete(srm.sessions, sessionID)

	// Prüfe, ob dies die letzte Session des Benutzers war
	hasOtherSessions := false
	for _, otherSession := range srm.sessions {
		if otherSession.Username == username {
			hasOtherSessions = true
			break
		}
	}

	// Entferne Benutzer nur, wenn keine anderen Sessions existieren
	if !hasOtherSessions {
		err := srm.UnregisterUser(username)
		if err != nil {
			log.Printf("[SESSION-RESOURCE] Warning: Failed to unregister user %s: %v", username, err)
		}
	}

	sessionDuration := time.Since(session.CreatedAt)
	log.Printf("[SESSION-RESOURCE] Session unregistered: %s (User: %s, Duration: %v, Messages: %d)",
		sessionID, username, sessionDuration, session.MessageCount)

	return nil
}

// CheckSessionLimits überprüft, ob eine Session ihre Ressourcenlimits einhält
func (srm *SessionResourceManager) CheckSessionLimits(sessionID string, messageSize int) error {
	srm.sessionsMutex.Lock()
	defer srm.sessionsMutex.Unlock()

	session, exists := srm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	now := time.Now()

	// Aktualisiere Session-Statistiken
	session.MessageCount++
	session.BandwidthUsed += int64(messageSize)
	session.LastActivity = now

	// Rate-Limiting pro Minute zurücksetzen
	if now.Sub(session.CreatedAt) > time.Minute {
		timeSinceReset := now.Sub(session.CreatedAt.Add(time.Duration(session.RateLimitResets+1) * time.Minute))
		if timeSinceReset >= 0 {
			session.MessageCount = 0
			session.BandwidthUsed = 0
			session.RateLimitResets++
		}
	}

	// Prüfe Nachrichten-Limit
	if session.MessageCount > session.MaxMessages {
		return fmt.Errorf("message rate limit exceeded for session %s: %d > %d per minute",
			sessionID, session.MessageCount, session.MaxMessages)
	}

	// Prüfe Bandbreiten-Limit
	if session.BandwidthUsed > session.MaxBandwidth {
		return fmt.Errorf("bandwidth limit exceeded for session %s: %d > %d bytes per minute",
			sessionID, session.BandwidthUsed, session.MaxBandwidth)
	}

	// Aktualisiere Benutzer-Aktivität
	srm.UpdateUserActivity(session.Username)

	return nil
}

// CreateResourceLimitedContextForSession erstellt einen Kontext mit Session-spezifischen Limits
func (srm *SessionResourceManager) CreateResourceLimitedContextForSession(sessionID string) (context.Context, error) {
	srm.sessionsMutex.RLock()
	session, exists := srm.sessions[sessionID]
	srm.sessionsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Erstelle Kontext mit Timeout basierend auf Benutzerlimits
	limits, err := srm.GetUserLimits(session.Username)
	if err != nil {
		return nil, err
	}

	ctx, _ := context.WithTimeout(context.Background(), limits.MaxExecutionTime)

	return ctx, nil
}

// GetSessionStats gibt Statistiken über alle Sessions zurück
func (srm *SessionResourceManager) GetSessionStats() map[string]interface{} {
	srm.sessionsMutex.RLock()
	defer srm.sessionsMutex.RUnlock()

	totalSessions := len(srm.sessions)
	totalMessages := int64(0)
	totalBandwidth := int64(0)
	userCounts := make(map[string]int)
	ipCounts := make(map[string]int)

	for _, session := range srm.sessions {
		totalMessages += session.MessageCount
		totalBandwidth += session.BandwidthUsed
		userCounts[session.Username]++
		ipCounts[session.IPAddress]++
	}

	// Kombiniere mit System-Statistiken
	systemStats := srm.GetSystemStats()

	return map[string]interface{}{
		"total_sessions":   totalSessions,
		"total_messages":   totalMessages,
		"total_bandwidth":  totalBandwidth,
		"unique_users":     len(userCounts),
		"unique_ips":       len(ipCounts),
		"system_stats":     systemStats,
		"basic_executions": len(srm.GetAllBasicExecutions()),
	}
}

// CleanupInactiveSessions entfernt inaktive Sessions
func (srm *SessionResourceManager) CleanupInactiveSessions(maxInactiveTime time.Duration) {
	srm.sessionsMutex.Lock()
	defer srm.sessionsMutex.Unlock()

	now := time.Now()
	inactiveSessions := []string{}

	for sessionID, session := range srm.sessions {
		if now.Sub(session.LastActivity) > maxInactiveTime {
			inactiveSessions = append(inactiveSessions, sessionID)
		}
	}

	for _, sessionID := range inactiveSessions {
		log.Printf("[SESSION-RESOURCE] Cleaning up inactive session: %s", sessionID)
		srm.UnregisterSession(sessionID)
	}

	if len(inactiveSessions) > 0 {
		log.Printf("[SESSION-RESOURCE] Cleaned up %d inactive sessions", len(inactiveSessions))
	}
}

// StartPeriodicCleanup startet die periodische Bereinigung inaktiver Sessions
func (srm *SessionResourceManager) StartPeriodicCleanup() {
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		for range ticker.C {
			srm.CleanupInactiveSessions(30 * time.Minute) // Sessions nach 30 Minuten Inaktivität entfernen
		}
	}()
}

// GetSessionResource gibt die Ressourcen-Information für eine Session zurück
func (srm *SessionResourceManager) GetSessionResource(sessionID string) (*SessionResource, error) {
	srm.sessionsMutex.RLock()
	defer srm.sessionsMutex.RUnlock()

	session, exists := srm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}
