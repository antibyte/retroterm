package editor

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
	"github.com/antibyte/retroterm/pkg/virtualfs"
)

// Editor limits - werden jetzt aus der Konfiguration gelesen
// Siehe [Editor] Sektion in settings.cfg

// EditorManager manages active editor sessions
type EditorManager struct {
	editors map[string]*Editor // sessionID -> Editor
	mu      sync.RWMutex
}

// Global editor manager instance
var globalEditorManager = &EditorManager{
	editors: make(map[string]*Editor),
}

// GetEditorManager returns the global editor manager instance
func GetEditorManager() *EditorManager {
	return globalEditorManager
}

// StartEditor creates and starts a new editor session
func (em *EditorManager) StartEditor(config EditorConfig) *Editor {
	logger.Info(logger.AreaEditor, "StartEditor called for session: %s, filename: %s", config.SessionID, config.Filename)
	em.mu.Lock()
	defer em.mu.Unlock()

	// Close existing editor for this session if any
	if existingEditor, exists := em.editors[config.SessionID]; exists {
		logger.Info(logger.AreaEditor, "Closing existing editor for session: %s", config.SessionID)
		existingEditor.Close()
	}
	// Create new editor
	editor := NewEditor(config)
	em.editors[config.SessionID] = editor
	logger.Info(logger.AreaEditor, "Created new editor for session: %s", config.SessionID)
	// CRITICAL FIX: Always load file BEFORE starting if filename is specified
	// This ensures that the first render contains the correct content, not empty/raw lines
	if config.Filename != "" {
		logger.Info(logger.AreaEditor, "Loading file before start: %s", config.Filename)
		err := editor.LoadFile(config.Filename)
		if err != nil {
			logger.Error(logger.AreaEditor, "Error loading file %s: %v", config.Filename, err)
		} else {
			logger.Info(logger.AreaEditor, "File loaded successfully: %s (%d lines)", config.Filename, len(editor.lines))
		}
	}

	// Send initial editor state to client (after content is loaded)
	editor.Start()
	logger.Info(logger.AreaEditor, "Editor started for session: %s", config.SessionID)

	return editor
}

// GetEditor returns the editor for a session, if any
func (em *EditorManager) GetEditor(sessionID string) *Editor {
	em.mu.RLock()
	defer em.mu.RUnlock()

	logger.Debug(logger.AreaEditor, "GetEditor called for session: %s", sessionID)
	logger.Debug(logger.AreaEditor, "Available editors: %d", len(em.editors))
	for sid := range em.editors {
		logger.Debug(logger.AreaEditor, "- Session: %s", sid)
	}

	editor := em.editors[sessionID]
	if editor != nil {
		logger.Debug(logger.AreaEditor, "Found editor for session: %s", sessionID)
	} else {
		logger.Debug(logger.AreaEditor, "No editor found for session: %s", sessionID)
	}

	return editor
}

// CloseEditor closes and removes an editor session
func (em *EditorManager) CloseEditor(sessionID string) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if editor, exists := em.editors[sessionID]; exists {
		editor.Close()
		delete(em.editors, sessionID)
	}
}

// ProcessEditorInput processes input for an active editor session
func (em *EditorManager) ProcessEditorInput(sessionID string, input string) bool {
	em.mu.RLock()
	editor := em.editors[sessionID]
	em.mu.RUnlock()

	if editor == nil {
		return false // No active editor
	}

	return editor.ProcessInput(input)
}

// Editor represents a full-screen text editor instance
type Editor struct {
	// Core editor state
	lines    []string // Text content, each line as a string
	cursorX  int      // Current column position (0-based)
	cursorY  int      // Current line position (0-based)
	scrollY  int      // Vertical scroll position (0-based)
	modified bool     // Whether content has been modified since last save
	filename string   // Current filename (empty for new file)
	readOnly bool     // If true, editor is in read-only mode (view mode)

	// Word wrap support
	wrappedLines []string // Display lines with word wrapping applied
	lineMapping  []int    // Maps wrapped line index to original line index

	// Display dimensions
	rows      int // Terminal rows (height)
	cols      int // Terminal columns (width)
	statusRow int // Row index for status line (typically rows-1)
	textRows  int // Available rows for text content (rows-1)
	// Communication
	outputChan chan shared.Message // Channel to send messages to client
	sessionID  string              // Session ID for this editor instance
	// Virtual file system access
	vfs                *virtualfs.VFS // Editor state
	active             bool           // Whether editor is currently active
	lastActivity       time.Time
	showingExitWarning bool // Whether currently showing exit warning (ESC to cancel)
	// Filename input state
	requestingFilename bool   // Whether currently requesting filename input
	filenameInput      string // Current filename being typed
	exitAfterSave      bool   // Whether to exit after successful save

	// Channel to wait for frontend ready signal
	readyChan chan bool

	// Mapping information for wrapped lines
	wrappedLineMap []WrappedLineInfo
}

// WrappedLineInfo holds information about a wrapped line segment
type WrappedLineInfo struct {
	logicalLine  int // Corresponding logical line number
	segmentIndex int // Index of the segment within the logical line
}

// EditorConfig holds configuration for creating a new editor
type EditorConfig struct {
	Filename   string
	Rows       int
	Cols       int
	SessionID  string
	OutputChan chan shared.Message
	VFS        *virtualfs.VFS
	ReadOnly   bool // If true, editor will be read-only (view mode)
}

// NewEditor creates a new editor instance
func NewEditor(config EditorConfig) *Editor {
	editor := &Editor{
		lines:        []string{""}, // Start with one empty line
		cursorX:      0,
		cursorY:      0,
		scrollY:      0,
		modified:     false,
		filename:     config.Filename,
		readOnly:     config.ReadOnly,
		rows:         config.Rows,
		cols:         config.Cols,
		statusRow:    config.Rows - 1,
		textRows:     config.Rows - 1,
		outputChan:   config.OutputChan,
		sessionID:    config.SessionID,
		vfs:          config.VFS,
		active:       true,
		lastActivity: time.Now(),
	}

	// Initialize wrapped lines immediately - essential for ReadOnly mode
	editor.updateWrappedLines()
	logger.Info(logger.AreaEditor, "NewEditor: Initialized with readOnly=%v, wrapped lines: %d", editor.readOnly, len(editor.wrappedLines))

	// Note: File loading is now handled asynchronously after Start() is called
	// to ensure proper message ordering

	return editor
}

// LoadFile loads content from a file

// ProcessInput handles keyboard input for the editor
func (e *Editor) ProcessInput(input string) bool {
	// CRITICAL: If editor is not active, ignore all input.
	// This prevents race conditions where input arrives after the editor has been closed.
	if !e.active {
		logger.Warn(logger.AreaEditor, "ProcessInput called on inactive editor for session %s. Input '%s' ignored.", e.sessionID, input)
		return false // Stop processing immediately
	}

	log.Printf("[EDITOR-BACKEND] ProcessInput called with input: %q, readOnly: %v", input, e.readOnly)
	logger.Info(logger.AreaEditor, "ProcessInput called with input: %q, readOnly: %v", input, e.readOnly)
	// IMMEDIATE BLOCK FOR READONLY - before any other processing
	if e.readOnly {
		logger.Info(logger.AreaEditor, "READONLY MODE: Blocking all input except navigation")
		logger.Debug(logger.AreaEditor, "READONLY: Processing input %q", input)
		// Only allow navigation and exit in read-only mode
		switch input {
		case "ArrowUp":
			logger.Info(logger.AreaEditor, "READONLY: Scrolling up")
			if e.scrollY > 0 {
				e.scrollY--
				logger.Debug(logger.AreaEditor, "READONLY: ScrollY now: %d", e.scrollY)
				e.Render() // Force immediate update to frontend
			}
		case "ArrowDown":
			logger.Info(logger.AreaEditor, "READONLY: Scrolling down")
			if e.scrollY < len(e.wrappedLines)-e.textRows {
				e.scrollY++
				logger.Debug(logger.AreaEditor, "READONLY: ScrollY now: %d", e.scrollY)
				e.Render() // Force immediate update to frontend
			}
		case "PageUp":
			logger.Info(logger.AreaEditor, "READONLY: Page up")
			oldScrollY := e.scrollY
			e.scrollY -= e.textRows
			if e.scrollY < 0 {
				e.scrollY = 0
			}
			if oldScrollY != e.scrollY {
				logger.Debug(logger.AreaEditor, "READONLY: PageUp scrollY: %d -> %d", oldScrollY, e.scrollY)
				e.Render() // Force immediate update to frontend
			}
		case "PageDown":
			logger.Info(logger.AreaEditor, "READONLY: Page down")
			oldScrollY := e.scrollY
			logger.Debug(logger.AreaEditor, "READONLY: PageDown - wrappedLines=%d, textRows=%d, scrollY=%d", len(e.wrappedLines), e.textRows, e.scrollY)
			e.scrollY += e.textRows
			if e.scrollY > len(e.wrappedLines)-e.textRows {
				e.scrollY = len(e.wrappedLines) - e.textRows
				if e.scrollY < 0 {
					e.scrollY = 0
				}
			}
			if oldScrollY != e.scrollY {
				logger.Debug(logger.AreaEditor, "READONLY: PageDown scrollY: %d -> %d", oldScrollY, e.scrollY)
				logger.Debug(logger.AreaEditor, "READONLY: DEBUG - About to call Render() from PageDown")
				e.Render() // Force immediate update to frontend
			} else {
				logger.Debug(logger.AreaEditor, "READONLY: PageDown - no scroll change, already at limit")
			}
		case "CTRL+X", "Escape":
			logger.Info(logger.AreaEditor, "READONLY: Exiting")
			return e.handleExit()
		default:
			logger.Info(logger.AreaEditor, "READONLY: Ignoring input %q", input)
		}
		// All navigation in ReadOnly mode is handled above with immediate Render() calls
		return true
	}

	logger.Debug(logger.AreaEditor, "ProcessInput called with input: %q, showingExitWarning: %v, readOnly: %v", input, e.showingExitWarning, e.readOnly)
	e.lastActivity = time.Now()

	// DEBUG: Log current editor state
	logger.Debug(logger.AreaEditor, "Editor state: readOnly=%v, scrollY=%d, totalLines=%d, textRows=%d", e.readOnly, e.scrollY, len(e.lines), e.textRows)

	// Special handling when requesting filename
	if e.requestingFilename {
		return e.handleFilenameInput(input)
	}
	// Special handling when exit warning is showing
	if e.showingExitWarning {
		log.Printf("[EDITOR-BACKEND] Exit warning active, processing input: %q", input)
		logger.Info(logger.AreaEditor, "ProcessInput: Exit warning active, processing input: %q", input)
		switch input {
		case "Escape":
			// Cancel exit warning with ESC
			log.Printf("[EDITOR-BACKEND] ESC pressed - canceling exit warning")
			logger.Info(logger.AreaEditor, "ProcessInput: ESC pressed - canceling exit warning")
			return e.handleCancelExit()
		case "CTRL+S":
			// Save and exit when exit warning is showing
			log.Printf("[EDITOR-BACKEND] CTRL+S pressed during exit warning - saving and exiting")
			e.showingExitWarning = false // Clear exit warning first

			// Try to save the file
			if e.filename == "" {
				e.sendStatusMessage("Error: No filename specified")
				return true
			}

			// Sofortiges Feedback für den Benutzer
			e.sendStatusMessage("Saving file...")

			// Synchrones Speichern mit Timeout-Schutz
			log.Printf("[EDITOR-BACKEND] Starting file save for: %s", e.filename)

			done := make(chan error, 1)
			go func() {
				done <- e.SaveFile()
			}()

			select {
			case err := <-done:
				if err != nil {
					log.Printf("[EDITOR-BACKEND] Error saving file: %v", err)
					e.sendStatusMessage("Error saving: " + err.Error())
					return true // Stay in editor if save failed
				} else {
					log.Printf("[EDITOR-BACKEND] File saved successfully, now exiting editor")
					e.sendStatusMessage("File saved: " + e.filename)
					// Save was successful, now exit
					e.sendEditorMessage("stop", nil)
					e.sendMessage(shared.MessageTypeInputControl, "", map[string]interface{}{
						"inputEnabled": true,
					})
					e.active = false
					return false
				}
			case <-time.After(5 * time.Second):
				log.Printf("[EDITOR-BACKEND] Save operation timed out for: %s", e.filename)
				e.sendStatusMessage("Save timeout - try again")
				return true // Stay in editor if save timed out
			}
		case "CTRL+C", "CTRL+c":
			// Force exit without saving when exit warning is showing
			log.Printf("[EDITOR-BACKEND] CTRL+C pressed during exit warning - forcing exit without save")
			logger.Info(logger.AreaEditor, "ProcessInput: CTRL+C pressed - forcing exit without save")

			// Send stop command to frontend
			e.sendEditorMessage("stop", nil)
			e.sendMessage(shared.MessageTypeInputControl, "", map[string]interface{}{
				"inputEnabled": true,
			})
			e.active = false
			return false
		default:
			// Any other key cancels the exit warning
			log.Printf("[EDITOR-BACKEND] Other key pressed (%q) - canceling exit warning", input)
			return e.handleCancelExit()
		}
	} // End of if e.showingExitWarning
	// Handle special key combinations first (but after exit warning handling)
	if strings.HasPrefix(input, "CTRL+") {
		return e.handleControlKey(input[5:])
	} // Handle special keys - ONLY IN EDIT MODE (ReadOnly handled at top of function)
	if !e.readOnly {
		switch input {
		case "ArrowUp":
			logger.Info(logger.AreaEditor, "ProcessInput: ArrowUp detected, calling moveCursor(0, -1)")
			e.moveCursor(0, -1)
		case "ArrowDown":
			logger.Info(logger.AreaEditor, "ProcessInput: ArrowDown detected, calling moveCursor(0, 1)")
			e.moveCursor(0, 1)
		case "ArrowLeft":
			e.moveCursor(-1, 0)
		case "ArrowRight":
			e.moveCursor(1, 0)
		case "Home":
			e.cursorX = 0
		case "End":
			if e.cursorY < len(e.lines) {
				e.cursorX = len(e.lines[e.cursorY])
			}
		case "PageUp":
			e.moveCursor(0, -e.textRows)
		case "PageDown":
			e.moveCursor(0, e.textRows)
		case "Backspace":
			e.handleBackspace()
		case "Delete":
			e.handleDelete()
		case "Enter":
			e.handleEnter()
		case "Tab":
			e.insertText("    ") // Insert 4 spaces for tab
		case "Escape":
			// ESC key - currently does nothing in normal editor mode
			// Could be used for future features like command mode
			break
		default:
			// Regular character input
			if len(input) == 1 {
				e.insertText(input)
			}
		}
	}

	// Update scroll position to keep cursor visible
	e.adjustScroll()

	// Redraw the editor
	e.draw()

	return true // Continue processing
}

// handleControlKey processes control key combinations

// handleSave saves the current file

// handleExit exits the editor

// handleCancelExit cancels the exit warning and returns to normal editor mode
// handleFilenameInput handles input when requesting a filename

// ProcessEditorMessage verarbeitet Editor-spezifische Kommandos
func (e *Editor) ProcessEditorMessage(command, data string) bool {
	// CRITICAL: If editor is not active, ignore all messages.
	// This prevents race conditions where messages arrive after the editor has been closed.
	if !e.active {
		logger.Warn(logger.AreaEditor, "ProcessEditorMessage called on inactive editor for session %s. Command '%s' ignored.", e.sessionID, command)
		return false // Stop processing immediately
	}

	logger.Debug(logger.AreaEditor, "ProcessEditorMessage: command=%s, data=%s, readOnly=%v, showingExitWarning=%v, requestingFilename=%v", command, data, e.readOnly, e.showingExitWarning, e.requestingFilename)
	e.lastActivity = time.Now() // Block all editing commands in read-only mode (except exit commands)
	if e.readOnly {
		switch command {
		case "exit":
			// Always allow exit
		case "key_input":
			// Allow key_input but log what keys are being sent
			logger.Info(logger.AreaEditor, "ReadOnly mode: allowing key_input: %q", data)
			// Allow all navigation keys - let ProcessInput handle the filtering
			// This allows Arrow keys, Page Up/Down, Home, End, etc.
		case "cursor_up":
			// ReadOnly navigation: scroll up
			logger.Info(logger.AreaEditor, "READONLY: Scrolling up")
			if e.scrollY > 0 {
				e.scrollY--
				logger.Debug(logger.AreaEditor, "READONLY: ScrollY now: %d", e.scrollY)
				e.Render()
			}
			return true
		case "cursor_down":
			// ReadOnly navigation: scroll down
			logger.Info(logger.AreaEditor, "READONLY: Scrolling down")
			if e.scrollY < len(e.wrappedLines)-e.textRows {
				e.scrollY++
				logger.Debug(logger.AreaEditor, "READONLY: ScrollY now: %d", e.scrollY)
				e.Render()
			}
			return true
		case "page_up":
			// ReadOnly navigation: page up
			logger.Info(logger.AreaEditor, "READONLY: Page up")
			oldScrollY := e.scrollY
			e.scrollY -= e.textRows
			if e.scrollY < 0 {
				e.scrollY = 0
			}
			if oldScrollY != e.scrollY {
				logger.Debug(logger.AreaEditor, "READONLY: PageUp scrollY: %d -> %d", oldScrollY, e.scrollY)
				e.Render()
			}
			return true
		case "page_down":
			// ReadOnly navigation: page down
			logger.Info(logger.AreaEditor, "READONLY: Page down")
			oldScrollY := e.scrollY
			e.scrollY += e.textRows
			if e.scrollY > len(e.wrappedLines)-e.textRows {
				e.scrollY = len(e.wrappedLines) - e.textRows
				if e.scrollY < 0 {
					e.scrollY = 0
				}
			}
			if oldScrollY != e.scrollY {
				logger.Debug(logger.AreaEditor, "READONLY: PageDown scrollY: %d -> %d", oldScrollY, e.scrollY)
				e.Render()
			}
			return true
		case "cursor_home_document":
			// ReadOnly navigation: scroll to top
			logger.Info(logger.AreaEditor, "READONLY: Scroll to top")
			e.scrollY = 0
			e.Render()
			return true
		case "cursor_end_document": // ReadOnly navigation: scroll to bottom
			logger.Info(logger.AreaEditor, "READONLY: Scroll to bottom")
			e.scrollY = len(e.wrappedLines) - e.textRows
			if e.scrollY < 0 {
				e.scrollY = 0
			}
			e.Render()
			return true
		case "ready":
			// Frontend signals it's ready to receive render message - allow in ReadOnly mode
			logger.Info(logger.AreaEditor, "READONLY: Received frontend ready signal")
			// Jetzt die öffentliche Ready() Methode verwenden
			e.Ready()
			return true
		default:
			logger.Info(logger.AreaEditor, "ReadOnly mode: blocking editor command: %s", command)
			return true
		}
	}
	// Special handling when requesting filename
	if e.requestingFilename {
		switch command {
		case "backspace":
			return e.handleFilenameInput("Backspace")
		case "char_input":
			return e.handleFilenameInput(data)
		case "key_input":
			return e.handleFilenameInput(data)
		case "filename_submit":
			return e.handleFilenameSubmit(data)
		case "filename_cancel":
			return e.handleFilenameCancel()
		default:
			// For other commands during filename input, ignore them
			log.Printf("[EDITOR-BACKEND] Ignoring command %s during filename input", command)
			return true
		}
	}
	switch command {
	case "ready":
		// Frontend signals it's ready to receive render message
		logger.Info(logger.AreaEditor, "Received frontend ready signal")
		if e.readyChan != nil {
			select {
			case e.readyChan <- true:
				logger.Info(logger.AreaEditor, "Sent ready signal to waiting goroutine")
			default:
				logger.Warn(logger.AreaEditor, "Ready channel is full or no one waiting")
			}
		} else {
			logger.Warn(logger.AreaEditor, "Ready signal received but no ready channel available")
		}
		return true
	case "save": // Check if exit warning is active - if so, save and exit
		if e.showingExitWarning {
			log.Printf("[EDITOR-BACKEND] Save command during exit warning - saving and exiting")

			// Try to save the file - but if no filename, request filename input first
			if e.filename == "" || e.filename == "<current program>" || e.filename == "<new file>" {
				log.Printf("[EDITOR-BACKEND] No real filename during exit save (filename='%s'), requesting filename input", e.filename)
				e.requestingFilename = true
				e.filenameInput = ""
				e.exitAfterSave = true // Remember to exit after save
				// Note: Keep showingExitWarning=true until save is complete

				// Send filename input command to frontend
				e.sendEditorMessage("filename_input", map[string]interface{}{
					"prompt": "Save as: ",
				})
				return true
			}

			e.showingExitWarning = false // Clear exit warning only when we have a filename
			// Sofortiges Feedback für den Benutzer
			e.sendStatusMessage("Saving file...")

			// Synchrones Speichern mit Timeout-Schutz
			log.Printf("[EDITOR-BACKEND] Starting file save for: %s", e.filename)

			done := make(chan error, 1)
			go func() {
				done <- e.SaveFile()
			}()

			select {
			case err := <-done:
				if err != nil {
					log.Printf("[EDITOR-BACKEND] Error saving file: %v", err)
					e.sendStatusMessage("Error saving: " + err.Error())
					return true // Stay in editor if save failed
				} else {
					log.Printf("[EDITOR-BACKEND] File saved successfully, now exiting editor")
					e.sendStatusMessage("File saved: " + e.filename)
					// Save was successful, now exit
					e.sendEditorMessage("stop", nil)
					e.sendMessage(shared.MessageTypeInputControl, "", map[string]interface{}{
						"inputEnabled": true,
					})
					e.active = false
					return false
				}
			case <-time.After(5 * time.Second):
				log.Printf("[EDITOR-BACKEND] Save operation timed out for: %s", e.filename)
				e.sendStatusMessage("Save timeout - try again")
				return true // Stay in editor if save timed out
			}
		} else {
			// Normal save without exit
			return e.handleSave()
		}
	case "exit":
		return e.handleExit()
	case "force_exit":
		return e.handleForceExit()
	case "cancel_exit":
		return e.handleCancelExit()
	case "filename_submit":
		return e.handleFilenameSubmit(data)
	case "filename_cancel":
		return e.handleFilenameCancel()
	case "open":
		return e.handleOpen(data)
	case "cursor_up":
		e.moveCursor(0, -1)
		e.adjustScroll()
		e.Render()
	case "cursor_down":
		e.moveCursor(0, 1)
		e.adjustScroll()
		e.Render()
	case "cursor_left":
		e.moveCursor(-1, 0)
		e.adjustScroll()
		e.Render()
	case "cursor_right":
		e.moveCursor(1, 0)
		e.adjustScroll()
		e.Render()
	case "cursor_home_line":
		e.cursorX = 0
		e.Render()
	case "cursor_end_line":
		if e.cursorY < len(e.lines) {
			e.cursorX = len(e.lines[e.cursorY])
		}
		e.Render()
	case "cursor_home_document":
		e.cursorX = 0
		e.cursorY = 0
		e.scrollY = 0
		e.Render()
	case "cursor_end_document":
		if len(e.lines) > 0 {
			e.cursorY = len(e.lines) - 1
			e.cursorX = len(e.lines[e.cursorY])
		}
		e.adjustScroll()
		e.Render()
	case "page_up":
		e.cursorY -= e.textRows
		if e.cursorY < 0 {
			e.cursorY = 0
		}
		e.adjustScroll()
		e.Render()
	case "page_down":
		e.cursorY += e.textRows
		if e.cursorY >= len(e.lines) {
			e.cursorY = len(e.lines) - 1
		}
		if e.cursorY < 0 {
			e.cursorY = 0
		}
		e.adjustScroll()
		e.Render()
	case "backspace":
		e.handleBackspace()
		e.Render()
	case "delete":
		e.handleDelete()
		e.Render()
	case "insert_newline":
		e.handleInsertNewline()
		e.Render()
	case "insert_tab":
		e.handleInsertChar('\t')
		e.Render()
	case "insert_char":
		log.Printf("[EDITOR-BACKEND] Processing insert_char: %q", data)
		if len(data) > 0 {
			e.clearExitWarning() // Clear any exit warning when typing
			// Correct UTF-8 handling: convert string to runes
			runes := []rune(data)
			if len(runes) > 0 {
				e.handleInsertChar(runes[0])
				log.Printf("[EDITOR-BACKEND] Character inserted: %q, calling Render()", runes[0])
			}
			e.Render()
		}
	case "key_input":
		// Parse the key data and extract the actual key
		log.Printf("[EDITOR-BACKEND] Processing key_input: %q", data)

		// Parse the JSON to extract the key
		var keyData map[string]interface{}
		if err := json.Unmarshal([]byte(data), &keyData); err != nil {
			log.Printf("[EDITOR-BACKEND] Error parsing key data: %v", err)
			e.sendStatusMessage("Error parsing key data")
			return true
		}

		// Extract the key value
		if keyValue, ok := keyData["key"]; ok {
			if keyStr, ok := keyValue.(string); ok {
				log.Printf("[EDITOR-BACKEND] Extracted key: %q", keyStr)

				// Handle character input differently
				if keyStr == "char" {
					if charValue, ok := keyData["char"]; ok {
						if charStr, ok := charValue.(string); ok {
							log.Printf("[EDITOR-BACKEND] Processing character: %q", charStr)
							// Handle character insertion
							if len(charStr) > 0 {
								runes := []rune(charStr)
								if len(runes) > 0 {
									e.handleInsertChar(runes[0])
									log.Printf("[EDITOR-BACKEND] Character inserted: %q", runes[0])
								}
							}
							e.Render()
							return true
						}
					}
					log.Printf("[EDITOR-BACKEND] Error: char key without char value")
					return true
				}

				// For other keys, call ProcessInput with the key string
				return e.ProcessInput(keyStr)
			}
		}

		log.Printf("[EDITOR-BACKEND] Error: invalid key data format")
		e.sendStatusMessage("Invalid key data format")
		return true
	default:
		e.sendStatusMessage("Unknown command: " + command)
	}

	return true
}

// handleBackspace verarbeitet Backspace-Taste
// handleDelete processes the Delete key
// handleInsertNewline inserts a new line
// Start initializes the editor and sends the initial state to the client
func (e *Editor) Start() {
	logger.Info(logger.AreaEditor, "Editor Start() called for session: %s, filename: %s", e.sessionID, e.filename)
	e.active = true
	e.readyChan = make(chan bool, 1) // Buffer to prevent blocking if frontend is slow
	// Send "start" message to client with initial content and parameters
	// Content is joined with \n which is most reliable format for frontend parsing
	content := strings.Join(e.lines, "\n")
	logger.Info(logger.AreaEditor, "Editor Start: Sending 'start' message with %d lines. First line: '%s'", len(e.lines), e.lines[0])
	logger.Debug(logger.AreaEditor, "Editor Start: Content sample (first 100 chars): %s", content[:min(100, len(content))])

	e.sendEditorMessage("start", map[string]interface{}{
		"filename":   e.filename,
		"content":    content,
		"readOnly":   e.readOnly,
		"cols":       e.cols,
		"rows":       e.rows, // Total editor window height
		"cursorX":    e.cursorX,
		"cursorY":    e.cursorY,
		"scrollY":    e.scrollY,
		"hideCursor": e.readOnly, // Hide cursor by default in ReadOnly mode
	})
	logger.Info(logger.AreaEditor, "Editor Start: 'start' message sent. Waiting for 'ready' from client...")
	// Wait for "ready" signal from frontend with reduced timeout
	select {
	case <-e.readyChan:
		logger.Info(logger.AreaEditor, "Editor Start: Received 'ready' signal from client for session %s.", e.sessionID)
		// Frontend is ready, now send the first full render command
		// This ensures the client has processed "start" and is ready for detailed state
		e.Render()
		logger.Info(logger.AreaEditor, "Editor Start: Initial Render() call complete after 'ready'.")
	case <-time.After(1 * time.Second): // REDUZIERT: 1-Sekunden-Timeout (war 5 Sekunden)
		logger.Error(logger.AreaEditor, "Editor Start: Timeout waiting for 'ready' signal from client for session %s. Proceeding with Render().", e.sessionID)
		// Auch wenn kein "ready" kommt, versuchen wir zu rendern, falls der Client es doch noch verarbeitet.
		e.Render()
	}
}

// Ready is called when the frontend signals it's ready to receive editor commands
func (e *Editor) Ready() {
	logger.Info(logger.AreaEditor, "Editor Ready() called for session: %s. Signaling readyChan.", e.sessionID)
	if e.readyChan != nil {
		select {
		case e.readyChan <- true:
			logger.Info(logger.AreaEditor, "Successfully sent true to readyChan.")
		default:
			logger.Warn(logger.AreaEditor, "readyChan is full or nil, could not send true. Frontend might have sent 'ready' multiple times or too late.")
		}
	} else {
		logger.Warn(logger.AreaEditor, "readyChan is nil in Ready(). This should not happen if Start() was called.")
	}
}

// Close signals the editor to shut down
func (e *Editor) Close() {
	// Prüfe, ob der Editor bereits inaktiv ist
	if !e.active {
		logger.Info(logger.AreaEditor, "Close called on already inactive editor for session %s.", e.sessionID)
		return
	}

	logger.Info(logger.AreaEditor, "Closing editor for session %s.", e.sessionID)
	e.active = false

	// Send editor stop command to frontend
	e.sendEditorMessage("stop", nil)

	// Close the output channel to signal the forwarding goroutine in tinyos to stop.
	// This must be done after all messages have been sent.
	close(e.outputChan)

	logger.Info(logger.AreaEditor, "Editor for session %s closed successfully.", e.sessionID)
}

// Render sends the current editor state to the frontend
// sendEditorMessage sends an editor-specific message to the frontend
// draw renders the editor display
func (e *Editor) draw() {
	// Ensure wrapped lines are up-to-date before any rendering
	if len(e.wrappedLines) == 0 || e.cols <= 0 {
		logger.Debug(logger.AreaEditor, "draw(): Ensuring wrapped lines are up-to-date before rendering")
		e.updateWrappedLines()
	}

	// For now, just call Render - this maintains compatibility
	e.Render()
}

// Note: The previous handleCursorMovement function has been removed as it was incomplete
// and redundant. Cursor movement is now handled by the improved moveCursor function.

// debugVerifyMappings performs a comprehensive round-trip test of cursor mapping functions
// This function tests every possible logical position to ensure mapping consistency
// Only runs if debug_mapping_verification is enabled in settings
