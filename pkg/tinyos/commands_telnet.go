package tinyos

import (
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// cmdTelnet handles telnet connections to predefined servers
func (os *TinyOS) cmdTelnet(args []string) []shared.Message {
	logger.Info(logger.AreaTerminal, "cmdTelnet called with args: %v", args)

	if len(args) == 0 {
		logger.Error(logger.AreaTerminal, "cmdTelnet: no session ID provided")
		return os.CreateWrappedTextMessage("", "telnet: session ID missing")
	}

	sessionID := args[0]
	logger.Info(logger.AreaTerminal, "cmdTelnet processing for session: %s", sessionID)

	// DIAGNOSTIC: Check session state before telnet attempt
	os.sessionMutex.RLock()
	session, sessionExists := os.sessions[sessionID]
	os.sessionMutex.RUnlock()

	logger.Info(logger.AreaTerminal, "TELNET_DIAGNOSTIC: Session exists=%t for sessionID=%s", sessionExists, sessionID)
	if sessionExists && session != nil {
		logger.Info(logger.AreaTerminal, "TELNET_DIAGNOSTIC: Session state - ChessActive=%t, LastActivity=%v",
			session.ChessActive, session.LastActivity)
	}

	// Show help if no server specified
	if len(args) < 2 {
		logger.Info(logger.AreaTerminal, "cmdTelnet: no server specified for session %s", sessionID)
		return os.CreateWrappedTextMessage(sessionID, "telnet: server name required\nUsage: telnet <servername> or telnet list")
	}

	serverArg := strings.ToLower(args[1])
	logger.Info(logger.AreaTerminal, "cmdTelnet: attempting to connect to server '%s' for session %s", serverArg, sessionID)
	// Handle special commands
	if serverArg == "list" {
		logger.Info(logger.AreaTerminal, "cmdTelnet: listing servers for session %s", sessionID)
		return os.getTelnetServerList(sessionID)
	} // Check if user is already in a telnet session
	logger.Info(logger.AreaTerminal, "TELNET_DIAGNOSTIC: Checking if session %s already in telnet process", sessionID)
	if os.isInTelnetProcess(sessionID) {
		logger.Warn(logger.AreaTerminal, "cmdTelnet: session %s already in telnet session", sessionID)
		return os.CreateWrappedTextMessage(sessionID, "Already in telnet session. Use Ctrl+X or ESC to exit first.")
	}
	logger.Info(logger.AreaTerminal, "TELNET_DIAGNOSTIC: Session %s not in telnet process, proceeding", sessionID)

	// Check connection limits to prevent resource exhaustion
	os.telnetMutex.RLock()
	totalConnections := len(os.telnetStates)
	logger.Info(logger.AreaTerminal, "TELNET_DIAGNOSTIC: Current telnet connections: %d/%d", totalConnections, 10)
	os.telnetMutex.RUnlock()

	const MAX_TELNET_CONNECTIONS = 10 // Limit concurrent telnet connections
	if totalConnections >= MAX_TELNET_CONNECTIONS {
		logger.Warn(logger.AreaTerminal, "cmdTelnet: connection limit reached (%d/%d)", totalConnections, MAX_TELNET_CONNECTIONS)
		return os.CreateWrappedTextMessage(sessionID, "Telnet connection limit reached. Please try again later.")
	}

	// Get server configuration
	logger.Info(logger.AreaTerminal, "cmdTelnet: getting server config for '%s'", serverArg)
	serverConfig, err := os.getTelnetServerConfig(serverArg)
	if err != nil {
		logger.Error(logger.AreaTerminal, "cmdTelnet: server config error for '%s': %v", serverArg, err)
		return os.CreateWrappedTextMessage(sessionID, fmt.Sprintf("telnet: %v", err))
	}
	logger.Info(logger.AreaTerminal, "cmdTelnet: server config found - %s at %s", serverConfig.DisplayName, serverConfig.Host)
	// Attempt to connect
	logger.Info(logger.AreaTerminal, "Attempting telnet connection to %s for session %s", serverConfig.Host, sessionID)
	conn, err := net.DialTimeout("tcp", serverConfig.Host, 10*time.Second)
	if err != nil {
		logger.Error(logger.AreaTerminal, "Telnet connection failed to %s: %v", serverConfig.Host, err)
		logger.Error(logger.AreaTerminal, "TELNET_DIAGNOSTIC: Connection failure for session %s to %s", sessionID, serverConfig.Host)
		return os.CreateWrappedTextMessage(sessionID, fmt.Sprintf("telnet: connection failed to %s", serverConfig.DisplayName))
	}
	logger.Info(logger.AreaTerminal, "Telnet connection established to %s for session %s", serverConfig.Host, sessionID)
	logger.Info(logger.AreaTerminal, "TELNET_DIAGNOSTIC: Connection successful for session %s", sessionID) // Create telnet state with much larger buffer to prevent blocking and shutdown channel
	outputChan := make(chan shared.Message, 5000)                                                          // Increased from 1000 to prevent blocking
	shutdownChan := make(chan struct{})
	telnetState := &TelnetState{
		ServerName:   serverConfig.DisplayName,
		ServerHost:   serverConfig.Host,
		Connection:   conn,
		SessionID:    sessionID,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		OutputChan:   outputChan,
		ShutdownChan: shutdownChan,
		ServerEcho:   false, // Will be determined during negotiation
		LocalEcho:    true,  // Default to local echo until server takes over
	}
	// Store telnet state
	os.telnetMutex.Lock()
	os.telnetStates[sessionID] = telnetState
	logger.Info(logger.AreaTerminal, "Telnet state stored for session %s, total sessions: %d", sessionID, len(os.telnetStates))
	logger.Info(logger.AreaTerminal, "TELNET_DIAGNOSTIC: Telnet state successfully stored for session %s", sessionID)
	os.telnetMutex.Unlock() // Send initial telnet negotiation - corrected for mapscii compatibility
	logger.Info(logger.AreaTerminal, "Sending corrected telnet negotiations to %s for session %s", serverConfig.Host, sessionID)
	// Standard telnet client behavior - send initial capabilities
	// IAC WILL TERMINAL_TYPE - we can provide terminal type
	conn.Write([]byte{255, 251, 24}) // IAC WILL TERMINAL_TYPE
	logger.Info(logger.AreaTerminal, "Sent WILL TERMINAL_TYPE to %s", serverConfig.Host)

	// IAC WILL NAWS - we can provide window size
	conn.Write([]byte{255, 251, 31})                                            // IAC WILL NAWS
	logger.Info(logger.AreaTerminal, "Sent WILL NAWS to %s", serverConfig.Host) // IAC WILL SUPPRESS_GO_AHEAD - we support suppressing go-ahead
	conn.Write([]byte{255, 251, 3})                                             // IAC WILL SUPPRESS_GO_AHEAD
	logger.Info(logger.AreaTerminal, "Sent WILL SUPPRESS_GO_AHEAD to %s", serverConfig.Host)

	// IAC WONT ECHO - we don't want to handle local echo (server should echo)
	// This is important for proper echo handling with telnet servers
	conn.Write([]byte{255, 252, 1}) // IAC WONT ECHO
	logger.Info(logger.AreaTerminal, "Sent WONT ECHO to %s (server should handle echo)", serverConfig.Host)

	// Get actual client terminal dimensions instead of hardcoded values
	clientCols, clientRows := os.GetTerminalDimensions(sessionID)

	// IAC SB NAWS width1 width2 height1 height2 IAC SE
	windowSizeCmd := []byte{
		255, 250, 31, // IAC SB NAWS
		byte(clientCols >> 8), byte(clientCols & 0xFF), // columns as 2 bytes
		byte(clientRows >> 8), byte(clientRows & 0xFF), // rows as 2 bytes
		255, 240, // IAC SE
	}
	conn.Write(windowSizeCmd)
	logger.Info(logger.AreaTerminal, "Sent window size (%dx%d) to %s", clientCols, clientRows, serverConfig.Host)
	logger.Info(logger.AreaTerminal, "Initial telnet negotiations completed for %s", serverConfig.Host)

	logger.Info(logger.AreaTerminal, "Telnet session started for %s to %s", sessionID, serverConfig.DisplayName)

	// Start telnet session in background
	go os.handleTelnetSession(telnetState)

	// Send an initial "activation" input after a short delay to prevent timeout
	// Many interactive telnet services like mapscii expect user activity to stay connected
	go func() {
		time.Sleep(1 * time.Second) // Wait for negotiation to complete
		logger.Info(logger.AreaTerminal, "Sending initial activation input to keep %s connection alive", serverConfig.Host)
		// Send a simple newline to show activity without affecting the display
		conn.Write([]byte{13}) // Carriage return
	}()

	// Send initial messages
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Connecting to %s...", serverConfig.DisplayName)},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Connected to %s", serverConfig.DisplayName)},
		{Type: shared.MessageTypeText, Content: "Press Ctrl+X or ESC to exit telnet session"},
		{Type: shared.MessageTypeText, Content: "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"},
		{Type: shared.MessageTypeTelnet, Content: "start", SessionID: sessionID, Params: map[string]interface{}{
			"serverName": serverConfig.DisplayName,
		}},
	}
}

// getTelnetServerConfig retrieves server configuration from settings
func (os *TinyOS) getTelnetServerConfig(serverName string) (*TelnetServerConfig, error) {
	// Get server configuration from settings.cfg [Telnet] section
	configValue := configuration.GetString("Telnet", serverName, "")
	if configValue == "" {
		return nil, fmt.Errorf("server '%s' not found. Use 'telnet list' to see available servers", serverName)
	}

	// Parse format: "Display Name|host:port"
	parts := strings.Split(configValue, "|")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid server configuration for '%s'", serverName)
	}

	return &TelnetServerConfig{
		DisplayName: parts[0],
		Host:        parts[1],
	}, nil
}

// isInTelnetProcess checks if a session is currently in a telnet process
func (os *TinyOS) isInTelnetProcess(sessionID string) bool {
	os.telnetMutex.RLock()
	defer os.telnetMutex.RUnlock()
	state, exists := os.telnetStates[sessionID]

	// ENHANCED DEBUG: Show all active telnet states for debugging
	logger.Debug(logger.AreaTerminal, "TELNET STATE CHECK for session %s: exists=%t, total_states=%d",
		sessionID, exists, len(os.telnetStates))

	if exists && state != nil {
		logger.Debug(logger.AreaTerminal, "TELNET STATE DETAILS for session %s: server=%s, host=%s, connection_alive=%t, created_at=%v",
			sessionID, state.ServerName, state.ServerHost, state.Connection != nil, state.CreatedAt)

		// Check if connection is actually alive - if not, mark for cleanup but don't block here
		if state.Connection == nil {
			logger.Warn(logger.AreaTerminal, "Found dead telnet session %s (no connection), will be cleaned up", sessionID)
			// Schedule cleanup in background to avoid deadlock
			go func() {
				os.CleanupTelnetSession(sessionID)
			}()
			return false
		}
	}

	// List all active telnet sessions for comprehensive debugging
	if len(os.telnetStates) > 0 {
		logger.Debug(logger.AreaTerminal, "ALL ACTIVE TELNET SESSIONS:")
		for sid, st := range os.telnetStates {
			logger.Debug(logger.AreaTerminal, "  Session %s: server=%s, alive=%t",
				sid, st.ServerName, st.Connection != nil)
		}
	}

	return exists && state != nil && state.Connection != nil
}

// getTelnetServerList returns a list of available telnet servers
func (os *TinyOS) getTelnetServerList(sessionID string) []shared.Message {
	var content strings.Builder
	content.WriteString("Available Telnet Servers:\n")
	content.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	// Get all keys from [Telnet] section
	servers := configuration.GetSection("Telnet")
	if len(servers) == 0 {
		content.WriteString("No telnet servers configured.\n")
	} else {
		for serverName, configValue := range servers {
			// Parse format: "Display Name|host:port"
			parts := strings.Split(configValue, "|")
			if len(parts) >= 2 {
				content.WriteString(fmt.Sprintf("%-15s - %s\n", serverName, parts[0]))
			}
		}
	}

	content.WriteString("\nUsage: telnet <servername>")
	return os.CreateWrappedTextMessage(sessionID, content.String())
}

// TelnetServerConfig holds configuration for a telnet server
type TelnetServerConfig struct {
	DisplayName string
	Host        string
}

// handleTelnetSession manages the telnet connection in a goroutine
func (os *TinyOS) handleTelnetSession(telnetState *TelnetState) {
	defer func() {
		// Recovery from panics to prevent backend crashes
		if r := recover(); r != nil {
			logger.Error(logger.AreaTerminal, "Telnet session panic recovered for %s: %v", telnetState.SessionID, r)
		}

		// Send "end" message to frontend before cleanup
		endMessage := shared.Message{
			Type:      shared.MessageTypeTelnet,
			Content:   "end",
			SessionID: telnetState.SessionID,
		}
		// Try to send via callback first (direct WebSocket route) - ASYNC to prevent deadlock
		endSentViaCallback := false
		if os.SendToClientCallback != nil {
			// CRITICAL FIX: Make callback asynchronous to prevent deadlock
			go func() {
				err := os.SendToClientCallback(telnetState.SessionID, endMessage)
				if err != nil {
					logger.Warn(logger.AreaTerminal, "Failed to send telnet end via callback: %v", err)
				} else {
					logger.Info(logger.AreaTerminal, "Telnet end message sent via callback for session %s", telnetState.SessionID)
				}
			}()
			endSentViaCallback = true // Assume it will succeed
		}
		// Also try via output channel as fallback (non-blocking)
		if !telnetState.safeSendToOutputChan(endMessage, 200*time.Millisecond) {
			if !endSentViaCallback {
				logger.Error(logger.AreaTerminal, "CRITICAL: Failed to send telnet end message via both callback and channel for session %s - channel closed or timeout", telnetState.SessionID)
			} else {
				logger.Debug(logger.AreaTerminal, "Output channel closed or timeout for telnet end message, but callback succeeded for session %s", telnetState.SessionID)
			}
		} else {
			logger.Info(logger.AreaTerminal, "Telnet end message sent via output channel for session %s", telnetState.SessionID)
		}
		// Clean up on exit
		if telnetState.Connection != nil {
			telnetState.Connection.Close()
		}

		// CRITICAL FIX: Use async cleanup to prevent deadlock with SendToClientCallback
		go func() {
			// Small delay to ensure any ongoing callbacks complete
			time.Sleep(50 * time.Millisecond)

			os.telnetMutex.Lock()
			delete(os.telnetStates, telnetState.SessionID)
			remainingSessions := len(os.telnetStates)
			os.telnetMutex.Unlock()

			logger.Info(logger.AreaTerminal, "Telnet session cleaned up for %s, remaining sessions: %d", telnetState.SessionID, remainingSessions)
		}()
	}()

	// Create channels for coordinated shutdown
	shutdownSignal := make(chan struct{})
	healthCheck := make(chan bool, 1)

	// Start health monitoring goroutine
	go os.monitorTelnetHealth(telnetState, shutdownSignal, healthCheck)
	// Monitor for shutdown signals in a separate goroutine
	go func() {
		select {
		case <-telnetState.ShutdownChan:
			logger.Info(logger.AreaTerminal, "Shutdown signal received for telnet session %s", telnetState.SessionID)
		case <-time.After(60 * time.Minute): // Increased maximum session duration for stability
			logger.Warn(logger.AreaTerminal, "Telnet session %s exceeded maximum duration, forcing shutdown", telnetState.SessionID)
		case healthy := <-healthCheck:
			if !healthy {
				logger.Warn(logger.AreaTerminal, "Telnet health check failed for session %s, forcing shutdown", telnetState.SessionID)
				// Health check failed - session will be terminated by closing shutdownSignal
			}
		}
		close(shutdownSignal)
	}()

	// Main read loop with improved error handling
	buffer := make([]byte, 4096)
	consecutiveErrors := 0
	maxConsecutiveErrors := 3

	for {
		// Check for shutdown signal first (non-blocking)
		select {
		case <-shutdownSignal:
			logger.Info(logger.AreaTerminal, "Exiting telnet read loop due to shutdown signal for session %s", telnetState.SessionID)
			return
		default:
			// Continue with normal operation
		}
		// Set more generous read timeout to prevent premature disconnections
		telnetState.Connection.SetReadDeadline(time.Now().Add(120 * time.Second)) // Increased from 30s
		n, err := telnetState.Connection.Read(buffer)
		if err != nil {
			consecutiveErrors++
			logger.Debug(logger.AreaTerminal, "Telnet read error %d/%d for session %s: %v", consecutiveErrors, maxConsecutiveErrors, telnetState.SessionID, err)

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// CRITICAL FIX: Handle excessive timeouts properly
				if consecutiveErrors >= maxConsecutiveErrors {
					logger.Warn(logger.AreaTerminal, "Too many consecutive timeouts (%d) for telnet session %s, forcing cleanup", consecutiveErrors, telnetState.SessionID) // Force cleanup of stuck session
					go func() {
						time.Sleep(100 * time.Millisecond)
						os.CleanupTelnetSession(telnetState.SessionID)
					}()
					return
				}

				// Check for extended inactivity (10 minutes of no activity to match config)
				if time.Since(telnetState.LastActivity) > 10*time.Minute {
					logger.Warn(logger.AreaTerminal, "Telnet session %s timed out due to inactivity, forcing cleanup", telnetState.SessionID)
					go func() {
						time.Sleep(50 * time.Millisecond)
						os.CleanupTelnetSession(telnetState.SessionID)
					}()
					return
				}

				// Reset consecutive errors for timeouts, they might be temporary
				if consecutiveErrors > 1 {
					consecutiveErrors = 1
				}
				continue
			}

			// For other errors, check if we should abort
			if consecutiveErrors >= maxConsecutiveErrors {
				logger.Error(logger.AreaTerminal, "Telnet session %s aborted after %d consecutive errors, forcing cleanup", telnetState.SessionID, consecutiveErrors)
				go func() {
					time.Sleep(50 * time.Millisecond)
					os.CleanupTelnetSession(telnetState.SessionID)
				}()
				return
			}

			// Check if it's EOF (normal connection close)
			if err == io.EOF {
				logger.Info(logger.AreaTerminal, "Telnet connection closed normally by server for session %s, forcing cleanup", telnetState.SessionID)
				go func() {
					time.Sleep(50 * time.Millisecond)
					os.CleanupTelnetSession(telnetState.SessionID)
				}()
				return
			}

			// For other errors, continue but don't process data
			continue
		}

		// Reset error counter on successful read
		consecutiveErrors = 0
		if n > 0 {
			telnetState.LastActivity = time.Now()
			logger.Debug(logger.AreaTerminal, "Telnet received %d bytes from %s for session %s", n, telnetState.ServerHost, telnetState.SessionID)

			// Process and filter telnet protocol bytes before sending
			filteredData := os.filterTelnetProtocolWithNegotiation(buffer[:n], telnetState)

			if len(filteredData) > 0 {
				// Send data with non-blocking approach
				message := shared.Message{
					Type:      shared.MessageTypeTelnet,
					Content:   string(filteredData),
					SessionID: telnetState.SessionID,
				}
				logger.Debug(logger.AreaTerminal, "Telnet sending %d bytes of content for session %s", len(filteredData), telnetState.SessionID) // Try to send via callback first (direct WebSocket route) - ASYNC to prevent deadlock
				dataSentViaCallback := false
				if os.SendToClientCallback != nil {
					// CRITICAL FIX: Make callback asynchronous to prevent deadlock
					go func(msg shared.Message) {
						err := os.SendToClientCallback(telnetState.SessionID, msg)
						if err != nil {
							logger.Debug(logger.AreaTerminal, "Failed to send telnet data via callback, client may be disconnected: %v", err)
							// If callback fails consistently, the client is likely disconnected
							// Note: Cannot terminate session from here as it's async now
						}
					}(message)
					dataSentViaCallback = true // Assume it will succeed
				} // Also try via output channel as fallback (non-blocking with reasonable timeout)
				if !telnetState.safeSendToOutputChan(message, 100*time.Millisecond) {
					if !dataSentViaCallback {
						logger.Warn(logger.AreaTerminal, "Failed to send telnet data via both callback and channel for session %s - dropping data to prevent blocking", telnetState.SessionID)
						// Drop data rather than block the entire session
					}
				} else {
					logger.Debug(logger.AreaTerminal, "Telnet output queued for session %s: %d bytes", telnetState.SessionID, len(filteredData))
				}
			} else {
				// Debug: Log when we receive data but have no filtered output
				logger.Debug(logger.AreaTerminal, "Telnet received %d bytes but filtered data is empty for session %s", n, telnetState.SessionID)
			}
		}
	}
}

// monitorTelnetHealth performs periodic health checks on telnet connections
func (os *TinyOS) monitorTelnetHealth(telnetState *TelnetState, shutdown <-chan struct{}, result chan<- bool) {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-shutdown:
			return
		case <-ticker.C:
			// Perform connection health check
			if !os.isConnectionHealthy(telnetState) {
				select {
				case result <- false:
				case <-time.After(1 * time.Second):
					// If we can't send the result, the session is probably already shutting down
				}
				return
			}
		}
	}
}

// isConnectionHealthy checks if a telnet connection is still healthy
func (os *TinyOS) isConnectionHealthy(telnetState *TelnetState) bool {
	if telnetState.Connection == nil {
		return false
	}

	// Instead of sending NOP commands that create empty frontend messages,
	// check connection health by verifying the connection state
	// and checking for excessive inactivity

	// Check for excessive inactivity first
	if time.Since(telnetState.LastActivity) > 10*time.Minute {
		logger.Debug(logger.AreaTerminal, "Telnet session %s inactive for too long", telnetState.SessionID)
		return false
	}

	// Check if the connection is still alive using TCP keepalive check
	// This avoids sending data that would appear as empty messages in frontend
	conn := telnetState.Connection
	if netConn, ok := conn.(*net.TCPConn); ok {
		// Try to set a very short read timeout to test connection
		originalDeadline := time.Time{}
		netConn.SetReadDeadline(time.Now().Add(1 * time.Millisecond))

		// Try a non-blocking read to detect connection state
		buffer := make([]byte, 1)
		_, err := netConn.Read(buffer)

		// Reset the deadline
		netConn.SetReadDeadline(originalDeadline)

		// If we get a timeout, that's actually good - connection is alive but no data
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			logger.Debug(logger.AreaTerminal, "Telnet health check passed for session %s (connection alive)", telnetState.SessionID)
			return true
		}

		// If we get other errors, connection might be dead
		if err != nil && err != io.EOF {
			logger.Debug(logger.AreaTerminal, "Telnet health check failed for session %s: %v", telnetState.SessionID, err)
			return false
		}
	}

	// If we can't do TCP-specific checks, assume connection is healthy
	// if we haven't exceeded the inactivity timeout
	logger.Debug(logger.AreaTerminal, "Telnet health check passed for session %s (default)", telnetState.SessionID)
	return true
}

// filterTelnetProtocol removes telnet protocol bytes from data and handles negotiations
func filterTelnetProtocol(data []byte) []byte {
	result := make([]byte, 0, len(data))
	i := 0

	for i < len(data) {
		if data[i] == 255 { // IAC (Interpret As Command)
			// Handle IAC sequences
			if i+1 < len(data) {
				command := data[i+1]
				if command >= 251 && command <= 254 { // WILL, WONT, DO, DONT
					if i+2 < len(data) {
						option := data[i+2]
						logger.Info(logger.AreaTerminal, "Received IAC %d option %d", command, option)

						// Handle specific telnet options for mapscii compatibility
						switch option {
						case 31: // NAWS (Negotiate About Window Size)
							if command == 253 { // DO NAWS
								logger.Info(logger.AreaTerminal, "Server requests NAWS - already negotiated")
							}
						case 24: // TERMINAL_TYPE
							if command == 253 { // DO TERMINAL_TYPE
								logger.Info(logger.AreaTerminal, "Server requests terminal type - already negotiated")
							}
						case 1: // ECHO
							if command == 251 { // WILL ECHO
								logger.Info(logger.AreaTerminal, "Server will echo")
							}
						}

						// 3-byte sequence: IAC + command + option
						i += 3
						continue
					}
				} else if command == 250 { // SB (Subnegotiation Begin)
					// Handle subnegotiation sequences
					logger.Info(logger.AreaTerminal, "Received subnegotiation")
					// Skip until SE (240)
					for j := i + 2; j < len(data); j++ {
						if data[j] == 240 { // SE (Subnegotiation End)
							i = j + 1
							break
						}
					}
					continue
				}
				// Other 2-byte sequences
				i += 2
				continue
			}
			i++ // Just skip IAC if at end
			continue
		}

		// Keep regular data
		result = append(result, data[i])
		i++
	}

	return result
}

// filterTelnetProtocolWithNegotiation filters telnet protocol bytes and handles negotiations
func (os *TinyOS) filterTelnetProtocolWithNegotiation(data []byte, telnetState *TelnetState) []byte {
	result := make([]byte, 0, len(data))
	i := 0

	for i < len(data) {
		if data[i] == 255 { // IAC (Interpret As Command)
			// Handle IAC sequences
			if i+1 < len(data) {
				command := data[i+1]
				if command >= 251 && command <= 254 { // WILL, WONT, DO, DONT
					if i+2 < len(data) {
						option := data[i+2]
						logger.Info(logger.AreaTerminal, "Telnet negotiation received: IAC %d option %d for session %s", command, option, telnetState.SessionID)

						// Handle specific telnet options and send appropriate responses
						switch option {
						case 31: // NAWS (Negotiate About Window Size)
							if command == 253 { // DO NAWS - Server asks us to send window size
								// Respond with WILL NAWS (we agree to send window size)
								response := []byte{255, 251, 31} // IAC WILL NAWS
								telnetState.Connection.Write(response)
								logger.Info(logger.AreaTerminal, "Sent WILL NAWS response for session %s", telnetState.SessionID)

								// Send current window size (already sent during connection, but send again)
								windowResponse := []byte{255, 250, 31, 0, 80, 0, 24, 255, 240} // IAC SB NAWS 80 24 IAC SE
								telnetState.Connection.Write(windowResponse)
								logger.Info(logger.AreaTerminal, "Sent window size (80x24) for session %s", telnetState.SessionID)
							}
						case 24: // TERMINAL_TYPE
							if command == 253 { // DO TERMINAL_TYPE - Server asks for our terminal type
								// Respond with WILL TERMINAL_TYPE (we agree to send terminal type)
								response := []byte{255, 251, 24} // IAC WILL TERMINAL_TYPE
								telnetState.Connection.Write(response)
								logger.Info(logger.AreaTerminal, "Sent WILL TERMINAL_TYPE response for session %s", telnetState.SessionID)
							}
						case 1: // ECHO
							if command == 251 { // WILL ECHO
								logger.Info(logger.AreaTerminal, "Server will echo for session %s", telnetState.SessionID)
								telnetState.ServerEcho = true
								telnetState.LocalEcho = false
								// Respond with DO ECHO to acknowledge
								response := []byte{255, 253, 1} // IAC DO ECHO
								telnetState.Connection.Write(response)
								logger.Info(logger.AreaTerminal, "Sent DO ECHO response for session %s", telnetState.SessionID)

								// Notify frontend about echo state change
								echoMessage := shared.Message{
									Type:      shared.MessageTypeTelnet,
									Content:   "echo_state",
									SessionID: telnetState.SessionID,
									Params: map[string]interface{}{
										"serverEcho": true,
										"localEcho":  false,
									}}
								if os.SendToClientCallback != nil {
									// CRITICAL FIX: Make callback asynchronous to prevent deadlock
									go func(msg shared.Message) {
										os.SendToClientCallback(telnetState.SessionID, msg)
									}(echoMessage)
								}
							} else if command == 252 { // WONT ECHO
								logger.Info(logger.AreaTerminal, "Server won't echo for session %s", telnetState.SessionID)
								telnetState.ServerEcho = false
								telnetState.LocalEcho = true
								// Respond with DONT ECHO to acknowledge
								response := []byte{255, 254, 1} // IAC DONT ECHO
								telnetState.Connection.Write(response)
								logger.Info(logger.AreaTerminal, "Sent DONT ECHO response for session %s", telnetState.SessionID)

								// Notify frontend about echo state change
								echoMessage := shared.Message{
									Type:      shared.MessageTypeTelnet,
									Content:   "echo_state",
									SessionID: telnetState.SessionID,
									Params: map[string]interface{}{
										"serverEcho": false,
										"localEcho":  true,
									}}
								if os.SendToClientCallback != nil {
									// CRITICAL FIX: Make callback asynchronous to prevent deadlock
									go func(msg shared.Message) {
										os.SendToClientCallback(telnetState.SessionID, msg)
									}(echoMessage)
								}
							}
						}

						// 3-byte sequence: IAC + command + option
						i += 3
						continue
					}
				} else if command == 250 { // SB (Subnegotiation Begin)
					// Handle subnegotiation sequences
					logger.Info(logger.AreaTerminal, "Received telnet subnegotiation for session %s", telnetState.SessionID)

					// Look for the subnegotiation type
					if i+2 < len(data) {
						subOption := data[i+2]
						logger.Info(logger.AreaTerminal, "Subnegotiation for option %d", subOption)

						switch subOption {
						case 24: // TERMINAL_TYPE
							if i+3 < len(data) && data[i+3] == 1 { // TERMINAL_TYPE SEND
								// Server wants our terminal type - respond with xterm for mapscii compatibility
								// MapSCII checks TERM environment variable and expects standard xterm
								terminalType := []byte{255, 250, 24, 0} // IAC SB TERMINAL_TYPE IS
								terminalType = append(terminalType, []byte("xterm")...)
								terminalType = append(terminalType, []byte{255, 240}...) // IAC SE
								telnetState.Connection.Write(terminalType)
								logger.Info(logger.AreaTerminal, "Sent terminal type 'xterm' for session %s (mapscii compatibility)", telnetState.SessionID)
							}
						case 31: // NAWS
							// Server requests window size subnegotiation
							logger.Info(logger.AreaTerminal, "Server requested NAWS subnegotiation for session %s", telnetState.SessionID)

							// Get terminal dimensions for this session
							clientCols, clientRows := os.GetTerminalDimensions(telnetState.SessionID)

							// Send window size: IAC SB NAWS width_high width_low height_high height_low IAC SE
							width := uint16(clientCols)
							height := uint16(clientRows)

							naws := []byte{255, 250, 31}                            // IAC SB NAWS
							naws = append(naws, byte(width>>8), byte(width&0xFF))   // Width (high, low)
							naws = append(naws, byte(height>>8), byte(height&0xFF)) // Height (high, low)
							naws = append(naws, 255, 240)                           // IAC SE

							telnetState.Connection.Write(naws)
							logger.Info(logger.AreaTerminal, "Sent NAWS subnegotiation: %dx%d for session %s", width, height, telnetState.SessionID)
						default:
							logger.Info(logger.AreaTerminal, "Unknown subnegotiation option %d", subOption)
						}
					}

					// Skip until SE (240)
					for j := i + 2; j < len(data); j++ {
						if data[j] == 240 { // SE (Subnegotiation End)
							i = j + 1
							break
						}
					}
					continue
				}
				// Other 2-byte sequences
				i += 2
				continue
			}
			i++ // Just skip IAC if at end
			continue
		}

		// Keep regular data
		result = append(result, data[i])
		i++
	}

	return result
}

// HandleTelnetInput processes input from the user to send to telnet server
func (os *TinyOS) HandleTelnetInput(input string, sessionID string) []shared.Message {
	// Use a timeout-protected mutex access to prevent deadlocks
	done := make(chan bool, 1)
	var telnetState *TelnetState
	var exists bool

	go func() {
		os.telnetMutex.RLock()
		telnetState, exists = os.telnetStates[sessionID]
		os.telnetMutex.RUnlock()
		done <- true
	}()
	// Wait for mutex access with increased timeout for better reliability
	select {
	case <-done:
		// Mutex access completed
	case <-time.After(10 * time.Second): // Increased from 5s for better stability
		logger.Error(logger.AreaTerminal, "Telnet input handler: mutex timeout for session %s", sessionID)
		return os.CreateWrappedTextMessage(sessionID, "Telnet session temporarily unavailable")
	}

	if !exists {
		return os.CreateWrappedTextMessage(sessionID, "No active telnet session")
	}

	// Check for exit commands
	if input == "\x18" || input == "\x1b" { // Ctrl+X or ESC
		return os.exitTelnetSession(sessionID)
	} // Send input to telnet server with timeout protection
	if telnetState.Connection != nil {
		// Use a more generous write timeout to prevent premature failures
		telnetState.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second)) // Increased from 5s

		// CRITICAL FIX: Use goroutine with timeout to prevent backend blocking
		done := make(chan error, 1)
		go func() {
			// Send input exactly as received (Unix telnet behavior)
			_, err := telnetState.Connection.Write([]byte(input))
			done <- err
		}()

		select {
		case err := <-done:
			if err != nil {
				logger.Error(logger.AreaTerminal, "Error sending telnet input for %s: %v", sessionID, err)
				// Connection is dead, clean up
				go os.CleanupTelnetSession(sessionID)
				return os.CreateWrappedTextMessage(sessionID, "Telnet connection lost")
			}
		case <-time.After(5 * time.Second):
			logger.Error(logger.AreaTerminal, "CRITICAL: Telnet write timeout for session %s - connection appears dead", sessionID)
			// Force cleanup of dead connection to prevent further blocking
			go os.CleanupTelnetSession(sessionID)
			return os.CreateWrappedTextMessage(sessionID, "Telnet connection timeout - session terminated")
		}
		// Reset write deadline immediately
		telnetState.Connection.SetWriteDeadline(time.Time{})

		telnetState.LastActivity = time.Now()
	}

	// Return empty message for now - output will come via the output channel
	return []shared.Message{}
}

// exitTelnetSession terminates a telnet session
func (os *TinyOS) exitTelnetSession(sessionID string) []shared.Message {
	logger.Info(logger.AreaTerminal, "=== TELNET EXIT START for session %s ===", sessionID)

	// CRITICAL FIX: Use timeout-protected lock to prevent deadlock
	lockAcquired := make(chan bool, 1)

	go func() {
		os.telnetMutex.Lock()
		lockAcquired <- true
	}()

	select {
	case <-lockAcquired:
		// Lock acquired successfully, proceed with cleanup
		defer os.telnetMutex.Unlock()
	case <-time.After(10 * time.Second):
		logger.Error(logger.AreaTerminal, "CRITICAL: Telnet exit mutex timeout for session %s - deadlock avoided", sessionID)
		return os.CreateWrappedTextMessage(sessionID, "Session cleanup temporarily unavailable")
	}

	// Now we have the lock, proceed with cleanup
	defer os.telnetMutex.Unlock()

	telnetState, exists := os.telnetStates[sessionID]

	// ENHANCED DEBUG: Log telnet state before cleanup
	logger.Debug(logger.AreaTerminal, "TELNET EXIT: Found telnet state: exists=%t", exists)
	if exists && telnetState != nil {
		logger.Debug(logger.AreaTerminal, "TELNET EXIT: State details - server=%s, connection_alive=%t",
			telnetState.ServerName, telnetState.Connection != nil)
	}

	if exists {
		logger.Info(logger.AreaTerminal, "Closing telnet connection for session %s", sessionID)

		// Signal shutdown to the goroutine first
		select {
		case telnetState.ShutdownChan <- struct{}{}:
			logger.Info(logger.AreaTerminal, "Shutdown signal sent for session %s", sessionID)
		default:
			logger.Warn(logger.AreaTerminal, "Shutdown channel full or closed for session %s", sessionID)
		}

		// Close connection
		if telnetState.Connection != nil {
			telnetState.Connection.Close()
		}

		// Send end message to frontend before cleanup
		endMessage := shared.Message{
			Type:      shared.MessageTypeTelnet,
			Content:   "end",
			SessionID: sessionID,
		}

		// Try to send via callback if available
		if os.SendToClientCallback != nil {
			err := os.SendToClientCallback(sessionID, endMessage)
			if err != nil {
				logger.Warn(logger.AreaTerminal, "Failed to send telnet end message via callback: %v", err)
			} else {
				logger.Info(logger.AreaTerminal, "Telnet end message sent via callback for session %s", sessionID)
			}
		} // Also try to send via output channel if still available
		if !telnetState.safeSendToOutputChan(endMessage, 200*time.Millisecond) {
			logger.Warn(logger.AreaTerminal, "Failed to send telnet end message via channel for session %s - channel closed or timeout", sessionID)
		} else {
			logger.Info(logger.AreaTerminal, "Telnet end message sent via channel for session %s", sessionID)
		}
		delete(os.telnetStates, sessionID)
		logger.Info(logger.AreaTerminal, "Telnet state deleted for session %s", sessionID)

		// ENHANCED DEBUG: Verify telnet state is really deleted
		_, stillExists := os.telnetStates[sessionID]
		logger.Debug(logger.AreaTerminal, "TELNET EXIT VERIFICATION: session %s still exists after delete: %t, total remaining: %d",
			sessionID, stillExists, len(os.telnetStates))

		logger.Info(logger.AreaTerminal, "Telnet session removed from state map for %s, remaining sessions: %d", sessionID, len(os.telnetStates))
	} else {
		logger.Info(logger.AreaTerminal, "No telnet state found for session %s", sessionID)
	}

	logger.Info(logger.AreaTerminal, "=== TELNET EXIT END ===")

	if exists {
		logger.Info(logger.AreaTerminal, "Telnet session manually terminated for %s", sessionID)
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"},
			{Type: shared.MessageTypeText, Content: "Telnet session terminated"},
			{Type: shared.MessageTypeTelnet, Content: "end", SessionID: sessionID},
		}
	}
	return os.CreateWrappedTextMessage(sessionID, "No active telnet session")
}

// CleanupTelnetSession safely cleans up a telnet session with deadlock prevention
func (os *TinyOS) CleanupTelnetSession(sessionID string) {
	logger.Info(logger.AreaTerminal, "=== TELNET CLEANUP START for session %s ===", sessionID)

	// CRITICAL FIX: Use timeout-protected lock to prevent deadlock
	lockAcquired := make(chan bool, 1)

	go func() {
		os.telnetMutex.Lock()
		lockAcquired <- true
	}()

	select {
	case <-lockAcquired:
		// Lock acquired successfully, proceed with cleanup
		defer os.telnetMutex.Unlock()
	case <-time.After(10 * time.Second):
		logger.Error(logger.AreaTerminal, "CRITICAL: Telnet cleanup mutex timeout for session %s - deadlock avoided, session may leak", sessionID)
		return
	}

	telnetState, exists := os.telnetStates[sessionID]
	if !exists {
		logger.Debug(logger.AreaTerminal, "Telnet cleanup: session %s not found in telnet states", sessionID)
		return
	}

	logger.Info(logger.AreaTerminal, "Cleaning up telnet session %s (server: %s)", sessionID, telnetState.ServerName)

	// Signal shutdown to the goroutine first
	select {
	case telnetState.ShutdownChan <- struct{}{}:
		logger.Debug(logger.AreaTerminal, "Shutdown signal sent for session %s", sessionID)
	default:
		logger.Debug(logger.AreaTerminal, "Shutdown channel full or closed for session %s", sessionID)
	}

	// Close connection if still open
	if telnetState.Connection != nil {
		err := telnetState.Connection.Close()
		if err != nil {
			logger.Debug(logger.AreaTerminal, "Error closing telnet connection for session %s: %v", sessionID, err)
		}
		telnetState.Connection = nil
	}

	// Send end message to frontend
	endMessage := shared.Message{
		Type:      shared.MessageTypeTelnet,
		Content:   "end",
		SessionID: sessionID,
	}

	// Try to send via callback if available (async to prevent deadlock)
	if os.SendToClientCallback != nil {
		go func() {
			err := os.SendToClientCallback(sessionID, endMessage)
			if err != nil {
				logger.Debug(logger.AreaTerminal, "Failed to send telnet end message via callback: %v", err)
			} else {
				logger.Info(logger.AreaTerminal, "Telnet end message sent via callback for session %s", sessionID)
			}
		}()
	}
	// Also try via output channel with timeout
	if !telnetState.safeSendToOutputChan(endMessage, 200*time.Millisecond) {
		logger.Debug(logger.AreaTerminal, "Timeout sending telnet end message via channel for session %s", sessionID)
	} else {
		logger.Info(logger.AreaTerminal, "Telnet end message sent via channel for session %s", sessionID)
	} // Remove from telnet states
	delete(os.telnetStates, sessionID)
	remainingSessions := len(os.telnetStates)

	logger.Info(logger.AreaTerminal, "=== TELNET CLEANUP COMPLETE for session %s, remaining sessions: %d ===", sessionID, remainingSessions)
}
