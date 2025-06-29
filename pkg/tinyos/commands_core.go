package tinyos

import (
	"context"

	"github.com/antibyte/retroterm/pkg/auth"
	"github.com/antibyte/retroterm/pkg/chess"
	"github.com/antibyte/retroterm/pkg/editor"

	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// ExecuteWithContext executes a command with the given context
// and extracts the SessionID from the context
func (os *TinyOS) ExecuteWithContext(ctx context.Context, input string) []shared.Message { // Extract session ID from context
	var sessionID string = auth.SessionIDFromContext(ctx)

	// Wenn keine Session-ID vorhanden ist, versuche eine Gast-Session zu erstellen
	if sessionID == "" {
		logger.Warn(logger.AreaGeneral, "No session ID in context, attempting to process without session")
		// Für bestimmte Befehle ohne Session-Anforderung fortfahren
		tokens := strings.Fields(input)
		if len(tokens) > 0 {
			cmd := strings.ToLower(tokens[0])
			if cmd == "help" || cmd == "echo" || cmd == "clear" {
				// Diese Befehle können ohne Session ausgeführt werden
				return os.ProcessCommand("", input)
			}
		}
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "No active session. Please reload the page to establish a new session."}}
	}

	// --- Authoritative Input Routing based on InputMode ---
	// This is the core of the race condition fix. It ensures input is always
	// routed to the correct handler based on the session's current mode.
	currentMode := os.GetInputMode(sessionID)
	logger.Info(logger.AreaSession, "ExecuteWithContext: session %s, mode %d, input '%s'", sessionID, currentMode, input)

	switch currentMode {
	case InputModeEditor:
		editorManager := editor.GetEditorManager()
		if editor := editorManager.GetEditor(sessionID); editor != nil {
			if editor.ProcessInput(input) {
				return nil // Editor handled input, no further processing
			}
			// ProcessInput returned false, which means the editor has exited.
			// We must now reset the input mode to the OS shell.
			logger.Info(logger.AreaEditor, "Editor for session %s exited. Resetting input mode to OS Shell.", sessionID)
			os.SetInputMode(sessionID, InputModeOSShell)
			// The input that caused the exit (e.g., Ctrl+X) should not be processed by the shell.
			return nil
		} else {
			// Editor instance not found, but mode is Editor. This is an inconsistent state.
			// Reset the mode to be safe and process as a shell command.
			logger.Warn(logger.AreaEditor, "Inconsistent state: InputMode is Editor, but no editor instance found for session %s. Resetting mode.", sessionID)
			os.SetInputMode(sessionID, InputModeOSShell)
		}

	case InputModeChess:
		os.sessionMutex.Lock()
		session, exists := os.sessions[sessionID]
		if exists && session.ChessGame != nil {
			chessGame := session.ChessGame
			os.sessionMutex.Unlock()
			return chessGame.HandleInput(input)
		}
		os.sessionMutex.Unlock()
		// Inconsistent state, reset mode
		os.SetInputMode(sessionID, InputModeOSShell)

	case InputModeTelnet:
		return os.HandleTelnetInput(input, sessionID)
	case InputModePager:
		return os.handleCatPagerInput(input, sessionID)
	case InputModeLoginProcess:
		return os.handleLoginInput(input, sessionID)
	case InputModeRegistrationProcess:
		return os.handleRegistrationInput(input, sessionID)
	case InputModePasswordChange:
		return os.handlePasswordChangeInput(input, sessionID)

	case InputModeOSShell:
		// Continue to normal command processing below
	default:
		// Unknown mode, reset to be safe
		logger.Warn(logger.AreaSession, "Unknown input mode %d for session %s. Resetting to OS Shell.", currentMode, sessionID)
		os.SetInputMode(sessionID, InputModeOSShell)
	}

	// --- OS Shell Command Processing ---

	// Check if we are in a registration process
	if sessionID != "" && os.isInRegistrationProcess(sessionID) {
		// Process registration input
		return os.handleRegistrationInput(input, sessionID)
	}
	// Check if we are in a password change process
	if sessionID != "" && os.isInPasswordChangeProcess(sessionID) {
		// Process password change input
		return os.handlePasswordChangeInput(input, sessionID)
	}
	// Check if we are in a login process
	if sessionID != "" && os.isInLoginProcess(sessionID) {
		// Process login input
		return os.handleLoginInput(input, sessionID)
	} // Check if we are in a CAT pager process
	if sessionID != "" && os.isInCatPagerProcess(sessionID) {
		// Process CAT pager input
		logger.Debug(logger.AreaTerminal, "CAT PAGER INPUT processing for session %s: input=%q", sessionID, input)
		return os.handleCatPagerInput(input, sessionID)
	}

	// Split command into tokens
	tokens := strings.Fields(input)

	// Check if a valid session exists
	if sessionID != "" {
		os.sessionMutex.RLock()
		_, exists := os.sessions[sessionID]
		os.sessionMutex.RUnlock()

		// If a valid session exists
		if exists {
			// For empty input, simply return an empty message
			if len(tokens) == 0 {
				return os.CreateWrappedTextMessage(sessionID, "")
			}
		} else {
			// Session does not exist in the map
			logger.Debug(logger.AreaAuth, "SessionID %s does not exist in the sessions map", sessionID) // Try to load the session from the database if available
			if os.db != nil {
				var username string
				err := os.db.QueryRow("SELECT username FROM user_sessions WHERE session_id = ?", sessionID).Scan(&username)
				err = os.db.QueryRow("SELECT username FROM user_sessions WHERE session_id = ?", sessionID).Scan(&username)
				if err == nil && username != "" {

					// Load session from database into the map
					os.sessionMutex.Lock()
					os.sessions[sessionID] = &Session{
						ID:           sessionID,
						Username:     username,
						CurrentPath:  "/home/" + username,
						CreatedAt:    time.Now(),
						LastActivity: time.Now(),
					}
					os.sessionMutex.Unlock()

					return os.ProcessCommand(sessionID, input)
				}
			}

			return os.CreateWrappedTextMessage(sessionID, "No valid session found")
		}
	}
	// If no Session-ID is present or other cases
	if len(tokens) == 0 {
		return os.CreateWrappedTextMessage("", "")
	}
	// Extract command and arguments
	cmd := strings.ToLower(tokens[0])
	args := tokens[1:] // For commands that need the Session-ID, we add it as the first argument
	if sessionID != "" {
		switch cmd {
		case "whoami", "logout", "passwd", "ls", "pwd", "cd", "mkdir", "cat", "write", "rm", "limits", "chat", "chathistory", "basic", "run", "edit", "view", "resources", "telnet":
			args = append([]string{sessionID}, args...)
		}
	}

	// Special handling for __BREAK__ - ignore in TinyOS
	if cmd == "__break__" {
		return []shared.Message{} // Empty response
	} // Execute command based on first token
	logger.Debug(logger.AreaTerminal, "Processing command '%s' with sessionID '%s', args: %v", cmd, sessionID, args)

	// ENHANCED DEBUG: Log final command execution route
	logger.Debug(logger.AreaTerminal, "EXECUTECONTEXT FINAL ROUTE: sessionID=%s, cmd=%s, about to execute command handler",
		sessionID, cmd)

	switch cmd {
	case "help":
		return os.cmdHelp(args)
	case "echo":
		return os.cmdEcho(args)
	case "clear":
		return os.cmdClear(args)
	case "register":
		// Start new registration process
		return os.cmdRegisterNew(args, nil, sessionID)
	case "login":
		return os.cmdLoginNew(args, sessionID)
	case "logout":
		return os.cmdLogout(args)
	case "whoami":
		return os.cmdWhoAmI(args)
	case "chat":
		return os.cmdChat(args)
	case "chathistory":
		return os.cmdChatHistory(args) // File system commands
	case "ls":
		return os.cmdLs(args)
	case "pwd":
		return os.cmdPwd(args)
	case "cd":
		return os.cmdCd(args)
	case "mkdir":
		return os.cmdMkdir(args)
	case "cat":
		return os.cmdCat(args)
	case "write":
		return os.cmdWrite(args)
	case "rm":
		return os.cmdRm(args)
	case "limits":
		logger.Debug(logger.AreaTerminal, "Calling cmdLimits with args: %v", args)
		return os.cmdLimits(args)
	case "resources":
		logger.Debug(logger.AreaTerminal, "Calling cmdResources with args: %v", args)
		return os.cmdResources(args)
	case "edit":
		return os.cmdEdit(args)
	case "view":
		return os.cmdView(args)
	case "basic":
		return os.cmdBasic(args)
	case "chess":
		return os.cmdChess(args, sessionID)
	case "telnet":
		return os.cmdTelnet(args)
	case "run":
		return os.cmdRun(args)
	case "date":
		return os.cmdDate(args)
	case "about":
		return os.cmdAbout(args)
	case "passwd":
		return os.cmdPasswd(args)
	default:
		logger.Debug(logger.AreaTerminal, "Unknown command: %s", cmd)
		return os.CreateWrappedTextMessage(sessionID, "Unknown command: "+cmd)
	}
}

// ProcessCommand is a helper method for processing commands after session restoration
func (os *TinyOS) ProcessCommand(sessionID string, input string) []shared.Message {
	// ENHANCED DEBUG: Log ProcessCommand entry
	logger.Debug(logger.AreaTerminal, "PROCESSCOMMAND START: sessionID=%s, input=%q", sessionID, input)
	// Check if we are in a telnet process first (defensive check)
	if sessionID != "" && os.isInTelnetProcess(sessionID) {
		logger.Debug(logger.AreaTerminal, "PROCESSCOMMAND: Still in telnet mode for session %s, routing back to telnet handler", sessionID)
		return os.HandleTelnetInput(input, sessionID)
	}

	// Check if we are in a registration process
	if os.isInRegistrationProcess(sessionID) {
		// Process registration input
		return os.handleRegistrationInput(input, sessionID)
	}

	// Check if we are in a login process
	if os.isInLoginProcess(sessionID) {
		// Process login input
		return os.handleLoginInput(input, sessionID)
	}
	// Check if we are in a password change process
	if os.isInPasswordChangeProcess(sessionID) {
		// Process password change input
		return os.handlePasswordChangeInput(input, sessionID)
	}
	// Check if we are in an active chess game
	if sessionID != "" {
		os.sessionMutex.Lock()
		session, exists := os.sessions[sessionID]
		if exists && session.ChessActive && session.ChessGame != nil {
			os.sessionMutex.Unlock()

			logger.Info(logger.AreaTerminal, "Redirecting input to active chess game: %s", input)
			// Handle chess input directly
			messages := session.ChessGame.HandleInput(input)

			// Check if chess game should be quit (look for special quit signal)
			for _, msg := range messages {
				if msg.Type == shared.MessageTypeText && msg.Content == "CHESS_QUIT_SIGNAL" {
					// End chess game and filter out the quit signal message
					os.sessionMutex.Lock()
					session.ChessGame = nil
					session.ChessActive = false
					os.sessions[sessionID] = session
					os.sessionMutex.Unlock()

					// Return filtered messages without the quit signal
					filteredMessages := make([]shared.Message, 0)
					for _, m := range messages {
						if m.Content != "CHESS_QUIT_SIGNAL" {
							filteredMessages = append(filteredMessages, m)
						}
					}
					// Add final message indicating return to TinyOS
					filteredMessages = append(filteredMessages, shared.Message{
						Type:    shared.MessageTypeText,
						Content: "Back to TinyOS.",
					})

					logger.Info(logger.AreaChess, "Chess game ended for session %s", sessionID)
					return filteredMessages
				}
			}

			return messages
		}
		os.sessionMutex.Unlock()
	}

	// Split command into tokens
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return os.CreateWrappedTextMessage(sessionID, "")
	} // Extract command and arguments
	cmd := strings.ToLower(tokens[0])
	args := tokens[1:]
	// DEBUG: Log welcher Befehl verarbeitet wird
	logger.Debug(logger.AreaTerminal, "Processing command: '%s' for session: '%s'", cmd, sessionID)
	// For commands that need the Session-ID, we add it as the first argument
	if sessionID != "" {
		switch cmd {
		case "passwd", "whoami", "logout", "ls", "pwd", "cd", "mkdir", "cat", "write", "rm", "limits", "chat", "chathistory", "basic", "run", "edit", "view", "resources", "debug", "telnet":
			args = append([]string{sessionID}, args...)
			logger.Debug(logger.AreaTerminal, "Added sessionID to args for command '%s', args length: %d", cmd, len(args))
		}
	}

	// Special handling for __BREAK__ - ignore in TinyOS
	if cmd == "__break__" {
		return []shared.Message{} // Empty response
	} // Execute command based on the first token
	logger.Debug(logger.AreaTerminal, "Processing command '%s' with sessionID '%s', args: %v", cmd, sessionID, args)

	// ENHANCED DEBUG: Log final command execution route
	logger.Debug(logger.AreaTerminal, "EXECUTECONTEXT FINAL ROUTE: sessionID=%s, cmd=%s, about to execute command handler",
		sessionID, cmd)

	switch cmd {
	case "help":
		return os.cmdHelp(args)
	case "echo":
		return os.cmdEcho(args)
	case "clear":
		return os.cmdClear(args)
	case "register":
		// Start new registration process
		return os.cmdRegisterNew(args, nil, sessionID)
	case "login":
		return os.cmdLoginNew(args, sessionID)
	case "logout":
		return os.cmdLogout(args)
	case "whoami":
		return os.cmdWhoAmI(args)
	case "chat":
		return os.cmdChat(args)
	case "chathistory":
		return os.cmdChatHistory(args) // File system commands
	case "ls":
		return os.cmdLs(args)
	case "pwd":
		return os.cmdPwd(args)
	case "cd":
		return os.cmdCd(args)
	case "mkdir":
		return os.cmdMkdir(args)
	case "cat":
		return os.cmdCat(args)
	case "write":
		return os.cmdWrite(args)
	case "rm":
		return os.cmdRm(args)
	case "limits":
		logger.Debug(logger.AreaTerminal, "Calling cmdLimits with args: %v", args)
		return os.cmdLimits(args)
	case "resources":
		logger.Debug(logger.AreaTerminal, "Calling cmdResources with args: %v", args)
		return os.cmdResources(args)
	case "edit":
		return os.cmdEdit(args)
	case "view":
		return os.cmdView(args)
	case "basic":
		return os.cmdBasic(args)
	case "chess":
		return os.cmdChess(args, sessionID)
	case "telnet":
		return os.cmdTelnet(args)
	case "run":
		return os.cmdRun(args)
	case "date":
		return os.cmdDate(args)
	case "about":
		return os.cmdAbout(args)
	case "passwd":
		return os.cmdPasswd(args)
	default:
		logger.Debug(logger.AreaTerminal, "Unknown command: %s", cmd)
		return os.CreateWrappedTextMessage(sessionID, "Unknown command: "+cmd)
	}
}

// Execute executes a command and returns the result
func (os *TinyOS) Execute(input string) []shared.Message {
	// Split command into tokens
	tokens := strings.Fields(input)
	if len(tokens) == 0 {
		return os.CreateWrappedTextMessage("", "")
	}

	// Extract command and arguments
	cmd := strings.ToLower(tokens[0])
	args := tokens[1:] // Execute command based on the first token
	switch cmd {
	case "help":
		return os.cmdHelp(args)
	case "echo":
		return os.cmdEcho(args)
	case "clear":
		return os.cmdClear(args)
	case "register":
		// Registration requires a valid session from the frontend
		return os.CreateWrappedTextMessage("", "Registration is only available through the web interface.")
	case "login":
		return os.cmdLoginNew(args, "")
	case "logout":
		return os.cmdLogout(args)
	case "whoami":
		return os.cmdWhoAmI(args)
	case "chat":
		return os.cmdChat(args)
	case "ls":
		return os.cmdLs(args)
	case "pwd":
		return os.cmdPwd(args)
	case "cd":
		return os.cmdCd(args)
	case "mkdir":
		return os.cmdMkdir(args)
	case "cat":
		return os.cmdCat(args)
	case "write":
		return os.cmdWrite(args)
	case "rm":
		return os.cmdRm(args)
	case "basic":
		return os.cmdBasic(args)
	case "edit":
		return os.cmdEdit(args)
	case "view":
		return os.cmdView(args)
	case "date":
		return os.cmdDate(args)
	case "about":
		return os.cmdAbout(args)
	case "passwd":
		return os.cmdPasswd(args)
	default:
		return os.CreateWrappedTextMessage("", "Unknown command: "+cmd)
	}
}

// IsInCatPagerProcess checks if a session is currently in a CAT pager process (exported version)
func (os *TinyOS) IsInCatPagerProcess(sessionID string) bool {
	os.catPagerMutex.RLock()
	defer os.catPagerMutex.RUnlock()
	_, exists := os.catPagerStates[sessionID]

	// Debug logging to track pager state
	logger.Debug(logger.AreaTerminal, "CAT PAGER CHECK for session %s: exists=%t, total_pager_states=%d",
		sessionID, exists, len(os.catPagerStates))

	return exists
}

// isInCatPagerProcess checks if a session is currently in a CAT pager process (internal version)
func (os *TinyOS) isInCatPagerProcess(sessionID string) bool {
	return os.IsInCatPagerProcess(sessionID)
}

// Helper functions for chess command
func getDifficultyName(difficulty int) string {
	switch difficulty {
	case 1:
		return "Easy"
	case 2:
		return "Medium"
	case 3:
		return "Hard"
	default:
		return "Medium"
	}
}

func getColorName(color chess.Color) string {
	if color == chess.White {
		return "White"
	}
	return "Black"
}

// handleCatPagerInput handles input during CAT pager process
func (os *TinyOS) handleCatPagerInput(input string, sessionID string) []shared.Message {
	os.catPagerMutex.RLock()
	state, exists := os.catPagerStates[sessionID]
	os.catPagerMutex.RUnlock()

	if !exists {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Error: No active CAT pager session"}}
	}
	// Process single character input (case insensitive)
	input = strings.ToLower(strings.TrimSpace(input))

	logger.Debug(logger.AreaTerminal, "CAT PAGER INPUT: session=%s, input=%q, currentLine=%d, totalLines=%d",
		sessionID, input, state.CurrentLine, len(state.Lines))

	// Handle quit commands
	if input == "q" || input == "quit" || input == "\x1b" || input == "\x03" {
		// Quit pager (q, quit, ESC, Ctrl+C)
		os.catPagerMutex.Lock()
		delete(os.catPagerStates, sessionID)
		os.catPagerMutex.Unlock()

		// Get username for proper prompt
		promptText := os.GetPromptForSession(sessionID)

		return []shared.Message{
			{Type: shared.MessageTypeEditor, EditorCommand: "status", EditorStatus: ""}, // Clear status line
			{Type: shared.MessageTypePager, Content: "deactivate"},                      // Tell frontend to exit pager mode
			{Type: shared.MessageTypeText, Content: promptText, NoNewline: true},        // OS prompt without newline
		}
	} else if input == "m" || input == "more" || input == "" || input == " " || input == "\r" || input == "\n" {
		// Show more - display next page (m, more, empty, SPACE, ENTER)
		return os.showNextCatPage(sessionID, state)
	} else {
		// Invalid input - show help but don't exit pager mode
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Press m for more, q to quit"}}
	}
}

// showNextCatPage displays the next page of content in CAT pager
func (os *TinyOS) showNextCatPage(sessionID string, state *CatPagerState) []shared.Message {
	startLine := state.CurrentLine
	endLine := startLine + state.PageSize

	logger.Debug(logger.AreaTerminal, "CAT PAGER NEXT: session=%s, startLine=%d, endLine=%d, totalLines=%d, pageSize=%d",
		sessionID, startLine, endLine, len(state.Lines), state.PageSize)

	if startLine >= len(state.Lines) {
		// No more content
		os.catPagerMutex.Lock()
		delete(os.catPagerStates, sessionID)
		os.catPagerMutex.Unlock()
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "--- End of file ---"},
			{Type: shared.MessageTypePager, Content: "deactivate"}, // Tell frontend to exit pager mode
		}
	}

	// Get the lines for this page
	var pageLines []string
	if endLine > len(state.Lines) {
		endLine = len(state.Lines)
	}

	pageLines = state.Lines[startLine:endLine]

	// Update current line position
	os.catPagerMutex.Lock()
	state.CurrentLine = endLine
	os.catPagerMutex.Unlock()

	// Prepare output
	content := strings.Join(pageLines, "\n") // Check if there are more lines to show
	if endLine < len(state.Lines) {
		// More content available - show pager prompt and activate pager mode
		basePrompt := "--- " + state.Filename + " --- m: more, q: quit"

		// Limit padding to avoid performance issues with very wide terminals
		prompt := basePrompt
		if state.Terminal.Cols > 0 && state.Terminal.Cols <= 120 {
			promptLen := len(basePrompt)
			if promptLen < state.Terminal.Cols {
				maxPadding := state.Terminal.Cols - promptLen
				if maxPadding > 50 {
					maxPadding = 50 // Limit padding to 50 characters max
				}
				padding := strings.Repeat(" ", maxPadding)
				prompt = basePrompt + padding
			}
		}

		return []shared.Message{
			{Type: shared.MessageTypeText, Content: content},
			{Type: shared.MessageTypeEditor, EditorCommand: "status", EditorStatus: prompt}, // Set status line like editor
			{Type: shared.MessageTypePager, Content: "activate"},                            // Tell frontend to enter pager mode
		}
	} else {
		// Last page - clean up state and show final content
		os.catPagerMutex.Lock()
		delete(os.catPagerStates, sessionID)
		os.catPagerMutex.Unlock()

		return []shared.Message{
			{Type: shared.MessageTypeEditor, EditorCommand: "status", EditorStatus: ""}, // Clear status line
			{Type: shared.MessageTypePager, Content: "deactivate"},                      // Tell frontend to exit pager mode first
			{Type: shared.MessageTypeText, Content: content},                            // Just the content, no prompt
		}
	}
}
