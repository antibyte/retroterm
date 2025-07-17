package tinyos

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net"
	gos "os"
	"strings"
	"sync"
	"time"

	"github.com/antibyte/retroterm/pkg/board"
	"github.com/antibyte/retroterm/pkg/chess"
	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/editor"
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/resources" // Neu hinzugefügt
	"github.com/antibyte/retroterm/pkg/shared"
	"github.com/antibyte/retroterm/pkg/virtualfs"

	_ "modernc.org/sqlite"

	"golang.org/x/crypto/bcrypt"
)

// logMessage schreibt eine Nachricht ins Log
func logMessage(format string, v ...interface{}) {
	// Wir verwenden die Standard-Logfunktion von Go
	log.Printf(format, v...)
}

// Helper function for TinyOS debug logging that respects configuration
func tinyOSDebugLog(format string, args ...interface{}) {
	logger.Debug(logger.AreaGeneral, format, args...)
}

// InputMode defines the authoritative input mode for a user session.
// This is used to correctly route user input and prevent race conditions.
type InputMode int

const (
	InputModeOSShell             InputMode = 0
	InputModeEditor              InputMode = 1
	InputModeChess               InputMode = 2
	InputModeTelnet              InputMode = 3
	InputModePager               InputMode = 4
	InputModeLoginProcess        InputMode = 5
	InputModeRegistrationProcess InputMode = 6
	InputModePasswordChange      InputMode = 7
	InputModeBasicInterpreter    InputMode = 8
	InputModeBoard               InputMode = 9
)

// isTemporaryUser checks if a username should receive temporary sessions
func isTemporaryUser(username string) bool {
	temporaryUsers := []string{"dyson"}
	for _, tmpUser := range temporaryUsers {
		if username == tmpUser {
			return true
		}
	}
	return false
}

// ChatMessage repräsentiert eine einzelne Chat-Nachricht
type ChatMessage struct {
	Role    string    `json:"role"`    // "user" oder "assistant"
	Content string    `json:"content"` // Der Nachrichteninhalt
	Time    time.Time `json:"time"`    // Zeitpunkt der Nachricht
}

// Session repräsentiert eine Benutzersitzung
type Session struct {
	ID           string             // Eindeutige ID der Session
	Username     string             // Benutzername
	IPAddress    string             // IP-Adresse des Clients
	CurrentPath  string             // Aktueller Pfad des Benutzers
	CreatedAt    time.Time          // Zeitpunkt der Erstellung
	LastActivity time.Time          // Zeitpunkt der letzten Aktivität
	ChatHistory  []ChatMessage      // Chat-Verlauf mit DeepSeek
	ChessGame    *chess.ChessUI     // Chess game state
	ChessActive  bool               // Flag whether chess game is active
	InputMode    InputMode          // The current authoritative input mode for the session.
	Terminal     TerminalDimensions // Terminal dimensions for this session
}

// SessionContext enthält Kontext-Informationen für die Ausführung von Befehlen
type SessionContext struct {
	SessionID   string // Die ID der aktuellen Session
	Username    string // Der angemeldete Benutzername (leer wenn nicht angemeldet)
	CurrentPath string // Aktueller Pfad des Benutzers
	IPAddress   string // IP-Adresse des verbundenen Clients
	IsGuest     bool   // Flag, ob es sich um einen Gastbenutzer handelt
}

// TinyOS ist die Hauptstruktur für das Betriebssystem
type TinyOS struct {
	Vfs                   *virtualfs.VFS                    // Feld exportiert
	ResourceManager       *resources.SessionResourceManager // Session-Ressourcenmanager
	SystemResourceManager *resources.SystemResourceManager  // System-Ressourcenmanager
	PromptManager         *shared.PromptManager             // System für externe Prompt-Dateien
	systemEnv             map[string]string
	lastDeepSeekQuery     string
	deepSeekHistory       []map[string]string

	db             *sql.DB
	chatRateLimits map[string]*RateLimit
	bannedUsers    map[string]time.Time
	mu             sync.Mutex
	guestSessions  []string // Liste für Gast-Sessions
	// Felder für das Session-Management
	sessions     map[string]*Session // Map von Session-IDs zu Sessions
	sessionMutex sync.RWMutex        // Mutex für Thread-sicheren Zugriff auf Sessions

	// Terminal-Dimensionen pro Session
	sessionTerminals map[string]*TerminalDimensions // Map von Session-IDs zu Terminal-Dimensionen

	// BASIC Session-Tracking für Session-Limits
	activeBasicSessions map[string]bool // Set von SessionIDs mit aktiven BASIC-Sitzungen
	basicSessionMutex   sync.RWMutex    // Mutex für Thread-sicheren Zugriff auf BASIC-Sitzungen
	cols                int             // Terminal-Breite in Spalten
	rows                int             // Terminal-Höhe in Zeilen

	// Login process tracking
	loginStates        map[string]*LoginState        // Map of session IDs to login status
	loginMutex         sync.RWMutex                  // Mutex for thread-safe access to login status	// Registration process tracking
	registrationStates map[string]*RegistrationState // Map von Session-IDs zu Registrierungs-Status
	registrationMutex  sync.RWMutex                  // Mutex für Thread-sicheren Zugriff auf Registrierungs-Status
	// Password change process tracking
	passwordChangeStates map[string]*PasswordChangeState // Map of session IDs to password change status
	passwordChangeMutex  sync.RWMutex                    // Mutex for thread-safe access to password change status
	// CAT pager process tracking
	catPagerStates map[string]*CatPagerState // Map von Session-IDs zu CAT-Pager-Status
	catPagerMutex  sync.RWMutex              // Mutex für Thread-sicheren Zugriff auf CAT-Pager-Status
	// Telnet session tracking
	telnetStates map[string]*TelnetState // Map von Session-IDs zu Telnet-Status
	telnetMutex  sync.RWMutex            // Mutex für Thread-sicheren Zugriff auf Telnet-Status

	// Failed login attempt tracking
	failedLoginAttempts map[string]*LoginAttemptTracker // Map von IP-Adressen zu Login-Versuch-Tracking
	loginAttemptMutex   sync.RWMutex                    // Mutex für Thread-sicheren Zugriff auf Login-Versuche

	// Shutdown channel for telnet output processor
	telnetOutputShutdown chan bool

	// Board system management
	boardManager  *board.BoardManager
	boardSessions map[string]*BoardSession

	// Callback function for sending messages to clients
	SendToClientCallback func(sessionID string, message shared.Message) error
}

// LoginState stores the status of a multi-step login process
type LoginState struct {
	Stage     string    // "username", "password"
	Username  string    // Cached username
	IPAddress string    // IP address for login
	CreatedAt time.Time // Time when login was initiated
}

// RegistrationState speichert den Status eines mehrstufigen Registrierungsprozesses
type RegistrationState struct {
	Stage     string    // "username", "password", "confirm_password"
	Username  string    // Zwischengespeicherter Benutzername
	Password  string    // Zwischengespeichertes Passwort
	IPAddress string    // IP-Adresse für die Registrierung
	CreatedAt time.Time // Zeitpunkt der Registrierungsinitiierung
}

// PasswordChangeState stores the state of a multi-step password change process
type PasswordChangeState struct {
	Stage           string    // "current", "new", "confirm"
	Username        string    // Username whose password is being changed
	CurrentPassword string    // Current password (temporarily stored for verification)
	NewPassword     string    // New password (temporarily stored for confirmation)
	CreatedAt       time.Time // Time when password change was initiated
}

// CatPagerState speichert den Status eines CAT-Pager-Prozesses
type CatPagerState struct {
	Lines       []string           // Alle Zeilen der Datei
	CurrentLine int                // Aktuelle Zeile (0-basiert)
	PageSize    int                // Anzahl der Zeilen pro Seite
	Filename    string             // Name der angezeigten Datei
	CreatedAt   time.Time          // Zeitpunkt der Pager-Initiierung
	Terminal    TerminalDimensions // Terminal dimensions for proper status line formatting
}

// TerminalDimensions speichert die Terminal-Abmessungen für eine Session
type TerminalDimensions struct {
	Cols int // Terminal-Breite in Spalten
	Rows int // Terminal-Höhe in Zeilen
}

// RateLimit speichert die Rate-Limiting-Informationen für einen Benutzer
type RateLimit struct {
	Count          int       // Anfragen in der aktuellen Minute
	LastReset      time.Time // Zeitpunkt des letzten Zurücksetzens
	RateLimitUntil time.Time // Zeitpunkt, bis zu dem das Rate-Limit gilt
}

// TelnetState stores the status of an active telnet connection
type TelnetState struct {
	ServerName    string              // Name of the connected server
	ServerHost    string              // Host and port of the server
	Connection    net.Conn            // The actual TCP connection
	SessionID     string              // Session ID of the user
	CreatedAt     time.Time           // Time when telnet was initiated
	LastActivity  time.Time           // Time of last activity
	OutputChan    chan shared.Message // Channel for sending output to the frontend
	ShutdownChan  chan struct{}       // Channel for graceful shutdown
	ServerEcho    bool                // Whether server is echoing our input
	LocalEcho     bool                // Whether we should echo locally
	channelClosed bool                // Flag to track if OutputChan is closed
	channelMutex  sync.RWMutex        // Mutex to protect channel operations
}

// LoginAttemptTracker tracks failed login attempts for an IP address
type LoginAttemptTracker struct {
	FailedAttempts int       // Number of failed attempts
	LastAttempt    time.Time // Time of last failed attempt
	LockedUntil    time.Time // Time until which login is locked (zero if not locked)
}

// NewTinyOS erstellt eine neue Instanz von TinyOS
func NewTinyOS(vfs *virtualfs.VFS, promptManager *shared.PromptManager) *TinyOS {
	os := &TinyOS{
		Vfs:                   vfs,
		ResourceManager:       resources.NewSessionResourceManager(), // Session-Ressourcenmanager
		SystemResourceManager: resources.NewSystemResourceManager(),  // System-Ressourcenmanager
		PromptManager:         promptManager,                         // PromptManager hinzufügen
		systemEnv:             make(map[string]string),
		deepSeekHistory:       make([]map[string]string, 0), chatRateLimits: make(map[string]*RateLimit),
		bannedUsers: make(map[string]time.Time), sessions: make(map[string]*Session), // Initialisiere die Sessions-Map
		sessionTerminals:     make(map[string]*TerminalDimensions),  // Initialisiere die Terminal-Dimensionen-Map
		activeBasicSessions:  make(map[string]bool),                 // Initialisiere das Set für aktive BASIC-Sitzungen
		registrationStates:   make(map[string]*RegistrationState),   // Initialisiere die Registrierungs-Status-Map
		passwordChangeStates: make(map[string]*PasswordChangeState), // Initialize password change states map
		loginStates:          make(map[string]*LoginState),          // Initialisiere die Login-Status-Map
		catPagerStates:       make(map[string]*CatPagerState),       // Initialisiere die CAT-Pager-Status-Map
		telnetStates:         make(map[string]*TelnetState),         // Initialisiere die Telnet-Status-Map
		failedLoginAttempts:  make(map[string]*LoginAttemptTracker), // Initialisiere die fehlgeschlagenen Login-Versuche-Map
		telnetOutputShutdown: make(chan bool),                       // Initialize the shutdown channel
	}

	// Registriere TinyOS als Provider beim VFS
	vfs.SetTinyOSProvider(os)

	// Initialisiere die Datenbank
	os.initDB()

	// Umgebungsvariablen laden
	os.loadEnvFromDB()

	// Lade gebannte Benutzer aus der Datenbank
	os.loadBannedUsers() // Starte Session-Cleanup-Goroutine (periodisch)
	go func() {
		ticker := time.NewTicker(5 * time.Minute) // Cleanup alle 5 Minuten
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				os.CleanupExpiredSessions()
			}
		}
	}()

	// Starte Ressourcenmanager-Cleanup
	os.ResourceManager.StartPeriodicCleanup()
	os.ResourceManager.MemoryGuard()

	// Initialisiere das Gast-Dateisystem (das passiert unabhängig davon,
	// ob ein Benutzer angemeldet ist oder nicht, damit wir immer ein Home für Gastbenutzer haben)
	fmt.Printf("[INIT] Initialisiere Gast-VFS beim Systemstart\n")
	err := vfs.InitializeGuestVFS()
	if err != nil {
		fmt.Printf("[INIT] Fehler beim Initialisieren des Gast-VFS: %v\n", err)
	} else {
		fmt.Printf("[INIT] Gast-VFS erfolgreich initialisiert\n")
	}
	// Prüfe nochmals, ob das Gast-Home existiert
	guestHomePath := "/home/guest"
	if !os.Vfs.Exists(guestHomePath, "") {
		err := os.Vfs.InitializeGuestVFS()
		if err != nil {
			fmt.Printf("[INIT] Fehler beim Initialisieren des Gast-VFS: %v\n", err)
		} else {
			fmt.Printf("[INIT] Gast-VFS erfolgreich initialisiert\n")
		}
	} else {
		entries, listErr := os.Vfs.ListDir("/home/guest")
		if listErr != nil {
			fmt.Printf("[INIT] Fehler beim Auflisten des Gast-Home-Verzeichnisses: %v\n", listErr)
		} else {
			fmt.Printf("[INIT] Gast-Home-Verzeichnis enthält %d Einträge\n", len(entries))
		}
	}
	// Telnet output processor background worker is disabled
	// We use direct WebSocket callback for telnet output instead
	// go os.processTelnetOutputs()

	// Shutdown any existing telnet output processor from previous runs
	os.ShutdownTelnetOutputProcessor()

	return os
}

// GetUsernameForSession gibt den Benutzernamen für eine Session zurück
func (os *TinyOS) GetUsernameForSession(sessionID string) string {
	os.sessionMutex.RLock()
	defer os.sessionMutex.RUnlock()

	session, exists := os.sessions[sessionID]
	if !exists {
		return ""
	}
	return session.Username
}

// GetSessionUsername returns the username for a given session ID (required by auth interface)
func (os *TinyOS) GetSessionUsername(sessionID string) string {
	return os.GetUsernameForSession(sessionID)
}

// GetUsernameBySessionID gibt den Benutzernamen für eine gegebene Session-ID zurück
func (os *TinyOS) GetUsernameBySessionID(sessionID string) string {
	return os.GetUsernameForSession(sessionID)
}

// GetPromptForSession generates a simple standard prompt
func (os *TinyOS) GetPromptForSession(sessionID string) string {
	return "> "
}

// VerifyPassword überprüft, ob das gegebene Passwort für den Benutzer korrekt ist
func (os *TinyOS) VerifyPassword(username, password string) bool {
	if os.db == nil {
		logger.Warn(logger.AreaAuth, "Database not available for password verification")
		return false
	}

	// Get password hash from database
	var storedHash string
	err := os.db.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Warn(logger.AreaAuth, "User '%s' not found for password verification", username)
		} else {
			logger.Warn(logger.AreaAuth, "Database error during password verification for user '%s': %v", username, err)
		}
		return false
	}

	// Compare password with stored hash
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		logger.SecurityWarn("Password verification failed for user '%s'", username)
		return false
	}

	logger.SecurityInfo("Password verification successful for user '%s'", username)
	return true
}

// UpdateUserPassword aktualisiert das Passwort eines Benutzers in der Datenbank
func (os *TinyOS) UpdateUserPassword(username, newPassword string) error {
	if os.db == nil {
		return fmt.Errorf("database not available")
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Warn(logger.AreaAuth, "Failed to hash new password for user '%s': %v", username, err)
		return fmt.Errorf("failed to hash password: %v", err)
	}

	// Update password in database
	_, err = os.db.Exec("UPDATE users SET password = ? WHERE username = ?", string(hashedPassword), username)
	if err != nil {
		logger.Warn(logger.AreaAuth, "Failed to update password for user '%s': %v", username, err)
		return fmt.Errorf("failed to update password: %v", err)
	}

	logger.SecurityInfo("Password successfully updated for user '%s'", username)
	return nil
}

// CurrentPathFromSession returns the current working directory for a session
func (os *TinyOS) CurrentPathFromSession(sessionID string) string {
	os.sessionMutex.RLock()
	defer os.sessionMutex.RUnlock()

	if session, exists := os.sessions[sessionID]; exists {
		return session.CurrentPath
	}

	return "" // Return empty string if session not found
}

// ExtractSessionID extrahiert die Session-ID aus den Befehlsargumenten
// und gibt sowohl die Session-ID als auch die bereinigten Argumente zurück
func (os *TinyOS) ExtractSessionID(args []string) (string, []string) {
	// Prüfe ob genügend Argumente vorhanden sind und das letzte Argument mit "sid=" beginnt
	if len(args) > 0 && strings.HasPrefix(args[len(args)-1], "sid=") {
		// Extrahiere die Session-ID aus dem letzten Argument
		sid := strings.TrimPrefix(args[len(args)-1], "sid=")
		// Entferne das Session-ID-Argument aus den Argumenten
		return sid, args[:len(args)-1]
	}

	// Prüfe ob das erste Argument eine Session-ID sein könnte
	if len(args) > 0 && !strings.HasPrefix(args[0], "/") && !strings.HasPrefix(args[0], ".") && !strings.HasPrefix(args[0], "-") {
		// Möglicherweise ist das erste Argument die Session-ID
		sid := args[0]

		// Prüfe ob es sich um eine gültige Session-ID handelt
		os.sessionMutex.RLock()
		_, exists := os.sessions[sid]
		os.sessionMutex.RUnlock()

		if exists {
			return sid, args[1:] // Session-ID aus den Argumenten entfernen
		}
	}

	return "", args
}

// UpdateSessionActivity aktualisiert den Zeitpunkt der letzten Aktivität einer Session
func (os *TinyOS) UpdateSessionActivity(sessionID string) {
	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()

	session, exists := os.sessions[sessionID]
	if !exists {
		return
	}

	session.LastActivity = time.Now()
	os.sessions[sessionID] = session

	// Aktualisiere die Session in der Datenbank
	if os.db != nil {
		_, err := os.db.Exec("UPDATE user_sessions SET last_activity = ? WHERE session_id = ?",
			session.LastActivity, sessionID)
		if err != nil {
			logMessage("[TINYOS] Fehler beim Aktualisieren der Session-Aktivität in der Datenbank: %v", err)
		}
	}
}

// UpdateSessionPath aktualisiert den aktuellen Pfad einer Session in der Datenbank
func (os *TinyOS) UpdateSessionPath(sessionID string, newPath string) error {
	if sessionID == "" {
		return nil // Keine Session-ID, nichts zu aktualisieren
	}

	tinyOSDebugLog("[UPDATE-PATH] Updating path for session %s to %s", sessionID, newPath)

	// Aktualisiere zuerst den Pfad im Speicher
	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()
	session, exists := os.sessions[sessionID]
	if exists {
		tinyOSDebugLog("[UPDATE-PATH] Found session, old path: %s, new path: %s", session.CurrentPath, newPath)
		session.CurrentPath = newPath
		os.sessions[sessionID] = session
		tinyOSDebugLog("[UPDATE-PATH] Updated session in memory")

		// Nur für reguläre Benutzer (nicht Gastbenutzer) Datenbank aktualisieren
		if !strings.HasPrefix(session.Username, "guest-") && os.db != nil {
			tinyOSDebugLog("[UPDATE-PATH] Updating database for user %s", session.Username)
			_, err := os.db.Exec("UPDATE user_sessions SET current_path = ? WHERE session_id = ?",
				newPath, sessionID)
			if err != nil {
				tinyOSDebugLog("[UPDATE-PATH] Database update error: %v", err)
				return fmt.Errorf("error updating session path: %v", err)
			}
			tinyOSDebugLog("[UPDATE-PATH] Database updated successfully")
		} else {
			tinyOSDebugLog("[UPDATE-PATH] Skipping database update (guest user or no db)")
		}
	} else {
		tinyOSDebugLog("[UPDATE-PATH] Session not found: %s", sessionID)
		return fmt.Errorf("session not found: %s", sessionID)
	}

	tinyOSDebugLog("[UPDATE-PATH] Path update completed successfully")
	return nil
}

// generateSessionID erzeugt eine eindeutige Session-ID
// GenerateSessionID erzeugt eine eindeutige Session-ID (exportiert)
func GenerateSessionID() string {
	// Erzeuge eine zufällige ID mit 32 Zeichen
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback, wenn kein zufälliger Wert erzeugt werden kann
		return fmt.Sprintf("s-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

// CleanupExpiredSessions entfernt abgelaufene Sessions
func (os *TinyOS) CleanupExpiredSessions() {
	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()

	now := time.Now()
	sessionTimeout := 24 * time.Hour // Sessions laufen nach 24 Stunden Inaktivität ab
	for id, session := range os.sessions {
		if now.Sub(session.LastActivity) > sessionTimeout {
			// Cleanup telnet sessions for expired sessions
			os.CleanupTelnetSessionSync(session.ID)

			// CRITICAL FIX: Also cleanup cat pager states for expired sessions
			os.catPagerMutex.Lock()
			if _, exists := os.catPagerStates[id]; exists {
				delete(os.catPagerStates, id)
				logMessage("[TINYOS] Cleaned up cat pager state for expired session %s", id)
			}
			os.catPagerMutex.Unlock()

			delete(os.sessions, id)

			if os.db != nil {
				_, err := os.db.Exec("DELETE FROM user_sessions WHERE session_id = ?", id)
				if err != nil {
					logMessage("[TINYOS] Fehler beim Löschen der abgelaufenen Session aus der Datenbank: %v", err)
				}
			}

			logMessage("[TINYOS] Abgelaufene Session %s für Benutzer %s entfernt", id, session.Username)
		}
	}

	// Also cleanup orphaned telnet sessions
	os.cleanupOrphanedTelnetSessions()
}

// BASIC Session Management Functions

// StartBasicSession startet eine neue BASIC-Sitzung, wenn das Limit nicht erreicht ist
func (os *TinyOS) StartBasicSession(sessionID string) bool {
	os.basicSessionMutex.Lock()
	defer os.basicSessionMutex.Unlock()

	// Prüfe, ob das globale Limit erreicht ist
	if len(os.activeBasicSessions) >= MaxBasicSessions {
		log.Printf("[BASIC-SESSION] Maximum BASIC sessions reached (%d/%d), denying session %s",
			len(os.activeBasicSessions), MaxBasicSessions, sessionID)
		return false
	}

	// Prüfe für Gastbenutzer das separate Limit
	if os.isGuestSession(sessionID) {
		guestCount := os.countGuestBasicSessions()
		if guestCount >= MaxGuestBasicSessions {
			log.Printf("[BASIC-SESSION] Maximum guest BASIC sessions reached (%d/%d), denying guest session %s",
				guestCount, MaxGuestBasicSessions, sessionID)
			return false
		}
	}

	// Sitzung hinzufügen
	os.activeBasicSessions[sessionID] = true
	log.Printf("[BASIC-SESSION] Started BASIC session %s (total: %d/%d)",
		sessionID, len(os.activeBasicSessions), MaxBasicSessions)
	return true
}

// EndBasicSession beendet eine BASIC-Sitzung
func (os *TinyOS) EndBasicSession(sessionID string) {
	os.basicSessionMutex.Lock()
	defer os.basicSessionMutex.Unlock()

	if _, exists := os.activeBasicSessions[sessionID]; exists {
		delete(os.activeBasicSessions, sessionID)
		log.Printf("[BASIC-SESSION] Ended BASIC session %s (remaining: %d/%d)",
			sessionID, len(os.activeBasicSessions), MaxBasicSessions)
	}
}

// IsBasicSessionActive prüft, ob eine BASIC-Sitzung aktiv ist
func (os *TinyOS) IsBasicSessionActive(sessionID string) bool {
	os.basicSessionMutex.RLock()
	defer os.basicSessionMutex.RUnlock()

	_, exists := os.activeBasicSessions[sessionID]
	return exists
}

// GetActiveBasicSessionCount gibt die Anzahl aktiver BASIC-Sitzungen zurück
func (os *TinyOS) GetActiveBasicSessionCount() int {
	os.basicSessionMutex.RLock()
	defer os.basicSessionMutex.RUnlock()

	return len(os.activeBasicSessions)
}

// isGuestSession prüft, ob es sich um eine Gastsitzung handelt
func (os *TinyOS) isGuestSession(sessionID string) bool {
	os.sessionMutex.RLock()
	defer os.sessionMutex.RUnlock()

	session, exists := os.sessions[sessionID]
	if !exists {
		return false
	}

	return session.Username == "" || session.Username == "guest" ||
		strings.HasPrefix(session.Username, "guest-")
}

// countGuestBasicSessions zählt die aktiven BASIC-Sitzungen für Gastbenutzer
func (os *TinyOS) countGuestBasicSessions() int {
	count := 0
	for sessionID := range os.activeBasicSessions {
		if os.isGuestSession(sessionID) {
			count++
		}
	}
	return count
}

// CheckChatRateLimit prüft, ob ein Benutzer das Rate-Limit überschritten hat
// Gibt zurück: (isLimited, shouldBan, isTemporarilyBlocked)
func (os *TinyOS) CheckChatRateLimit(username, ip string) (bool, bool, bool) {
	os.mu.Lock()
	defer os.mu.Unlock()

	// Kombiniere Benutzername und IP für den Schlüssel
	key := username + ":" + ip

	now := time.Now()

	// Prüfe zuerst, ob der Benutzer ein temporäres Rate-Limit hat
	if limit, ok := os.chatRateLimits[key]; ok {
		if now.Before(limit.RateLimitUntil) {
			return true, false, true // Benutzer hat ein temporäres Rate-Limit
		}
		// Prüfe, ob die letzte Zurücksetzung mehr als das konfigurierte Intervall her ist
		resetInterval := configuration.GetDuration("ChatRateLimit", "rate_limit_reset_interval", time.Minute)
		if now.Sub(limit.LastReset) > resetInterval {
			// Zurücksetzen des Zählers
			limit.Count = 0
			limit.LastReset = now
		}

		// Zähler erhöhen
		limit.Count++
		// Prüfen auf Überschreitung
		maxRequestsBan := configuration.GetInt("ChatRateLimit", "max_requests_per_minute_ban", 20)
		maxRequests := configuration.GetInt("ChatRateLimit", "max_requests_per_minute", 10)
		rateLimitDuration := configuration.GetDuration("ChatRateLimit", "rate_limit_duration", 2*time.Minute)

		if limit.Count > maxRequestsBan {
			// Benutzer sollte gebannt werden
			return true, true, false
		}

		if limit.Count > maxRequests {
			// Benutzer hat das Rate-Limit überschritten
			limit.RateLimitUntil = now.Add(rateLimitDuration)
			return true, false, true
		}

		return false, false, false
	}

	// Neue Rate-Limit-Eintrag erstellen, wenn noch keiner existiert
	os.chatRateLimits[key] = &RateLimit{
		Count:     1,
		LastReset: now,
	}

	return false, false, false
}

// CheckChatTimeLimits prüft die zeitlichen Beschränkungen für Chat-Sitzungen
func (os *TinyOS) CheckChatTimeLimits(username string) (bool, string) {
	if username == "" || os.db == nil {
		return false, "Benutzer nicht angemeldet"
	}

	// Heutiges Datum im Format YYYY-MM-DD
	today := time.Now().Format("2006-01-02")

	// Prüfe, ob der Benutzer heute bereits einen Eintrag hat
	var timeUsed int
	var lastSessionStart int64
	err := os.db.QueryRow(
		"SELECT time_used, last_session_start FROM chat_usage WHERE username = ? AND date = ?",
		username, today).Scan(&timeUsed, &lastSessionStart)

	if err == sql.ErrNoRows {
		// Kein Eintrag vorhanden - einen neuen erstellen
		now := time.Now().Unix()
		_, err = os.db.Exec(
			"INSERT INTO chat_usage (username, date, time_used, last_session_start) VALUES (?, ?, 0, ?)",
			username, today, now)
		if err != nil {
			logMessage("[TINYOS] Fehler beim Erstellen des Chat-Nutzungseintrags: %v", err)
		}

		return false, ""
	} else if err != nil {
		logMessage("[TINYOS] Fehler beim Abrufen der Chat-Nutzung: %v", err)
		return true, "Datenbankfehler"
	}

	now := time.Now().Unix()
	// Wenn die letzte Sitzung weniger als eine Stunde her ist, prüfe, ob das Stundenlimit erreicht wurde
	if now-lastSessionStart < 3600 {
		if timeUsed >= 300 { // 5 Minuten in Sekunden
			return true, "Du hast das Stundenlimit von 5 Minuten erreicht. Versuche es später erneut."
		}
	} else {
		// Letzte Sitzung ist mehr als eine Stunde her, setze den Startzeitpunkt zurück
		_, err = os.db.Exec(
			"UPDATE chat_usage SET last_session_start = ? WHERE username = ? AND date = ?",
			now, username, today)

		if err != nil {
			logMessage("[TINYOS] Fehler beim Aktualisieren des Chat-Sitzungsstarts: %v", err)
		}
	}
	// Prüfe das Tageslimit
	if timeUsed >= 900 { // 15 Minuten in Sekunden
		return true, "Du hast das Tageslimit von 15 Minuten erreicht. Versuche es morgen erneut."
	}

	return false, ""
}

// IsBanned prüft, ob ein Benutzer oder eine IP gebannt ist
func (os *TinyOS) IsBanned(username, ip string) (bool, string) {
	if username == "" && ip == "" {
		return false, ""
	}

	os.mu.Lock()
	defer os.mu.Unlock()

	now := time.Now()
	bannedEntities := []string{}
	longestBan := time.Duration(0)
	var latestExpiry time.Time

	// Alle gebannten Entitäten sammeln (Username und/oder IP)
	// Prüfe für Benutzername, wenn vorhanden
	if username != "" {
		if expiry, ok := os.bannedUsers[username]; ok {
			if now.Before(expiry) {
				// Benutzer ist noch gebannt
				bannedEntities = append(bannedEntities, "username")
				remaining := expiry.Sub(now)
				if remaining > longestBan {
					longestBan = remaining
					latestExpiry = expiry
				}
			} else {
				// Ban ist abgelaufen, entferne ihn
				delete(os.bannedUsers, username)
				// Auch aus der Datenbank entfernen
				if os.db != nil {
					_, _ = os.db.Exec("DELETE FROM banned_users WHERE identifier = ?", username)
				}
			}
		}
	}

	// Prüfe für IP-Adresse, wenn vorhanden
	if ip != "" {
		if expiry, ok := os.bannedUsers[ip]; ok {
			if now.Before(expiry) {
				// IP ist noch gebannt
				bannedEntities = append(bannedEntities, "IP")
				remaining := expiry.Sub(now)
				if remaining > longestBan {
					longestBan = remaining
					latestExpiry = expiry
				}
			} else {
				// Ban ist abgelaufen, entferne ihn
				delete(os.bannedUsers, ip)
				// Auch aus der Datenbank entfernen
				if os.db != nil {
					_, _ = os.db.Exec("DELETE FROM banned_users WHERE identifier = ?", ip)
				}
			}
		}
	}

	// Wenn keine Entität gebannt ist, früh zurückkehren
	if len(bannedEntities) == 0 {
		return false, ""
	}

	// Formatiere die Zeitangabe je nach verbleibender Zeit
	var timeStr string
	hours := int(longestBan.Hours())
	minutes := int(longestBan.Minutes()) % 60

	if hours > 24 {
		days := hours / 24
		hours = hours % 24
		timeStr = fmt.Sprintf("%d Tage, %d Stunden", days, hours)
	} else if hours > 0 {
		timeStr = fmt.Sprintf("%d Stunden, %d Minuten", hours, minutes)
	} else {
		timeStr = fmt.Sprintf("%d Minuten", minutes)
	}

	// Erstelle eine aussagekräftige Nachricht
	reason := "Chat-Missbrauch"
	message := fmt.Sprintf("Du bist aufgrund von %s gebannt. Der Ban läuft ab in: %s",
		reason, timeStr)

	logMessage("[TINYOS] Ban-Check: %s und/oder IP %s ist gebannt bis %v",
		username, ip, latestExpiry.Format("02.01.2006 15:04:05"))

	return true, message
}

// BanUserAndIP sperrt einen Benutzer und seine IP-Adresse für die angegebene Dauer
func (os *TinyOS) BanUserAndIP(username, ip string, duration time.Duration) {
	os.mu.Lock()
	defer os.mu.Unlock()

	expiry := time.Now().Add(duration)

	// Speichere den Ban im Speicher
	os.bannedUsers[username] = expiry
	os.bannedUsers[ip] = expiry

	// Speichere den Ban in der Datenbank
	if os.db != nil {
		_, err := os.db.Exec("INSERT OR REPLACE INTO banned_users (identifier, expiry) VALUES (?, ?)",
			username, expiry.Unix())
		if err != nil {
			fmt.Printf("Fehler beim Speichern des Benutzerbans: %v\n", err)
		}

		_, err = os.db.Exec("INSERT OR REPLACE INTO banned_users (identifier, expiry) VALUES (?, ?)",
			ip, expiry.Unix())
		if err != nil {
			fmt.Printf("Fehler beim Speichern des IP-Bans: %v\n", err)
		}
	}
}

// loadBannedUsers lädt die gesperrten Benutzer aus der Datenbank
func (os *TinyOS) loadBannedUsers() {
	if os.db == nil {
		return
	}

	// Aktuelle Zeit für Vergleich mit Ablaufdatum
	now := time.Now().Unix()

	rows, err := os.db.Query("SELECT identifier, expiry FROM banned_users WHERE expiry > ?", now)
	if err != nil {
		fmt.Printf("Fehler beim Laden der umgebensvariablen: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var identifier string
		var expiryTimestamp int64
		if err := rows.Scan(&identifier, &expiryTimestamp); err != nil {
			fmt.Printf("Fehler beim Lesen eines gebannten Benutzers: %v\n", err)
			continue
		}

		// Konvertiere Unix-Timestamp zurück zu time.Time
		expiry := time.Unix(expiryTimestamp, 0)
		os.bannedUsers[identifier] = expiry
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Fehler beim Iterieren über gebannte Benutzer: %v\n", err)
	}
}

// initDB initialisiert die SQLite-Datenbank
func (os *TinyOS) initDB() {
	// Datenbankdatei öffnen/erstellen
	db, err := sql.Open("sqlite", "tinyos.db")
	if err != nil {
		fmt.Printf("Fehler beim Öffnen der Datenbank: %v\n", err)
		return
	}
	os.db = db

	// Tabellen erstellen, falls sie nicht existieren
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		username TEXT PRIMARY KEY,
		password TEXT NOT NULL,
		last_login INTEGER,
		login_attempts INTEGER DEFAULT 0,
		is_admin INTEGER DEFAULT 0,
		is_active INTEGER DEFAULT 1,
		created_at INTEGER NOT NULL,
		ip_address TEXT
	)`)
	if err != nil {
		fmt.Printf("Fehler beim Erstellen der Benutzertabelle: %v\n", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS env_vars (
		name TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`)
	if err != nil {
		fmt.Printf("Fehler beim Erstellen der Umgebungsvariablentabelle: %v\n", err)
	}

	// Erstelle die Tabelle für gebannte Benutzer
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS banned_users (
		identifier TEXT PRIMARY KEY,
		expiry INTEGER NOT NULL
	)`)
	if err != nil {
		fmt.Printf("Fehler beim Erstellen der Tabelle für gebannte Benutzer: %v\n", err)
	}

	// Erstelle die Tabelle für Registrierungsversuche
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS registration_attempts (
		ip_address TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		PRIMARY KEY (ip_address, timestamp)
	)`)
	if err != nil {
		fmt.Printf("Fehler beim Erstellen der Tabelle für Registrierungsversuche: %v\n", err)
	}

	// Erstelle die Tabelle für virtuelle Dateien
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS virtual_files (
		username TEXT NOT NULL,
		path TEXT NOT NULL,
		content TEXT,
		is_dir INTEGER NOT NULL,
		mod_time INTEGER NOT NULL,
		PRIMARY KEY (username, path)
	)`)
	if err != nil {
		fmt.Printf("Fehler beim Erstellen der Tabelle für virtuelle Dateien: %v\n", err)
	}
	// Erstelle die Tabelle für Benutzersitzungen
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS user_sessions (
		session_id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		last_activity INTEGER NOT NULL,
		current_path TEXT
	)`)
	if err != nil {
		fmt.Printf("Fehler beim Erstellen der Tabelle für Benutzersitzungen: %v\n", err)
	}

	// Add current_path column if it doesn't exist (for existing installations)
	_, err = db.Exec(`ALTER TABLE user_sessions ADD COLUMN current_path TEXT`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		fmt.Printf("Warning: Could not add current_path column to user_sessions: %v\n", err)
	}

	// Erstelle die Tabelle für Chat-Nutzung
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS chat_usage (
		username TEXT NOT NULL,
		date TEXT NOT NULL,
		time_used INTEGER NOT NULL,
		last_session_start INTEGER NOT NULL,
		PRIMARY KEY (username, date)
	)`)
	if err != nil {
		fmt.Printf("Fehler beim Erstellen der Tabelle für Chat-Nutzung: %v\n", err)
	}
}

// Getenv gibt den Wert einer Umgebungsvariable zurück
func (os *TinyOS) Getenv(name string) string {
	os.mu.Lock()
	defer os.mu.Unlock()

	// Zuerst in der In-Memory-Map nachschauen
	if value, exists := os.systemEnv[name]; exists {
		return value
	}

	// Dann im tatsächlichen Betriebssystem
	return ""
}

// loadEnvFromDB lädt Umgebungsvariablen aus der Datenbank
func (os *TinyOS) loadEnvFromDB() {
	if os.db == nil {
		return
	}

	rows, err := os.db.Query("SELECT name, value FROM env_vars")
	if err != nil {
		fmt.Printf("Fehler beim Laden der Umgebungsvariablen: %v\n", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var name, value string
		if err := rows.Scan(&name, &value); err != nil {
			fmt.Printf("Fehler beim Lesen einer Umgebungsvariable: %v\n", err)
			continue
		}
		os.systemEnv[name] = value
	}

	if err := rows.Err(); err != nil {
		fmt.Printf("Fehler beim Iterieren über Umgebungsvariablen: %v\n", err)
	}
}

// syncExamplePrograms kopiert alle Beispielprogramme aus dem examples-Ordner ins Heimatverzeichnis des Benutzers
// und aktualisiert bestehende Beispiele, wenn sie sich geändert haben
func (os *TinyOS) syncExamplePrograms(username string) error {
	tinyOSDebugLog("[SYNC] syncExamplePrograms aufgerufen für %s", username)

	// Prüfe, ob der Nutzer angemeldet ist
	if username == "" {
		tinyOSDebugLog("[SYNC] Kein Username")
		return fmt.Errorf("kein Benutzer angemeldet")
	}

	// Prüfe, ob das virtuelle Dateisystem initialisiert wurde
	if os.Vfs == nil {
		tinyOSDebugLog("[SYNC] VFS ist nil")
		return fmt.Errorf("virtuelles Dateisystem nicht initialisiert")
	}
	tinyOSDebugLog("[SYNC] Beginne Synchronisation")
	logMessage("[TINYOS] Synchronisiere Beispielprogramme für Benutzer %s", username)
	// Special handling for dyson user - copy files from dyson directory instead of examples
	if username == "dyson" {
		tinyOSDebugLog("[SYNC] Spezielle Dyson-Synchronisation")
		err := os.syncDysonFiles(username)
		if err != nil {
			tinyOSDebugLog("[SYNC] Dyson-Sync fehlgeschlagen: %v", err)
			logMessage("[TINYOS] Fehler bei Dyson-Dateien-Sync: %v", err)
		} else {
			tinyOSDebugLog("[SYNC] Dyson-Dateien erfolgreich synchronisiert")
			logMessage("[TINYOS] Dyson-spezifische Dateien erfolgreich synchronisiert")
		}
		return nil // Early return for dyson - don't sync normal examples
	}
	// Create home directory if it doesn't exist
	homePath := "/home/" + username
	if !os.Vfs.Exists(homePath, "") {
		logMessage("[TINYOS] Home directory for %s doesn't exist, creating it", username)
		err := os.Vfs.MkdirAll(homePath)
		if err != nil {
			logMessage("[TINYOS] Error creating home directory: %v", err)
			return fmt.Errorf("error creating home directory: %v", err)
		}
	}

	// Create basic subdirectory for BASIC programs and SID files
	basicPath := homePath + "/basic"
	if !os.Vfs.Exists(basicPath, "") {
		logMessage("[TINYOS] Basic directory for %s doesn't exist, creating it", username)
		err := os.Vfs.MkdirAll(basicPath)
		if err != nil {
			logMessage("[TINYOS] Error creating basic directory: %v", err)
			return fmt.Errorf("error creating basic directory: %v", err)
		}
	}
	// Beispielprogramme dynamisch aus dem examples-Ordner lesen
	tinyOSDebugLog("[SYNC] Lese physische Beispiele")
	examples, err := os.readPhysicalExamples()
	if err != nil {
		tinyOSDebugLog("[SYNC] Fehler beim Lesen der Beispiele: %v", err)
		logMessage("[TINYOS] Fehler beim Lesen der Beispielprogramme: %v", err)
		return err
	}
	tinyOSDebugLog("[SYNC] %d Beispiele gelesen", len(examples))

	// Kopiere/aktualisiere jedes Beispiel mit direktem Datenbankzugriff
	var copyErrors []error
	logMessage("[TINYOS] Beginne mit der Synchronisierung von %d Beispielprogrammen", len(examples))
	for filename, content := range examples {
		filePath := fmt.Sprintf("/home/%s/basic/%s", username, filename)
		logMessage("[TINYOS] Writing example file %s directly", filePath)

		// Direkter DB-Zugriff als zuverlässigster Weg
		if os.db != nil {
			isDir := 0 // Datei, nicht Verzeichnis
			modTime := time.Now().Unix()

			// Direkt ausführen ohne Zuweisung der nicht benutzten Variable
			_, err := os.db.Exec(
				`INSERT OR REPLACE INTO virtual_files (username, path, content, is_dir, mod_time) 
				VALUES (?, ?, ?, ?, ?)`,
				username, filePath, content, isDir, modTime)

			if err != nil {
				logMessage("[TINYOS] DB-Fehler beim Kopieren des Beispielprogramms %s: %v", filename, err)
				copyErrors = append(copyErrors, err)
			} else {
				logMessage("[TINYOS] Beispiel %s erfolgreich in DB", filename)
			}
		} else {
			logMessage("[TINYOS] WARNUNG: Datenbank-Handle ist nil! Direkte DB-Operationen nicht möglich.")
		}
	}

	// Prüfe, ob Fehler bei der DB-Operation aufgetreten sind
	if len(copyErrors) > 0 {
		return fmt.Errorf("fehler beim Kopieren von %d Beispielprogrammen (aber einige könnten trotzdem erfolgreich kopiert worden sein)", len(copyErrors))
	}
	// Überprüfe mit direkter Datenbankabfrage, ob die Dateien jetzt wirklich vorhanden sind
	if os.db != nil {
		var count int
		err := os.db.QueryRow(
			`SELECT COUNT(*) FROM virtual_files 
			WHERE username = ? 
			AND is_dir = 0 
			AND path LIKE ? 
			AND (LOWER(path) LIKE '%.bas' OR LOWER(path) LIKE '%.sid')`,
			username, "/home/"+username+"/basic/%").Scan(&count)

		if err != nil {
			logMessage("[TINYOS] Fehler bei der Überprüfung der kopierten Dateien: %v", err)
		} else {
			logMessage("[TINYOS] %d Programmdateien (.bas/.sid) wurden für Benutzer %s erfolgreich gespeichert", count, username)

			// Liste alle Dateien zur Überprüfung auf
			rows, err := os.db.Query(
				`SELECT path FROM virtual_files 
				WHERE username = ? 
				AND is_dir = 0 
				AND path LIKE ? 
				AND (LOWER(path) LIKE '%.bas' OR LOWER(path) LIKE '%.sid')`,
				username, "/home/"+username+"/basic/%")

			if err != nil {
				logMessage("[TINYOS] Fehler beim Abfragen der Dateipfade: %v", err)
			} else {
				defer rows.Close()
				logMessage("[TINYOS] Gefundene Programmdateien (.bas/.sid) für Benutzer %s:", username)
				i := 0
				for rows.Next() {
					var path string
					if err := rows.Scan(&path); err == nil {
						logMessage("[TINYOS]  - %s", path)
						i++
					}
				}
				if i == 0 {
					logMessage("[TINYOS] Keine Dateien gefunden, obwohl Count=%d", count)
				}
			}

			if count == 0 {
				return fmt.Errorf("konnte keine Beispielprogramme für Benutzer %s speichern", username)
			}
		}
	}

	// Füge eine erfolgreiche Abschlussmeldung hinzu
	logMessage("[TINYOS] Synchronisierung der Beispielprogramme für Benutzer %s erfolgreich abgeschlossen", username)
	return nil
}

// syncDysonFiles copies all files from the dyson directory to Dyson's home directory
func (os *TinyOS) syncDysonFiles(username string) error {
	fmt.Printf("[DYSON-SYNC] Starting Dyson file synchronization for %s\n", username)

	// Ensure home directory exists
	homePath := "/home/" + username
	if !os.Vfs.Exists(homePath, "") {
		err := os.Vfs.MkdirAll(homePath)
		if err != nil {
			return fmt.Errorf("failed to create home directory: %v", err)
		}
	}

	// Create basic subdirectory for BASIC files
	basicPath := homePath + "/basic"
	if !os.Vfs.Exists(basicPath, "") {
		err := os.Vfs.MkdirAll(basicPath)
		if err != nil {
			return fmt.Errorf("failed to create basic directory: %v", err)
		}
	}
	// Read files from physical dyson directory
	dysonDir := "dyson"
	files, err := os.readPhysicalDirectory(dysonDir)
	if err != nil {
		return fmt.Errorf("failed to read dyson directory: %v", err)
	}

	fmt.Printf("[DYSON-SYNC] Found %d files in dyson directory\n", len(files))
	// Copy each file to appropriate location using VFS
	var copyErrors []error
	for filename, content := range files {
		var targetPath string

		// Determine target path based on file extension
		if strings.HasSuffix(strings.ToLower(filename), ".bas") {
			targetPath = fmt.Sprintf("%s/basic/%s", homePath, filename)
		} else {
			targetPath = fmt.Sprintf("%s/%s", homePath, filename)
		}

		// Use VFS to write file (this will handle both in-memory and database storage)
		err := os.Vfs.WriteFile(targetPath, content, username)
		if err != nil {
			logMessage("[DYSON-SYNC] VFS error copying %s: %v", filename, err)
			copyErrors = append(copyErrors, err)
		} else {
			logMessage("[DYSON-SYNC] Successfully copied %s to %s", filename, targetPath)
		}
	}

	if len(copyErrors) > 0 {
		return fmt.Errorf("failed to copy %d Dyson files", len(copyErrors))
	}

	logMessage("[DYSON-SYNC] Successfully synchronized %d Dyson files", len(files))
	return nil
}

// readPhysicalExamples liest alle .bas, .sid und .txt Dateien aus dem examples-Ordner
func (os *TinyOS) readPhysicalExamples() (map[string]string, error) {
	examples := make(map[string]string)
	files, err := gos.ReadDir("examples")
	if err != nil {
		return nil, fmt.Errorf("Fehler beim Lesen des examples-Ordners: %v", err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		lowerName := strings.ToLower(name) // Unterstütze .bas, .sid und .txt Dateien
		if strings.HasSuffix(lowerName, ".bas") || strings.HasSuffix(lowerName, ".sid") || strings.HasSuffix(lowerName, ".txt") {
			contentBytes, err := gos.ReadFile("examples/" + name)
			if err != nil {
				fmt.Printf("Fehler beim Lesen der Beispieldatei %s: %v\n", name, err)
				continue
			}
			examples[name] = string(contentBytes)
		}
	}
	return examples, nil
}

// readPhysicalDirectory reads all files from a specified directory
func (os *TinyOS) readPhysicalDirectory(dirPath string) (map[string]string, error) {
	files := make(map[string]string)
	dirEntries, err := gos.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %v", dirPath, err)
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		contentBytes, err := gos.ReadFile(dirPath + "/" + name)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", name, err)
			continue
		}
		files[name] = string(contentBytes)
	}

	return files, nil
}

// readPhysicalFile liest eine Datei aus dem physischen Dateisystem
func (os *TinyOS) readPhysicalFile(path string) (string, error) {
	// Diese Funktion würde in einer echten Implementierung os.ReadFile verwenden
	// Hier verwenden wir einen spezifischen Pfad, da wir direkt mit dem lokalen Dateisystem arbeiten

	// Spezialfall für sound_demo.bas
	if path == "examples/sound_demo.bas" {
		// Hier den Inhalt der Datei zurückgeben
		return `10 REM Sound und Sprachausgabe Demo
20 CLS
30 PRINT "TinyBASIC Sound und Sprach-Demo"
40 PRINT "------------------------------"
50 PRINT
60 PRINT "BEEP-Befehl:"
70 BEEP
80 FOR I = 1 TO 2000
90 NEXT I
100 PRINT
110 PRINT "Sprachausgabe mit SAY:"
120 SAY "Hallo, ich bin der Sprach Synthetisator"
130 FOR I = 1 TO 4000
140 NEXT I
150 PRINT
160 PRINT "Sprachausgabe mit SPEAK:"
170 SPEAK "Ich kann auch mit dem SPEAK Befehl sprechen"
180 FOR I = 1 TO 4000
190 NEXT I
200 PRINT
210 PRINT "Toene mit SOUND:"
220 FOR F = 100 TO 1000 STEP 100
230   PRINT "Frequenz: "; F
240   SOUND F,200
250   FOR I = 1 TO 500
260   NEXT I
270 NEXT F
280 PRINT
290 PRINT "Demo beendet!"
300 END`, nil
	}

	// Für andere Dateien einen Fehler zurückgeben
	return "", fmt.Errorf("datei nicht gefunden: %s", path)
}

// copyExamplePrograms kopiert Beispielprogramme ins Heimatverzeichnis eines neuen Benutzers
// und gibt einen Fehler zurück, falls der Kopiervorgang fehlschlägt
func (os *TinyOS) copyExamplePrograms(username string) error {
	// Prüfen, ob das virtuelle Dateisystem initialisiert wurde
	if os.Vfs == nil {
		logMessage("[TINYOS] Fehler beim Kopieren der Beispielprogramme: virtuelles Dateisystem nicht initialisiert")
		return fmt.Errorf("virtuelles Dateisystem nicht initialisiert")
	}

	logMessage("[TINYOS] Beginne das Kopieren der Beispielprogramme für Benutzer %s", username)

	// Beispielprogramme dynamisch aus dem examples-Ordner lesen
	examples, err := os.readPhysicalExamples()
	if err != nil {
		logMessage("[TINYOS] Fehler beim Lesen der Beispielprogramme: %v", err)
		return err
	}

	// Prüfe, ob das Home-Verzeichnis existiert, falls nicht, erstelle es
	homePath := "/home/" + username
	logMessage("[TINYOS] Prüfe Home-Verzeichnis %s für Benutzer %s", homePath, username)
	if !os.Vfs.Exists(homePath, "") {
		logMessage("[TINYOS] Home-Verzeichnis existiert nicht, erstelle es")
		err := os.Vfs.Mkdir(homePath)
		if err != nil {
			logMessage("[TINYOS] Fehler beim Erstellen des Home-Verzeichnisses: %v", err)
			return fmt.Errorf("fehler beim Erstellen des Home-Verzeichnisses: %v", err)
		}
	}

	// Direkte Schreiboperation über das VFS für alle Beispielprogramme mit Timeout
	var copyErrors []error
	for filename, content := range examples {
		filePath := fmt.Sprintf("/home/%s/%s", username, filename)
		logMessage("[TINYOS] Kopiere Beispiel %s nach %s", filename, filePath)

		// Erstellen eines Channels für den Timeout
		done := make(chan error, 1)

		// Ausführen der Schreiboperation in einer Goroutine
		go func(path, content string) {
			done <- os.Vfs.WriteFile(path, content, "")
		}(filePath, content)

		// Warten auf Abschluss mit Timeout
		select {
		case err := <-done:
			if err != nil {
				logMessage("[TINYOS] Fehler beim Kopieren des Beispielprogramms %s: %v", filename, err)
				copyErrors = append(copyErrors, err)
			} else {
				logMessage("[TINYOS] Beispielprogramm %s erfolgreich kopiert", filename)
			}
		case <-time.After(5 * time.Second):
			logMessage("[TINYOS] Timeout beim Kopieren des Beispielprogramms %s", filename)
			copyErrors = append(copyErrors, fmt.Errorf("timeout beim Kopieren von %s", filename))
		}
	}

	// Prüfe, ob Fehler aufgetreten sind
	if len(copyErrors) > 0 {
		return fmt.Errorf("fehler beim Kopieren von %d Beispielprogrammen", len(copyErrors))
	}

	// Überprüfe, ob die Dateien tatsächlich erstellt wurden (mit Timeout)
	if os.db != nil {
		homeDir := "/home/" + username

		// Channel für Timeout-Kontrolle
		done := make(chan bool, 1)
		var filesFound int
		// Datenbankabfrage in Goroutine
		go func() {
			query := `SELECT COUNT(*) FROM virtual_files 
				WHERE username = ? 
				AND is_dir = 0 
				AND path LIKE ? 
				AND (LOWER(path) LIKE '%.bas' OR LOWER(path) LIKE '%.sid')`

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Verwenden eines vorbereiteten Statements mit Timeout-Kontext
			stmt, err := os.db.PrepareContext(ctx, query)
			if err != nil {
				logMessage("[TINYOS] Fehler beim Vorbereiten der Datenbankabfrage: %v", err)
				done <- false
				return
			}
			defer stmt.Close()

			err = stmt.QueryRowContext(ctx, username, homeDir+"/%").Scan(&filesFound)
			if err != nil {
				logMessage("[TINYOS] Fehler bei der Datenbankabfrage: %v", err)
				done <- false
				return
			}

			done <- true
		}()
		// Warten auf Abschluss mit Timeout
		select {
		case <-done:
			logMessage("[TINYOS] Insgesamt %d Programmdateien (.bas/.sid) im Heimatverzeichnis gefunden", filesFound)
			// Wenn keine Dateien gefunden wurden, könnte ein Problem vorliegen
			if filesFound == 0 {
				return fmt.Errorf("keine Programmdateien (.bas/.sid) im Heimatverzeichnis gefunden")
			}
		case <-time.After(10 * time.Second):
			logMessage("[TINYOS] Timeout bei der Überprüfung der kopierten Dateien")
			return fmt.Errorf("timeout bei der Überprüfung der kopierten Dateien")
		}
	}

	return nil
}

// CleanupGuestSession bereinigt die temporären Ressourcen einer Gast-Session
func (os *TinyOS) CleanupGuestSession() error {
	logMessage("[TINYOS] CleanupGuestSession wird aufgerufen")

	// Wenn die VFS-Instanz nicht initialisiert ist, können wir nichts tun
	if os.Vfs == nil {
		return fmt.Errorf("virtuelles Dateisystem nicht initialisiert")
	}

	// Bereinigung des Gast-VFS durchführen
	err := os.Vfs.CleanupGuestVFS()
	if err != nil {
		logMessage("[TINYOS] Fehler beim Bereinigen des Gast-VFS: %v", err)
		return fmt.Errorf("fehler beim Bereinigen des Gast-VFS: %v", err)
	}

	// Alle Gast-Sessions aus der internen Speicherung entfernen
	os.mu.Lock()
	os.guestSessions = nil
	os.mu.Unlock()

	logMessage("[TINYOS] Gast-Session erfolgreich bereinigt")
	return nil
}

// IsIPRestricted checks if an IP address is restricted for registration
func (os *TinyOS) IsIPRestricted(ip string) bool {
	if os.db == nil {
		return false
	}

	// Check if a registration from this IP has already occurred in the last 24 hours
	cutoffTime := time.Now().Add(-24 * time.Hour).Unix()

	var count int
	err := os.db.QueryRow(
		"SELECT COUNT(*) FROM registration_attempts WHERE ip_address = ? AND timestamp > ?",
		ip, cutoffTime).Scan(&count)

	if err != nil {
		logger.Error(logger.AreaAuth, "Error checking IP restriction for %s: %v", ip, err)
		return false
	}

	// If a registration already exists, the IP is restricted
	logger.Info(logger.AreaAuth, "IP restriction check for %s: %d registrations in last 24h", ip, count)
	return count > 0
}

// AddRegistrationAttempt stores a registration attempt for an IP address
func (os *TinyOS) AddRegistrationAttempt(ip string) {
	if os.db == nil {
		return
	}

	// Current time as Unix timestamp
	now := time.Now().Unix()

	_, err := os.db.Exec(
		"INSERT INTO registration_attempts (ip_address, timestamp) VALUES (?, ?)",
		ip, now)

	if err != nil {
		logger.Error(logger.AreaAuth, "Error storing registration attempt for IP %s: %v", ip, err)
	} else {
		logger.Info(logger.AreaAuth, "Recorded registration attempt for IP %s", ip)
	}
}

// UserExists prüft, ob ein Benutzer bereits existiert
func (os *TinyOS) UserExists(username string) bool {
	var exists int
	err := os.db.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username).Scan(&exists)

	if err != nil {
		// Bei Datenbankfehlern nehmen wir an, dass der Benutzer nicht existiert
		return false
	}

	return exists > 0
}

// RegisterUser registriert einen neuen Benutzer
func (os *TinyOS) RegisterUser(username, password, ipAddress string) error {
	tinyOSDebugLog("[REGISTER] RegisterUser aufgerufen für %s", username)

	// Prüfe, ob der Benutzername bereits existiert
	var exists int
	err := os.db.QueryRow(
		"SELECT COUNT(*) FROM users WHERE username = ?",
		username).Scan(&exists)

	if err != nil {
		tinyOSDebugLog("[REGISTER] Datenbankfehler beim Prüfen: %v", err)
		return fmt.Errorf("fehler bei der Benutzerprüfung: %v", err)
	}

	if exists > 0 {
		tinyOSDebugLog("[REGISTER] Benutzer existiert bereits")
		return fmt.Errorf("benutzername bereits vergeben")
	}

	tinyOSDebugLog("[REGISTER] Benutzer existiert noch nicht, erstelle...")

	// Passwort hashen
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("fehler beim Hashen des Passworts: %v", err)
	}

	// Benutzer in Datenbank eintragen
	now := time.Now().Unix()
	_, err = os.db.Exec(
		"INSERT INTO users (username, password, created_at, ip_address) VALUES (?, ?, ?, ?)",
		username, string(hashedPassword), now, ipAddress)

	if err != nil {
		return fmt.Errorf("fehler beim Erstellen des Benutzers: %v", err)
	}

	// Erstelle das Home-Verzeichnis für den neuen Benutzer
	homePath := "/home/" + username
	tinyOSDebugLog("[REGISTER] Erstelle Home-Verzeichnis: %s", homePath)
	err = os.Vfs.MkdirAll(homePath) // Korrigiert: Nur ein Argument
	if err != nil {
		tinyOSDebugLog("[REGISTER] Fehler beim Erstellen des Home-Verzeichnisses: %v", err)
		// Versuche, den Benutzer zu löschen, wenn das Home-Verzeichnis nicht erstellt werden konnte
		// Dies ist wichtig, um inkonsistente Zustände zu vermeiden
		if delErr := os.deleteUserFromDB(username); delErr != nil {
			logMessage("[TINYOS] Kritischer Fehler: Benutzer konnte nicht gelöscht werden nach fehlgeschlagener Home-Verzeichnis-Erstellung: %v", delErr)
		}
		return fmt.Errorf("fehler beim Erstellen des Home-Verzeichnisses: %v", err)
	}
	tinyOSDebugLog("[REGISTER] Home-Verzeichnis erstellt: %s", homePath)

	// Initialisiere das Dateisystem für den neuen Benutzer
	tinyOSDebugLog("[REGISTER] Initialisiere VFS")
	err = os.Vfs.InitializeUserVFS(username) // Beibehaltung des Aufrufs, Überprüfung nach anderen Fixes
	if err != nil {
		tinyOSDebugLog("[REGISTER] VFS-Initialisierung fehlgeschlagen: %v", err)
		// Optional: Hier könnte man überlegen, ob der Benutzer wieder gelöscht werden soll,
		// falls die Initialisierung des Dateisystems fehlschlägt.
		// Für den Moment loggen wir den Fehler und fahren fort.
	} else {
		tinyOSDebugLog("[REGISTER] VFS initialisiert")
	}
	tinyOSDebugLog("[REGISTER] RegisterUser erfolgreich abgeschlossen")
	return nil
}

// isLoginBlocked checks if login attempts are blocked for a given IP address
func (os *TinyOS) isLoginBlocked(ipAddress string) (bool, int) {
	os.loginAttemptMutex.RLock()
	defer os.loginAttemptMutex.RUnlock()

	tracker, exists := os.failedLoginAttempts[ipAddress]
	if !exists {
		return false, 0
	}

	// Check if lockout has expired
	if time.Now().After(tracker.LockedUntil) {
		return false, 0
	}

	// Calculate remaining seconds
	remaining := int(tracker.LockedUntil.Sub(time.Now()).Seconds())
	if remaining < 0 {
		remaining = 0
	}

	return !tracker.LockedUntil.IsZero() && time.Now().Before(tracker.LockedUntil), remaining
}

// recordFailedLoginAttempt records a failed login attempt for an IP address
func (os *TinyOS) recordFailedLoginAttempt(ipAddress string) {
	os.loginAttemptMutex.Lock()
	defer os.loginAttemptMutex.Unlock()

	maxAttempts := configuration.GetInt("Authentication", "max_failed_login_attempts", 5)
	lockoutDuration := time.Duration(configuration.GetInt("Authentication", "login_lockout_duration_seconds", 30)) * time.Second

	tracker, exists := os.failedLoginAttempts[ipAddress]
	if !exists {
		tracker = &LoginAttemptTracker{
			FailedAttempts: 0,
			LastAttempt:    time.Time{},
			LockedUntil:    time.Time{},
		}
		os.failedLoginAttempts[ipAddress] = tracker
	}

	// Reset attempts if enough time has passed (1 hour)
	if time.Since(tracker.LastAttempt) > time.Hour {
		tracker.FailedAttempts = 0
	}

	tracker.FailedAttempts++
	tracker.LastAttempt = time.Now()

	// Lock account if max attempts reached
	if tracker.FailedAttempts >= maxAttempts {
		tracker.LockedUntil = time.Now().Add(lockoutDuration)
		logger.SecurityInfo("Login blocked for IP %s after %d failed attempts. Locked until %v",
			ipAddress, tracker.FailedAttempts, tracker.LockedUntil)
	}
}

// clearFailedLoginAttempts clears failed login attempts for an IP address after successful login
func (os *TinyOS) clearFailedLoginAttempts(ipAddress string) {
	os.loginAttemptMutex.Lock()
	defer os.loginAttemptMutex.Unlock()

	delete(os.failedLoginAttempts, ipAddress)
}

// LoginUser meldet einen Benutzer an
func (os *TinyOS) LoginUser(username, password, ipAddress string) ([]shared.Message, string, error) {
	tinyOSDebugLog("[LOGIN] LoginUser aufgerufen für %s", username)
	logMessage("[TINYOS] LoginUser aufgerufen für Benutzer %s von IP %s", username, ipAddress)
	// Check if login is blocked for this IP address
	isBlocked, remainingSeconds := os.isLoginBlocked(ipAddress)
	if isBlocked {
		logger.SecurityWarn("Login attempt blocked for IP %s. %d seconds remaining", ipAddress, remainingSeconds)
		return nil, "", fmt.Errorf("Too many login attempts. Try again in %d seconds", remainingSeconds)
	}

	// Check database connection
	if os.db == nil {
		return nil, "", fmt.Errorf("keine Datenbankverbindung verfügbar")
	}
	// Passwort-Hash aus der Datenbank abrufen
	var storedHash string
	var userID int64
	err := os.db.QueryRow("SELECT rowid, password FROM users WHERE username = ?", username).Scan(&userID, &storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			// Record failed login attempt for unknown username
			os.recordFailedLoginAttempt(ipAddress)
			logger.SecurityWarn("Login failed for unknown user '%s' from IP %s", username, ipAddress)
			return nil, "", fmt.Errorf("invalid username or password")
		}
		return nil, "", fmt.Errorf("database error: %v", err)
	}
	// Password verification
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		// Record failed login attempt for wrong password
		os.recordFailedLoginAttempt(ipAddress)
		logger.SecurityWarn("Login failed for user '%s' from IP %s: incorrect password", username, ipAddress)
		return nil, "", fmt.Errorf("invalid username or password")
	}
	// Prüfen, ob der Benutzer oder die IP gebannt ist
	isBanned, banMessage := os.IsBanned(username, ipAddress)
	if isBanned {
		logMessage("[TINYOS] Anmeldung verweigert für gebannten Benutzer %s oder IP %s", username, ipAddress)
		return nil, "", fmt.Errorf(banMessage)
	}

	// Clear failed login attempts for this IP after successful authentication
	os.clearFailedLoginAttempts(ipAddress)
	logger.SecurityInfo("Successful login for user '%s' from IP %s - cleared failed login attempts", username, ipAddress)

	// Anmeldestatus in der Datenbank aktualisieren
	_, err = os.db.Exec("UPDATE users SET is_logged_in = 1, last_login = CURRENT_TIMESTAMP, ip_address = ? WHERE username = ?", ipAddress, username)
	if err != nil {
		logMessage("[TINYOS] Fehler beim Aktualisieren der Anmeldestatus: %v", err)
	}
	// Home-Verzeichnis des Benutzers setzen	// Home-Verzeichnis des Benutzers setzen
	homePath := "/home/" + username

	// Neue Session erstellen
	sessionID := GenerateSessionID()
	session := &Session{
		ID:           sessionID,
		Username:     username,
		IPAddress:    ipAddress,
		CurrentPath:  homePath, // Setze das aktuelle Verzeichnis der Session auf das Home-Verzeichnis
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// Session speichern
	os.sessionMutex.Lock()
	os.sessions[sessionID] = session
	os.sessionMutex.Unlock()

	// Session in der Datenbank speichern
	if os.db != nil {
		_, err := os.db.Exec(
			"INSERT INTO user_sessions (session_id, user_id, username, ip_address, created_at, last_activity) VALUES (?, ?, ?, ?, ?, ?)",
			sessionID, userID, username, ipAddress, session.CreatedAt, session.LastActivity,
		)
		if err != nil {
			logMessage("[TINYOS] Fehler beim Speichern der Session in der Datenbank: %v", err)
		}
	}

	// Benutzer-VFS aus der Datenbank laden und aktivieren
	tinyOSDebugLog("[LOGIN] Initialisiere VFS")
	err = os.Vfs.InitializeUserVFS(username)
	if err != nil {
		tinyOSDebugLog("[LOGIN] VFS-Initialisierung fehlgeschlagen: %v", err)
		// Wenn das Home-Verzeichnis nicht existiert, versuche es explizit zu erstellen
		if !os.Exists(homePath) {
			err = os.Vfs.MkdirAll(homePath)
			if err != nil {
				logMessage("[TINYOS] Fehler beim Erstellen des Home-Verzeichnisses: %v", err)
			}
		}
	} else {
		tinyOSDebugLog("[LOGIN] VFS initialisiert")
	}

	// Beispielprogramme synchronisieren für den Benutzer
	tinyOSDebugLog("[LOGIN] Synchronisiere Beispielprogramme")
	err = os.syncExamplePrograms(username)
	if err != nil {
		tinyOSDebugLog("[LOGIN] Beispielprogramme-Sync fehlgeschlagen: %v", err)
		// Kein fataler Fehler, da dies nur die Beispielprogramme betrifft	} else {
		tinyOSDebugLog("[LOGIN] Beispielprogramme synchronisiert")
	}

	tinyOSDebugLog("[LOGIN] LoginUser erfolgreich abgeschlossen")
	logger.Info(logger.AreaAuth, "Benutzer %s erfolgreich angemeldet mit Session %s", username, sessionID)

	// Return welcome messages and session ID
	messages := []shared.Message{
		{Type: shared.MessageTypeText, Content: "Login successful!"},
		{Type: shared.MessageTypeText, Content: "Welcome, " + username + "!"},
		{Type: shared.MessageTypeSound, Content: "beep"},
		{Type: shared.MessageTypePrompt, PromptSymbol: os.GetPromptForSession(sessionID)}, // Set prompt with current path
	}

	// For temporary users, add a special message to trigger token refresh in frontend
	if isTemporaryUser(username) {
		logger.Info(logger.AreaAuth, "Temporary user %s logged in - requesting frontend token refresh", username)
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeAuthRefresh,
			Content: "temporary_user_login",
		})
	}

	return messages, sessionID, nil
}

// LogoutUser meldet einen Benutzer ab
func (os *TinyOS) LogoutUser(sessionID string) []shared.Message {
	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()

	session, exists := os.sessions[sessionID]
	if !exists {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Keine aktive Sitzung zum Abmelden gefunden."}}
	}

	username := session.Username
	delete(os.sessions, sessionID)

	// Session aus der Datenbank löschen
	if os.db != nil {
		_, err := os.db.Exec("DELETE FROM user_sessions WHERE session_id = ?", sessionID)
		if err != nil {
			logMessage("[TINYOS] Fehler beim Löschen der Session aus der Datenbank: %v", err)
		}
	}

	logMessage("[TINYOS] Benutzer %s (Session %s) abgemeldet.", username, sessionID)
	bFalse := false // Definiere bFalse für den Pointer
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "Erfolgreich abgemeldet."},
		{Type: shared.MessageTypeInputControl, InputEnabled: &bFalse}, // Korrigiert: InputEnabled verwenden
		{Type: shared.MessageTypeSession, Content: ""},                // Session-ID beim Client löschen
	}
}

// deleteUserFromDB löscht einen Benutzer aus der Datenbank.
// Dies ist eine grundlegende Implementierung und muss möglicherweise erweitert werden.
func (os *TinyOS) deleteUserFromDB(username string) error {
	if os.db == nil {
		return fmt.Errorf("datenbank nicht initialisiert")
	}
	_, err := os.db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("fehler beim Löschen des Benutzers %s aus der datenbank: %v", username, err)
	}
	logMessage("[TINYOS] Benutzer %s aus der Datenbank gelöscht.", username)
	return nil
}

// Exists prüft, ob eine Datei oder ein Verzeichnis im virtuellen Dateisystem existiert
func (os *TinyOS) Exists(path string) bool {
	if os.Vfs == nil {
		return false
	}
	// Füge leeren String als sessionID-Parameter hinzu
	return os.Vfs.Exists(path, "")
}

// ExistsWithSession prüft, ob eine Datei oder ein Verzeichnis im virtuellen Dateisystem existiert,
// und berücksichtigt dabei die Session-ID
func (os *TinyOS) ExistsWithSession(path string, sessionID string) bool {
	if os.Vfs == nil {
		return false
	}
	return os.Vfs.Exists(path, sessionID)
}

// Username gibt den Benutzernamen basierend auf dem aktuellen Thread-lokalen Kontext zurück
func (os *TinyOS) Username() string {
	// In einer korrekten Implementierung würden wir den Thread-lokalen Kontext abfragen
	// Da Go kein direktes Thread-Local Storage hat, verwenden wir eine kontextbasierte Lösung

	// Zuerst versuchen wir, den Benutzernamen aus dem aktuellen Thread-lokalen Kontext zu holen
	// // In einer tatsächlichen Implementierung könnte dies z.B. über Context-Objekte oder ein Request-spezifisches Mapping erfolgen

	// Da wir keinen direkten Zugriff auf den Kontext haben, verwenden wir eine alternative Strategie:
	// 1. Überprüfen, ob es eine aktive Session im aktuellen Goroutine-Kontext gibt

	// 2. Wenn nicht, als Fallback die letzte aktive Session finden

	// Aktuelle Implementierung: Fallback-Strategie - finde den Benutzer mit der neuesten Aktivität
	os.sessionMutex.RLock()
	defer os.sessionMutex.RUnlock()

	var mostRecentSession *Session
	var mostRecentTime time.Time

	for _, session := range os.sessions {
		// Ignoriere Gast-Sessions (beginnen mit "guest-")
		if strings.HasPrefix(session.Username, "guest-") {
			continue
		}

		// Finde die Session mit der neuesten Aktivität
		if mostRecentSession == nil || session.LastActivity.After(mostRecentTime) {
			mostRecentSession = session
			mostRecentTime = session.LastActivity
		}
	}

	if mostRecentSession != nil {
		return mostRecentSession.Username
	}

	// Wenn keine aktive Session gefunden wurde, geben wir einen leeren String zurück
	return ""
}

// ListDirBasFiles listet alle BASIC-Dateien (*.bas) im Home-Verzeichnis des aktuellen Benutzers auf
// DEPRECATED: Use ListDirProgramFiles instead
func (os *TinyOS) ListDirBasFiles() ([]string, error) {
	return os.ListDirProgramFiles()
}

// ListDirBasFilesWithSession listet alle BASIC-Dateien (*.bas) für eine bestimmte Session auf
// DEPRECATED: Use ListDirProgramFilesWithSession instead
func (os *TinyOS) ListDirBasFilesWithSession(sessionID string) ([]string, error) {
	return os.ListDirProgramFilesWithSession(sessionID)
}

// ListDirProgramFiles listet alle Programmdateien (*.bas und *.sid) im Home-Verzeichnis des aktuellen Benutzers auf
func (os *TinyOS) ListDirProgramFiles() ([]string, error) { // Da keine Session-ID verfügbar ist, verwenden wir eine leere Session für Gast
	return os.ListDirProgramFilesWithSession("")
}

// ListDirProgramFilesWithSession listet alle Programmdateien (*.bas und *.sid) für eine bestimmte Session auf
func (os *TinyOS) ListDirProgramFilesWithSession(sessionID string) ([]string, error) {
	if os.Vfs == nil {
		return nil, fmt.Errorf("virtuelles Dateisystem nicht initialisiert")
	}
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		username = "guest"
	}

	return os.Vfs.ListDirProgramFilesForUser(username)
}

// ListDirProgramFilesForUser gibt eine Liste von Programmdateien (.bas und .sid) für den angegebenen Benutzer zurück.
func (os *TinyOS) ListDirProgramFilesForUser(username string) ([]string, error) {
	if os.Vfs == nil {
		return nil, fmt.Errorf("virtuelles Dateisystem nicht initialisiert")
	}

	// Für Gastbenutzer die RAM-VFS verwenden
	if username == "guest" {
		if err := os.syncExamplePrograms(username); err != nil {
			return nil, fmt.Errorf("fehler beim Synchronisieren der Gast-Programme: %v", err)
		}
		return os.Vfs.ListDirProgramFiles("") // Die RAM-VFS-Version verwenden
	}

	// Für reguläre Benutzer
	return os.Vfs.ListDirProgramFilesForUser(username)
}

// ReadFile liest den Inhalt einer Datei im virtuellen Dateisystem
func (os *TinyOS) ReadFile(path string) (string, error) {
	if os.Vfs == nil {
		return "", fmt.Errorf("virtuelles Dateisystem nicht initialisiert")
	}
	// Verwende die vorhandene Methode des VFS ohne SessionID
	return os.Vfs.ReadFile(path, "")
}

// ReadFileWithSession liest eine Datei mit einer bestimmten Session-ID
func (os *TinyOS) ReadFileWithSession(filename string, sessionID string) (string, error) {
	log.Printf("[DEBUG-TINYOS] ReadFileWithSession: filename=%s, sessionID=%s", filename, sessionID)
	resolvedPath, err := os.ResolvePath(filename, sessionID)
	if err != nil {
		log.Printf("[DEBUG-TINYOS] ReadFileWithSession FEHLER: Pfad konnte nicht aufgelöst werden: %v", err)
		return "", err
	}
	log.Printf("[DEBUG-TINYOS] ReadFileWithSession: resolvedPath=%s", resolvedPath)
	return os.Vfs.ReadFile(resolvedPath, sessionID)
}

// WriteFile schreibt Inhalt in eine Datei im virtuellen Dateisystem
func (os *TinyOS) WriteFile(path, content string) error {
	if os.Vfs == nil {
		return fmt.Errorf("virtuelles Dateisystem nicht initialisiert")
	}
	// Verwende die vorhandene Methode des VFS ohne SessionID
	return os.Vfs.WriteFile(path, content, "")
}

// WriteFileWithSession schreibt in eine Datei mit einer bestimmten Session-ID
func (os *TinyOS) WriteFileWithSession(path, content string, sessionID string) error {
	if os.Vfs == nil {
		return fmt.Errorf("virtuelles Dateisystem nicht initialisiert")
	}
	// Leite den Aufruf an die VFS-Methode mit SessionID weiter
	return os.Vfs.WriteFile(path, content, sessionID)
}

// UsernameFromSession gibt den Benutzernamen für eine bestimmte Session-ID zurück
func (os *TinyOS) UsernameFromSession(sessionID string) string {
	// Diese Methode verwendet die bereits vorhandene GetUsernameForSession
	return os.GetUsernameForSession(sessionID)
}

// CreateGuestSession legt eine neue Gast-Session mit gegebener SessionID und IP an und gibt sie zurück
func (os *TinyOS) CreateGuestSession(sessionID, ip string) (*Session, error) {
	session := &Session{
		ID:           sessionID,
		Username:     "guest",
		IPAddress:    ip,
		CurrentPath:  "/home/guest", // Gast-Sessions starten im Home-Verzeichnis
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}
	os.sessionMutex.Lock()
	os.sessions[sessionID] = session
	os.sessionMutex.Unlock()
	// Gast-VFS für neue Session zurücksetzen und neu initialisieren
	if os.Vfs != nil {
		// Zunächst alte Gast-Dateien bereinigen
		err := os.Vfs.CleanupGuestVFS()
		if err != nil {
			logMessage("[TINYOS] Warnung: Fehler beim Bereinigen des Gast-VFS: %v", err)
		}

		// Dann frisches VFS initialisieren
		err = os.Vfs.InitializeGuestVFS()
		if err != nil {
			logMessage("[TINYOS] Warnung: Fehler beim Initialisieren des Gast-VFS: %v", err)
			// Nicht als kritischer Fehler behandeln, Session ist trotzdem gültig
		} else {
			logMessage("[TINYOS] Gast-VFS erfolgreich für Session %s zurückgesetzt und initialisiert", sessionID)
		}
	}
	logMessage("[TINYOS] Gast-Session %s erstellt für IP %s", sessionID, ip)
	return session, nil
}

// RestoreUserSession restores a user session from a JWT token
func (os *TinyOS) RestoreUserSession(sessionID, username, ip string) (*Session, error) {
	// For temporary users, check if they already have an existing session
	// If the session is older than 15 minutes, don't restore it
	if isTemporaryUser(username) {
		os.sessionMutex.Lock()
		existingSession, exists := os.sessions[sessionID]
		os.sessionMutex.Unlock()

		if exists {
			// Check if the existing session is older than 15 minutes
			sessionAge := time.Since(existingSession.CreatedAt)
			if sessionAge > 15*time.Minute {
				logger.Info(logger.AreaAuth, "Temporary session %s for user %s expired (age: %v), not restoring", sessionID, username, sessionAge)
				// Remove the expired session
				os.sessionMutex.Lock()
				delete(os.sessions, sessionID)
				os.sessionMutex.Unlock()
				return nil, fmt.Errorf("temporary session expired")
			}

			logger.Info(logger.AreaAuth, "Restoring existing temporary session for user %s (age: %v)", username, sessionAge)
			return existingSession, nil
		}

		// Create new temporary session only if none exists
		logger.Info(logger.AreaAuth, "Creating new temporary session for user %s (not persisted)", username)
		session := &Session{
			ID:           sessionID,
			Username:     username,
			IPAddress:    ip,
			CurrentPath:  "/home/" + username,
			CreatedAt:    time.Now(),
			LastActivity: time.Now(),
		}

		os.sessionMutex.Lock()
		os.sessions[sessionID] = session
		os.sessionMutex.Unlock()

		// Initialize VFS for the temporary user if needed
		if os.Vfs != nil {
			err := os.Vfs.InitializeUserVFS(username)
			if err != nil {
				logger.Warn(logger.AreaAuth, "Error initializing user VFS for %s: %v", username, err)
			} else {
				logger.Info(logger.AreaAuth, "User VFS successfully initialized for temporary session %s, user %s", sessionID, username)
			}
		}

		logger.Info(logger.AreaAuth, "Temporary user session %s created for user %s from IP %s", sessionID, username, ip)
		return session, nil
	}

	// First check if session exists in database (for regular users only)
	var dbUsername, dbIP, dbCurrentPath string
	var dbCreatedAt, dbLastActivity int64

	if os.db != nil {
		err := os.db.QueryRow(`
			SELECT username, ip_address, created_at, last_activity, 
			COALESCE(current_path, '/home/' || username) as current_path
			FROM user_sessions WHERE session_id = ?`, sessionID).Scan(
			&dbUsername, &dbIP, &dbCreatedAt, &dbLastActivity, &dbCurrentPath)

		if err == nil {
			// Session exists in database, restore it
			session := &Session{
				ID:           sessionID,
				Username:     dbUsername,
				IPAddress:    dbIP,
				CurrentPath:  dbCurrentPath,
				CreatedAt:    time.Unix(dbCreatedAt, 0),
				LastActivity: time.Unix(dbLastActivity, 0),
			}

			os.sessionMutex.Lock()
			os.sessions[sessionID] = session
			os.sessionMutex.Unlock()

			logger.Info(logger.AreaAuth, "User session %s restored from database for user %s", sessionID, dbUsername)

			// Update last activity
			session.LastActivity = time.Now()
			_, err = os.db.Exec("UPDATE user_sessions SET last_activity = ? WHERE session_id = ?",
				session.LastActivity.Unix(), sessionID)
			if err != nil {
				logger.Warn(logger.AreaAuth, "Failed to update session activity: %v", err)
			}

			// Initialize VFS for the user if needed
			if os.Vfs != nil {
				err := os.Vfs.InitializeUserVFS(dbUsername)
				if err != nil {
					logger.Warn(logger.AreaAuth, "Error initializing user VFS for %s: %v", dbUsername, err)
				}
			}

			return session, nil
		} else {
			logMessage("[TINYOS] Session %s not found in database, creating new session: %v", sessionID, err)
		}
	}

	// Session not in database, create new one
	session := &Session{
		ID:           sessionID,
		Username:     username,
		IPAddress:    ip,
		CurrentPath:  "/home/" + username, // User sessions start in their home directory
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	os.sessionMutex.Lock()
	os.sessions[sessionID] = session
	os.sessionMutex.Unlock()
	// Store new session in database (skip for temporary users)
	if os.db != nil && !isTemporaryUser(username) {
		// Get user ID
		var userID int
		err := os.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
		if err != nil {
			logger.Warn(logger.AreaAuth, "Could not find user ID for %s: %v", username, err)
		} else {
			_, err = os.db.Exec(`				INSERT INTO user_sessions (session_id, user_id, username, ip_address, created_at, last_activity)
				VALUES (?, ?, ?, ?, ?, ?)`,
				sessionID, userID, username, ip, session.CreatedAt.Unix(), session.LastActivity.Unix())
			if err != nil {
				logger.Warn(logger.AreaAuth, "Failed to save session to database: %v", err)
			}
		}
	} else if isTemporaryUser(username) {
		logger.Info(logger.AreaAuth, "Temporary user session %s created for user %s (not persisted in database)", sessionID, username)
	}

	// Initialize VFS for the user if needed
	if os.Vfs != nil {
		err := os.Vfs.InitializeUserVFS(username)
		if err != nil {
			logger.Warn(logger.AreaAuth, "Error initializing user VFS for %s: %v", username, err)
			// Not treating as critical error, session is still valid
		} else {
			logger.Info(logger.AreaAuth, "User VFS successfully initialized for session %s, user %s", sessionID, username)
		}
	}

	logger.Info(logger.AreaAuth, "User session %s created and stored for user %s from IP %s", sessionID, username, ip)
	return session, nil
}

// ResolvePath löst einen gegebenen Pfad auf und gibt den kanonischen Pfad zurück
func (os *TinyOS) ResolvePath(path string, sessionID string) (string, error) {
	log.Printf("[DEBUG-TINYOS] ResolvePath: Eingabe-Pfad=%s, SessionID=%s", path, sessionID)
	// Ermittle Username und CurrentPath für Session
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		username = "guest"
	}
	homeDir := "/home/" + username
	currentPath := homeDir
	if session, ok := os.sessions[sessionID]; ok && session.CurrentPath != "" {
		currentPath = session.CurrentPath
	}
	log.Printf("[DEBUG-TINYOS] ResolvePath: Username=%s, HomeDir=%s, CurrentPath=%s", username, homeDir, currentPath)
	// Wenn Pfad absolut ist, direkt verwenden, sonst relativ zu CurrentPath
	resolvedPath := path
	if !strings.HasPrefix(path, "/") {
		resolvedPath = currentPath + "/" + path
	}
	log.Printf("[DEBUG-TINYOS] ResolvePath: Aufgelöster Pfad=%s, SessionID=%s", resolvedPath, sessionID)
	return resolvedPath, nil
}

// ForceResetTelnetState forcefully resets telnet state for a session (for frontend reconnections)
func (os *TinyOS) ForceResetTelnetState(sessionID string) {
	logger.Info(logger.AreaTerminal, "Force resetting telnet state for session %s", sessionID)

	// CRITICAL FIX: Use timeout-protected lock to prevent deadlock (similar to exitTelnetSession)
	lockAcquired := make(chan bool, 1)
	var telnetState *TelnetState
	var exists bool

	go func() {
		os.telnetMutex.Lock()
		telnetState, exists = os.telnetStates[sessionID]
		lockAcquired <- true
	}()

	select {
	case <-lockAcquired:
		// Lock acquired successfully, proceed with cleanup
		defer os.telnetMutex.Unlock()
	case <-time.After(5 * time.Second):
		logger.Error(logger.AreaTerminal, "CRITICAL: Force reset mutex timeout for session %s - deadlock avoided", sessionID)
		return
	}

	if exists && telnetState != nil {
		logger.Info(logger.AreaTerminal, "Force resetting existing telnet state for session %s", sessionID)

		// Aggressively terminate the telnet session
		if telnetState.Connection != nil {
			telnetState.Connection.Close()
		}

		// Signal shutdown to any running goroutines
		if telnetState.ShutdownChan != nil {
			select {
			case telnetState.ShutdownChan <- struct{}{}:
				// Signal sent successfully
			default:
				// Channel might be full or closed, continue anyway
			}
		}
		// Close output channel
		if telnetState.OutputChan != nil {
			select {
			case <-telnetState.OutputChan:
				// Drain any pending messages
			default:
				// Channel is empty or closed
			}
			telnetState.safeCloseOutputChan()
		}

		// Force delete state
		delete(os.telnetStates, sessionID)

		logger.Info(logger.AreaTerminal, "Force reset completed for session %s, remaining sessions: %d",
			sessionID, len(os.telnetStates))
	} else {
		logger.Debug(logger.AreaTerminal, "No telnet state to force reset for session %s", sessionID)
	}
}

// Debug-Log für Terminal-Dimensionen hinzufügen
func (os *TinyOS) SetTerminalDimensions(cols, rows int) {
	os.sessionMutex.Lock()

	defer os.sessionMutex.Unlock()

	// Validierung
	if cols <= 0 {
		cols = 80 // Standard-Fallback
	}
	if rows <= 0 {
		rows = 24 // Standard-Fallback
	}

	if os.sessionTerminals == nil {
		os.sessionTerminals = make(map[string]*TerminalDimensions)
	}
	// Hier wird ein sessionID Parameter benötigt
	if os.sessionTerminals == nil {
		os.sessionTerminals = make(map[string]*TerminalDimensions)
	}

	// Speichere die Dimensionen ohne an eine bestimmte Session zu binden
	os.cols = cols
	os.rows = rows

	// Ausführliches Debug-Log hinzufügen
	log.Printf("[DEBUG-TINYOS] Terminal-Dimensionen gesetzt auf %dx%d (vorher: %dx%d)",
		cols, rows, os.cols, os.rows)

	// Werte setzen
	os.cols = cols
	os.rows = rows
}

// UpdateTerminalDimensions updates the terminal dimensions for a session
func (os *TinyOS) UpdateTerminalDimensions(sessionID string, cols, rows int) {
	if sessionID == "" {
		return
	}

	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()

	session, exists := os.sessions[sessionID]
	if exists {
		session.Terminal.Cols = cols
		session.Terminal.Rows = rows
		os.sessions[sessionID] = session
		logger.Debug(logger.AreaTerminal, "Updated terminal dimensions for session %s: %dx%d", sessionID, cols, rows)
	}
}

// GetTerminalDimensions returns the terminal dimensions for a session
func (os *TinyOS) GetTerminalDimensions(sessionID string) (int, int) {
	if sessionID == "" {
		return 80, 24 // Default dimensions
	}

	os.sessionMutex.RLock()
	defer os.sessionMutex.RUnlock()

	session, exists := os.sessions[sessionID]
	if exists {
		if session.Terminal.Cols > 0 && session.Terminal.Rows > 0 {
			return session.Terminal.Cols, session.Terminal.Rows
		}
	}

	return 80, 24 // Default dimensions
}

// wrapText umbricht Text basierend auf den Terminal-Dimensionen einer Session
func (os *TinyOS) wrapText(sessionID, text string) []string {
	if text == "" {
		return []string{""}
	}

	cols, _ := os.GetTerminalDimensions(sessionID)

	// Berücksichtige bereits vorhandene Zeilenumbrüche
	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		if len(line) <= cols {
			// Zeile passt, keine Umbruch nötig
			wrappedLines = append(wrappedLines, line)
		} else {
			// Zeile muss umgebrochen werden
			wrappedLines = append(wrappedLines, os.wrapLine(line, cols)...)
		}
	}

	return wrappedLines
}

// wrapLine umbricht eine einzelne Zeile an Wortgrenzen (TinyOS-Version)
func (os *TinyOS) wrapLine(line string, width int) []string {
	if len(line) <= width {
		return []string{line}
	}

	var result []string
	words := strings.Fields(line)

	if len(words) == 0 {
		// Leere Zeile oder nur Leerzeichen
		return []string{line}
	}

	currentLine := ""

	for _, word := range words {
		// Prüfe ob das Wort allein schon zu lang ist
		if len(word) > width {
			// Wort ist zu lang, muss hart umgebrochen werden
			if currentLine != "" {
				result = append(result, currentLine)
				currentLine = ""
			}

			// Hartes Umbrechen des zu langen Wortes
			for len(word) > width {
				result = append(result, word[:width])
				word = word[width:]
			}

			if len(word) > 0 {
				currentLine = word
			}
		} else {
			// Normales Wort
			testLine := currentLine
			if testLine != "" {
				testLine += " "
			}
			testLine += word

			if len(testLine) <= width {
				// Wort passt in die aktuelle Zeile
				currentLine = testLine
			} else {
				// Wort passt nicht, neue Zeile beginnen
				if currentLine != "" {
					result = append(result, currentLine)
				}
				currentLine = word
			}
		}
	}

	// Letzte Zeile hinzufügen falls vorhanden
	if currentLine != "" {
		result = append(result, currentLine)
	}

	return result
}

// CreateWrappedTextMessage erstellt eine Textnachricht mit automatischem Zeilenumbruch
func (os *TinyOS) CreateWrappedTextMessage(sessionID string, text string) []shared.Message {
	// Debug-Ausgabe für die Nachrichtenerzeugung

	// Wenn kein Text vorhanden ist, gib eine leere Nachricht zurück
	if text == "" {
		return []shared.Message{
			{
				Type:      shared.MessageTypeText,
				Content:   "",
				SessionID: sessionID,

				NoNewline: false,
			},
		}
	}

	// Terminal-Dimensionen holen (wenn verfügbar für die Session)
	width := 80 // Standard-Breite
	if os.cols > 0 {
		width = os.cols
	}

	// Debug-Ausgabe für die Terminalbreite

	// Text in Zeilen umbrechen und umformatieren
	var wrappedText string

	// Bei langen Zeilen Umbrüche einfügen
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if len(line) > width {
			// Zeilenumbruch hinzufügen
			currentLine := ""
			words := strings.Fields(line)

			for _, word := range words {
				// Wenn das Wort zu lang ist, muss es getrennt werden
				if len(word) > width {
					if currentLine != "" {
						wrappedText += currentLine + "\n"
						currentLine = ""

					}

					// Langes Wort in Teile zerlegen
					for len(word) > width {
						wrappedText += word[:width] + "\n"
						word = word[width:]
					}
					currentLine = word
				} else if len(currentLine)+len(word)+1 > width {
					// Wenn die Zeile zu lang wird, füge einen Umbruch ein
					wrappedText += currentLine + "\n"
					currentLine = word
				} else {
					// Füge das Wort zur aktuellen Zeile hinzu
					if currentLine == "" {
						currentLine = word
					} else {
						currentLine += " " + word
					}
				}
			}

			// Letzte Zeile hinzufügen
			if currentLine != "" {
				wrappedText += currentLine
			}
		} else {
			wrappedText += line
		}

		// Zeilenumbruch hinzufügen, außer bei der letzten Zeile
		if i < len(lines)-1 {
			wrappedText += "\n"
		}
	} // Debug-Ausgabe für den umgebrochenen Text

	// Slice mit einer Nachricht erstellen und zurückgeben
	return []shared.Message{
		{
			Type:      shared.MessageTypeText,
			Content:   wrappedText,
			SessionID: sessionID, // sessionID wird als Parameter übergeben
			NoNewline: false,
		},
	}
}

// ValidateSession prüft, ob eine Session-ID gültig und nicht abgelaufen ist
func (os *TinyOS) ValidateSession(sessionID string) (*Session, bool) {
	if sessionID == "" {
		return nil, false
	}

	os.sessionMutex.RLock()
	defer os.sessionMutex.RUnlock()

	session, exists := os.sessions[sessionID]
	if !exists {
		return nil, false
	}

	// Prüfe, ob die Session abgelaufen ist (24 Stunden für Gast-Sessions)
	if time.Since(session.LastActivity) > 24*time.Hour {
		// Session ist abgelaufen, entferne sie
		go func() {
			os.sessionMutex.Lock()
			delete(os.sessions, sessionID)
			os.sessionMutex.Unlock()
		}()
		return nil, false
	}

	// Aktualisiere LastActivity
	session.LastActivity = time.Now()

	return session, true
}

// ShowChatHistory zeigt die Chat-Historie für eine Session an
func (os *TinyOS) ShowChatHistory(sessionID string) []shared.Message {
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		return []shared.Message{{
			Content: "Error: You must be logged in to view chat history.",

			Type: shared.MessageTypeText,
		}}
	}

	os.sessionMutex.RLock()
	session, exists := os.sessions[sessionID]
	os.sessionMutex.RUnlock()

	if !exists {
		return []shared.Message{{
			Content: "Error: Session not found.",
			Type:    shared.MessageTypeText,
		}}
	}

	if len(session.ChatHistory) == 0 {
		return []shared.Message{{
			Content: "No chat history available.",
			Type:    shared.MessageTypeText,
		}}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Chat History (%d messages):\n\n", len(session.ChatHistory)))

	for i, msg := range session.ChatHistory {
		timeStr := msg.Time.Format("15:04:05")
		if msg.Role == "user" {
			result.WriteString(fmt.Sprintf("[%s] You: %s\n", timeStr, msg.Content))
		} else {
			result.WriteString(fmt.Sprintf("[%s] MCP: %s\n", timeStr, msg.Content))
		}
		if i < len(session.ChatHistory)-1 {
			result.WriteString("\n")
		}
	}

	return []shared.Message{{
		Content: result.String(),
		Type:    shared.MessageTypeText,
	}}
}

// ResetChessStateForSession resets the chess game state for a specific session
// This is called when a client reconnects to prevent chess remaining active after page reload
func (os *TinyOS) ResetChessStateForSession(sessionID string) {
	logger.Info(logger.AreaChess, "ResetChessStateForSession called for session: %s", sessionID)

	if sessionID == "" {
		logger.Debug(logger.AreaChess, "ResetChessStateForSession: Empty session ID provided")
		return
	}

	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()

	session, exists := os.sessions[sessionID]
	if !exists {
		logger.Warn(logger.AreaChess, "ResetChessStateForSession: Session %s not found in sessions map", sessionID)
		return
	}

	// Debug: Log current chess state before reset
	logger.Info(logger.AreaChess, "Session %s chess state before reset - ChessActive: %t, ChessGame: %v",
		sessionID, session.ChessActive, session.ChessGame != nil)

	// Reset chess state
	if session.ChessActive || session.ChessGame != nil {
		logger.Info(logger.AreaChess, "Resetting chess state for session %s (ChessActive: %t)", sessionID, session.ChessActive)
		session.ChessActive = false
		session.ChessGame = nil
		logger.Info(logger.AreaChess, "Chess state successfully reset for session %s", sessionID)
	} else {
		logger.Info(logger.AreaChess, "No chess state to reset for session %s", sessionID)
	}

	// Debug: Log chess state after reset
	logger.Info(logger.AreaChess, "Session %s chess state after reset - ChessActive: %t, ChessGame: %v",
		sessionID, session.ChessActive, session.ChessGame != nil)
}

// cleanupOrphanedTelnetSessions removes telnet sessions without active client sessions
func (os *TinyOS) cleanupOrphanedTelnetSessions() {
	now := time.Now()
	telnetTimeout := 10 * time.Minute // Telnet sessions timeout after 10 minutes of inactivity

	// Step 1: Create a snapshot of sessions to check with minimal mutex lock time
	var sessionsToCheck []struct {
		sessionID    string
		telnetState  *TelnetState
		lastActivity time.Time
	}

	os.telnetMutex.RLock()
	for sessionID, telnetState := range os.telnetStates {
		sessionsToCheck = append(sessionsToCheck, struct {
			sessionID    string
			telnetState  *TelnetState
			lastActivity time.Time
		}{
			sessionID:    sessionID,
			telnetState:  telnetState,
			lastActivity: telnetState.LastActivity,
		})
	}
	os.telnetMutex.RUnlock()

	// Step 2: Perform checks outside of mutex lock
	var sessionsToCleanup []string

	for _, session := range sessionsToCheck {
		shouldCleanup := false

		// Check if session is inactive for too long
		if now.Sub(session.lastActivity) > telnetTimeout {
			logger.Info(logger.AreaTerminal, "Marking inactive telnet session %s for cleanup (last activity: %v ago)",
				session.sessionID, now.Sub(session.lastActivity))
			shouldCleanup = true
		} else {
			// Check if the corresponding user session still exists
			os.sessionMutex.RLock()
			_, sessionExists := os.sessions[session.sessionID]
			os.sessionMutex.RUnlock()

			if !sessionExists {
				logger.Info(logger.AreaTerminal, "Marking orphaned telnet session %s for cleanup (no corresponding user session)", session.sessionID)
				shouldCleanup = true
			}
		}

		if shouldCleanup {
			sessionsToCleanup = append(sessionsToCleanup, session.sessionID)

			// Close connection outside of telnet mutex to avoid blocking
			if session.telnetState.Connection != nil {
				session.telnetState.Connection.Close()
			}
		}
	}

	// Step 3: Cleanup sessions with minimal mutex lock time
	if len(sessionsToCleanup) > 0 {
		os.telnetMutex.Lock()
		for _, sessionID := range sessionsToCleanup {
			delete(os.telnetStates, sessionID)
			logger.Info(logger.AreaTerminal, "Cleaned up telnet session %s", sessionID)
		}
		os.telnetMutex.Unlock()
	}
}

// CleanupGhostTelnetSessions removes telnet sessions that are not actually in telnet mode
func (os *TinyOS) CleanupGhostTelnetSessions() {
	os.telnetMutex.Lock()
	defer os.telnetMutex.Unlock()

	// Get list of sessions that have telnet states but are not actually connected
	var ghostSessions []string
	for sessionID, telnetState := range os.telnetStates {
		// Check if the connection is nil or closed
		if telnetState.Connection == nil {
			ghostSessions = append(ghostSessions, sessionID)
		}
	}
	// Clean up ghost sessions
	for _, sessionID := range ghostSessions {
		logger.Info(logger.AreaTerminal, "Removing ghost telnet session: %s", sessionID)
		telnetState := os.telnetStates[sessionID]

		// Close connection if exists
		if telnetState.Connection != nil {
			telnetState.Connection.Close()
		}
		// Close output channel
		if telnetState.OutputChan != nil {
			telnetState.safeCloseOutputChan()
		}

		// Remove from map
		delete(os.telnetStates, sessionID)
		logger.Info(logger.AreaTerminal, "Ghost telnet session cleanup completed for session %s", sessionID)
	}
}

// cleanupTelnetSessionSync performs synchronous telnet session cleanup
func (os *TinyOS) CleanupTelnetSessionSync(sessionID string) {
	os.telnetMutex.Lock()

	telnetState, exists := os.telnetStates[sessionID]
	if !exists {
		os.telnetMutex.Unlock()
		logger.Info(logger.AreaTerminal, "No telnet session found to cleanup for session %s", sessionID)
		return
	}

	logger.Info(logger.AreaTerminal, "Synchronously cleaning up telnet session for session %s", sessionID)

	// Signal shutdown to the session goroutine
	if telnetState.ShutdownChan != nil {
		close(telnetState.ShutdownChan)
	}

	// Close the connection
	if telnetState.Connection != nil {
		telnetState.Connection.Close()
	}
	// Close the output channel
	if telnetState.OutputChan != nil {
		telnetState.safeCloseOutputChan()
	}

	delete(os.telnetStates, sessionID)

	// CRITICAL FIX: Release mutex BEFORE sending callback to prevent deadlock
	os.telnetMutex.Unlock()

	// Send end message to frontend AFTER releasing mutex
	if os.SendToClientCallback != nil {
		endMessage := shared.Message{
			Type:      shared.MessageTypeTelnet,
			Content:   "end",
			SessionID: sessionID,
		}
		err := os.SendToClientCallback(sessionID, endMessage)
		if err != nil {
			logger.Warn(logger.AreaTerminal, "Failed to send telnet end message during cleanup: %v", err)
		} else {
			logger.Info(logger.AreaTerminal, "Telnet end message sent during cleanup for session %s", sessionID)
		}
	}

	logger.Info(logger.AreaTerminal, "Synchronous telnet session cleanup completed for session %s", sessionID)
}

// processTelnetOutputs continuously monitors all active telnet sessions
// for output and forwards it to the appropriate clients
// This function is DISABLED - use direct WebSocket callback instead
func (os *TinyOS) processTelnetOutputs() {
	// Check for shutdown signal immediately
	select {
	case <-os.telnetOutputShutdown:
		logger.Info(logger.AreaTerminal, "Telnet output processor shutdown requested")
		return
	default:
		// Continue with processing
	}

	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-os.telnetOutputShutdown:
			logger.Info(logger.AreaTerminal, "Telnet output processor shutting down")
			return
		case <-ticker.C:
			// DISABLED: This processor is no longer used
			// We use direct WebSocket callback for telnet output instead
			return
		}
	}
}

// ShutdownTelnetOutputProcessor immediately stops the telnet output processor
func (os *TinyOS) ShutdownTelnetOutputProcessor() {
	select {
	case os.telnetOutputShutdown <- true:
		logger.Info(logger.AreaTerminal, "Telnet output processor shutdown signal sent")
	default:
		logger.Warn(logger.AreaTerminal, "Telnet output processor shutdown signal already sent or channel full")
	}
}

// IsTelnetSessionActive checks if a session is currently running a telnet connection
// This function uses a timeout to prevent deadlocks in WebSocket processing
func (os *TinyOS) IsTelnetSessionActive(sessionID string) bool {
	// Use a channel to implement timeout for mutex acquisition
	result := make(chan bool, 1)

	go func() {
		// Use RLock for read-only operations
		os.telnetMutex.RLock()
		defer os.telnetMutex.RUnlock()
		state, exists := os.telnetStates[sessionID]

		if exists && state != nil {
			// Check if connection is actually alive - if not, mark for cleanup but don't block here
			if state.Connection == nil {
				logger.Warn(logger.AreaTerminal, "Found dead telnet session %s (no connection) in frontend check, will be cleaned up", sessionID)
				// Schedule cleanup in background to avoid deadlock
				go func() {
					os.CleanupTelnetSessionSync(sessionID)
				}()
				result <- false
				return
			}
		}

		// Return true only if session exists AND has active connection (consistent with isInTelnetProcess)
		result <- (exists && state != nil && state.Connection != nil)
	}()
	// Wait for result with timeout to prevent blocking WebSocket processing
	select {
	case res := <-result:
		return res
	case <-time.After(15 * time.Second): // Increased timeout to account for cleanup operations
		logger.Warn(logger.AreaTerminal, "DEADLOCK PREVENTION: IsTelnetSessionActive timeout for session %s, assuming false", sessionID)
		return false
	}
}

// CleanupSessionResources is called when a client disconnects to clean up all associated resources.
// This prevents "stuck" states (e.g., in editor, telnet) when the user reloads the page.
func (os *TinyOS) CleanupSessionResources(sessionID string) {
	if sessionID == "" {
		return
	}

	// CRITICAL: Reset the input mode to default immediately.
	// This is the most important step to prevent race conditions on reconnect.
	// The new connection will see the correct OS_SHELL mode even if other cleanup
	// routines are still running.
	os.SetInputMode(sessionID, InputModeOSShell)

	logger.Info(logger.AreaSession, "Cleaning up resources for disconnected session: %s", sessionID)

	// 1. Clean up active editor session
	editorManager := editor.GetEditorManager()
	if editorManager.GetEditor(sessionID) != nil {
		logger.Info(logger.AreaEditor, "Closing active editor for session: %s", sessionID)
		editorManager.CloseEditor(sessionID)
	}

	// 2. Clean up active telnet session
	os.CleanupTelnetSessionSync(sessionID)

	// 3. Clean up active cat pager state
	os.catPagerMutex.Lock()
	if _, exists := os.catPagerStates[sessionID]; exists {
		delete(os.catPagerStates, sessionID)
		logger.Info(logger.AreaTerminal, "Cleaned up cat pager state for session: %s", sessionID)
	}
	os.catPagerMutex.Unlock()

	// 4. Clean up any pending registration, login, or password change processes
	os.registrationMutex.Lock()
	delete(os.registrationStates, sessionID)
	os.registrationMutex.Unlock()

	os.loginMutex.Lock()
	delete(os.loginStates, sessionID)
	os.loginMutex.Unlock()

	os.passwordChangeMutex.Lock()
	delete(os.passwordChangeStates, sessionID)
	os.passwordChangeMutex.Unlock()

	// 5. Clean up active chess game state
	os.sessionMutex.Lock()
	if session, exists := os.sessions[sessionID]; exists {
		if session.ChessActive || session.ChessGame != nil {
			logger.Info(logger.AreaChess, "Resetting chess state for disconnected session: %s", sessionID)
			session.ChessActive = false
			session.ChessGame = nil
		}
	}
	os.sessionMutex.Unlock()
}

// SetInputMode atomically sets the input mode for a given session.
// This is the central function for managing the user's state.
func (os *TinyOS) SetInputMode(sessionID string, mode InputMode) {
	if sessionID == "" {
		return
	}

	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()

	if session, exists := os.sessions[sessionID]; exists {
		logger.Info(logger.AreaSession, "Setting input mode for session %s to %d", sessionID, mode)
		session.InputMode = mode
	} else {
		logger.Warn(logger.AreaSession, "Attempted to set input mode for non-existent session %s", sessionID)
	}
}

// GetInputMode atomically gets the input mode for a given session.
func (os *TinyOS) GetInputMode(sessionID string) InputMode {
	if sessionID == "" {
		return InputModeOSShell
	}

	os.sessionMutex.RLock()
	defer os.sessionMutex.RUnlock()

	if session, exists := os.sessions[sessionID]; exists {
		return session.InputMode
	}

	return InputModeOSShell
}

// GetActiveTelnetSessionCount returns the number of active telnet sessions
func (os *TinyOS) GetActiveTelnetSessionCount() int {
	os.telnetMutex.RLock()
	defer os.telnetMutex.RUnlock()
	return len(os.telnetStates)
}

// GetActiveTelnetSessionIDs returns a list of active telnet session IDs
func (os *TinyOS) GetActiveTelnetSessionIDs() []string {
	os.telnetMutex.RLock()
	defer os.telnetMutex.RUnlock()

	sessionIDs := make([]string, 0, len(os.telnetStates))
	for sessionID := range os.telnetStates {
		sessionIDs = append(sessionIDs, sessionID)
	}
	return sessionIDs
}

// safeSendToOutputChan safely sends a message to the telnet output channel
// Returns true if the message was sent successfully, false if channel is closed
func (ts *TelnetState) safeSendToOutputChan(message shared.Message, timeout time.Duration) bool {
	ts.channelMutex.RLock()
	defer ts.channelMutex.RUnlock()

	if ts.channelClosed {
		return false // Channel is already closed
	}

	select {
	case ts.OutputChan <- message:
		return true
	case <-time.After(timeout):
		return false // Timeout
	}
}

// safeCloseOutputChan safely closes the telnet output channel
func (ts *TelnetState) safeCloseOutputChan() {
	ts.channelMutex.Lock()
	defer ts.channelMutex.Unlock()

	if !ts.channelClosed {
		close(ts.OutputChan)
		ts.channelClosed = true
	}
}
