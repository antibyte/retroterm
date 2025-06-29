package terminal

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/antibyte/retroterm/pkg/shared"
)

// Konstanten für Client-Management
const (
	MaxClientsDefault = 100 // Maximale Anzahl gleichzeitiger Clients
)

// RateLimitInfo speichert Rate-Limiting-Informationen pro IP
type RateLimitInfo struct {
	requests  int
	lastReset time.Time
}

// ClientManager verwaltet Client-Verbindungen mit Session-IDs
type ClientManager struct {
	clients    map[string]*Client        // sessionID -> Client
	rateLimits map[string]*RateLimitInfo // ipAddress -> RateLimitInfo
	mu         sync.RWMutex
}

// NewClientManager erstellt einen neuen ClientManager
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients:    make(map[string]*Client),
		rateLimits: make(map[string]*RateLimitInfo),
	}
}

// AddClient fügt einen neuen Client hinzu
func (cm *ClientManager) AddClient(sessionID string, client *Client) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.clients[sessionID] = client
	log.Printf("[CLIENT-MANAGER] Client added for session %s", sessionID)
}

// RemoveClient entfernt einen Client
func (cm *ClientManager) RemoveClient(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if client, exists := cm.clients[sessionID]; exists {
		// Channel sicher schließen
		select {
		case <-client.send:
			// Channel bereits geschlossen
		default:
			close(client.send)
		}
		delete(cm.clients, sessionID)
		log.Printf("[CLIENT-MANAGER] Client removed for session %s", sessionID)
	}
}

// TransferClient moves a client from one session ID to another without closing the connection
func (cm *ClientManager) TransferClient(oldSessionID, newSessionID string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if client, exists := cm.clients[oldSessionID]; exists {
		// Remove from old session without closing the channel
		delete(cm.clients, oldSessionID)

		// Add to new session
		cm.clients[newSessionID] = client

		log.Printf("[CLIENT-MANAGER] Client transferred from session %s to session %s", oldSessionID, newSessionID)
		return true
	}
	return false
}

// SendToClient sendet eine Nachricht an einen spezifischen Client
func (cm *ClientManager) SendToClient(sessionID string, message shared.Message) error {
	cm.mu.RLock()
	client, exists := cm.clients[sessionID]
	cm.mu.RUnlock()
	if !exists {
		return fmt.Errorf("client not found for session %s", sessionID)
	}

	// Message zu JSON konvertieren
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	// Channel-Status prüfen vor dem Senden
	select {
	case client.send <- jsonData:
		log.Printf("[CLIENT-MANAGER] Message sent to session %s: type=%d", sessionID, message.Type)
		return nil
	case <-time.After(time.Second):
		log.Printf("[CLIENT-MANAGER] Send timeout for session %s", sessionID)
		return fmt.Errorf("send timeout")
	}
}

// GetClientCount gibt die Anzahl der verbundenen Clients zurück
func (cm *ClientManager) GetClientCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return len(cm.clients)
}

// HasClient prüft, ob ein Client für die Session existiert
func (cm *ClientManager) HasClient(sessionID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, exists := cm.clients[sessionID]
	return exists
}

// CheckRateLimit prüft das Rate-Limiting für eine IP-Adresse
func (cm *ClientManager) CheckRateLimit(ipAddress string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	now := time.Now()

	// Initialisiere Rate-Limit-Eintrag wenn nicht vorhanden
	if _, exists := cm.rateLimits[ipAddress]; !exists {
		cm.rateLimits[ipAddress] = &RateLimitInfo{
			requests:  0,
			lastReset: now,
		}
	}

	rateLimit := cm.rateLimits[ipAddress]

	// Reset Zähler wenn mehr als eine Minute vergangen ist
	if now.Sub(rateLimit.lastReset) > time.Minute {
		rateLimit.requests = 0
		rateLimit.lastReset = now
	}
	// Zähler erhöhen
	rateLimit.requests++
	// Check rate limit (200 requests per minute - increased for better UX with held keys)
	if rateLimit.requests > 200 {
		log.Printf("[SECURITY] Rate limit exceeded for IP %s: %d requests in last minute", ipAddress, rateLimit.requests)
		return fmt.Errorf("rate limit exceeded: too many requests from %s", ipAddress)
	}

	// Warning for high request rates
	if rateLimit.requests > 300 {
		log.Printf("[SECURITY] High request rate from IP %s: %d requests in last minute", ipAddress, rateLimit.requests)
	}

	return nil
}

// ValidateClientMessage validiert eine eingehende Client-Nachricht
func (cm *ClientManager) ValidateClientMessage(messageData []byte) error {
	// Einfache JSON-Validierung
	var temp interface{}
	return json.Unmarshal(messageData, &temp)
}

// UpdateClientActivity aktualisiert die letzte Aktivität eines Clients
func (cm *ClientManager) UpdateClientActivity(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if client, exists := cm.clients[sessionID]; exists {
		// Hier könnte man einen lastActivity Timestamp setzen
		// Für jetzt nur ein Log
		_ = client // Verwende client Variable um Compiler-Warnung zu vermeiden
		log.Printf("[CLIENT-MANAGER] Updated activity for session %s", sessionID)
	}
}
