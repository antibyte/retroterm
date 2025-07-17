package tinyos

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/chess"
	"github.com/antibyte/retroterm/pkg/editor"
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// cmdHelp displays help information
func (os *TinyOS) cmdHelp(args []string) []shared.Message {
	commands := []string{
		"help", "echo", "clear", "basic", "run", "chess", "chat", "chathistory", "register", "login", "logout", "whoami", "ls", "pwd", "cd", "mkdir", "cat", "write", "rm", "limits", "resources", "edit", "view", "debug", "telnet", "date", "about", "passwd", "board",
	}
	helpTexts := map[string]string{
		"help":  "help [command]\nShows a list of all commands or help for a specific command.\nExample: help ls",
		"echo":  "echo <text>\nReturns the text.\nExample: echo Hello World",
		"clear": "clear\nClears the screen.\nExample: clear", "basic": "basic\nStarts BASIC mode.\nExample: basic",
		"run":         "run <filename>\nRun a BASIC program directly from TinyOS.\nFilename can be with or without .bas extension.\nExample: run graphics\nExample: run sprites.bas",
		"chess":       "chess [difficulty] [color]\nStarts a chess game with the computer.\nDifficulty: easy/1, medium/2, hard/3 (default: medium)\nColor: white/w, black/b (default: white)\nExample: chess\nExample: chess easy white\nExample: chess hard black",
		"chat":        "chat\nStarts chat mode (login required).\nExample: chat",
		"chathistory": "chathistory\nShows the chat history (login required).\nExample: chathistory",
		"register":    "register\nStarts the registration process.\nExample: register",
		"login":       "login\nStarts the login process.\nExample: login",
		"logout":      "logout\nLogs out the current user.\nExample: logout",
		"whoami":      "whoami\nShows the current username.\nExample: whoami",
		"ls":          "ls [directory]\nLists the contents of a directory.\nExample: ls\nExample: ls /home/alice",
		"pwd":         "pwd\nShows the current directory.\nExample: pwd",
		"cd":          "cd <directory>\nChanges the directory.\nExample: cd /home/alice",
		"mkdir":       "mkdir <directory>\nCreates a new directory.\nExample: mkdir testdir",
		"cat":         "cat <file>\nShows the contents of a file.\nExample: cat readme.txt",
		"write":       "write <file> <content>\nWrites text to a file.\nExample: write test.txt Hello World", "rm": "rm <file/directory>\nDeletes a file or empty directory.\nExample: rm test.txt",
		"limits":    "limits\nShows your current resource limits and file usage.\nExample: limits",
		"resources": "resources\nShows detailed system resource statistics.\nExample: resources", "edit": "edit [filename]\nOpens the full-screen text editor.\nExample: edit\nExample: edit myfile.bas",
		"view":   "view <filename>\nOpens a file in read-only mode (view only).\nExample: view readme.txt\nExample: view myfile.bas",
		"telnet": "telnet <servername>\nConnect to a predefined telnet server.\nUse 'telnet list' to see available servers.\nExample: telnet towel\nExample: telnet list",
		"date":   "date\nShows the current date and time with year set to 1984.\nExample: date",
		"about":  "about\nShows information about this terminal system.\nExample: about",
		"passwd": "passwd\nChanges the password of the current user.\nExample: passwd",
		"board":  "board\nAccess the RetroTerm BBS message board system.\nGuests can read messages, registered users can post.\nExample: board",
	}

	// SessionID aus args extrahieren, wenn vorhanden
	sessionID := ""
	if len(args) > 0 && strings.HasPrefix(args[0], "session_") {
		sessionID = args[0]
		args = args[1:]
	}

	if len(args) == 0 {
		result := "Available commands:\n" + strings.Join(commands, ", ") + "\n\nUse help <command> for details"
		return os.CreateWrappedTextMessage(sessionID, result)
	}

	cmd := strings.ToLower(args[0])
	if txt, ok := helpTexts[cmd]; ok {
		return os.CreateWrappedTextMessage(sessionID, txt)
	}

	return os.CreateWrappedTextMessage(sessionID, "No help text available for this command.")
}

// cmdEcho returns the passed text
func (os *TinyOS) cmdEcho(args []string) []shared.Message {
	// SessionID aus args extrahieren, wenn vorhanden
	sessionID := ""
	if len(args) > 0 && strings.HasPrefix(args[0], "session_") {
		sessionID = args[0]
		args = args[1:]
	}

	text := strings.Join(args, " ")
	return os.CreateWrappedTextMessage(sessionID, text)
}

// cmdClear clears the screen
func (os *TinyOS) cmdClear(args []string) []shared.Message {
	// Für Clear verwenden wir weiterhin den direkten MessageTypeClear ohne Wrapping
	return []shared.Message{
		{Type: shared.MessageTypeClear},
	}
}

// cmdChat startet den Chat-Modus mit Rate-Limit- und Authentifizierungsprüfung
func (os *TinyOS) cmdChat(args []string) []shared.Message {
	// Überprüfen, ob eine SessionID im Argument übergeben wurde
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}
	// Prüfen, ob eine gültige Session vorhanden ist
	if sessionID == "" {
		return os.CreateWrappedTextMessage("", "Register and login to use the chat.")
	}
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		return os.CreateWrappedTextMessage(sessionID, "Register and login to use the chat.")
	}

	// Prüfen, ob es sich um einen Gast-Benutzer handelt
	if username == "guest" || strings.HasPrefix(username, "guest-") {
		return os.CreateWrappedTextMessage(sessionID, "Register and login to use the chat.")
	}

	// Prüfen, ob IP und/oder Benutzer gebannt sind
	os.sessionMutex.RLock()
	session, exists := os.sessions[sessionID]
	ipAddress := "127.0.0.1" // Standardwert
	if exists {
		ipAddress = session.IPAddress
	}
	os.sessionMutex.RUnlock()
	isBanned, banMessage := os.IsBanned(username, ipAddress)
	if isBanned {
		return os.CreateWrappedTextMessage(sessionID, banMessage)
	}

	// Chat-Zeitlimits prüfen
	isLimited, limitMsg := os.CheckChatTimeLimits(username)
	if isLimited {
		return os.CreateWrappedTextMessage(sessionID, limitMsg)
	}
	// Für Sondernachrichten wie Sound müssen wir weiterhin direkt shared.Message verwenden
	// und können diese dann mit der Textnachricht kombinieren
	messages := []shared.Message{
		{Type: shared.MessageTypeSound, Content: "floppy"},
		{Type: shared.MessageTypeChat, Content: "chat"}, // Neuer MessageType für Chat-Aktivierung
	}

	// Nachricht mit Text-Wrapping hinzufügen
	wrappedMsg := os.CreateWrappedTextMessage(sessionID, "Connecting to Master control program...")
	messages = append(messages, wrappedMsg...)
	wrappedMsg = os.CreateWrappedTextMessage(sessionID, "Ask your questions now. Enter 'exit' to return to the OS.")
	messages = append(messages, wrappedMsg...)
	return messages
}

// cmdChatHistory shows the chat history for the current session
func (os *TinyOS) cmdChatHistory(args []string) []shared.Message {
	// Check if a SessionID was passed as argument
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	// Check if we have a valid session
	if sessionID == "" {
		return os.CreateWrappedTextMessage("", "You must be logged in to view chat history.")
	}

	return os.ShowChatHistory(sessionID)
}

// cmdBasic starts the BASIC mode (mode switch for the frontend)
func (os *TinyOS) cmdBasic(args []string) []shared.Message {

	// Extract session ID if present
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	// Check session limit before starting BASIC session
	if sessionID != "" && !os.StartBasicSession(sessionID) {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: SessionLimitMessage},
		}
	}

	// Get username from session ID
	var username string
	if sessionID != "" {
		os.sessionMutex.RLock()
		session, exists := os.sessions[sessionID]
		if exists {
			username = session.Username
		}
		os.sessionMutex.RUnlock()
	} // For logged-in users
	if username != "" {
		// Ensure user VFS is initialized BEFORE syncing examples
		err := os.Vfs.InitializeUserVFS(username)
		if err != nil {
			logger.Error(logger.AreaAuth, "User VFS could not be initialized for %s: %v", username, err)
		}

		if err := os.syncExamplePrograms(username); err != nil {
			logger.Error(logger.AreaTerminal, "Example programs could not be copied: %v", err)
		}

		// Switch to basic directory for logged-in users
		if sessionID != "" {
			os.sessionMutex.Lock()
			if session, exists := os.sessions[sessionID]; exists {
				session.CurrentPath = "/home/" + username + "/basic"
				os.sessions[sessionID] = session
			}
			os.sessionMutex.Unlock()
		}
	} else {
		// FFor guest users - special handling for example files
		// **IMPORTANT**: Reinitialize guest VFS on each BASIC start
		// to ensure all example files are available

		// First clean up, then reinitialize
		err := os.Vfs.CleanupGuestVFS()
		if err != nil {
			logger.Error(logger.AreaTerminal, "Guest VFS could not be cleaned up: %v", err)
		}
		// Reinitialize with all example programs
		err = os.Vfs.InitializeGuestVFS()
		if err != nil {
			logger.Error(logger.AreaTerminal, "Guest VFS could not be initialized: %v", err)
		} else {
			// Set current path to basic directory for guest session
			if sessionID != "" {
				os.sessionMutex.Lock()
				if session, exists := os.sessions[sessionID]; exists {
					session.CurrentPath = "/home/guest/basic"
					os.sessions[sessionID] = session
				}
				os.sessionMutex.Unlock()
			}
		}
	}
	// Liste of program files (.bas and .sid) to display in BASIC mode
	var programFiles []string
	var err error
	if username == "" {
		// For guest users, list files from guest basic directory
		entries, listErr := os.Vfs.ListDir("/home/guest/basic")
		if listErr == nil {
			for _, entry := range entries {
				lowerEntry := strings.ToLower(entry)
				if strings.HasSuffix(lowerEntry, ".bas") || strings.HasSuffix(lowerEntry, ".sid") {
					programFiles = append(programFiles, entry)
				}
			}
		} else {
			logger.Error(logger.AreaTerminal, "Error listing guest program files: %v", listErr)
			// Fallback to example files if listing fails
			programFiles = []string{"hello.bas", "graphics.bas", "sound_demo.bas", "ull.sid"}
		}
	} else {
		// For logged-in users, list files from user's basic directory
		basicDir := "/home/" + username + "/basic"
		entries, listErr := os.Vfs.ListDir(basicDir)
		if listErr == nil {
			for _, entry := range entries {
				lowerEntry := strings.ToLower(entry)
				if strings.HasSuffix(lowerEntry, ".bas") || strings.HasSuffix(lowerEntry, ".sid") {
					programFiles = append(programFiles, entry)
				}
			}
		} else {
			logger.Error(logger.AreaTerminal, "Error listing program files for %s: %v", username, listErr)
			// Fallback to ListDirProgramFilesForUser method
			programFiles, err = os.ListDirProgramFilesForUser(username)
			if err != nil {
				logger.Error(logger.AreaTerminal, "Fallback error listing program files: %v", err)
			}
		}
	}
	basicFilesMsg := ""
	if len(programFiles) > 0 {
		basicFilesMsg = fmt.Sprintf("Available programs: %s", strings.Join(programFiles, ", "))
	}
	// Important: This message contains the mode switch command
	messages := []shared.Message{
		{Type: shared.MessageTypeMode, Content: "basic"}, // Signals frontend to switch to BASIC mode
		{Type: shared.MessageTypeSound, Content: "floppy"},
		{Type: shared.MessageTypeText, Content: "TinyBASIC v1.0"},
		{Type: shared.MessageTypeText, Content: "Ready"},
	}

	// Add message about available programs only if present
	if basicFilesMsg != "" {
		messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: basicFilesMsg})
	}

	// Empty line for better readability
	messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: ""})

	return messages
}

// cmdLimits shows current filesystem and resource limits
func (os *TinyOS) cmdLimits(args []string) []shared.Message {
	logger.Debug(logger.AreaTerminal, "cmdLimits called with args: %v", args)

	if len(args) == 0 {
		logger.Debug(logger.AreaTerminal, "No session ID provided in args")
		return os.CreateWrappedTextMessage("", "Error: No session ID provided")
	}

	sessionID := args[0]
	logger.Debug(logger.AreaTerminal, "Using sessionID: '%s'", sessionID)
	username := os.GetUsernameForSession(sessionID)
	logger.Debug(logger.AreaTerminal, "Username for session '%s': '%s'", sessionID, username)

	// For guests use "guest" as username
	if username == "" {
		username = "guest"
		logger.Debug(logger.AreaTerminal, "Using guest username for session '%s'", sessionID)
	}

	// Ensure that the Guest user is registered in the resource manager
	if username == "guest" {
		err := os.SystemResourceManager.RegisterUser(username)
		if err != nil && !strings.Contains(err.Error(), "already registered") {
			logger.Debug(logger.AreaTerminal, "Error registering guest user: %v", err)
		} else {
			logger.Debug(logger.AreaTerminal, "Guest user registered successfully")
		}

		// Also register the session - try to get the real IP address
		ipAddress := "127.0.0.1" // Default fallback
		os.sessionMutex.RLock()
		if session, exists := os.sessions[sessionID]; exists && session.IPAddress != "" {
			ipAddress = session.IPAddress
		}
		os.sessionMutex.RUnlock()

		err = os.ResourceManager.RegisterSession(sessionID, username, ipAddress)
		if err != nil && !strings.Contains(err.Error(), "already registered") {
			logger.Debug(logger.AreaTerminal, "Error registering guest session: %v", err)
		} else {
			logger.Debug(logger.AreaTerminal, "Guest session registered successfully with IP: %s", ipAddress)
		}
	}

	var response strings.Builder
	response.WriteString("=== YOUR LIMITS ===\n")
	response.WriteString(fmt.Sprintf("User: %s\n\n", username))
	// Ressourcenlimits vom Ressourcenmanager
	limits, err := os.SystemResourceManager.GetUserLimits(username)
	if err == nil {
		response.WriteString("SYSTEM RESOURCES:\n")
		response.WriteString(fmt.Sprintf("CPU Share: %.1f%%\n", limits.MaxCPUPercent))
		response.WriteString(fmt.Sprintf("Memory: %d MB\n", limits.MaxMemoryMB))
		response.WriteString(fmt.Sprintf("Program Runtime: %v\n", limits.MaxExecutionTime))
		response.WriteString("\n")

		response.WriteString("BASIC PROGRAM LIMITS:\n")
		response.WriteString(fmt.Sprintf("Max Commands: %d million\n", 20))       // 20 Millionen
		response.WriteString(fmt.Sprintf("Max Loop Iterations: %d million\n", 3)) // 3 Millionen
		response.WriteString("\n")
	}

	// Filesystem-Limits
	stats, err := os.Vfs.GetUserStats(username)
	if err == nil {
		response.WriteString("FILE SYSTEM:\n")
		response.WriteString(fmt.Sprintf("Directories: %d/%d\n", stats.DirectoryCount, stats.MaxDirectories))
		response.WriteString(fmt.Sprintf("Files per Directory: max %d\n", stats.MaxFilesPerDir))
		response.WriteString(fmt.Sprintf("File Size: max %.1f KB\n", float64(limits.MaxFileSize)/1024))
		response.WriteString(fmt.Sprintf("Home Directory: %d/%d files\n", stats.HomeDirectoryFiles, stats.MaxFilesPerDir))
		// Status-Warnungen
		if stats.DirectoryCount >= stats.MaxDirectories {
			response.WriteString("\n!!! Directory limit reached!\n")
		}
		if stats.HomeDirectoryFiles >= int(0.9*float64(stats.MaxFilesPerDir)) {
			response.WriteString("\n!!! Home directory nearly full!\n")
		}
	}

	// Warnung für Gast-Benutzer
	if username == "guest" {
		response.WriteString("\n*** Guest files are not saved permanently! ***")
		response.WriteString("\nRegister an account for persistent storage.")
	} else {
		response.WriteString("\nTip: Use 'resources' for detailed system info.")
	}

	return os.CreateWrappedTextMessage(sessionID, response.String())
}

// cmdEdit opens the full-screen editor
func (os *TinyOS) cmdEdit(args []string) []shared.Message {
	if len(args) < 1 {
		return os.CreateWrappedTextMessage("", "edit: session ID missing")
	}

	sessionID := args[0]

	// Editor is now available for all users including guests

	filename := ""
	if len(args) > 1 {
		filename = args[1]
	}

	// Start editor session
	editorManager := editor.GetEditorManager() // Create output channel for editor messages
	outputChan := make(chan shared.Message, 100)
	// Get terminal dimensions (default values, could be made configurable)
	rows := 24
	cols := 80
	config := editor.EditorConfig{
		Filename:   filename,
		Rows:       rows,
		Cols:       cols,
		SessionID:  sessionID,
		OutputChan: outputChan,
		VFS:        os.Vfs,
	}
	editorInstance := editorManager.StartEditor(config)

	// Start a goroutine to forward messages from the editor's output channel
	// to the main WebSocket connection via the SendToClientCallback.
	go func() {
		editorChan := editorInstance.GetOutputChannel()
		logger.Info(logger.AreaEditor, "Starting editor message forwarder for session %s", sessionID)

		for msg := range editorChan { // This loop exits when the channel is closed by editor.Close()
			if os.SendToClientCallback != nil {
				err := os.SendToClientCallback(sessionID, msg)
				if err != nil {
					logger.Warn(logger.AreaEditor, "Failed to forward editor message for session %s: %v. Client may be disconnected.", sessionID, err)
					return // Stop forwarding if the client connection is broken
				}
			}
		}
		logger.Info(logger.AreaEditor, "Editor message forwarder for session %s has shut down.", sessionID)
	}()

	// Set the authoritative input mode to Editor
	os.SetInputMode(sessionID, InputModeEditor)

	// The forwarder goroutine will handle sending messages from the editor.
	// We return nil so the shell doesn't print a prompt.
	return nil
}

// cmdView opens the full-screen editor in read-only mode (view only)
func (os *TinyOS) cmdView(args []string) []shared.Message {
	if len(args) < 1 {
		return os.CreateWrappedTextMessage("", "view: session ID missing")

	}

	sessionID := args[0]
	// View command requires a filename
	if len(args) < 2 {
		return os.CreateWrappedTextMessage("", "view: filename required\nUsage: view <filename>")
	}
	fileArg := args[1]

	// Get username for proper path resolution
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		username = "guest"
	}

	// Determine target path with proper resolution
	var targetPath string
	os.sessionMutex.RLock()
	session, exists := os.sessions[sessionID]
	os.sessionMutex.RUnlock()

	if exists {
		currentPath := session.CurrentPath
		// If absolute path, use it directly
		if filepath.IsAbs(fileArg) {
			targetPath = filepath.Clean(fileArg)
		} else {
			// Add relative path to current user's path
			targetPath = filepath.Join(currentPath, fileArg)
		}
		// Normalize to Unix format
		targetPath = strings.ReplaceAll(targetPath, "\\", "/")
	} else {
		// Fallback: Use user's home directory
		targetPath = filepath.Join("/home", username, fileArg)
		targetPath = strings.ReplaceAll(targetPath, "\\", "/")
	}

	// Check if file exists before starting editor (view only works with existing files)
	_, err := os.Vfs.ReadFile(targetPath, sessionID)
	if err != nil {
		return os.CreateWrappedTextMessage(sessionID, "File not found")
	}

	// Use the resolved target path as filename for the editor
	filename := targetPath

	// Start editor session in read-only mode
	editorManager := editor.GetEditorManager()
	outputChan := make(chan shared.Message, 100)

	// Get terminal dimensions (default values, could be made configurable)
	rows := 24
	cols := 80

	config := editor.EditorConfig{
		Filename:   filename,
		Rows:       rows,
		Cols:       cols,
		SessionID:  sessionID,
		OutputChan: outputChan,
		VFS:        os.Vfs,
		ReadOnly:   true, // Enable read-only mode
	}
	editorInstance := editorManager.StartEditor(config)

	// Start a goroutine to forward messages from the editor's output channel
	// to the main WebSocket connection via the SendToClientCallback.
	go func() {
		editorChan := editorInstance.GetOutputChannel()
		logger.Info(logger.AreaEditor, "Starting editor message forwarder for session %s (view mode)", sessionID)

		for msg := range editorChan { // This loop exits when the channel is closed by editor.Close()
			if os.SendToClientCallback != nil {
				err := os.SendToClientCallback(sessionID, msg)
				if err != nil {
					logger.Warn(logger.AreaEditor, "Failed to forward editor message for session %s: %v. Client may be disconnected.", sessionID, err)
					return // Stop forwarding if the client connection is broken
				}
			}
		}
		logger.Info(logger.AreaEditor, "Editor message forwarder for session %s (view mode) has shut down.", sessionID)
	}()

	// Set the authoritative input mode to Editor AFTER the editor instance is created and the forwarder is running.
	os.SetInputMode(sessionID, InputModeEditor)

	logger.Info(logger.AreaTerminal, "View command initiated for file: %s, ReadOnly mode enabled", filename)

	// We return nil so the shell doesn't print a prompt. The main input loop
	// will now delegate input to the editor because the mode has been set.
	return nil
}

// getTelnetOutput retrieves pending telnet output for a session
// cmdResources shows resource statistics and limits for the current user
func (os *TinyOS) cmdResources(args []string) []shared.Message {
	if len(args) < 1 {
		return os.CreateWrappedTextMessage("", "Error: Session ID required")
	}

	sessionID := args[0]

	// Check if session exists first
	os.sessionMutex.RLock()
	_, sessionExists := os.sessions[sessionID]
	os.sessionMutex.RUnlock()

	if !sessionExists {
		return os.CreateWrappedTextMessage("", "Error: Session not found")
	}

	username := os.GetUsernameForSession(sessionID)

	// For guests use "guest" as username
	if username == "" {
		username = "guest"
	}

	// Ensure that the user is registered in the resource manager (both guest and normal users)
	err := os.SystemResourceManager.RegisterUser(username)
	if err != nil && !strings.Contains(err.Error(), "already registered") {
		logger.ResourcesError("Error registering user '%s': %v", username, err)
	}

	// Also register the session - try to get the real IP address
	ipAddress := "127.0.0.1" // Default fallback
	os.sessionMutex.RLock()
	if session, exists := os.sessions[sessionID]; exists && session.IPAddress != "" {
		ipAddress = session.IPAddress
	}
	os.sessionMutex.RUnlock()
	err = os.ResourceManager.RegisterSession(sessionID, username, ipAddress)
	if err != nil && !strings.Contains(err.Error(), "already registered") {
		logger.ResourcesError("Error registering session '%s': %v", sessionID, err)
	}

	var content strings.Builder
	content.WriteString("=== SYSTEM RESOURCES ===\n")

	// Compact system statistics
	systemStats := os.SystemResourceManager.GetSystemStats()
	sessionStats := os.ResourceManager.GetSessionStats()

	content.WriteString(fmt.Sprintf("RAM: %v/%v MB | CPU: %v%% reserved | Sessions: %v | Users: %v\n",
		systemStats["current_ram_mb"], systemStats["total_ram_mb"],
		systemStats["cpu_reserved"], sessionStats["total_sessions"], sessionStats["unique_users"]))

	// User-specific limits (compact)
	limits, err := os.SystemResourceManager.GetUserLimits(username)
	if err != nil {
		content.WriteString(fmt.Sprintf("Error: %v\n", err))
	} else {
		content.WriteString(fmt.Sprintf("\nYour Limits (%s): CPU %.1f%% | RAM %d MB | Runtime %v\n",
			username, limits.MaxCPUPercent, limits.MaxMemoryMB, limits.MaxExecutionTime))
		content.WriteString(fmt.Sprintf("Files: %d max | Size: %.0f KB max\n",
			limits.MaxTotalFiles, float64(limits.MaxFileSize)/1024))
	}
	// Session info (compact)
	logger.ResourcesDebug("About to call GetSessionResource for sessionID: '%s'", sessionID)
	sessionResource, err := os.ResourceManager.GetSessionResource(sessionID)
	if err != nil {
		logger.ResourcesError("Error getting session resource: %v", err)
		// Add debug info about available sessions
		sessionStats := os.ResourceManager.GetSessionStats()
		logger.ResourcesDebug("Total sessions in ResourceManager: %v", sessionStats["total_sessions"])
		content.WriteString(fmt.Sprintf("Session Error: %v\n", err))
	} else {
		sessionDuration := time.Since(sessionResource.CreatedAt)

		// Format duration: show seconds if less than 1 minute, otherwise show minutes
		var durationStr string
		if sessionDuration < time.Minute {
			durationStr = fmt.Sprintf("%.0fs", sessionDuration.Seconds())
		} else {
			durationStr = sessionDuration.Round(time.Minute).String()
		}

		content.WriteString(fmt.Sprintf("\nSession: %s... | Duration: %s\n",
			sessionID[:8], durationStr))
	}

	// BASIC program info (compact, if available)
	basicStats, err := os.ResourceManager.GetBasicExecutionStats(username)
	if err == nil {
		content.WriteString(fmt.Sprintf("\nBASIC: %s | Runtime: %v ms | Cmds: %v/%v | Loops: %v/%v\n",
			basicStats["program"], basicStats["runtime_ms"],
			basicStats["commands"], basicStats["max_commands"],
			basicStats["loops"], basicStats["max_loops"]))
	}
	content.WriteString("\nTip: Resources auto-balance as users join/leave. Use 'limits' for details.")
	return os.CreateWrappedTextMessage(sessionID, content.String())
}

// cmdRun executes a BASIC program file directly from TinyOS
func (os *TinyOS) cmdRun(args []string) []shared.Message {
	logger.Info(logger.AreaTerminal, "cmdRun called with args: %v", args)

	if len(args) < 2 {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Usage: run <filename>"},
		}
	}

	sessionID := args[0]
	filename := args[1]

	// Validate session
	os.sessionMutex.RLock()
	session, exists := os.sessions[sessionID]
	os.sessionMutex.RUnlock()

	if !exists {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "No active session found."},
		}
	}

	// Add .bas extension if not present
	if !strings.Contains(filename, ".") {
		filename += ".bas"
	}

	// Validate that only .bas files are accepted
	if !strings.HasSuffix(strings.ToLower(filename), ".bas") {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Error: Only .bas files are supported."},
		}
	}

	// Check if file exists
	_, err := os.Vfs.ReadFile(filename, sessionID)
	if err != nil {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: fmt.Sprintf("Error: File '%s' not found.", filename)},
		}
	}

	// Get username for proper BASIC initialization
	username := session.Username

	// Check session limit before starting BASIC session
	if !os.StartBasicSession(sessionID) {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: SessionLimitMessage},
		}
	}

	// Initialize VFS for the user
	if username != "" {
		err := os.Vfs.InitializeUserVFS(username)
		if err != nil {
			logger.Error(logger.AreaTerminal, "User VFS could not be initialized for %s: %v", username, err)
		}

		if err := os.syncExamplePrograms(username); err != nil {
			logger.Error(logger.AreaTerminal, "Example programs could not be synced: %v", err)
		}

		// Switch to basic directory for logged-in users
		os.sessionMutex.Lock()
		if session, exists := os.sessions[sessionID]; exists {
			session.CurrentPath = "/home/" + username + "/basic"
			os.sessions[sessionID] = session
		}
		os.sessionMutex.Unlock()
	} else {
		// For guest users - reinitialize guest VFS
		err := os.Vfs.CleanupGuestVFS()
		if err != nil {
			logger.Error(logger.AreaTerminal, "Guest VFS could not be cleaned up: %v", err)
		}
		err = os.Vfs.InitializeGuestVFS()
		if err != nil {
			logger.Error(logger.AreaTerminal, "Guest VFS could not be initialized: %v", err)
		} else {
			// Set current path to basic directory for guest session
			os.sessionMutex.Lock()
			if session, exists := os.sessions[sessionID]; exists {
				session.CurrentPath = "/home/guest/basic"
				os.sessions[sessionID] = session
			}
			os.sessionMutex.Unlock()
		}
	} // Switch to BASIC mode with autorun parameter
	messages := []shared.Message{
		{Type: shared.MessageTypeMode, Content: "basic-autorun:" + filename}, // Special mode with autorun filename
		{Type: shared.MessageTypeSound, Content: "floppy"},
		{Type: shared.MessageTypeText, Content: "TinyBASIC v1.0"},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Loading and running %s...", filename)},
		{Type: shared.MessageTypeText, Content: ""}, // Empty line for better readability
	}

	return messages
}

// cmdChessFixed handles chess game commands with proper parameter handling
func (os *TinyOS) cmdChess(args []string, sessionID string) []shared.Message {
	logger.Info(logger.AreaChess, "Chess command called with args: %v, sessionID: %s", args, sessionID)

	// Get or create chess game session
	os.sessionMutex.Lock()
	session, exists := os.sessions[sessionID]
	if !exists {
		os.sessionMutex.Unlock()
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Error: No active session found",
		}}
	}

	// If chess game already exists, handle input directly
	if session.ChessGame != nil && session.ChessActive {
		chessGame := session.ChessGame
		os.sessionMutex.Unlock()

		// Handle chess input commands
		if len(args) > 0 {
			input := strings.Join(args, " ")
			switch strings.ToLower(args[0]) {
			case "quit", "exit":
				// End chess game
				os.sessionMutex.Lock()
				session.ChessGame = nil
				session.ChessActive = false
				os.sessions[sessionID] = session
				os.sessionMutex.Unlock()

				return []shared.Message{
					{Type: shared.MessageTypeClear},
					{Type: shared.MessageTypeText, Content: "Chess game ended. Back to TinyOS."},
				}

			case "help":
				return []shared.Message{
					{Type: shared.MessageTypeText, Content: "Chess Commands:"},
					{Type: shared.MessageTypeText, Content: "  move <from> <to> - Make a move (e.g., move e2 e4)"},
					{Type: shared.MessageTypeText, Content: "  help - Show this help"},
					{Type: shared.MessageTypeText, Content: "  quit - Exit chess game"},
					{Type: shared.MessageTypeText, Content: ""},
					{Type: shared.MessageTypeText, Content: "Notation: Use standard chess notation (a1-h8)"}}

			case "move":
				if len(args) < 3 {
					return []shared.Message{{
						Type:    shared.MessageTypeText,
						Content: "Usage: move <from> <to> (e.g., move e2 e4)",
					}}
				}

				from := args[1]
				to := args[2]
				moveInput := from + " " + to

				// Process the move using HandleInput
				return chessGame.HandleInput(moveInput)

			default:
				// Try to parse as move notation (e.g., "e2e4" or "e2 e4")
				if len(input) >= 4 {
					// Handle various move formats
					move := strings.ReplaceAll(input, " ", "")
					if len(move) >= 4 {
						moveInput := move[:2] + " " + move[2:4]
						return chessGame.HandleInput(moveInput)
					}
				}

				return []shared.Message{{
					Type:    shared.MessageTypeText,
					Content: "Unknown command. Type 'help' for available commands.",
				}}
			}
		} else {
			// No arguments - show current board
			return chessGame.RenderBoard()
		}
	}

	// Initialize new chess game
	// Default settings: Medium difficulty, player plays white
	difficulty := 2
	playerColor := chess.White

	// Parse arguments for difficulty and color (only when creating new game)
	for _, arg := range args {
		switch strings.ToLower(arg) {
		case "easy", "1":
			difficulty = 1
		case "medium", "2":
			difficulty = 2
		case "hard", "3":
			difficulty = 3
		case "black", "b":
			playerColor = chess.Black
		case "white", "w":
			playerColor = chess.White
		}
	}

	session.ChessGame = chess.NewChessUI(difficulty, playerColor)
	session.ChessActive = true
	os.sessions[sessionID] = session
	os.sessionMutex.Unlock() // Unlock after all session modifications

	logger.Info(logger.AreaChess, "New chess game started for session %s, difficulty: %d, player color: %d", sessionID, difficulty, int(playerColor))

	chessGame := session.ChessGame

	// Set the authoritative input mode to Chess
	// This message MUST be sent first to ensure the frontend is in the correct mode
	// before processing any UI rendering messages.
	messages := []shared.Message{
		{Type: shared.MessageTypeClear},
		{Type: shared.MessageTypeMode, Content: "CHESS"},
		{Type: shared.MessageTypeText, Content: "Welcome to TinyOS Chess!"},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Difficulty: %s", getDifficultyName(difficulty))},
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("You are playing as: %s", getColorName(playerColor))},
		{Type: shared.MessageTypeText, Content: ""},
	}

	// Add board rendering messages
	logger.Debug(logger.AreaChess, "Rendering initial chess board for new game")
	boardMessages := chessGame.RenderBoard()
	messages = append(messages, boardMessages...)
	logger.Debug(logger.AreaChess, "Chess command completed, returning %d messages", len(messages))

	return messages
}

// cmdDate displays the current date and time with year set to 1984
func (os *TinyOS) cmdDate(args []string) []shared.Message {
	// Extract sessionID from args if present
	sessionID := ""
	if len(args) > 0 && strings.HasPrefix(args[0], "session_") {
		sessionID = args[0]
	}
	// Get current time
	now := time.Now()

	// Create a new time with year 1984
	dateWith1984 := time.Date(1984, now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())

	// Format the date in a retro-style format (2006 represents the year in Go's time format)
	dateStr := dateWith1984.Format("Mon Jan 02 15:04:05 MST 2006")

	return os.CreateWrappedTextMessage(sessionID, dateStr)
}

// cmdAbout shows information about the terminal system
func (os *TinyOS) cmdAbout(args []string) []shared.Message {
	// Extract sessionID from args if present
	sessionID := ""
	if len(args) > 0 && strings.HasPrefix(args[0], "session_") {
		sessionID = args[0]
	}

	return os.CreateWrappedTextMessage(sessionID, "https://github.com/antibyte/retroterm")
}
