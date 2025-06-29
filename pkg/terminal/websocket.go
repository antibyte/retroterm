package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/auth"
	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/editor"
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"

	"github.com/gorilla/websocket"
)

// WebSocket-Konfigurationskonstanten - werden jetzt aus der Konfiguration gelesen
// Siehe [Network] Sektion in settings.cfg

// Hilfsfunktionen für WebSocket-Konfigurationswerte
func getWriteWait() time.Duration {
	return configuration.GetDuration("Network", "write_wait_timeout", 10*time.Second)
}

func getPongWait() time.Duration {
	return configuration.GetDuration("Network", "pong_timeout", 90*time.Second)
}

func getPingPeriod() time.Duration {
	pongWait := getPongWait()
	return (pongWait * 9) / 10
}

func getMaxMessageSize() int64 {
	return int64(configuration.GetInt("Network", "max_message_size_kb", 64) * 1024)
}

func getMaxChannelBuffer() int {
	return configuration.GetInt("Network", "max_channel_buffer", 10000)
}

func getClientTimeout() time.Duration {
	return configuration.GetDuration("Network", "client_timeout", 30*time.Second)
}

func getMaxMessagesPerSecond() int {
	return configuration.GetInt("Network", "max_messages_per_second", 50)
}

// Chat-spezifische Konstanten
const (
	chatRoleSystem       = "system"
	chatCmdExit          = "exit" // ChatResponseTypeText entspricht shared.MessageTypeText
	ChatResponseTypeText = 0      // Normal text
	// ChatResponseTypeBeep entspricht shared.MessageTypeBeep
	ChatResponseTypeBeep = 2 // Beep sound
	// ChatResponseTypeSpeak entspricht shared.MessageTypeSpeak
	ChatResponseTypeSpeak = 3 // TTS output
	// ChatResponseTypeEvil entspricht shared.MessageTypeEvil
	ChatResponseTypeEvil = 26 // Evil effect - dramatic noise increase for MCP
)

var newline = []byte{'\n'} // Korrigiert: Zeilenumbruch für WebSocket-Nachrichten

// convertKeyToBasicFormat konvertiert JavaScript Key-Namen zu BASIC-kompatiblen Strings
func convertKeyToBasicFormat(jsKey string) string {
	switch jsKey {
	case "Escape":
		return "\x1B" // ESC-Taste
	case "ArrowUp":
		return "\x1B[A" // Pfeil nach oben
	case "ArrowDown":
		return "\x1B[B" // Pfeil nach unten
	case "ArrowRight":
		return "\x1B[C" // Pfeil nach rechts
	case "ArrowLeft":
		return "\x1B[D" // Pfeil nach links
	case "Backspace":
		return "\x7F" // Backspace/Delete (127)
	case "Delete":
		return "\x1B[3~" // Delete key
	case "Enter":
		return "\r" // Enter-Taste
	case "Space":
		return " " // Leertaste
	case "Tab":
		return "\t" // Tab-Taste
	default:
		// Für normale Zeichen, verwende das Zeichen direkt
		if len(jsKey) == 1 {
			return jsKey
		}
		// Für unbekannte Tasten, gib leeren String zurück
		return ""
	}
}

// Hilfsfunktion zum Extrahieren der Keys aus einer map[int]bool
func getKeysFromBoolMap(m map[int]bool) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ChatRequest repräsentiert eine Anfrage über den Chat-WebSocket
type ChatRequest struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
	SessionID string `json:"sessionId,omitempty"` // Für sichere Session-ID-Übertragung
}

// ChatResponse repräsentiert eine Antwort über den Chat-WebSocket
type ChatResponse struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Type    int    `json:"type"`
	Error   string `json:"error,omitempty"`
}

// HandleChatWebSocket handles WebSocket connections for chat mode - ENHANCED SECURITY
func (h *TerminalHandler) HandleChatWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get client IP address
	ipAddress := r.RemoteAddr
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 {
			ipAddress = strings.TrimSpace(parts[0])
		}
	} // Check if IP is banned
	if h.isIPBanned(ipAddress) {
		logger.SecurityWarn("Chat connection from banned IP rejected: %s", ipAddress)
		http.Error(w, "Access denied", http.StatusForbidden)
		return
	}

	// Check client limits before upgrade
	if len(h.chatClients) >= MaxClientsDefault {
		logger.SecurityWarn("Maximum chat clients reached, connection rejected: %s", ipAddress)
		http.Error(w, "Chat server overloaded", http.StatusServiceUnavailable)
		return
	}

	// Check if valid CSRF token is present in query parameters
	csrfToken := r.URL.Query().Get("token")
	sessionToken := r.URL.Query().Get("session")

	if !h.validateCSRFToken(csrfToken, sessionToken, r) {
		logger.SecurityError("Invalid CSRF token in chat WebSocket request from: %s", ipAddress)
		http.Error(w, "Unauthorized: Invalid CSRF Token", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP to WebSocket
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.WebSocketError("Chat WebSocket upgrade failed for %s: %v", ipAddress, err)
		return
	} // Get SessionID from header or query parameter (header preferred for security)
	requestedSessionID := r.Header.Get("X-Session-ID")
	if requestedSessionID == "" {
		// Fallback to query parameter (less secure as visible in logs)
		requestedSessionID = r.URL.Query().Get("sessionId")
	}

	// SECURITY: Chat can start without initial Session-ID and receive Session-ID
	// securely via WebSocket message. This enhances security.
	var username string
	requiresAuth := false

	if requestedSessionID != "" { // Session ID validation
		if err := h.securityValidator.ValidateSessionID(requestedSessionID); err != nil {
			logger.SecurityError("Invalid session ID format from %s: %v", ipAddress, err)
			errorMsg := ChatResponse{
				Role:  chatRoleSystem,
				Error: "Invalid session format. Please reconnect.",
			}
			jsonMsg, _ := json.Marshal(errorMsg)
			conn.WriteMessage(websocket.TextMessage, jsonMsg)
			conn.Close()
			return
		}

		// Check if user is logged in (based on SessionID)
		username = h.os.GetUsernameForSession(requestedSessionID)
	} else {
		// Keine Session-ID vorhanden - warte auf Auth-Nachricht über WebSocket
		requiresAuth = true
		username = "" // Temporär leer, bis Auth-Nachricht empfangen wird
	} // Benutzer-Validierung nur wenn Session-ID vorhanden, sonst später über WebSocket
	if !requiresAuth && username == "" {
		// Session ungültig
		errorMsg := ChatResponse{
			Role:  chatRoleSystem,
			Error: "You must be logged in to use the chat. Invalid session.",
		}
		jsonMsg, err := json.Marshal(errorMsg)
		if err != nil {
			conn.Close()
			return
		}
		conn.WriteMessage(websocket.TextMessage, jsonMsg)
		conn.Close()
		return
	}

	// Gast-Benutzer-Prüfung nur wenn bereits authentifiziert
	if !requiresAuth && (username == "guest" || strings.HasPrefix(username, "guest-")) {
		errorMsg := ChatResponse{
			Role:  chatRoleSystem,
			Error: "Chat is only available for registered users. Please log in with a registered account.",
		}
		jsonMsg, err := json.Marshal(errorMsg)
		if err != nil {
			conn.Close()
			return
		}
		conn.WriteMessage(websocket.TextMessage, jsonMsg)
		conn.Close()
		return
	}
	// Bann-Prüfung nur wenn bereits authentifiziert
	if !requiresAuth {
		// Prüfen, ob der Benutzer gebannt ist
		isBanned, banMsg := h.os.IsBanned(username, ipAddress)
		if isBanned {
			errorMsg := ChatResponse{
				Role:  chatRoleSystem,
				Error: banMsg,
			}
			jsonMsg, err := json.Marshal(errorMsg)
			if err != nil {
				conn.Close()
				return
			}
			conn.WriteMessage(websocket.TextMessage, jsonMsg)
			conn.Close()
			return
		}

		// Rate-Limit-Prüfung nur für authentifizierte Benutzer
		isLimited, shouldBan, isTemporarilyBlocked := h.os.CheckChatRateLimit(username, ipAddress)
		if shouldBan {
			// Benutzer bannen
			h.os.BanUserAndIP(username, ipAddress, 24*time.Hour)
			errorMsg := ChatResponse{
				Role:  chatRoleSystem,
				Error: "You have been banned for 24 hours due to abuse.",
			}
			jsonMsg, err := json.Marshal(errorMsg)
			if err != nil {
				conn.Close()
				return
			}
			conn.WriteMessage(websocket.TextMessage, jsonMsg)
			conn.Close()
			return
		}

		if isLimited || isTemporarilyBlocked {
			errorMsg := ChatResponse{
				Role:  chatRoleSystem,
				Error: "Chat rate limit exceeded. Please try again later.",
			}
			jsonMsg, err := json.Marshal(errorMsg)
			if err != nil {
				conn.Close()
				return
			}
			conn.WriteMessage(websocket.TextMessage, jsonMsg)
			conn.Close()
			return
		}
	} // Neuen Client erstellen mit sicherer Channel-Größe
	client := &Client{conn: conn,
		send:      make(chan []byte, getMaxChannelBuffer()), // Konfigurierbare Puffergröße
		handler:   h,
		ipAddress: ipAddress,
		cols:      80, // Standardwerte
		rows:      24,
		mode:      ModeChat,           // Setze den Modus auf Chat
		sessionID: requestedSessionID, // SessionID hier setzen (kann leer sein)
	}

	// Chat clients should NOT be added to clientManager
	// Only terminal clients should be in clientManager for EVIL message routing
	// Chat clients are managed separately in chatClients map

	// Client zur Chat-Client-Liste hinzufügen
	h.chatMutex.Lock()
	h.chatClients[client] = true
	h.chatMutex.Unlock()

	// Chat-Willkommensnachricht - unterschiedlich je nach Auth-Status
	var welcomeContent string
	if requiresAuth {
		welcomeContent = "Please authenticate to continue..."
	} else {
		welcomeContent = "Connection established. MCP ready to assist."
	}

	welcomeMsg1 := ChatResponse{
		Role:    chatRoleSystem,
		Content: welcomeContent,
		Type:    ChatResponseTypeText,
	}
	jsonMsg1, err := json.Marshal(welcomeMsg1)
	if err != nil {
		// Verbindung schließen, da die Initialisierung fehlschlägt
		h.cleanupChatClient(client)
		return
	}
	// Senden über client.Send, um korrekte Behandlung im writePump sicherzustellen
	client.Send(jsonMsg1)

	// Routinen für das Lesen und Schreiben starten
	go h.chatReadPump(client)
	go client.writePump() // writePump ist generisch und kann wiederverwendet werden
}

// cleanupChatClient entfernt einen Client aus der Chat-Liste und schließt die Verbindung - DEADLOCK FIX
func (h *TerminalHandler) cleanupChatClient(client *Client) {
	// CRITICAL FIX: Close connection before removing from map to prevent race conditions
	if client.conn != nil {
		client.conn.Close()
	}

	// CRITICAL FIX: Signal shutdown to writePump to prevent goroutine leaks
	select {
	case <-client.shutdown:
		// Already closed
	default:
		close(client.shutdown)
	}

	// Remove from chat clients map with proper locking
	h.chatMutex.Lock()
	delete(h.chatClients, client)
	h.chatMutex.Unlock()

	// CRITICAL FIX: Safe channel close with recovery
	defer func() {
		if r := recover(); r != nil {
			logger.Error(logger.AreaTerminal, "Panic during chat client cleanup: %v", r)
		}
	}()

	// Check if send channel is already closed to prevent double close panic
	if client.send != nil {
		select {
		case <-client.send:
			// Channel already closed
		default:
			close(client.send)
		}
	}
}

// chatReadPump liest Nachrichten vom Chat-WebSocket
func (h *TerminalHandler) chatReadPump(client *Client) {
	defer func() {
		h.cleanupChatClient(client)
	}()
	client.conn.SetReadLimit(getMaxMessageSize()) // Konfigurierbare Nachrichtengröße
	// Verwende Konfigurationswerte für Timeouts
	client.conn.SetReadDeadline(time.Now().Add(getPongWait()))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(getPongWait()))
		client.lastPong = time.Now() // Aktualisiere lastPong auch für Chat-Clients
		return nil
	})
	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
			}
			break
		}

		// SICHERHEIT: JSON-Validierung für Chat
		sanitizedJSON, err := h.jsonValidator.ValidateAndSanitize(message)
		if err != nil {
			log.Printf("[SECURITY] Invalid chat JSON from client %s: %v", client.conn.RemoteAddr(), err)

			errorResponse := ChatResponse{
				Role:  chatRoleSystem,
				Error: "Invalid message format.",
				Type:  ChatResponseTypeText,
			}
			jsonErrorMsg, _ := json.Marshal(errorResponse)
			client.Send(jsonErrorMsg)
			continue
		} // Nachricht parsen
		var request ChatRequest
		if err := json.Unmarshal(sanitizedJSON, &request); err != nil {
			// Optional: Fehlermeldung an Client senden
			errorResponse := ChatResponse{
				Role:  chatRoleSystem,
				Error: "Invalid message format.",
				Type:  ChatResponseTypeText,
			}
			jsonErrorMsg, _ := json.Marshal(errorResponse)
			client.Send(jsonErrorMsg)
			continue
		}
		// SICHERHEIT: Auth-Nachricht verarbeiten
		if request.Type == "auth" && request.SessionID != "" {
			// Additional IP validation for auth attempts
			if h.isIPBanned(client.ipAddress) {
				log.Printf("[SECURITY] Auth attempt from banned IP: %s", client.ipAddress)
				errorResponse := ChatResponse{
					Role:  chatRoleSystem,
					Error: "Access denied.",
					Type:  ChatResponseTypeText,
				}
				jsonErrorMsg, _ := json.Marshal(errorResponse)
				client.Send(jsonErrorMsg)
				break // Close connection
			}

			// Rate limit auth attempts
			if err := h.clientManager.CheckRateLimit(client.ipAddress); err != nil {
				log.Printf("[SECURITY] Auth rate limit exceeded for IP %s: %v", client.ipAddress, err)
				errorResponse := ChatResponse{
					Role:  chatRoleSystem,
					Error: "Too many authentication attempts. Please wait.",
					Type:  ChatResponseTypeText,
				}
				jsonErrorMsg, _ := json.Marshal(errorResponse)
				client.Send(jsonErrorMsg)
				time.Sleep(2 * time.Second) // Extra delay for auth attempts
				continue
			}

			// Session-ID validieren
			if err := h.securityValidator.ValidateSessionID(request.SessionID); err != nil {
				log.Printf("[SECURITY] Invalid session ID in auth message from %s: %v", client.ipAddress, err)
				errorResponse := ChatResponse{
					Role:  chatRoleSystem,
					Error: "Invalid session format.",
					Type:  ChatResponseTypeText,
				}
				jsonErrorMsg, _ := json.Marshal(errorResponse)
				client.Send(jsonErrorMsg)
				continue
			}

			// Session-ID setzen und Benutzer authentifizieren
			client.sessionID = request.SessionID
			username := h.os.GetUsernameForSession(client.sessionID)

			if username == "" {
				errorResponse := ChatResponse{
					Role:  chatRoleSystem,
					Error: "Invalid session. Please log in.",
					Type:  ChatResponseTypeText,
				}
				jsonErrorMsg, _ := json.Marshal(errorResponse)
				client.Send(jsonErrorMsg)
				continue
			} // Gast-Benutzer prüfen
			if username == "guest" || strings.HasPrefix(username, "guest-") {
				errorResponse := ChatResponse{
					Role:  chatRoleSystem,
					Error: "Chat is only available for registered users.",
					Type:  ChatResponseTypeText,
				}
				jsonErrorMsg, _ := json.Marshal(errorResponse)
				client.Send(jsonErrorMsg)
				// Close connection for guest users
				log.Printf("[SECURITY] Guest user %s attempted to use chat, connection closed", username)
				break // Exit the readPump loop, which will close the connection
			}

			// Chat clients should NOT be added to clientManager
			// Only terminal clients should be in clientManager for EVIL message routing			// Erfolgreiche Authentifizierung
			log.Printf("[SECURITY] Chat authentication successful for user %s from IP %s", username, client.ipAddress)
			successResponse := ChatResponse{
				Role:    chatRoleSystem,
				Content: "Authentication successful. MCP ready to assist.",
				Type:    ChatResponseTypeText,
			}
			jsonSuccessMsg, _ := json.Marshal(successResponse)
			client.Send(jsonSuccessMsg)
			continue
		} // SICHERHEIT: Nur authentifizierte Clients können Chat-Nachrichten senden
		if client.sessionID == "" {
			errorResponse := ChatResponse{
				Role:  chatRoleSystem,
				Error: "Please authenticate first.",
				Type:  ChatResponseTypeText,
			}
			jsonErrorMsg, _ := json.Marshal(errorResponse)
			client.Send(jsonErrorMsg)
			continue
		}

		// SICHERHEIT: Additional guest user check for chat messages
		username := h.os.GetUsernameForSession(client.sessionID)
		if username == "guest" || strings.HasPrefix(username, "guest-") {
			errorResponse := ChatResponse{
				Role:  chatRoleSystem,
				Error: "Chat is only available for registered users.",
				Type:  ChatResponseTypeText,
			}
			jsonErrorMsg, _ := json.Marshal(errorResponse)
			client.Send(jsonErrorMsg)
			log.Printf("[SECURITY] Guest user %s attempted to send chat message, blocked", username)
			break // Close connection for guest users
		}

		// SICHERHEIT: Additional validation for chat messages
		// 1. Check message length to prevent spam
		if len(request.Content) > 1000 { // Max 1000 characters per message
			log.Printf("[SECURITY] Message too long from chat client %s: %d characters", client.conn.RemoteAddr(), len(request.Content))

			errorResponse := ChatResponse{
				Role:  chatRoleSystem,
				Error: "Message too long. Please keep messages under 1000 characters.",
				Type:  ChatResponseTypeText,
			}
			jsonErrorMsg, _ := json.Marshal(errorResponse)
			client.Send(jsonErrorMsg)
			continue
		}

		// 2. Check for suspicious patterns (basic injection prevention)
		suspiciousPatterns := []string{"<script>", "javascript:", "data:", "vbscript:", "onload=", "onerror="}
		content := strings.ToLower(request.Content)
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(content, pattern) {
				log.Printf("[SECURITY] Suspicious pattern detected in chat message from %s: %s", client.conn.RemoteAddr(), pattern)

				errorResponse := ChatResponse{
					Role:  chatRoleSystem,
					Error: "Message contains prohibited content.",
					Type:  ChatResponseTypeText,
				}
				jsonErrorMsg, _ := json.Marshal(errorResponse)
				client.Send(jsonErrorMsg)
				continue
			}
		}

		// SICHERHEIT: Content-Validierung für Chat-Nachrichten
		if err := h.securityValidator.ValidateChatContent(request.Content); err != nil {
			log.Printf("[SECURITY] Malicious content detected from chat client %s: %v", client.conn.RemoteAddr(), err)

			errorResponse := ChatResponse{
				Role:  chatRoleSystem,
				Error: "Message contains potentially harmful content and was blocked.",
				Type:  ChatResponseTypeText,
			}
			jsonErrorMsg, _ := json.Marshal(errorResponse)
			client.Send(jsonErrorMsg)
			continue
		}

		// Rate-Limiting nur für echte Chat-Nachrichten anwenden
		// Leere Nachrichten und exit-Befehle sind exempt
		isRateLimitExempt := strings.TrimSpace(request.Content) == "" ||
			strings.ToLower(request.Content) == chatCmdExit

		if !isRateLimitExempt {
			// SICHERHEIT: Rate-Limiting auch für Chat
			if err := h.clientManager.CheckRateLimit(client.ipAddress); err != nil {
				log.Printf("[SECURITY] Chat rate limit exceeded for client %s: %v", client.conn.RemoteAddr(), err)

				errorResponse := ChatResponse{
					Role:  chatRoleSystem,
					Error: "Too many requests. Please slow down.",
					Type:  ChatResponseTypeText,
				}
				jsonErrorMsg, _ := json.Marshal(errorResponse)
				client.Send(jsonErrorMsg)

				time.Sleep(1 * time.Second)
				continue
			}
		}

		if strings.TrimSpace(request.Content) == "" {
			continue
		}
		if strings.ToLower(request.Content) == chatCmdExit { // Use constant and ToLower
			log.Printf("[CHAT] User %s from IP %s initiated chat exit", h.os.GetUsernameForSession(client.sessionID), client.ipAddress)
			exitMsg := ChatResponse{
				Role:    chatRoleSystem,
				Content: "Connection to the Master Control Program terminated.",
				Type:    ChatResponseTypeText,
			}
			jsonMsg, _ := json.Marshal(exitMsg) // ignore error, safe for static struct
			client.Send(jsonMsg)                // Send has no return value
			break                               // Ends the readPump loop and triggers defer
		}
		// Die SessionID ist bereits im client-Objekt gespeichert und sollte von dort verwendet werden.		// SECURITY: Input sanitization for DeepSeek
		sanitizedContent := h.securityValidator.SanitizeForDeepSeek(request.Content)

		// Security audit log for chat messages
		chatUsername := h.os.GetUsernameForSession(client.sessionID)
		log.Printf("[CHAT-AUDIT] User %s from IP %s sent message (length: %d)", chatUsername, client.ipAddress, len(request.Content))

		messages := h.os.AskDeepSeek(sanitizedContent, client.sessionID)

		for _, msg := range messages {
			response := ChatResponse{
				Role:    "ai", // "ai" könnte auch eine Konstante sein
				Content: msg.Content,
			}
			switch msg.Type {
			case shared.MessageTypeBeep: // Korrigiert von MessageTypeSound für Beep
				response.Type = ChatResponseTypeBeep
				response.Content = "" // Content für Beep leer lassen
			case shared.MessageTypeSpeak: // Korrigiert von MessageTypeSay
				response.Type = ChatResponseTypeSpeak
				// Im Chat-Modus keine speechId erforderlich, da keine SAY_DONE-Kommunikation
			case shared.MessageTypeEvil: // MCP Evil effect - dramatic noise increase
				response.Type = ChatResponseTypeEvil
				response.Content = "" // Content für Evil effect leer lassen
			// Optional: Behandlung für shared.MessageTypeSound (nicht-Beep)
			// case shared.MessageTypeSound:
			// response.Type = ChatResponseTypeText // Oder eine andere spezifische Behandlung
			// response.Content = "[Sound played]" // Oder msg.Content, falls es Text ist
			default: // shared.MessageTypeText und andere
				response.Type = ChatResponseTypeText
			}
			jsonMsg, err := json.Marshal(response)
			if err != nil {
				continue
			}
			client.Send(jsonMsg)
		}
	}
}

// writeMessage sendet eine Nachricht an einen Client
func (c *Client) writeMessage(msg shared.Message) {
	// Special handling for sprite messages - reduced logging to prevent spam
	if msg.Type == shared.MessageTypeSprite {
		logger.Debug(logger.AreaTerminal, "Sending sprite message: Command=%s, ID=%d", msg.Command, msg.ID)
	}

	// Serialize message
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// Timeout für das Schreiben setzen
	c.conn.SetWriteDeadline(time.Now().Add(getWriteWait()))

	// Nachricht senden
	err = c.conn.WriteMessage(websocket.TextMessage, jsonBytes)
	if err != nil {
		c.conn.Close()
		return
	}

	// Debug-Ausgabe für erfolgreichen Versand (verkürzt für große Nachrichten)
	if len(jsonBytes) > 500 {
	}
}

// readPump pumpt Nachrichten vom WebSocket zum Hub - DEADLOCK FIX
func (c *Client) readPump() {
	defer func() {
		// CRITICAL FIX: Safe cleanup with recovery to prevent panics
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic in readPump cleanup for client %s: %v", c.conn.RemoteAddr(), r)
		}
		c.handler.cleanupClient(c)
	}()

	// CRITICAL FIX: Configure WebSocket reader with proper limits
	c.conn.SetReadLimit(getMaxMessageSize())
	c.conn.SetReadDeadline(time.Now().Add(getPongWait()))
	c.conn.SetPongHandler(func(string) error {
		logger.Debug(logger.AreaTerminal, "Received pong from client %s", c.conn.RemoteAddr())
		c.conn.SetReadDeadline(time.Now().Add(getPongWait()))
		c.lastPong = time.Now()
		return nil
	})

	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			// CRITICAL FIX: Log all connection errors for debugging
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
				log.Printf("[WEBSOCKET] Unexpected close error for client %s: %v", c.conn.RemoteAddr(), err)
			} else {
				log.Printf("[WEBSOCKET] Normal close for client %s: %v", c.conn.RemoteAddr(), err)
			}
			break
		}
		// Add debug logging for all incoming WebSocket messages
		logger.Debug(logger.AreaTerminal, "WebSocket message received: type=%d, length=%d, from=%s",
			messageType, len(message), c.conn.RemoteAddr())
		logger.Debug(logger.AreaTerminal, "RAW MESSAGE CONTENT: %q", string(message))
		if messageType == websocket.TextMessage {
			// Message processing for session
			logger.Debug(logger.AreaTerminal, "Processing WebSocket message for session %s", c.sessionID)

			// Prüfe, ob es sich um eine JSON-Nachricht handelt
			var testJSON map[string]interface{}
			isJSON := json.Unmarshal(message, &testJSON) == nil

			var request TerminalRequest

			if isJSON { // Spezialbehandlung für keepalive-Nachrichten mit String-Type
				if rawMap, ok := testJSON["type"].(string); ok && rawMap == "keepalive" {
					logger.Debug(logger.AreaTerminal, "DEBUG-KEEPALIVE Received keepalive from client %s", c.conn.RemoteAddr())
					// Keepalive-Nachrichten verwerfen - keine weitere Verarbeitung nötig
					continue
				} // SICHERHEIT: JSON-Validierung nur im Terminal-Modus, NICHT im BASIC-Modus oder Telnet-Modus
				// BASIC-Kommandos und Telnet-Eingaben sollen nicht durch die Sicherheitsvalidierung blockiert werden				// Check if this is a telnet session to bypass sanitization
				isTelnetSession := c.handler.os.IsTelnetSessionActive(c.sessionID)

				if c.mode != "basic" && !isTelnetSession {
					// JSON-Nachricht: Validierung und Sanitisierung anwenden (nur für normalen Terminal-Modus)
					sanitizedJSON, err := c.handler.jsonValidator.ValidateAndSanitize(message)
					if err != nil {
						log.Printf("[SECURITY] Invalid JSON from client %s: %v", c.conn.RemoteAddr(), err)

						// Sende Sicherheitswarnung an Client
						errorMsg := shared.Message{
							Type:    shared.MessageTypeText,
							Content: "Access denied: Invalid input format",
						}
						jsonMsg, _ := json.Marshal(errorMsg)
						c.Send(jsonMsg)
						continue
					}

					// JSON parsen
					err = json.Unmarshal(sanitizedJSON, &request)
					if err != nil {
						log.Printf("[SECURITY] Failed to parse sanitized JSON from client %s: %v", c.conn.RemoteAddr(), err)
						continue
					}
				} else { // BASIC-Modus oder Telnet-Modus: JSON direkt ohne Sicherheitsvalidierung parsen
					// BASIC-Kommandos und Telnet-Eingaben sollen ungehindert funktionieren
					err := json.Unmarshal(message, &request)
					if err != nil {
						logger.Warn(logger.AreaWebSocket, "Failed to parse JSON from client %s: %v", c.conn.RemoteAddr(), err)
						continue
					} // Debug for telnet input tracking
					if isTelnetSession {
						logger.Debug(logger.AreaTerminal, "Received telnet message: type=%d, content=%q, length=%d",
							request.Type, request.Content, len(request.Content))
						if request.Content != "" {
							for i, b := range []byte(request.Content) {
								logger.Debug(logger.AreaTerminal, "Telnet byte %d: 0x%02x (%d) char:'%c'", i, b, b, b)
							}
						} else {
							// Show raw JSON when content is empty to debug the issue
							logger.Debug(logger.AreaTerminal, "Empty telnet content received, raw JSON: %q", string(message))
						}
					}
				}
			} else {
				// Einfache Text-Eingabe: Als Text-Nachricht behandeln
				request = TerminalRequest{
					Type:    int(shared.MessageTypeText),
					Content: string(message),
				} // KEINE Content-Validierung für normale Terminal-Eingaben!
				// Terminal-Befehle wie "load", "run", "help" sind gültige Eingaben
				// Content-Validierung ist nur für Chat-Nachrichten notwendig
			}

			// Rate-Limiting nur für echte Benutzereingaben anwenden
			// Ausnahmen: Session-Init (type 7), Config-Updates, Key-Events, BASIC-Modus, Editor-Nachrichten, Telnet-Sessions

			// Check if this is a telnet session
			isTelnetSession := c.handler.os.IsTelnetSessionActive(c.sessionID)

			isRateLimitExempt := (request.Type == int(shared.MessageTypeSession) && request.Content == "") ||
				request.IsConfig ||
				request.Type == int(shared.MessageTypeKeyDown) ||
				request.Type == int(shared.MessageTypeKeyUp) ||
				request.Type == int(shared.MessageTypeEditor) || // Editor-Nachrichten vom Rate-Limiting ausnehmen
				c.mode == "basic" || // BASIC-Kommandos vom Rate-Limiting ausnehmen
				isTelnetSession // Telnet-Sessions vom Rate-Limiting ausnehmen

			if !isRateLimitExempt {
				// SICHERHEIT: Rate-Limiting prüfen
				if err := c.handler.clientManager.CheckRateLimit(c.ipAddress); err != nil {
					log.Printf("[SECURITY] Rate limit exceeded for client %s: %v", c.conn.RemoteAddr(), err)

					// Sende Rate-Limit-Warnung
					errorMsg := shared.Message{
						Type:    shared.MessageTypeText,
						Content: "Access denied: Too many requests",
					}
					jsonMsg, _ := json.Marshal(errorMsg)
					c.Send(jsonMsg)

					// Kurze Pause um weitere Versuche zu verlangsamen
					time.Sleep(1 * time.Second)
					continue
				}
			} // Behandlung für Session-Initialisierungsanfrage (type 7) - nur für JSON-Nachrichten
			if isJSON {
				var msgTypeRaw struct {
					Type int `json:"type"`
				}
				if err := json.Unmarshal(message, &msgTypeRaw); err == nil && msgTypeRaw.Type == int(shared.MessageTypeSession) && request.Content == "" && request.Mode == "" && request.SessionID == "" && !request.IsConfig {
					response := shared.Message{
						Type:      shared.MessageTypeSession,
						SessionID: c.sessionID,
					}
					jsonMsg, _ := json.Marshal(response)
					c.Send(jsonMsg)
					continue
				}
			} // Terminal-Konfiguration verarbeiten
			if request.IsConfig {
				// Prüfe zuerst, ob die Konfiguration gültige Werte für Spalten und Zeilen enthält
				if request.Cols <= 0 || request.Rows <= 0 {
					// Ungültige Werte, ignorieren
					continue
				}

				// Prüfe auf unsinnige Terminal-Auflösungen als Sicherheitsmaßnahme
				if request.Cols > 128 || request.Rows > 50 || request.Cols < 32 || request.Rows < 20 {
					// Sende Fehlermeldung
					errorMsg := shared.Message{
						Type:    shared.MessageTypeText,
						Content: "Access denied: Invalid terminal dimensions",
					}
					jsonMsg, _ := json.Marshal(errorMsg)
					c.Send(jsonMsg)
					// Verbindung beenden
					c.conn.Close()
					return
				} // Gültige Konfiguration speichern
				c.cols = request.Cols
				c.rows = request.Rows

				// Update terminal dimensions in TinyOS session if session ID is available
				if c.sessionID != "" {
					c.handler.os.UpdateTerminalDimensions(c.sessionID, request.Cols, request.Rows)
				}
				continue
			} // BASIC/OS-Modus und Sitzungs-ID aus der Anfrage extrahieren
			// Wenn diese Felder existieren, werden sie aktualisiert
			if request.Mode != "" {
				c.mode = request.Mode
			} // Session-ID Verarbeitung: Nur wenn explizit gesendet und nicht leer
			if request.SessionID != "" && request.SessionID != c.sessionID {
				// Check if the session belongs to a temporary user - if so, reject session restoration
				sessionData, sessionValid := c.handler.os.ValidateSession(request.SessionID)
				if sessionValid && sessionData != nil && auth.IsTemporaryUser(sessionData.Username) {
					log.Printf("[SECURITY] Blocking session restoration for temporary user '%s' via sessionId: %s", sessionData.Username, request.SessionID)
					// Create new guest session instead of restoring the temporary user session
					err := c.handler.createGuestSession(c)
					if err != nil {
						log.Printf("[ERROR] Failed to create guest session after blocking temporary user: %v", err)
					} else {
						log.Printf("[SESSION] Created new guest session %s after blocking temporary user", c.sessionID)
					}
					continue
				}

				// Validiere die neue Session-ID bevor sie übernommen wird
				if c.handler.validateSessionWithIP(request.SessionID, c.ipAddress) {
					oldSessionID := c.sessionID

					// Transfer client to new session ID without closing connection
					if oldSessionID != "" && oldSessionID != request.SessionID { // Try to transfer the client first
						if c.handler.clientManager.TransferClient(oldSessionID, request.SessionID) {
							logger.Debug(logger.AreaTerminal, "Client transferred from session %s to %s", oldSessionID, request.SessionID)
						} else {
							// Fallback: add client normally if transfer failed
							c.handler.clientManager.AddClient(request.SessionID, c)
							logger.Debug(logger.AreaTerminal, "Client added for new session %s", request.SessionID)
						}
					} else {
						// First time session ID assignment
						c.handler.clientManager.AddClient(request.SessionID, c)
						logger.Debug(logger.AreaTerminal, "Client added for session %s", request.SessionID)
					}

					// Update session ID after successful transfer/addition
					c.sessionID = request.SessionID
					logger.Debug(logger.AreaTerminal, "Session ID updated for client %s: %s", c.ipAddress, c.sessionID)
				} else {
					log.Printf("[WARNING] Invalid session ID rejected for client %s: %s", c.ipAddress, request.SessionID)
				}
			}

			// Fallback: Wenn keine Session-ID vorhanden ist, verwende die ursprüngliche Session-ID des Clients
			if c.sessionID == "" {
				log.Printf("[ERROR] Client %s has no session ID, creating new guest session", c.ipAddress)
				// Erstelle eine neue Gast-Session
				if err := c.handler.createGuestSession(c); err != nil {
					log.Printf("[ERROR] Failed to create guest session for client %s: %v", c.ipAddress, err)
					continue
				}
			} // Behandlung für Key-Events (INKEY$ Support)
			if request.Type == int(shared.MessageTypeKeyDown) {
				// Taste wurde gedrückt
				if c.mode == "basic" {
					// Hole die BASIC-Instanz für diese Session
					if basicInstance, exists := c.handler.basicInstances[c.sessionID]; exists {
						// Konvertiere spezielle Tasten
						key := convertKeyToBasicFormat(request.Key)
						basicInstance.SetKeyPressed(key)
					}
				}
				// Note: Pager input is handled in main text input processing, not here
				continue
			}

			if request.Type == int(shared.MessageTypeKeyUp) {
				// Taste wurde losgelassen
				if c.mode == "basic" {
					// Hole die BASIC-Instanz für diese Session
					if basicInstance, exists := c.handler.basicInstances[c.sessionID]; exists {
						// Konvertiere spezielle Tasten und setze nur diese spezifische Taste als losgelassen
						key := convertKeyToBasicFormat(request.Key)
						basicInstance.SetKeyReleased(key)
					}
				}
				// Note: Pager input is handled in main text input processing, not here
				continue
			}

			// CRITICAL FIX: Handle MessageTypePager for cat pager input
			if request.Type == int(shared.MessageTypePager) {
				logger.Debug(logger.AreaTerminal, "CAT PAGER MESSAGE: Processing pager input %q for session %s",
					request.Content, c.sessionID) // Process as pager input
				messages := c.handler.ProcessInputWithSession(request.Content, c.sessionID)
				// Send messages through the proper channel instead of direct WriteJSON
				c.handler.SendMessagesToClient(c, messages)
				continue
			}

			// --- Special case: If the original message is a SAY_DONE, forward the original JSON ---
			input := request.Content

			// EMERGENCY DEBUG: Log immediately when any input is received			// Process user input

			// DEBUG: Log the input immediately after assignment
			logger.Debug(logger.AreaTerminal, "input assigned from request.Content: %q (length: %d)", input, len(input)) // Skip empty or whitespace-only inputs, except for SAY_DONE and editor messages
			// IMPORTANT: For Telnet, whitespace characters like \r\n MUST NOT be filtered as they are important commands
			// IMPORTANT: For Cat Pager, whitespace characters like SPACE, ENTER are important pager commands
			// Check if this is a telnet session to bypass all input filtering
			isTelnetSession = c.handler.os.IsTelnetSessionActive(c.sessionID)
			isPagerSession := c.handler.os.IsInCatPagerProcess(c.sessionID)
			logger.Debug(logger.AreaTerminal, "CAT PAGER INPUT CHECK for session %s: pager_session=%t, input=%q",
				c.sessionID, isPagerSession, input)
			if !isTelnetSession && !isPagerSession && strings.TrimSpace(input) == "" && request.Type != 6 && request.Type != int(shared.MessageTypeEditor) {
				logger.Debug(logger.AreaTerminal, "Empty input, skipping")
				continue
			} // SAY_DONE special handling - skip for telnet sessions to avoid input corruption
			if !isTelnetSession {
				var origMap map[string]interface{}
				if err := json.Unmarshal(message, &origMap); err == nil {
					// Check for both "SAY_DONE" string and type: 6 (numeric)
					if t, ok := origMap["type"]; ok {
						logger.Debug(logger.AreaTerminal, "JSON type found: %v", t)
						if (t == "SAY_DONE") || (t == 6.0) || (t == 6) {
							// Forward the original JSON as string
							logger.Debug(logger.AreaTerminal, "SAY_DONE or type 6 detected, overwriting input with JSON")
							input = string(message)
						}
					}
				} else {
					logger.Debug(logger.AreaTerminal, "JSON unmarshal failed: %v", err)
				}
			} // DEBUG: Log the final input value
			logger.Debug(logger.AreaTerminal, "Final input value: %q (length: %d)", input, len(input))

			// Create a new context with Session-ID and Client
			ctx := auth.NewContextWithSessionID(context.Background(), c.sessionID)
			// Debug: Log Session-ID for troubleshooting - with enhanced character display			// Log input processing with minimal details

			// ENHANCED DEBUG: Log current session state and routing decision
			isTelnetSessionCheck := c.handler.os.IsTelnetSessionActive(c.sessionID)
			logger.Debug(logger.AreaTerminal, "SESSION ROUTING DEBUG for %s: telnet_active=%t, mode=%s, input=%q",
				c.sessionID, isTelnetSessionCheck, c.mode, input) // Add client to context
			ctx = context.WithValue(ctx, "client", c)

			var messages []shared.Message
			// EMERGENCY DEBUG: Log before telnet session check			// Check for telnet session routing
			isTelnetSession = c.handler.os.IsTelnetSessionActive(c.sessionID) // CRITICAL FIX: Check for pager session and handle pager input BEFORE other routing			isPagerSession = c.handler.os.IsInCatPagerProcess(c.sessionID)
			if isPagerSession {
				logger.Debug(logger.AreaTerminal, "CAT PAGER TEXT INPUT: Processing input %q for session %s", input, c.sessionID)
				// Process as pager input
				messages := c.handler.ProcessInputWithSession(input, c.sessionID)
				// Send messages through the proper channel instead of direct WriteJSON
				c.handler.SendMessagesToClient(c, messages)
				continue
			}

			if isTelnetSession {
				// For telnet sessions: direct input forwarding without any processing
				logger.Debug(logger.AreaTerminal, "Direct telnet input forwarding for session %s", c.sessionID)
				// Send input directly to telnet session
				messages = c.handler.os.HandleTelnetInput(input, c.sessionID)

				// CRITICAL FIX: Verify telnet session is still active after input processing
				// This ensures that if the session died during processing, we clean up properly
				go func() {
					time.Sleep(100 * time.Millisecond) // Small delay to allow processing to complete

					// Check if telnet session is still actually active
					if !c.handler.os.IsTelnetSessionActive(c.sessionID) {
						logger.Debug(logger.AreaTerminal, "Session %s is no longer active after input processing", c.sessionID)
					}
				}()

				// Send response immediately and continue to next input
				c.handler.SendMessagesToClient(c, messages)
				continue
			} else if c.mode == "basic" {
				// PRIORITY 2: BASIC mode
				// BASIC-Eingabe wird über ProcessBasicInput verarbeitet
				// Die session-spezifische TinyBASIC-Instanz wird dort automatisch geholt
				c.handler.ProcessBasicInput(c, input)
				continue
			} else if activeEditor := editor.GetEditorManager().GetEditor(c.sessionID); activeEditor != nil {
				// PRIORITY 3: Editor is active - process input as editor command
				// Prüfe, ob es sich um eine Editor-Message handelt (Type 20)

				if request.Type == int(shared.MessageTypeEditor) {
					logger.Debug(logger.AreaEditor, "Processing editor message: command=%s, data length=%d",
						request.EditorCommand, len(request.EditorData)) // Special handling for ready message
					if request.EditorCommand == "ready" {
						log.Printf("[EDITOR-WEBSOCKET] Received READY signal from frontend for session: %s", c.sessionID)
						// Enhanced readiness logging
						dataStr := "nil"
						if request.EditorData != "" {
							dataStr = fmt.Sprintf("(length: %d)", len(request.EditorData))

							// Try to fix potential JSON structure issue in editorData - if it looks like JSON
							if strings.HasPrefix(request.EditorData, "{") && strings.HasSuffix(request.EditorData, "}") {
								var jsonData map[string]interface{}
								if err := json.Unmarshal([]byte(request.EditorData), &jsonData); err == nil {
									// Successfully parsed as JSON object, extract status if available
									if status, ok := jsonData["status"].(string); ok {
										log.Printf("[EDITOR-WEBSOCKET] Extracted status from JSON: %s", status)
										// Just use the status as editorData string
										request.EditorData = status
									}
								}
							}
						}
						log.Printf("[EDITOR-WEBSOCKET] READY signal details: data=%s", dataStr)

						// ENHANCED: Höhere Priorität für ready-Signal
						// Direkt readyChan signalisieren für sofortiges Feedback
						if activeEditor != nil {
							activeEditor.Ready()

							// Nach ready sofort eine Render-Anforderung senden
							go func() {
								// Kurze Verzögerung für Terminal-Stabilität
								time.Sleep(50 * time.Millisecond)
								activeEditor.Render()
							}()
						}
					}

					// Always process editor output first
					c.handler.processEditorOutputForSession(c.sessionID, activeEditor.GetOutputChannel())

					if !activeEditor.ProcessEditorMessage(request.EditorCommand, request.EditorData) {
						log.Printf("[EDITOR-WEBSOCKET] Editor was closed by command")
						// Process output again for stop command
						c.handler.processEditorOutputForSession(c.sessionID, activeEditor.GetOutputChannel()) // Editor has ended - cleanup
						editor.GetEditorManager().CloseEditor(c.sessionID)
					} else {
						log.Printf("[EDITOR-WEBSOCKET] Processing editor output...")
						// Process editor output
						c.handler.processEditorOutputForSession(c.sessionID, activeEditor.GetOutputChannel())
					}
				} else {
					log.Printf("[EDITOR-WEBSOCKET] Processing text input for editor: %s", input)
					// Text-Input für Editor
					if !activeEditor.ProcessInput(input) {
						// Editor wurde beendet - cleanup
						editor.GetEditorManager().CloseEditor(c.sessionID)
					} else {
						// Editor-Ausgaben verarbeiten
						c.handler.processEditorOutputForSession(c.sessionID, activeEditor.GetOutputChannel())
					}
				}
				continue
			} else {
				// PRIORITY 4: Normal terminal mode
				// For normal terminal sessions, use the standard processing
				messages = c.handler.ProcessInputWithSession(input, c.sessionID)
			}

			// Sende alle Antwortnachrichten zurück an den Client
			c.handler.SendMessagesToClient(c, messages)
		}
	}
}

// writePump pumpt Nachrichten vom Hub zum WebSocket - DEADLOCK FIX
func (c *Client) writePump() {
	ticker := time.NewTicker(getPingPeriod())
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			// Reduced debug logging to prevent log spam
			c.conn.SetWriteDeadline(time.Now().Add(getWriteWait()))
			if !ok {
				// Channel was closed
				logger.Debug(logger.AreaTerminal, "Send channel closed for client %s", c.conn.RemoteAddr())
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// CRITICAL FIX: Safely drain additional messages with timeout protection
			// This prevents deadlocks when channel is being closed during drain
			timeout := time.NewTimer(10 * time.Millisecond)
			n := len(c.send)
			for i := 0; i < n; i++ {
				select {
				case additionalMsg := <-c.send:
					w.Write(newline)
					w.Write(additionalMsg)
				case <-timeout.C:
					// Timeout during drain - stop to prevent deadlock
					break
				}
			}
			timeout.Stop()

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(getWriteWait()))
			// Reduced ping logging frequency to prevent log spam
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error(logger.AreaTerminal, "Failed to send ping to client %s: %v", c.conn.RemoteAddr(), err)
				return
			}
		case <-c.shutdown:
			// Graceful shutdown requested
			logger.Debug(logger.AreaTerminal, "Shutdown signal received for client %s", c.conn.RemoteAddr())
			return
		}
	}
}
