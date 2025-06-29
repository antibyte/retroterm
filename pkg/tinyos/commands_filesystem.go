package tinyos

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// cmdLs listet den Inhalt eines Verzeichnisses auf
func (os *TinyOS) cmdLs(args []string) []shared.Message {
	// Debug-Ausgaben für eingehende Argumente
	// SessionID mit der standardisierten Methode extrahieren
	sessionID, cleanArgs := os.ExtractSessionID(args)
	// Wenn keine gültige Session vorhanden ist
	if sessionID == "" {
		logger.Error(logger.AreaSession, "No valid session ID found in args=%v", args)
		return os.CreateWrappedTextMessage("", "No session id found.")
	}

	// Username aus Session abrufen
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		return os.CreateWrappedTextMessage(sessionID, "Error: Invalid session")
	}

	// Aktuellen Pfad aus der Session abrufen
	os.sessionMutex.RLock()
	session, exists := os.sessions[sessionID]
	currentPath := "/home/" + username // Default, falls nicht in der Session
	if exists {
		currentPath = session.CurrentPath
	} else {
		logger.Info(logger.AreaSession, "Session %s does not exist in sessions map", sessionID)
	}
	os.sessionMutex.RUnlock()

	// Ziel-Verzeichnis bestimmen
	targetPath := currentPath
	if len(cleanArgs) > 0 {
		targetArg := cleanArgs[0]

		// Absoluten oder relativen Pfad behandeln
		if filepath.IsAbs(targetArg) {
			targetPath = filepath.Clean(targetArg)
		} else {
			targetPath = filepath.Clean(filepath.Join(currentPath, targetArg))
		}

		// Normalisiere den Pfad auf Unix-Format
		targetPath = strings.ReplaceAll(targetPath, "\\", "/")
	}

	// Zugriffsprüfung: Benutzer darf nur auf /home und sein eigenes Home-Verzeichnis zugreifen
	userHome := "/home/" + username
	if targetPath != "/home" && !strings.HasPrefix(targetPath, userHome) {
		logger.Error(logger.AreaAuth, "Access denied to path: %s for user: %s", targetPath, username)

		return os.CreateWrappedTextMessage(sessionID, "Error: Access denied to "+targetPath)
	}

	// Hole Verzeichnisinhalt vom VFS
	entries, err := os.Vfs.ListDir(targetPath)
	if err != nil {
		return os.CreateWrappedTextMessage(sessionID, "Error: "+err.Error())
	}
	// Formatiere die Ausgabe mit aktueller Pfadanzeige
	var result strings.Builder

	// Zeige den aktuellen Pfad vor der Auflistung an
	result.WriteString(fmt.Sprintf("Current directory: %s\n", targetPath))

	// Simple output of files separated by spaces to enhance the retro feel
	// Max 5 files per line for better layout
	const filesPerLine = 5
	fileCount := 0

	for _, entry := range entries {
		result.WriteString(entry)
		fileCount++

		// After each entry a space or line break
		if fileCount%filesPerLine == 0 {
			result.WriteString("\n")
		} else {
			result.WriteString("  ") // Two spaces for better spacing
		}
	} // Add a final line break if the last line is not full
	if fileCount%filesPerLine != 0 {
		result.WriteString("\n")
	}

	// Add storage usage information
	if storageInfo, err := os.Vfs.GetUserStorageInfo(username); err == nil {
		result.WriteString(fmt.Sprintf("\n%d of %d KB used", storageInfo.UsedKB, storageInfo.TotalKB))
	}

	return os.CreateWrappedTextMessage(sessionID, result.String())
}

// cmdPwd shows the current working directory
func (os *TinyOS) cmdPwd(args []string) []shared.Message {
	var sessionID string
	if len(args) > 0 {
		sessionID = args[0]
	}

	// Get the current path from the session
	if sessionID != "" {
		os.sessionMutex.RLock()
		session, exists := os.sessions[sessionID]
		os.sessionMutex.RUnlock()
		if exists {
			return os.CreateWrappedTextMessage(sessionID, session.CurrentPath)
		} else {

		}
	}

	// Fallback for invalid session
	return os.CreateWrappedTextMessage("", "No session found.")
}

// cmdCd changes the current directory
func (os *TinyOS) cmdCd(args []string) []shared.Message {
	// Extract the session ID using the standardized method
	sessionID, cleanArgs := os.ExtractSessionID(args)
	// If no valid session exists
	if sessionID == "" {
		return os.CreateWrappedTextMessage("", "No session found.")
	}

	// Get username from session
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		return os.CreateWrappedTextMessage(sessionID, "Error: Invalid session")
	}

	// Get current path from session
	os.sessionMutex.RLock()
	session, exists := os.sessions[sessionID]
	currentPath := "/home/" + username // Default value if not found in session
	if exists {
		currentPath = session.CurrentPath
	}
	os.sessionMutex.RUnlock()
	// If no directory is specified, switch to home directory
	if len(cleanArgs) == 0 {
		newPath := "/home/" + username

		// Update the path in the session and database using the new method
		err := os.UpdateSessionPath(sessionID, newPath)
		if err != nil {
			return os.CreateWrappedTextMessage(sessionID, "Error: Could not update path")
		}

		// Send simple confirmation message
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: "Current path: " + newPath},
		}
	}

	targetDir := cleanArgs[0]
	var newPath string

	// Handle absolute or relative paths
	if strings.HasPrefix(targetDir, "/") {
		newPath = filepath.Clean(targetDir)
	} else {
		newPath = filepath.Clean(filepath.Join(currentPath, targetDir))
	}

	// Normalize the path to Unix format
	newPath = strings.ReplaceAll(newPath, "\\", "/")

	// DEBUG: Log the path resolution process
	// Check if the target directory exists
	isDir := os.Vfs.IsDir(newPath)

	if !isDir {
		return os.CreateWrappedTextMessage(sessionID, "Error: Directory not found")
	}

	// Access check: User may only access /home and their own home directory
	userHome := "/home/" + username
	if newPath != "/home" && !strings.HasPrefix(newPath, userHome) {
		return os.CreateWrappedTextMessage(sessionID, "Error: Access denied")
	}
	// Update the path in the session and database using the new method
	os.UpdateSessionPath(sessionID, newPath)

	// Send simple confirmation message
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: "Current path: " + newPath},
	}
}

// cmdMkdir creates a new directory
func (os *TinyOS) cmdMkdir(args []string) []shared.Message {
	if len(args) == 0 {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Usage: mkdir <directory_name>"}}
	}

	// Extract session ID if present
	var sessionID string
	var dirArg string
	var username string

	// Extract session ID if present (first parameter not starting with / or .)
	for i, arg := range args {
		if i == 0 && !strings.HasPrefix(arg, "/") && !strings.HasPrefix(arg, ".") {
			// First parameter may be a session ID
			sessionID = arg
			args = args[1:] // Remove session ID from arguments
			break
		}
	}
	// After extracting the session ID, the directory name must still be present
	if len(args) < 1 {
		return os.CreateWrappedTextMessage(sessionID, "Usage: mkdir <directory_name>")
	}

	dirArg = args[0]

	// Get username from session ID
	if sessionID != "" {
		username = os.GetUsernameForSession(sessionID)
	}

	// Determine target path
	var targetPath string

	if username != "" {
		// User is logged in - get current path from session
		os.sessionMutex.RLock()
		session, exists := os.sessions[sessionID]
		os.sessionMutex.RUnlock()

		if exists {
			currentPath := session.CurrentPath
			// If absolute path, use it directly
			if filepath.IsAbs(dirArg) {
				targetPath = filepath.Clean(dirArg)
			} else {
				// Add relative path to current user's path
				targetPath = filepath.Join(currentPath, dirArg)
			}
			// Normalize to Unix format
			targetPath = strings.ReplaceAll(targetPath, "\\", "/")
		} else {
			// Fallback: Use user's home directory
			targetPath = filepath.Join("/home", username, dirArg)
			targetPath = strings.ReplaceAll(targetPath, "\\", "/")
		}
	} else {
		// No user logged in, use guest user path
		targetPath = filepath.Join("/home/guest", dirArg)
		targetPath = strings.ReplaceAll(targetPath, "\\", "/")
	}

	err := os.Vfs.Mkdir(targetPath)
	if err != nil {
		return os.CreateWrappedTextMessage(sessionID, "Error: "+err.Error())
	}

	// Success - empty but formatted message
	return os.CreateWrappedTextMessage(sessionID, "")
}

// cmdCat shows the content of a file
func (os *TinyOS) cmdCat(args []string) []shared.Message {
	if len(args) == 0 {
		return os.CreateWrappedTextMessage("", "Usage: cat <filename>")
	} // Extract session ID using standardized method
	sessionID, cleanArgs := os.ExtractSessionID(args)
	if sessionID == "" {
		return os.CreateWrappedTextMessage("", "No session ID found.")
	}

	if len(cleanArgs) == 0 {
		return os.CreateWrappedTextMessage(sessionID, "Usage: cat <filename>")
	}

	fileArg := cleanArgs[0]
	username := os.GetUsernameForSession(sessionID)

	// Determine target path

	var targetPath string

	if username != "" {
		// User is logged in - get current path from session
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
	} else {
		// No user logged in, use guest user path
		targetPath = filepath.Join("/home/guest", fileArg)
		targetPath = strings.ReplaceAll(targetPath, "\\", "/")
	}

	content, err := os.Vfs.ReadFile(targetPath, sessionID)
	if err != nil {
		return os.CreateWrappedTextMessage(sessionID, "Error: "+err.Error())
	}

	// Normalize line breaks: Windows (\r\n) to Unix (\n)
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Remove any remaining \r (if present)
	content = strings.ReplaceAll(content, "\r", "\n")

	// Get terminal dimensions for proper line wrapping and pager formatting
	cols, rows := os.GetTerminalDimensions(sessionID)

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Apply automatic line wrapping based on terminal width
	lines = os.wrapLinesForTerminal(lines, cols)

	// Check if file is small enough to display all at once
	pageSize := 20

	logger.Debug(logger.AreaTerminal, "CAT COMMAND: Processing file %s for session %s, original_lines=%d, wrapped_lines=%d", fileArg, sessionID, len(strings.Split(content, "\n")), len(lines))

	if len(lines) <= pageSize {
		// Small file - display all content at once
		logger.Debug(logger.AreaTerminal, "CAT SMALL FILE: File %s has %d lines, showing all content", fileArg, len(lines))
		wrappedContent := strings.Join(lines, "\n")
		return os.CreateWrappedTextMessage(sessionID, wrappedContent)
	}
	// Large file - use pager
	logger.Debug(logger.AreaTerminal, "CAT PAGER INIT: Creating pager state for session %s, file %s, lines=%d",
		sessionID, fileArg, len(lines))

	// Initialize pager state
	pagerState := &CatPagerState{
		Lines:       lines,
		CurrentLine: 0,
		PageSize:    pageSize,
		Filename:    fileArg,
		CreatedAt:   time.Now(),
		Terminal:    TerminalDimensions{Cols: cols, Rows: rows},
	}

	os.catPagerMutex.Lock()
	os.catPagerStates[sessionID] = pagerState
	logger.Debug(logger.AreaTerminal, "CAT PAGER STORED: Session %s added to pager states, total states=%d",
		sessionID, len(os.catPagerStates))
	os.catPagerMutex.Unlock()

	// Clear screen and show first page
	messages := []shared.Message{
		{Type: shared.MessageTypeClear}, // Clear screen properly
	}

	// Show first page manually (lines 0 to pageSize-1)
	firstPageLines := lines[0:pageSize]
	firstPageContent := strings.Join(firstPageLines, "\n")

	// Update CurrentLine to point to the first line of NEXT page
	pagerState.CurrentLine = pageSize

	// Add first page content and pager status
	basePrompt := "--- " + fileArg + " --- m: more, q: quit"
	prompt := basePrompt
	if cols > 0 && cols <= 120 {
		promptLen := len(basePrompt)
		if promptLen < cols {
			maxPadding := cols - promptLen
			if maxPadding > 50 {
				maxPadding = 50
			}
			padding := strings.Repeat(" ", maxPadding)
			prompt = basePrompt + padding
		}
	}

	messages = append(messages, []shared.Message{
		{Type: shared.MessageTypeText, Content: firstPageContent},
		{Type: shared.MessageTypeEditor, EditorCommand: "status", EditorStatus: prompt},
		{Type: shared.MessageTypePager, Content: "activate"},
	}...)

	return messages
}

// cmdWrite writes text to a file (overwrites existing)
func (os *TinyOS) cmdWrite(args []string) []shared.Message {
	if len(args) < 2 {
		return os.CreateWrappedTextMessage("", "Usage: write <filename> <content...>")
	}

	// Extract session ID if present
	var sessionID string
	var fileArg string
	var content string

	// Extract session ID if present (first parameter not starting with / or .)
	for i, arg := range args {
		if i == 0 && !strings.HasPrefix(arg, "/") && !strings.HasPrefix(arg, ".") {
			// First parameter may be a session ID
			sessionID = arg
			args = args[1:] // Remove session ID from arguments
			break
		}
	}

	// Get username from session ID
	username := os.GetUsernameForSession(sessionID)

	// After extracting the session ID, the file name and content must still be present
	if len(args) < 2 {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Usage: write <filename> <content...>"}}
	}

	fileArg = args[0]
	content = strings.Join(args[1:], " ")

	// Determine target path
	var targetPath string

	if username != "" {
		// User is logged in - get current path from session
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
	} else {
		// No user logged in, use guest user path
		targetPath = filepath.Join("/home/guest", fileArg)
		targetPath = strings.ReplaceAll(targetPath, "\\", "/")
	}

	err := os.Vfs.WriteFile(targetPath, content, sessionID)
	if err != nil {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Error: " + err.Error()}}
	}

	return []shared.Message{} // No output on success
}

// cmdRm deletes a file or empty directory
func (os *TinyOS) cmdRm(args []string) []shared.Message {
	if len(args) == 0 {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Usage: rm <file/directory>"}}
	}

	// Extract session ID if present
	var sessionID string
	var fileArg string
	var username string

	// Extract session ID if present (first parameter not starting with / or .)
	for i, arg := range args {
		if i == 0 && !strings.HasPrefix(arg, "/") && !strings.HasPrefix(arg, ".") {
			// First parameter may be a session ID
			sessionID = arg
			args = args[1:] // Remove session ID from arguments
			break
		}
	}

	// After extracting the session ID, the file/directory name must still be present
	if len(args) < 1 {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Usage: rm <file/directory>"}}
	}

	fileArg = args[0]

	// Get username from session ID
	if sessionID != "" {
		username = os.GetUsernameForSession(sessionID)

	}

	// Determine target path
	var targetPath string

	if username != "" {
		// User is logged in - get current path from session
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
	} else {
		// No user logged in, use guest user path
		targetPath = filepath.Join("/home/guest", fileArg)
		targetPath = strings.ReplaceAll(targetPath, "\\", "/")

	}

	// Security measures - certain directories must not be deleted
	if targetPath == "/home" || (username != "" && targetPath == "/home/"+username) {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Error: Cannot delete this directory."}}
	}

	// Check if the user has write access
	if username != "" {
		// Logged-in user may only delete within their home directory
		if !strings.HasPrefix(targetPath, "/home/"+username) {
			return []shared.Message{{Type: shared.MessageTypeText, Content: "Error: No permission for this operation."}}
		}
	}

	err := os.Vfs.Remove(targetPath)
	if err != nil {
		return []shared.Message{{Type: shared.MessageTypeText, Content: "Error: " + err.Error()}}
	}

	return []shared.Message{} // No output on success
}

// ListDirBasFilesForUser returns a list of .bas files for the specified user.
// DEPRECATED: Use ListDirProgramFilesForUser instead
func (os *TinyOS) ListDirBasFilesForUser(username string) ([]string, error) {
	return os.ListDirProgramFilesForUser(username)
}

// wrapLinesForTerminal wraps long lines to fit within terminal width
func (os *TinyOS) wrapLinesForTerminal(lines []string, terminalWidth int) []string {
	if terminalWidth <= 0 {
		terminalWidth = 80 // Default terminal width
	}

	var wrappedLines []string

	for _, line := range lines {
		if len(line) <= terminalWidth {
			// Line fits within terminal width
			wrappedLines = append(wrappedLines, line)
		} else {
			// Line needs to be wrapped
			for len(line) > 0 {
				if len(line) <= terminalWidth {
					// Remaining part fits
					wrappedLines = append(wrappedLines, line)
					break
				}

				// Find best break point (prefer space near end of line)
				breakPoint := terminalWidth

				// Look for a space near the end to break at word boundary
				for i := terminalWidth - 1; i >= terminalWidth*3/4 && i > 0; i-- {
					if line[i] == ' ' {
						breakPoint = i
						break
					}
				}

				// Add the wrapped portion
				wrappedLines = append(wrappedLines, line[:breakPoint])

				// Continue with remaining text (skip space if we broke at a space)
				if breakPoint < len(line) && line[breakPoint] == ' ' {
					line = line[breakPoint+1:]
				} else {
					line = line[breakPoint:]
				}
			}
		}
	}

	return wrappedLines
}
