package tinyos

import (
	"fmt"
	"strings"

	"github.com/antibyte/retroterm/pkg/board"
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// BoardSession represents an active board session
type BoardSession struct {
	UI        *board.BoardUI
	Active    bool
	SessionID string
}

// cmdBoard handles the board command
func (os *TinyOS) cmdBoard(args []string) []shared.Message {
	if len(args) == 0 {
		return os.CreateWrappedTextMessage("", "board: missing session ID")
	}
	
	sessionID := args[0]
	
	// Get session info
	os.sessionMutex.RLock()
	session, exists := os.sessions[sessionID]
	os.sessionMutex.RUnlock()
	
	if !exists {
		return os.CreateWrappedTextMessage(sessionID, "No active session found")
	}
	
	username := session.Username
	if username == "" {
		username = "guest"
	}
	
	isGuest := (username == "guest")
	
	// Initialize board manager if not already done
	if os.boardManager == nil {
		os.boardManager = board.NewBoardManager(os.db)
		if err := os.boardManager.InitializeDatabase(); err != nil {
			logger.Error(logger.AreaGeneral, "Failed to initialize board database: %v", err)
			return os.CreateWrappedTextMessage(sessionID, fmt.Sprintf("Board system error: %v", err))
		}
	}
	
	// Get terminal dimensions
	terminalWidth := 80
	terminalHeight := 24
	if session.Terminal.Cols > 0 {
		terminalWidth = session.Terminal.Cols
	}
	if session.Terminal.Rows > 0 {
		terminalHeight = session.Terminal.Rows
	}
	
	// Create board UI
	boardUI := board.NewBoardUI(os.boardManager, sessionID, username, isGuest, terminalWidth, terminalHeight)
	
	// Store board session
	os.mu.Lock()
	if os.boardSessions == nil {
		os.boardSessions = make(map[string]*BoardSession)
	}
	os.boardSessions[sessionID] = &BoardSession{
		UI:        boardUI,
		Active:    true,
		SessionID: sessionID,
	}
	os.mu.Unlock()
	
	// Set input mode to board
	os.SetInputMode(sessionID, InputModeBoard)
	
	// Start the board UI
	messages, err := boardUI.Start()
	if err != nil {
		logger.Error(logger.AreaGeneral, "Error starting board UI: %v", err)
		return os.CreateWrappedTextMessage(sessionID, fmt.Sprintf("Board error: %v", err))
	}
	
	// Add mode change message
	result := []shared.Message{
		{
			Type:    shared.MessageTypeMode,
			Content: "BOARD",
		},
	}
	result = append(result, messages...)
	
	return result
}

// HandleBoardInput handles input when in board mode
func (os *TinyOS) HandleBoardInput(input string, sessionID string) []shared.Message {
	os.mu.Lock()
	boardSession, exists := os.boardSessions[sessionID]
	os.mu.Unlock()
	
	if !exists || !boardSession.Active {
		// Board session not found or inactive, return to OS shell
		os.SetInputMode(sessionID, InputModeOSShell)
		return []shared.Message{
			{
				Type:    shared.MessageTypeMode,
				Content: "OS_SHELL",
			},
			{
				Type:    shared.MessageTypeText,
				Content: "Board session ended.",
			},
		}
	}
	
	// Handle quit commands
	trimmedInput := strings.TrimSpace(input)
	if trimmedInput == "q" || trimmedInput == "quit" || trimmedInput == "exit" {
		return os.exitBoard(sessionID)
	}
	
	// For ViewMessage state, pass raw input to handle Enter correctly
	// For other states, use trimmed input
	var processInput string
	if boardSession.UI.IsViewingMessage() {
		processInput = input
	} else {
		processInput = trimmedInput
	}
	
	// Process input through board UI
	messages, err := boardSession.UI.ProcessInput(processInput)
	if err != nil {
		logger.Error(logger.AreaGeneral, "Error processing board input: %v", err)
		return os.CreateWrappedTextMessage(sessionID, fmt.Sprintf("Board error: %v", err))
	}
	
	// Check if we should exit board mode
	if !boardSession.UI.IsActive() {
		return os.exitBoard(sessionID)
	}
	
	return messages
}

// exitBoard exits board mode and returns to OS shell
func (os *TinyOS) exitBoard(sessionID string) []shared.Message {
	// Clean up board session
	os.mu.Lock()
	if os.boardSessions != nil {
		delete(os.boardSessions, sessionID)
	}
	os.mu.Unlock()
	
	// Set input mode back to OS shell
	os.SetInputMode(sessionID, InputModeOSShell)
	
	return []shared.Message{
		{
			Type:    shared.MessageTypeMode,
			Content: "OS_SHELL",
		},
		{
			Type:    shared.MessageTypeText,
			Content: "Goodbye from RetroTerm BBS!",
		},
	}
}

// isInBoardMode checks if a session is in board mode
func (os *TinyOS) isInBoardMode(sessionID string) bool {
	os.mu.Lock()
	defer os.mu.Unlock()
	
	if os.boardSessions == nil {
		return false
	}
	
	boardSession, exists := os.boardSessions[sessionID]
	return exists && boardSession.Active
}