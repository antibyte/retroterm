package terminal

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/antibyte/retroterm/pkg/auth"
	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
	"github.com/antibyte/retroterm/pkg/tinybasic"
	"github.com/antibyte/retroterm/pkg/tinyos"

	"context"

	"github.com/gorilla/websocket"
)

// Modus-Konstanten für den Terminal
const (
	ModeOS    = "os"
	ModeBasic = "basic"
	ModeChat  = "chat"
)

// Rate-Limiting Konstanten - werden jetzt aus der Konfiguration gelesen
// Siehe [Terminal] Sektion in settings.cfg

// TerminalHandler verwaltet WebSocket-Verbindungen und Terminal-Sitzungen
type TerminalHandler struct {
	os             *tinyos.TinyOS
	basicInstances map[string]*tinybasic.TinyBASIC // Map von SessionID zu TinyBASIC-Instanz
	clients        map[*Client]bool
	chatClients    map[*Client]bool // Separate Map für Chat-Clients
	mutex          sync.RWMutex     // Geändert zu RWMutex für bessere Performance
	chatMutex      sync.RWMutex     // Geändert zu RWMutex für Chat-Clients
	upgrader       websocket.Upgrader
	LogFunc        func(string, ...interface{}) // Funktion zum Logging

	// Rate-Limiting für Session-Erstellung
	sessionRequests map[string][]time.Time // IP -> Liste der Session-Anfrage-Zeitstempel
	bannedIPs       map[string]time.Time   // IP -> Ban-Zeitstempel
	rateLimitMutex  sync.Mutex             // Mutex für Rate-Limiting Maps
	// Sicherheits-Komponenten
	clientManager     *ClientManager
	jsonValidator     *JSONValidator
	securityValidator *SecurityValidator
}

// Client repräsentiert einen verbundenen WebSocket-Client
type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	handler   *TerminalHandler
	ipAddress string
	cols      int
	rows      int
	mode      string // "os" oder "basic"
	lastPong  time.Time
	sessionID string
	shutdown  chan struct{} // Channel for graceful shutdown
}

// Send sendet eine Nachricht an den Client über den send-Kanal - DEADLOCK FIX
func (c *Client) Send(message []byte) {
	// CRITICAL FIX: Check client existence without holding mutex during send operation
	// This prevents deadlocks during timeout scenarios
	clientExists := false
	if c.mode == ModeChat {
		c.handler.chatMutex.RLock()
		_, clientExists = c.handler.chatClients[c]
		c.handler.chatMutex.RUnlock()
	} else {
		c.handler.mutex.RLock()
		_, clientExists = c.handler.clients[c]
		c.handler.mutex.RUnlock()
	}

	if !clientExists {
		// Client already removed, no action needed
		return
	}

	// CRITICAL FIX: Use shorter timeout and non-blocking send with immediate cleanup scheduling
	select {
	case c.send <- message:
		// Message successfully queued
	case <-time.After(100 * time.Millisecond): // Shorter timeout to prevent deadlocks
		// Send timeout - schedule async cleanup to prevent deadlock
		logger.Warn(logger.AreaTerminal, "Send timeout for client %s, scheduling async cleanup", c.conn.RemoteAddr())

		// CRITICAL: Schedule cleanup asynchronously to prevent mutex deadlock
		go func() {
			if c.mode == ModeChat {
				c.handler.cleanupChatClient(c)
			} else {
				c.handler.cleanupClient(c)
			}
		}()
	default:
		// Channel is full but not blocked - try one more time with immediate timeout
		select {
		case c.send <- message:
			// Successfully sent on retry
		default:
			// Channel completely blocked - schedule cleanup
			logger.Warn(logger.AreaTerminal, "Send channel blocked for client %s, scheduling cleanup", c.conn.RemoteAddr())
			go func() {
				if c.mode == ModeChat {
					c.handler.cleanupChatClient(c)
				} else {
					c.handler.cleanupClient(c)
				}
			}()
		}
	}
}

// TerminalRequest repräsentiert eine Anfrage vom Client
type TerminalRequest struct {
	IsConfig      bool   `json:"isConfig,omitempty"`
	Content       string `json:"content,omitempty"`
	Cols          int    `json:"cols,omitempty"`
	Rows          int    `json:"rows,omitempty"`
	Mode          string `json:"mode,omitempty"`          // Hinzugefügt: Mode-Feld
	SessionID     string `json:"sessionId,omitempty"`     // Hinzugefügt: SessionID-Feld
	Type          int    `json:"type,omitempty"`          // Nachrichtentyp für Key-Events
	Key           string `json:"key,omitempty"`           // Taste für Key-Events
	EditorCommand string `json:"editorCommand,omitempty"` // Editor-Befehl
	EditorData    string `json:"editorData,omitempty"`    // Editor-Daten
	SuppressEcho  bool   `json:"suppressEcho,omitempty"`  // Unterdrückt lokales Echo in TELNET-Modus
}

// NewTerminalHandler erstellt einen neuen TerminalHandler
func NewTerminalHandler(os *tinyos.TinyOS) *TerminalHandler {
	h := &TerminalHandler{
		os:             os,
		basicInstances: make(map[string]*tinybasic.TinyBASIC), // Session-basierte TinyBASIC-Instanzen
		clients:        make(map[*Client]bool), chatClients: make(map[*Client]bool),
		sessionRequests:   make(map[string][]time.Time), // Rate-Limiting für Session-Anfragen
		bannedIPs:         make(map[string]time.Time),   // Gesperrte IPs
		clientManager:     NewClientManager(),           // Sicherheits-Manager
		jsonValidator:     NewJSONValidator(),           // JSON-Validator
		securityValidator: NewSecurityValidator(),       // Security-Validator
		upgrader: websocket.Upgrader{
			ReadBufferSize:  configuration.GetInt("WebSocket", "read_buffer_size", 16384),
			WriteBufferSize: configuration.GetInt("WebSocket", "write_buffer_size", 16384),
			CheckOrigin: func(r *http.Request) bool {
				// Strict Origin checking to prevent CSRF attacks
				origin := r.Header.Get("Origin")

				// Check if Origin header exists
				if origin == "" {
					log.Printf("WebSocket request without Origin header rejected")
					return false
				}

				// Get allowed origins from configuration
				allowedOriginsStr := configuration.GetString("WebSocket", "allowed_origins", "http://localhost:8080,http://127.0.0.1:8080")
				allowedOrigins := strings.Split(allowedOriginsStr, ",")

				// Trim whitespace from each origin
				for i, origin := range allowedOrigins {
					allowedOrigins[i] = strings.TrimSpace(origin)
				}

				// Check if origin is in the list of allowed origins
				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}

				// Deny access and log
				log.Printf("WebSocket request from disallowed origin rejected: %s", origin)
				return false
			},
		},
	}

	// Set the callback function for sending messages to clients
	os.SendToClientCallback = h.clientManager.SendToClient
	// Starte die Goroutine, um BASIC-Ausgaben zu verarbeiten
	go h.processBasicOutput()

	// Starte die Goroutine für den Ping-Mechanismus
	go h.pingClients()

	return h
}

// HandleWebSocket verarbeitet eingehende WebSocket-Verbindungen - SICHERHEIT ERHÖHT
func (h *TerminalHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// IP-Adresse des Clients ermitteln
	ipAddress := r.RemoteAddr
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			ipAddress = strings.TrimSpace(parts[0])
		}
	}

	log.Printf("[WEBSOCKET] New WebSocket connection attempt from %s", ipAddress)
	log.Printf("[WEBSOCKET] Request details - URL: %s, Host: %s, Origin: %s", r.URL.String(), r.Host, r.Header.Get("Origin"))

	// Prüfe, ob die IP gesperrt ist
	if h.isIPBanned(ipAddress) {
		log.Printf("[SECURITY] Verbindung von gesperrter IP abgelehnt: %s", ipAddress)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Prüfe Client-Limits bevor Upgrade
	if len(h.clients) >= MaxClientsDefault {
		log.Printf("[SECURITY] Maximale Anzahl Clients erreicht, Verbindung abgelehnt: %s", ipAddress)
		http.Error(w, "Server overloaded", http.StatusServiceUnavailable)
		return
	}

	// Validiere CSRF-Token
	csrfToken := r.URL.Query().Get("token")
	sessionToken := r.URL.Query().Get("session")
	log.Printf("[WEBSOCKET] Token validation starting for %s - token provided: %t, session provided: %t",
		ipAddress, csrfToken != "", sessionToken != "")

	if !h.validateCSRFToken(csrfToken, sessionToken, r) {
		logger.Error(logger.AreaTerminal, "Invalid CSRF token in WebSocket request from: %s", ipAddress)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	logger.Info(logger.AreaTerminal, "Token validation successful for %s, proceeding with upgrade", ipAddress)

	// WebSocket Upgrade
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(logger.AreaTerminal, "WebSocket upgrade failed for %s: %v", ipAddress, err)
		return
	}

	logger.Info(logger.AreaTerminal, "WebSocket upgrade successful for %s", ipAddress)

	// EMERGENCY DEBUG: WebSocket upgrade debug
	logger.Debug(logger.AreaTerminal, "WebSocket upgrade completed for %s", ipAddress)

	// Client mit sicherer Channel-Größe erstellen
	client := &Client{
		conn:      conn,
		send:      make(chan []byte, getMaxChannelBuffer()), // Konfigurierbare Puffergröße
		handler:   h,
		ipAddress: ipAddress,
		cols:      80,
		rows:      24,
		mode:      ModeOS,
		lastPong:  time.Now(),
		shutdown:  make(chan struct{}),
	}

	// EMERGENCY DEBUG: Client created debug
	logger.Debug(logger.AreaTerminal, "Client created for %s, starting session handling", ipAddress)

	// Session-Management mit Sicherheitsprüfungen
	if err := h.handleClientSession(client, r); err != nil {
		log.Printf("[SECURITY] Session handling failed for %s: %v", ipAddress, err)
		conn.Close()
		return
	}

	// Debug: Überprüfe, ob Session-ID korrekt gesetzt wurde
	if client.sessionID == "" {
		log.Printf("[ERROR] Client session ID is empty after session handling for %s", ipAddress)
		conn.Close()
		return
	}
	log.Printf("[WEBSOCKET] Session established for %s - SessionID: %s", ipAddress, client.sessionID) // CRITICAL: Avoid force reset during WebSocket reconnections to prevent deadlocks
	// The existing telnet session cleanup mechanisms will handle dead sessions
	if h.os != nil {
		log.Printf("[WEBSOCKET] WebSocket reconnection for session %s - skipping force reset to avoid deadlocks", client.sessionID)
		// Note: Active telnet sessions will continue to work with the new WebSocket connection
		// Dead sessions will be cleaned up by the existing health check mechanisms
	}

	// Client-Manager-Validierung (nach Session-Setup)
	h.clientManager.AddClient(client.sessionID, client)

	log.Printf("[DEBUG] Client successfully initialized with session: %s", client.sessionID)

	h.mutex.Lock()
	h.clients[client] = true
	h.mutex.Unlock()

	log.Printf("[WEBSOCKET] Client added to active connections - Total clients: %d", len(h.clients))

	// Routinen für das Lesen und Schreiben starten
	go client.readPump()
	go client.writePump()

	log.Printf("[WEBSOCKET] Read/Write pumps started for %s (Session: %s)", ipAddress, client.sessionID)

	// Send welcome message asynchronously to prevent blocking
	go func() {
		// Small delay to ensure connection is fully established
		time.Sleep(100 * time.Millisecond) // Reduced from 1500ms to 100ms
		welcomeMsg := shared.Message{Type: shared.MessageTypeText, Content: "Welcome to Skynet Systems"}
		jsonMsg, err := json.Marshal(welcomeMsg)
		if err != nil {
			log.Printf("Error marshalling welcome message: %v", err)
		} else {
			h.SendToClient(client, jsonMsg)
		}

		// Anzahl der online Benutzer anzeigen
		userCount := h.getOnlineUserCount()
		var userCountMsg string
		if userCount == 1 {
			userCountMsg = "1 user online"
		} else {
			userCountMsg = fmt.Sprintf("%d users online", userCount)
		}

		onlineMsg := shared.Message{Type: shared.MessageTypeText, Content: userCountMsg}
		jsonOnlineMsg, err := json.Marshal(onlineMsg)
		if err != nil {
			log.Printf("Error marshalling online users message: %v", err)
		} else {
			h.SendToClient(client, jsonOnlineMsg)
		}
		infoMsg := shared.Message{Type: shared.MessageTypeText, Content: "Type 'help' for help"}
		jsonInfoMsg, err := json.Marshal(infoMsg)
		if err != nil {
			log.Printf("Error marshalling info message: %v", err)
		} else {
			h.SendToClient(client, jsonInfoMsg)
		}
	}()
}

// handleClientSession verwaltet Session-Erstellung und -Validierung sicher
func (h *TerminalHandler) handleClientSession(client *Client, r *http.Request) error {
	// Zunächst prüfen, ob ein gültiger JWT-Token vorhanden ist (Header oder Cookie)
	tokenString, tokenErr := auth.ExtractTokenFromRequest(r)
	// Wenn kein Token im Header/Cookie, prüfe Query-Parameter (für WebSocket-URLs)
	if tokenErr != nil {
		tokenString = r.URL.Query().Get("token")
		if tokenString != "" {
			tokenErr = nil // Token in Query-Parameter gefunden
		}
	}
	if tokenErr == nil && tokenString != "" {
		// JWT-Token gefunden, verwende intelligente Validierung
		claims, isUserToken, err := auth.ValidateToken(tokenString)
		if err == nil {
			var sessionID string
			var username string

			if isUserToken {
				// User token
				userClaims := claims.(*auth.UserClaims)
				sessionID = userClaims.SessionID
				username = userClaims.Username

				// Special handling: For temporary tokens, ALWAYS invalidate on new WebSocket connection (browser refresh)
				if userClaims.IsTempToken {
					return h.createGuestSession(client)
				}

				// Additional check: For temporary users, even if token doesn't have IsTempToken claim,
				// still invalidate if it's a temporary user (fallback for old tokens)
				if auth.IsTemporaryUser(username) {
					return h.createGuestSession(client)
				}
			} else {
				// Guest token
				guestClaims := claims.(*auth.GuestClaims)
				sessionID = guestClaims.SessionID
			}

			client.sessionID = sessionID // Reset chess state when reconnecting to existing session via JWT token
			h.resetChessStateForSession(sessionID)

			// CRITICAL FIX: Clean up any existing telnet sessions for this session ID
			// This handles the case where a user reloads the page but telnet sessions remain active
			if h.os.IsTelnetSessionActive(sessionID) {
				logger.Info(logger.AreaTerminal, "Cleaning up existing telnet session for reconnecting session %s", sessionID)
				h.os.CleanupTelnetSessionSync(sessionID)
			}

			// Prüfen, ob Session noch im TinyOS existiert, sonst wiederherstellen
			if !h.validateSessionWithIP(sessionID, client.ipAddress) {
				// Session existiert nicht mehr, aber Token ist gültig - Session wiederherstellen
				if isUserToken {
					// Restore user session
					_, err := h.os.RestoreUserSession(sessionID, username, client.ipAddress)
					if err != nil {
						return h.createGuestSession(client)
					}
				} else {
					// Restore guest session
					_, err := h.os.CreateGuestSession(sessionID, client.ipAddress)
					if err != nil {
						return h.createGuestSession(client)
					}
				}
			}
			return nil
		}
	}

	// Kein gültiger JWT-Token - Fallback auf alte Session-ID Validierung
	requestedSessionID := r.Header.Get("X-Session-ID")
	if requestedSessionID == "" {
		// Fallback auf Query-Parameter (weniger sicher, da in Logs sichtbar)
		requestedSessionID = r.URL.Query().Get("sessionId")
	}

	// Session-ID validieren falls vorhanden
	if requestedSessionID != "" {
		if err := h.securityValidator.ValidateSessionID(requestedSessionID); err != nil {
			log.Printf("[SECURITY] Invalid session ID format from %s: %v", client.ipAddress, err)
			return h.createGuestSession(client)
		}
	}

	if requestedSessionID != "" { // Validiere vorhandene SessionID mit IP-Binding
		if h.validateSessionWithIP(requestedSessionID, client.ipAddress) {
			client.sessionID = requestedSessionID
			log.Printf("[SESSION] Existing session validated: %s for IP: %s", requestedSessionID, client.ipAddress)

			// Reset chess state when reconnecting to existing session
			// This fixes the issue where chess remains active after page reload
			h.resetChessStateForSession(requestedSessionID)
		} else {
			// Session ungültig, erstelle neue Gast-Session
			log.Printf("[SECURITY] Invalid session attempt: %s from IP: %s", requestedSessionID, client.ipAddress)
			return h.createGuestSession(client)
		}
	} else {
		// Keine SessionID, erstelle Gast-Session
		return h.createGuestSession(client)
	}

	log.Printf("[CLIENT] Session validated for: %s (Session: %s)",
		client.ipAddress, client.sessionID)

	return nil
}

// createGuestSession erstellt eine neue Gast-Session mit Rate-Limiting
func (h *TerminalHandler) createGuestSession(client *Client) error {
	// Rate-Limiting prüfen
	if !h.checkAndUpdateSessionRateLimit(client.ipAddress) {
		return errors.New("session creation rate limit exceeded")
	}

	// Neue Gast-Session erstellen
	guestSessionID := tinyos.GenerateSessionID()
	_, err := h.os.CreateGuestSession(guestSessionID, client.ipAddress)
	if err != nil {
		return fmt.Errorf("failed to create guest session: %w", err)
	}

	client.sessionID = guestSessionID

	// Session-Info an Client senden
	sessionMsg := shared.Message{
		Type:      shared.MessageTypeSession,
		SessionID: client.sessionID,
		Content:   "Guest session created",
	}

	if msgBytes, err := json.Marshal(sessionMsg); err == nil {
		select {
		case client.send <- msgBytes:
		default:
			log.Printf("[WARNING] Failed to send session message to client")
		}
	}

	log.Printf("[SESSION] New guest session created: %s for IP: %s", guestSessionID, client.ipAddress)
	return nil
}

// validateSessionWithIP validiert Session mit IP-Binding für erhöhte Sicherheit
func (h *TerminalHandler) validateSessionWithIP(sessionID, clientIP string) bool {
	// Basis-Session-Validierung
	sessionData, valid := h.os.ValidateSession(sessionID)
	if !valid {
		return false
	}

	// IP-Binding prüfen (falls implementiert)
	if sessionData != nil {
		// Hier könnte IP-Binding-Logik implementiert werden
		// Für jetzt: Session ist gültig wenn TinyOS sie validiert
		return true
	}

	return false
}

// checkAndUpdateSessionRateLimit prüft ob eine IP zu viele Session-Anfragen gestellt hat
func (h *TerminalHandler) checkAndUpdateSessionRateLimit(ipAddress string) bool {
	h.rateLimitMutex.Lock()
	defer h.rateLimitMutex.Unlock()
	now := time.Now()
	ipBanDuration := configuration.GetDuration("Terminal", "ip_ban_duration", 24*time.Hour)

	// Prüfe, ob die IP bereits gesperrt ist
	if banTime, banned := h.bannedIPs[ipAddress]; banned {
		if now.Sub(banTime) < ipBanDuration {
			log.Printf("[SECURITY] IP %s ist noch gesperrt (verbleibend: %v)", ipAddress, ipBanDuration-now.Sub(banTime))
			return false
		} else {
			// Ban ist abgelaufen, entferne aus der Liste
			delete(h.bannedIPs, ipAddress)
			delete(h.sessionRequests, ipAddress) // Lösche auch die Anfrage-Historie
			log.Printf("[SECURITY] IP-Sperre für %s ist abgelaufen", ipAddress)
		}
	}

	// Hole die bisherigen Anfragen für diese IP
	requests, exists := h.sessionRequests[ipAddress]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// Konfigurationswerte lesen
	sessionRequestTimeWindow := configuration.GetDuration("Terminal", "session_request_time_window", time.Minute)
	maxSessionRequestsPerMinute := configuration.GetInt("Terminal", "max_session_requests_per_minute", 3)

	// Entferne Anfragen, die älter als das Zeitfenster sind
	cutoff := now.Add(-sessionRequestTimeWindow)
	validRequests := make([]time.Time, 0)
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Prüfe, ob die Anzahl der Anfragen das Limit überschreitet
	if len(validRequests) >= maxSessionRequestsPerMinute {
		// Rate-Limit überschritten - sperre die IP
		h.bannedIPs[ipAddress] = now
		delete(h.sessionRequests, ipAddress) // Lösche die Anfrage-Historie
		log.Printf("[SECURITY] IP %s wurde wegen zu vieler Session-Anfragen gesperrt (%d Anfragen in %v)",
			ipAddress, len(validRequests), sessionRequestTimeWindow)
		return false
	}

	// Füge die aktuelle Anfrage hinzu
	validRequests = append(validRequests, now)
	h.sessionRequests[ipAddress] = validRequests

	log.Printf("[RATE-LIMIT] Session-Anfrage von IP %s erlaubt (%d/%d in letzter Minute)",
		ipAddress, len(validRequests), maxSessionRequestsPerMinute)
	return true
}

// isIPBanned prüft ob eine IP gesperrt ist
func (h *TerminalHandler) isIPBanned(ipAddress string) bool {
	h.rateLimitMutex.Lock()
	defer h.rateLimitMutex.Unlock()

	banTime, banned := h.bannedIPs[ipAddress]
	if !banned {
		return false
	}

	// Prüfe, ob die Sperre noch aktiv ist
	ipBanDuration := configuration.GetDuration("Terminal", "ip_ban_duration", 24*time.Hour)
	if time.Since(banTime) >= ipBanDuration {
		// Sperre ist abgelaufen, entferne aus der Liste
		delete(h.bannedIPs, ipAddress)
		delete(h.sessionRequests, ipAddress)
		return false
	}
	return true
}

// ProcessInput verarbeitet Eingaben und führt entsprechende Befehle aus
func (h *TerminalHandler) ProcessInput(input string, sessionID string) {
	// Input normalisieren
	input = strings.TrimSpace(input)

	// Leere Eingaben ignorieren
	if input == "" {
		return
	}

	// Spezial-Befehle prüfen
	switch input {
	case "__START_BASIC__": // Wichtig: Statt eine eigene Nachricht zu senden, leiten wir die Nachrichten vom cmdBasic weiter
		// Dies stellt sicher, dass die Floppy-Sound-Nachricht korrekt übermittelt wird

		// WICHTIG: Erstelle Kontext mit der SessionID statt context.Background()!
		ctx := auth.NewContextWithSessionID(context.Background(), sessionID)
		messages := h.os.ExecuteWithContext(ctx, input)
		// Falls das OS keine Nachrichten zurückgibt, zeigen wir wenigstens eine Grundinformation an
		if len(messages) == 0 {
			h.sendWebSocketMessage(shared.Message{
				Type:      shared.MessageTypeText,
				Content:   "BASIC interpreter started. Type 'exit' to return.",
				SessionID: sessionID,
			}, sessionID)
		} else { // Nachrichten an den Client senden
			for _, msg := range messages {
				msg.SessionID = sessionID // SessionID setzen
				h.sendWebSocketMessage(msg, sessionID)
			}
		}
	case "__BREAK__":
		// WICHTIG: Erstelle Kontext mit der SessionID statt context.Background()!
		ctx := auth.NewContextWithSessionID(context.Background(), sessionID)
		responses := h.os.ExecuteWithContext(ctx, input) // Antworten verarbeiten und senden
		for _, msg := range responses {
			// SessionID für die ausgehende Nachricht setzen
			msg.SessionID = sessionID // Nachricht an Client senden
			h.sendWebSocketMessage(msg, sessionID)
		}
	default:
		// Normale Befehle über TinyOS ausführen
		// WICHTIG: Erstelle Kontext mit der SessionID statt context.Background()!
		ctx := auth.NewContextWithSessionID(context.Background(), sessionID)
		responses := h.os.ExecuteWithContext(ctx, input)

		// Antworten verarbeiten und senden
		for _, msg := range responses {
			// SessionID für die ausgehende Nachricht setzen
			msg.SessionID = sessionID

			// Nachricht an Client senden
			h.sendWebSocketMessage(msg, sessionID)
		}
	}
}

// ProcessInputWithSession ist die ältere Version der Methode und wird aus Kompatibilitätsgründen beibehalten
// In neuem Code sollte ProcessInput verwendet werden
func (h *TerminalHandler) ProcessInputWithSession(input string, sessionID string) []shared.Message {
	// Debug: Log Session-ID und Input für Troubleshooting
	logger.Debug(logger.AreaTerminal, "ProcessInputWithSession called with input: '%.50s', sessionID: '%s'", input, sessionID)

	// Prüfe, ob Session-ID vorhanden ist
	if sessionID == "" {
		logger.Error(logger.AreaTerminal, "ProcessInputWithSession called with empty sessionID")
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Error: No session ID provided",
		}}
	}

	// Erstelle einen Kontext mit der SessionID
	ctx := auth.NewContextWithSessionID(context.Background(), sessionID)
	log.Printf("[DEBUG-TERMINAL] About to call ExecuteWithContext with sessionID: '%s', input: '%s'", sessionID, input)
	// Verwende TinyOS für die Verarbeitung mit dem korrekten Kontext
	result := h.os.ExecuteWithContext(ctx, input)
	log.Printf("[DEBUG-TERMINAL] ExecuteWithContext returned %d messages", len(result))
	return result
}

// ProcessMessageFromClient verarbeitet eine Nachricht vom Client
func (h *TerminalHandler) ProcessMessageFromClient(msg *shared.Message, sessionID string) []shared.Message {
	// Erstelle einen Kontext mit der SessionID
	ctx := auth.NewContextWithSessionID(context.Background(), sessionID)
	// Input mit TinyOS verarbeiten
	return h.os.ExecuteWithContext(ctx, msg.Content)
}

// ProcessBasicInput verarbeitet Eingaben im BASIC-Modus
func (h *TerminalHandler) ProcessBasicInput(client *Client, input string) {
	log.Printf("[DEBUG-PROCESS-START] ProcessBasicInput started with input: '%s'", input)

	// SICHERHEIT: Update Client-Aktivität (aber KEINE Content-Validierung für BASIC-Befehle!)
	// BASIC-Befehle wie "load", "run", "help" sind völlig normale Eingaben
	// Content-Validierung ist nur für Chat-Nachrichten notwendig
	h.clientManager.UpdateClientActivity(client.sessionID)
	log.Printf("ProcessBasicInput: %q for session %s", input, client.sessionID)
	// Hole die session-spezifische TinyBASIC-Instanz
	log.Printf("[DEBUG-BEFORE-GETBASIC] About to call getBasicInstance for session %s", client.sessionID)
	basic := h.getBasicInstance(client.sessionID)
	log.Printf("[DEBUG-BASIC-INSTANCE] Got basic instance for session %s", client.sessionID)

	// --- Spezialfall: SAY_DONE-Nachricht vom Frontend erkennen und an Interpreter weiterleiten ---
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || strings.HasPrefix(trimmed, "{") { // Prüft, ob es sich um eine leere oder JSON-ähnliche Eingabe handelt
		var reqMap map[string]interface{}
		if err := json.Unmarshal([]byte(input), &reqMap); err == nil {
			// Prüfe sowohl auf "SAY_DONE" String als auch auf type: 6 (numerisch)
			if t, ok := reqMap["type"]; ok {
				isSayDone := false
				if t == "SAY_DONE" {
					isSayDone = true
				} else if typeNum, isNum := t.(float64); isNum && (typeNum == 6.0) {
					isSayDone = true
				}

				if isSayDone {
					var sayIDFromClientInterface interface{}
					var sayIDFromClientFloat float64
					var sayIDToBasic int
					var idOk bool

					// Versuche zuerst "speechId" (neues Format), dann "sayID" (altes Format)
					sayIDFromClientInterface, idOk = reqMap["speechId"]
					if !idOk {
						sayIDFromClientInterface, idOk = reqMap["sayID"]
					}
					if !idOk {
						log.Printf("[WARN-TERMINAL] SAY_DONE vom Client empfangen, aber ohne 'speechId' oder 'sayID'. Input: %s", input)
						basic.HandleSayDone(0)
						return
					}

					sayIDFromClientFloat, idOk = sayIDFromClientInterface.(float64)
					if !idOk {
						log.Printf("[WARN-TERMINAL] SAY_DONE vom Client empfangen, aber 'speechId'/'sayID' ist kein numerischer Typ. Wert: %v, Typ: %T. Input: %s", sayIDFromClientInterface, sayIDFromClientInterface, input)
						basic.HandleSayDone(0)
						return
					}
					sayIDToBasic = int(sayIDFromClientFloat)

					basic.SetLastSayDoneID(int64(sayIDToBasic))
					basic.HandleSayDone(sayIDToBasic)
					log.Printf("[DEBUG-TERMINAL] SAY_DONE vom Client empfangen und an Interpreter weitergeleitet. (ID: %d)", sayIDToBasic)
					return
				}
			}
		}
	}
	// Die Terminal-Dimensionen an den BASIC-Interpreter übergeben
	basic.SetTerminalDimensions(client.cols, client.rows) // Spezialbefehl __BREAK__ zur Beendigung des BASIC-Programms
	log.Printf("[DEBUG-BREAK-CHECK] Checking input: '%s', length: %d", input, len(input))
	if input == "__BREAK__" {
		stopMessages := basic.StopExecution()
		log.Printf("[DEBUG-BREAK] StopExecution returned %d messages", len(stopMessages))
		// Sende die Nachrichten direkt an den Client
		for _, message := range stopMessages {
			log.Printf("[DEBUG-BREAK] Sending message: type=%d, content=%s", message.Type, message.Content)
			jsonMsg, err := json.Marshal(message)
			if err != nil {
				log.Printf("Error marshalling stop message to client: %v", err)
				continue
			}
			h.SendToClient(client, jsonMsg)
		}
		log.Printf("[DEBUG-BREAK] All stop messages sent")
		return
	}
	// Behandle die INPUT-Antwort im Basic-Modus
	if basic.IsWaitingForInput() {
		if input == "" {
			input = " "
		}
		responseChannel := make(chan []shared.Message, 1)
		go func() {
			msgs := basic.ExecuteInputResponse(input)
			responseChannel <- msgs
		}()

		select {
		case msgs := <-responseChannel:
			for _, msg := range msgs {
				jsonMsg, err := json.Marshal(msg)
				if err != nil {
					log.Printf("Error marshalling basic message to client: %v", err)
					continue
				}
				h.SendToClient(client, jsonMsg)
			}
			if basic.IsRunning() && !basic.IsWaitingForInput() {
				disableInputMsg := shared.Message{Type: shared.MessageTypeInputControl, Content: "disable"}
				jsonDisableMsg, _ := json.Marshal(disableInputMsg)
				h.SendToClient(client, jsonDisableMsg)
			}
		case <-time.After(1 * time.Second):
			errorMsg := []byte(`{"type":0,"content":"[Error] BASIC interpreter Timeout bei Verarbeitung der Eingabe"}`) // Assuming type 0 is Text
			h.SendToClient(client, errorMsg)
			log.Printf("Warning: BASIC Timeout bei Verarbeitung von INPUT: %s", input)
		}
		// WICHTIG: Nach INPUT-Response die Ausgaben verarbeiten
		h.processBasicOutputForSession(client.sessionID, basic)
		return
	}

	if basic.IsRunning() {
		warningMsg := []byte(`{"type":0,"content":"Program already running. Use __BREAK__ to stop it."}`) // Assuming type 0 is Text
		h.SendToClient(client, warningMsg)
		return
	} // Führe den BASIC-Befehl synchron aus
	basic.SetSessionID(client.sessionID)
	returnedMessages := basic.Execute(input) // Capture messages

	// WICHTIG: Nach Execute die Ausgaben aus dem OutputChan verarbeiten
	// Dies verhindert, dass der OutputChan blockiert und MESSAGE_SEND_FAILED auftritt
	h.processBasicOutputForSession(client.sessionID, basic)
	// Überprüfe, ob der EXIT-Befehl ausgeführt wurde, indem wir nach der speziellen Fehlermeldung suchen.
	// Wir gehen davon aus, dass der Interpreter ErrExit als Textnachricht mit speziellem Inhalt sendet.
	var isExitCommand bool
	for _, msg := range returnedMessages {
		if msg.Type == shared.MessageTypeText && msg.Content == tinybasic.ErrExit.Error() {
			isExitCommand = true
			break
		}
	}
	if isExitCommand {
		client.mode = ModeOS
		// BASIC Session beenden (Session-Cleanup)
		if h.os != nil {
			h.os.EndBasicSession(client.sessionID)
			log.Printf("[BASIC-SESSION] Session %s exited BASIC mode, cleaned up session tracking", client.sessionID)
		}

		// Sende Mode-Wechsel-Nachricht an das Frontend
		modeMsg := shared.Message{Type: shared.MessageTypeMode, Content: "os"}
		jsonModeMsg, _ := json.Marshal(modeMsg)
		h.SendToClient(client, jsonModeMsg)
		// Sende Bestätigungsnachricht an den Client
		confirmMsg := shared.Message{Type: shared.MessageTypeText, Content: "Back to TinyOS."}
		jsonConfirmMsg, _ := json.Marshal(confirmMsg)
		h.SendToClient(client, jsonConfirmMsg)
		// Eingabe wieder freigeben
		enableInputMsg := shared.Message{Type: shared.MessageTypeInputControl, Content: "enable"}
		jsonEnableMsg, _ := json.Marshal(enableInputMsg)
		h.SendToClient(client, jsonEnableMsg)

		return // Wichtig: Beenden nach Behandlung des EXIT-Befehls
	}

	// Sende alle Nachrichten (beinhaltet normale Ausgabe oder Fehler, die von BASIC formatiert wurden, oder synthetisierte Fehler)
	for _, message := range returnedMessages {
		jsonMsg, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshalling basic message to client: %v", err)
			continue
		}
		h.SendToClient(client, jsonMsg)

		if message.Type == shared.MessageTypeMode && message.Content == "os" {
			client.mode = ModeOS
		}
	} // Prüfe, ob BASIC-Programm läuft und sperre/entsperre Eingabe entsprechend
	isRunning := basic.IsRunning()
	isWaitingInput := basic.IsWaitingForInput()
	log.Printf("[DEBUG-TERMINAL] After MCP command: isRunning=%v, isWaitingInput=%v", isRunning, isWaitingInput)

	if isRunning {
		disableInputMsg := shared.Message{Type: shared.MessageTypeInputControl, Content: "disable"}
		jsonDisableMsg, _ := json.Marshal(disableInputMsg)
		h.SendToClient(client, jsonDisableMsg)
	} else if isWaitingInput {
		// Wenn das System auf eine Eingabe wartet (z.B. nach MCP-Befehl), aktiviere die Eingabe
		log.Printf("[DEBUG-TERMINAL] Enabling input for MCP filename")
		enableInputMsg := shared.Message{Type: shared.MessageTypeInputControl, Content: "enable"}
		jsonEnableMsg, _ := json.Marshal(enableInputMsg)
		h.SendToClient(client, jsonEnableMsg)
	}
}

// startAutorun automatically loads and runs a BASIC program for a client
func (h *TerminalHandler) startAutorun(client *Client, filename string) {
	// Get the session-specific TinyBASIC instance
	basic := h.getBasicInstance(client.sessionID) // Set callback to return to TinyOS when program ends
	basic.SetOnProgramEnd(func() {
		// Clean up graphics and restore text mode
		clearMsg := shared.Message{Type: shared.MessageTypeClear, Command: "CLS"}
		jsonClearMsg, _ := json.Marshal(clearMsg)
		h.SendToClient(client, jsonClearMsg)
		// Clear any remaining graphics
		clearGfxMsg := shared.Message{
			Type:    shared.MessageTypeGraphics,
			Command: "CLEAR_GRAPHICS",
		}
		jsonClearGfxMsg, _ := json.Marshal(clearGfxMsg)
		h.SendToClient(client, jsonClearGfxMsg)
		// Explicitly stop any music to ensure clean sound state for next autorun
		musicStopMsg := shared.Message{
			Type: shared.MessageTypeSound,
			Params: map[string]interface{}{
				"action": "music_stop",
			},
		}
		jsonMusicStopMsg, _ := json.Marshal(musicStopMsg)
		h.SendToClient(client, jsonMusicStopMsg)

		// Small delay to ensure frontend processes the stop command
		time.Sleep(100 * time.Millisecond)

		// Send an additional sound reset to ensure clean state
		soundResetMsg := shared.Message{
			Type: shared.MessageTypeSound,
			Params: map[string]interface{}{
				"action": "reset",
			},
		}
		jsonSoundResetMsg, _ := json.Marshal(soundResetMsg)
		h.SendToClient(client, jsonSoundResetMsg)

		// Clean up the BASIC instance to ensure fresh state for next autorun
		h.cleanupBasicInstanceForAutorun(client.sessionID)

		// Switch back to OS mode
		client.mode = ModeOS

		// End BASIC session
		if h.os != nil {
			h.os.EndBasicSession(client.sessionID)
			log.Printf("[BASIC-SESSION] Session %s exited BASIC mode after autorun, cleaned up session tracking", client.sessionID)
		}

		// Send mode switch message to frontend
		modeMsg := shared.Message{Type: shared.MessageTypeMode, Content: "os"}
		jsonModeMsg, _ := json.Marshal(modeMsg)
		h.SendToClient(client, jsonModeMsg)

		// Send confirmation message
		confirmMsg := shared.Message{Type: shared.MessageTypeText, Content: "Program completed. Back to TinyOS."}
		jsonConfirmMsg, _ := json.Marshal(confirmMsg)
		h.SendToClient(client, jsonConfirmMsg)

		// Re-enable input
		enableInputMsg := shared.Message{Type: shared.MessageTypeInputControl, Content: "enable"}
		jsonEnableMsg, _ := json.Marshal(enableInputMsg)
		h.SendToClient(client, jsonEnableMsg)
	})
	// Execute LOAD command first - filename must be quoted for TinyBASIC
	loadCmd := fmt.Sprintf(`LOAD "%s"`, filename)
	loadMessages := basic.Execute(loadCmd)

	// Send load messages to client
	for _, message := range loadMessages {
		jsonMsg, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshalling load message: %v", err)
			continue
		}
		h.SendToClient(client, jsonMsg)
	}

	// Process any output from the load command
	h.processBasicOutputForSession(client.sessionID, basic)

	// Small delay to ensure load is complete
	time.Sleep(100 * time.Millisecond)

	// Execute RUN command
	runMessages := basic.Execute("RUN")

	// Send run messages to client
	for _, message := range runMessages {
		jsonMsg, err := json.Marshal(message)
		if err != nil {
			log.Printf("Error marshalling run message: %v", err)
			continue
		}
		h.SendToClient(client, jsonMsg)
	}
	// Process output from the run command
	h.processBasicOutputForSession(client.sessionID, basic)

	// Note: Input control is managed by TinyBASIC itself during execution
	// Programs that need user input (like invaders.bas) will handle input control automatically
}

// cleanupBasicInstanceForAutorun removes the TinyBASIC instance for a session after autorun
// This ensures a clean state for the next autorun execution
func (h *TerminalHandler) cleanupBasicInstanceForAutorun(sessionID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if basic, exists := h.basicInstances[sessionID]; exists {
		// Stop all running programs
		basic.StopExecution()

		// Clear the output channel completely
		outputChan := basic.GetOutputChannel()
		go func() {
			for {
				select {
				case <-outputChan:
					// Discard any remaining messages
				default:
					return
				}
			}
		}()

		// Remove the instance so a fresh one is created next time
		delete(h.basicInstances, sessionID)
		log.Printf("[BASIC-AUTORUN] Cleaned up BASIC instance for session %s after autorun", sessionID)
	}
}

// processBasicOutput überwacht alle Session-basierten BASIC-Instanzen und leitet Ausgaben weiter
func (h *TerminalHandler) processBasicOutput() {
	ticker := time.NewTicker(10 * time.Millisecond) // Reduziert von 100ms auf 10ms für bessere Responsiveness
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.processAllBasicOutputs()
		}
	}
}

// processAllBasicOutputs überprüft alle aktiven BASIC-Instanzen auf Ausgaben
func (h *TerminalHandler) processAllBasicOutputs() {
	h.mutex.Lock()
	// Erstelle eine Kopie der basicInstances, um Deadlocks zu vermeiden
	instancesCopy := make(map[string]*tinybasic.TinyBASIC)
	for sessionID, basic := range h.basicInstances {
		instancesCopy[sessionID] = basic
	}
	h.mutex.Unlock()

	// Verarbeite Ausgaben für jede Instanz
	for sessionID, basic := range instancesCopy {
		h.processBasicOutputForSession(sessionID, basic)
	}
}

// processBasicOutputForSession verarbeitet Ausgaben für eine spezifische Session
func (h *TerminalHandler) processBasicOutputForSession(sessionID string, basic *tinybasic.TinyBASIC) {
	outputChan := basic.GetOutputChannel()
	// Verarbeite maximal 100 Nachrichten pro Aufruf um Backlog zu vermeiden
	messagesProcessed := 0
	maxMessagesPerCycle := 100

	// Nicht-blockierendes Lesen von Nachrichten
	for messagesProcessed < maxMessagesPerCycle {
		select {
		case msg := <-outputChan:
			h.sendBasicOutputToClient(sessionID, msg)
			messagesProcessed++
		default:
			// Keine weiteren Nachrichten verfügbar
			return
		}
	}
}

// sendBasicOutputToClient sendet BASIC-Ausgabe an den entsprechenden Client
func (h *TerminalHandler) sendBasicOutputToClient(sessionID string, msg shared.Message) {
	// Sicherheitsüberprüfung: Stelle sicher, dass die SessionID in der Nachricht korrekt ist
	if msg.SessionID != "" && msg.SessionID != sessionID {
		log.Printf("[CRITICAL] Session ID mismatch! Expected: %s, Got: %s. Dropping message for security.", sessionID, msg.SessionID)
		return
	}

	// Setze die SessionID in der Nachricht explizit auf die erwartete SessionID
	msg.SessionID = sessionID

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshalling basic output message: %v", err)
		return
	}
	// Finde den Client mit der passenden SessionID im BASIC-Modus
	h.mutex.Lock()
	var targetClient *Client
	clientsChecked := 0
	for client := range h.clients {
		clientsChecked++
		if client.sessionID == sessionID && client.mode == ModeBasic {
			targetClient = client
			break
		}
	}
	h.mutex.Unlock() // Mutex sofort freigeben nach Client-Suche

	// Nachricht senden außerhalb des Mutex
	if targetClient != nil {
		targetClient.Send(jsonMsg)
	} else {
		log.Printf("[CRITICAL-WARNING] No BASIC client found for session %s (checked %d clients total)", sessionID, clientsChecked)
	}
}

// processEditorOutputForSession verarbeitet Editor-Ausgaben für eine spezifische Session
func (h *TerminalHandler) processEditorOutputForSession(sessionID string, outputChan <-chan shared.Message) {
	log.Printf("[EDITOR-OUTPUT] processEditorOutputForSession called for session %s", sessionID)

	// Verarbeite maximal 100 Nachrichten pro Aufruf um Backlog zu vermeiden
	messagesProcessed := 0
	maxMessagesPerCycle := 100

	// Nicht-blockierendes Lesen von Nachrichten
	for messagesProcessed < maxMessagesPerCycle {
		select {
		case msg := <-outputChan:
			log.Printf("[EDITOR-OUTPUT] Received message from editor output channel: type=%d, command=%s",
				msg.Type, msg.EditorCommand)

			// Enhance robustness for start and render messages
			if msg.Type == shared.MessageTypeEditor {
				if msg.EditorCommand == "start" || msg.EditorCommand == "render" {
					log.Printf("[EDITOR-OUTPUT] Processing special editor command: %s", msg.EditorCommand)
					// Extra logging for important commands
					if msg.Params != nil {
						if filename, ok := msg.Params["filename"].(string); ok {
							log.Printf("[EDITOR-OUTPUT] %s command for file: %s", msg.EditorCommand, filename)
						}
					}
				}
			}

			h.sendEditorOutputToClient(sessionID, msg)
			messagesProcessed++
		default:
			// Keine weiteren Nachrichten verfügbar
			if messagesProcessed == 0 {
				log.Printf("[EDITOR-OUTPUT] No messages available in output channel for session %s", sessionID)
			}
			return
		}
	}
}

// sendEditorOutputToClient sendet Editor-Ausgabe an den entsprechenden Client
func (h *TerminalHandler) sendEditorOutputToClient(sessionID string, msg shared.Message) {
	log.Printf("[EDITOR-OUTPUT] sendEditorOutputToClient called for session %s with message: %+v", sessionID, msg)

	// Suche den Client mit der entsprechenden SessionID
	h.mutex.RLock()
	var targetClient *Client
	for client := range h.clients {
		if client.sessionID == sessionID {
			targetClient = client
			break
		}
	}
	h.mutex.RUnlock()
	if targetClient != nil {
		log.Printf("[EDITOR-OUTPUT] Found target client for session %s", sessionID)
		// Nachricht an Client senden
		jsonMsg, err := json.Marshal(msg)
		if err != nil {
			log.Printf("[ERROR] Failed to marshal editor message: %v", err)
			return
		}
		log.Printf("[EDITOR-OUTPUT] Marshaled message, sending to client: %s", string(jsonMsg))
		// Verwende Client.Send direkt statt h.SendToClient
		targetClient.Send(jsonMsg)
	} else {
		log.Printf("[EDITOR-OUTPUT] WARNING: No target client found for session %s", sessionID)
	}
}

// SendToClient sendet eine Nachricht an einen bestimmten Client - DEADLOCK FIX
func (h *TerminalHandler) SendToClient(client *Client, message []byte) {
	// Safety checks
	if client == nil {
		logger.Error(logger.AreaTerminal, "SendToClient: Client ist nil")
		return
	}

	if client.send == nil {
		logger.Error(logger.AreaTerminal, "SendToClient: client.send Channel ist nil für Client %s", client.conn.RemoteAddr())
		return
	}

	// CRITICAL FIX: Use client's Send method which has proper deadlock prevention
	// This delegates to the improved Send method that handles timeouts safely
	client.Send(message)
}

// SendMessagesToClient sendet eine Reihe von Nachrichten an einen bestimmten Client
func (h *TerminalHandler) SendMessagesToClient(client *Client, messages []shared.Message) {
	if len(messages) == 0 {
		return
	}

	// Check for login confirmation and extract SessionID
	for _, message := range messages {
		if message.Type == 8 { // MessageTypeSession (not MessageTypeMode!)
		}
	}
	// Send each message as JSON to the client
	for _, message := range messages {
		jsonMsg, err := json.Marshal(message)
		if err != nil {
			logger.Error(logger.AreaTerminal, "Error marshalling message: %v", err)
			continue
		}

		h.SendToClient(client, jsonMsg)
		// Check for mode change messages
		if message.Type == shared.MessageTypeMode {
			if message.Content == "basic" {
				client.mode = "basic"
			} else if strings.HasPrefix(message.Content, "basic-autorun:") {
				// Extract filename from basic-autorun:filename
				filename := strings.TrimPrefix(message.Content, "basic-autorun:")
				client.mode = "basic"

				// Start autorun for this client's BASIC instance
				go h.startAutorun(client, filename)
			} else if message.Content == "os" {
				client.mode = "os"
			}
		}
	}

}

// Broadcast sendet eine Nachricht an alle verbundenen Terminal-Clients
func (h *TerminalHandler) Broadcast(message []byte) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for client := range h.clients {
		h.SendToClient(client, message) // Nutzt SendToClient für Fehlerbehandlung
	}
}

// BroadcastChat sendet eine Nachricht an alle verbundenen Chat-Clients
func (h *TerminalHandler) BroadcastChat(message []byte) {
	h.chatMutex.Lock()
	defer h.chatMutex.Unlock()
	for client := range h.chatClients {
		h.SendToClient(client, message) // Nutzt SendToClient für Fehlerbehandlung
	}
}

// pingClients sendet alle 30 Sekunden einen Ping an alle verbundenen Clients
// und überprüft, ob sie innerhalb von 10 Sekunden antworten
func (h *TerminalHandler) pingClients() {
	// Verwende einen längeren Ping-Intervall von 50 Sekunden anstatt 30 Sekunden
	// Dies verhindert Konflikte mit dem pingPeriod in websocket.go
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		h.mutex.Lock()
		clientsToCheck := make([]*Client, 0, len(h.clients))
		for client := range h.clients {
			// Chat-Clients überspringen, da sie ihren eigenen Ping/Pong-Mechanismus haben
			if client.mode != ModeChat {
				clientsToCheck = append(clientsToCheck, client)
			}
		}
		h.mutex.Unlock()
		// Check each client outside the mutex lock to avoid blocking
		for _, client := range clientsToCheck {
			// Use the lastPong time from the client without overriding it
			// The pong handler in readPump already updates lastPong

			// Updated: Use the new timeout value from configuration (120 seconds + buffer)
			// This ensures that a client is not prematurely marked as disconnected
			if time.Since(client.lastPong) > 130*time.Second { // Slightly higher than config pong_timeout
				logger.Warn(logger.AreaWebSocket, "No pong response from client %s for more than 130 seconds, disconnecting", client.conn.RemoteAddr())
				h.mutex.Lock()
				h.cleanupClient(client)
				h.mutex.Unlock()
			}
		}
	}
}

// cleanupClient bereinigt Ressourcen eines Clients, der nicht mehr verbunden ist
func (h *TerminalHandler) cleanupClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Verwende die sichere Unsafe-Version da bereits unter Lock
	h.cleanupClientUnsafe(client)
}

// cleanupClientUnsafe bereinigt einen Client ohne zusätzliche Mutex-Locks - DEADLOCK FIX
// WARNUNG: Diese Methode muss unter h.mutex Lock aufgerufen werden!
func (h *TerminalHandler) cleanupClientUnsafe(client *Client) {
	// Client aus allen Maps entfernen
	delete(h.clients, client)

	// Chat-Client auch aus chatClients entfernen (mit separatem Mutex)
	h.chatMutex.Lock()
	delete(h.chatClients, client)
	h.chatMutex.Unlock()

	// Client-Manager informieren
	h.clientManager.RemoveClient(client.sessionID)
	// BASIC-Programm beenden falls aktiv
	if client.mode == "basic" {
		logger.Debug(logger.AreaTerminal, "Client %s disconnected, stopping BASIC program for session %s",
			client.conn.RemoteAddr(), client.sessionID)
		h.cleanupBasicInstanceUnsafe(client.sessionID)
	}

	// TinyOS über Client-Disconnection informieren (für Telnet-Cleanup)
	if h.os != nil {
		h.os.CleanupSessionResources(client.sessionID)
	}

	// CRITICAL FIX: Signal shutdown before closing connection to prevent race conditions
	select {
	case <-client.shutdown:
		// Already closed
	default:
		close(client.shutdown)
	}

	// CRITICAL FIX: Close connection before closing send channel to prevent write errors
	if client.conn != nil {
		client.conn.Close()
	}
	// CRITICAL FIX: Safe channel close with recovery to prevent panics
	defer func() {
		if r := recover(); r != nil {
			logger.Error(logger.AreaTerminal, "Panic during send channel close for client %s: %v", client.conn.RemoteAddr(), r)
		}
	}()

	// Check if channel is already closed to prevent double close panic
	select {
	case <-client.send:
		// Channel already closed
	default:
		close(client.send)
	}
}

// cleanupBasicInstanceUnsafe entfernt BASIC-Instanz ohne zusätzliche Locks
// WARNUNG: Diese Methode muss unter h.mutex Lock aufgerufen werden!
func (h *TerminalHandler) cleanupBasicInstanceUnsafe(sessionID string) {
	if basic, exists := h.basicInstances[sessionID]; exists {
		// Stoppe alle laufenden Programme
		basic.StopExecution()

		// Leere den OutputChannel
		outputChan := basic.GetOutputChannel()
		go func() {
			for {
				select {
				case <-outputChan:
					// Verwerfe alle verbleibenden Nachrichten
				default:
					return
				}
			}
		}()

		delete(h.basicInstances, sessionID)
	}

	// BASIC Session aus Tracking entfernen
	if h.os != nil {
		h.os.EndBasicSession(sessionID)
		log.Printf("[BASIC-SESSION] Session %s cleaned up from BASIC tracking", sessionID)
	}
}

// getBasicInstance gibt die TinyBASIC-Instanz für eine Session zurück oder erstellt eine neue
func (h *TerminalHandler) getBasicInstance(sessionID string) *tinybasic.TinyBASIC {
	h.mutex.Lock()

	// Prüfe, ob bereits eine Instanz für diese Session existiert
	if basic, exists := h.basicInstances[sessionID]; exists {
		h.mutex.Unlock() // WICHTIG: Mutex früh freigeben!
		return basic
	}

	// Erstelle eine neue TinyBASIC-Instanz für diese Session
	basic := tinybasic.NewTinyBASIC(h.os)
	basic.SetSessionID(sessionID)
	h.basicInstances[sessionID] = basic
	h.mutex.Unlock() // WICHTIG: Mutex früh freigeben!

	return basic
}

// cleanupBasicInstance entfernt die TinyBASIC-Instanz für eine Session
func (h *TerminalHandler) cleanupBasicInstance(sessionID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if basic, exists := h.basicInstances[sessionID]; exists {
		// Stoppe alle laufenden Programme
		basic.StopExecution()

		// Leere den OutputChannel vollständig
		outputChan := basic.GetOutputChannel()
		go func() {
			for {
				select {
				case <-outputChan:
					// Verwerfe alle verbleibenden Nachrichten
				default:
					return
				}
			}
		}()

		delete(h.basicInstances, sessionID)

	}

	// BASIC Session aus dem Tracking entfernen (Session-Cleanup bei Disconnect)
	if h.os != nil {
		h.os.EndBasicSession(sessionID)
		log.Printf("[BASIC-SESSION] Session %s disconnected, cleaned up BASIC session tracking", sessionID)
	}
}

// validateCSRFToken validates CSRF tokens or JWT tokens
func (h *TerminalHandler) validateCSRFToken(csrfToken, sessionToken string, r *http.Request) bool {
	// Detailed logging for diagnostic purposes
	logger.Debug(logger.AreaAuth, "Starting token validation for %s", r.RemoteAddr)
	logger.Debug(logger.AreaAuth, "Request Host: %s, User-Agent: %s", r.Host, r.UserAgent())

	// If no token was provided, deny access
	if csrfToken == "" {
		logger.SecurityWarn("No token provided in request from %s", r.RemoteAddr)
		logger.Debug(logger.AreaAuth, "Request URL: %s, Method: %s", r.URL.String(), r.Method)
		return false
	}

	// Log token length and type for diagnostic purposes
	tokenLength := len(csrfToken)
	tokenPrefix := ""
	if tokenLength > 10 {
		tokenPrefix = csrfToken[:10]
	} else {
		tokenPrefix = csrfToken
	}
	logger.Debug(logger.AreaAuth, "Token received - Length: %d, Prefix: %s...", tokenLength, tokenPrefix)

	// Check first for JWT token (starts with "eyJ" - Base64 encoded JWT header)
	if strings.HasPrefix(csrfToken, "eyJ") {
		log.Printf("[CSRF_VALIDATION] JWT token detected, validating...")

		// Das ist ein JWT-Token, verwende die auth-Validierung
		claims, isUserToken, err := auth.ValidateToken(csrfToken)
		if err != nil {
			log.Printf("[CSRF_VALIDATION] FAILED: JWT validation error from %s: %v", r.RemoteAddr, err)
			log.Printf("[CSRF_VALIDATION] JWT token was: %s...", tokenPrefix)
			return false
		}
		if claims == nil {
			log.Printf("[CSRF_VALIDATION] FAILED: JWT validation returned nil claims from %s", r.RemoteAddr)
			return false
		}

		// JWT ist gültig - logge Benutzerinformationen falls verfügbar
		if isUserToken {
			// Dies ist ein Benutzer-Token
			if userClaims, ok := claims.(*auth.UserClaims); ok && userClaims != nil {
				log.Printf("[CSRF_VALIDATION] SUCCESS: Valid JWT user token accepted from %s for user: %s", r.RemoteAddr, userClaims.Username)
			} else {
				log.Printf("[CSRF_VALIDATION] SUCCESS: Valid JWT user token accepted from %s (could not extract username)", r.RemoteAddr)
			}
		} else {
			// Dies ist ein Gast-Token
			if guestClaims, ok := claims.(*auth.GuestClaims); ok && guestClaims != nil {
				log.Printf("[CSRF_VALIDATION] SUCCESS: Valid JWT guest token accepted from %s for session: %s", r.RemoteAddr, guestClaims.SessionID)
			} else {
				log.Printf("[CSRF_VALIDATION] SUCCESS: Valid JWT guest token accepted from %s (could not extract session ID)", r.RemoteAddr)
			}
		}
		return true
	}

	// Entwicklungsmodus für localhost - akzeptiere alle Verbindungen ohne weitere Prüfung
	host := r.Host
	remoteAddr := r.RemoteAddr
	log.Printf("[CSRF_VALIDATION] Checking development mode - Host: %s, RemoteAddr: %s", host, remoteAddr)

	if strings.HasPrefix(host, "localhost:") ||
		strings.HasPrefix(host, "127.0.0.1:") ||
		strings.HasPrefix(remoteAddr, "127.0.0.1:") ||
		strings.HasPrefix(remoteAddr, "[::1]:") ||
		strings.HasPrefix(remoteAddr, "localhost:") {
		log.Printf("[CSRF_VALIDATION] SUCCESS: Development mode token validation skipped for local connection from %s", r.RemoteAddr)
		return true
	}
	// Prüfe auf spezielle Entwicklungstoken
	if csrfToken == "DEV_TOKEN" {
		log.Printf("[CSRF_VALIDATION] SUCCESS: Development token accepted from %s", r.RemoteAddr)
		return true
	}
	// Prüfe auf Chat-Token (für Chat-WebSocket-Verbindungen)
	// SECURITY WARNING: This is a development fallback token that should be replaced
	// with proper JWT tokens in production
	if csrfToken == "chat-token" {
		log.Printf("[CSRF_VALIDATION] WARNING: Chat fallback token accepted from %s - should use JWT in production", r.RemoteAddr)
		// Only allow in development mode or with additional IP validation
		if strings.HasPrefix(host, "localhost:") ||
			strings.HasPrefix(host, "127.0.0.1:") ||
			strings.HasPrefix(remoteAddr, "127.0.0.1:") ||
			strings.HasPrefix(remoteAddr, "[::1]:") ||
			strings.HasPrefix(remoteAddr, "localhost:") {
			return true
		} else {
			log.Printf("[CSRF_VALIDATION] FAILED: Chat token only allowed for local connections, rejected for %s", r.RemoteAddr)
			return false
		}
	}

	// Fallback: Token-Prüfung ohne Datenbank (temporäre Lösung)
	if csrfToken == "TERMINAL_SECURE_TOKEN" {
		log.Printf("[CSRF_VALIDATION] SUCCESS: Fallback secure token accepted from %s", r.RemoteAddr)
		return true
	}

	// Alle anderen Tokens ablehnen
	log.Printf("[CSRF_VALIDATION] FAILED: Unknown/invalid token from %s", r.RemoteAddr)
	log.Printf("[CSRF_VALIDATION] Token details - Length: %d, Full token: %s", tokenLength, csrfToken)
	log.Printf("[CSRF_VALIDATION] Request details - URL: %s, Referer: %s", r.URL.String(), r.Referer())
	return false
}

// CleanupGuestSession bereinigt die Gastbenutzersitzung
// Diese Methode wird aufgerufen, wenn ein Gastbenutzer den Browser schließt
func (h *TerminalHandler) CleanupGuestSession(w http.ResponseWriter, r *http.Request) {
	// CORS-Header setzen
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// OPTIONS-Anfrage für CORS-Preflight behandeln
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Nur POST-Methode erlauben
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed"))
		return
	}

	log.Printf("[TERMINAL] CleanupGuestSession wird aufgerufen")

	// JWT-Token aus Request extrahieren und validieren
	tokenString, err := auth.ExtractTokenFromRequest(r)
	if err != nil {
		log.Printf("[TERMINAL] Kein JWT-Token im Request gefunden: %v", err)
		http.Error(w, "Unauthorized: Token fehlt", http.StatusUnauthorized)
		return
	}

	// Token validieren
	claims, err := auth.ValidateGuestToken(tokenString)
	if err != nil {
		log.Printf("[TERMINAL] Ungültiger JWT-Token: %v", err)
		http.Error(w, "Unauthorized: Ungültiger Token", http.StatusUnauthorized)
		return
	}

	// Session-ID aus Token extrahieren
	sessionID := claims.SessionID
	log.Printf("[TERMINAL] Gastsitzung mit Session-ID %s wird bereinigt", sessionID)

	// Benutzer überprüfen - für Gäste ist os.currentUser == ""
	username := h.os.Username()
	if username != "" {
		// Wenn ein Benutzer angemeldet ist, keine Aktion erforderlich
		log.Printf("[TERMINAL] CleanupGuestSession: Angemeldeter Benutzer %s erkannt, keine Bereinigung nötig", username)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("No cleanup needed for logged in user"))
		return
	}

	// Führe die Bereinigung durch
	err = h.os.CleanupGuestSession()
	if err != nil {
		log.Printf("[TERMINAL] Fehler bei der Gast-Session-Bereinigung: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to cleanup guest session"))
		return
	} // Erfolgreiche Antwort
	log.Printf("[TERMINAL] Gast-Session erfolgreich bereinigt")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Guest session cleaned up successfully"))
}

// sendWebSocketMessage sendet eine WebSocket-Nachricht an den Client
func (h *TerminalHandler) sendWebSocketMessage(msg shared.Message, sessionID string) {
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshalling message to client: %v", err)
		return
	}

	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Finde den Client mit der passenden SessionID
	for client := range h.clients {
		if client.sessionID == sessionID {
			client.Send(jsonMsg)
			break
		}
	}
}

// getOnlineUserCount zählt die Anzahl der einzigartigen online Benutzer
func (h *TerminalHandler) getOnlineUserCount() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	uniqueUsers := make(map[string]bool)

	for client := range h.clients {
		if client.sessionID != "" {
			// Hole den Benutzernamen für diese Session
			username := h.os.GetUsernameForSession(client.sessionID)
			if username != "" && username != "guest" && !strings.HasPrefix(username, "guest-") {
				// Nur eingeloggte, nicht-Gast Benutzer zählen
				uniqueUsers[username] = true
			} else {
				// Für Gäste verwenden wir die SessionID als eindeutigen Identifier
				// um doppelte Zählung zu vermeiden
				uniqueUsers["guest_"+client.sessionID] = true
			}
		}
	}

	return len(uniqueUsers)
}

// resetChessStateForSession resets the chess game state for a session
// This fixes the issue where chess remains active after page reload
func (h *TerminalHandler) resetChessStateForSession(sessionID string) {
	if sessionID == "" {
		return
	}
	// Reset chess state in TinyOS session
	h.os.ResetChessStateForSession(sessionID)
}
