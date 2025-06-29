package editor

import (
	"fmt"
	"log"
	"strings"

	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/shared"
)

// This file contains text insertion and utility functions for the editor

// handleInsertNewline handles inserting a new line at cursor position
func (e *Editor) handleInsertNewline() {
	log.Printf("[EDITOR-UTF8-DEBUG] handleInsertNewline called, cursorY: %d, cursorX: %d", e.cursorY, e.cursorX)

	if e.cursorY < len(e.lines) {
		line := e.lines[e.cursorY]
		lineRunes := []rune(line)

		log.Printf("[EDITOR-UTF8-DEBUG] Splitting line for newline: %q (runes: %d), cursor at: %d", line, len(lineRunes), e.cursorX)
		log.Printf("[EDITOR-UTF8-DEBUG] Line bytes: %v", []byte(line))

		// Fix cursor position if it's beyond the line length
		if e.cursorX > len(lineRunes) {
			log.Printf("[EDITOR-UTF8-DEBUG] ERROR: cursorX (%d) > line length (%d), correcting to %d", e.cursorX, len(lineRunes), len(lineRunes))
			e.cursorX = len(lineRunes)
		}

		var beforeCursor, afterCursor string
		if e.cursorX <= len(lineRunes) {
			beforeCursor = string(lineRunes[:e.cursorX])
			afterCursor = string(lineRunes[e.cursorX:])
		} else {
			beforeCursor = line
			afterCursor = ""
		}

		log.Printf("[EDITOR-UTF8-DEBUG] Split result - beforeCursor: %q, afterCursor: %q", beforeCursor, afterCursor)

		e.lines[e.cursorY] = beforeCursor
		newLines := make([]string, len(e.lines)+1)
		copy(newLines[:e.cursorY+1], e.lines[:e.cursorY+1])
		newLines[e.cursorY+1] = afterCursor
		copy(newLines[e.cursorY+2:], e.lines[e.cursorY+1:])
		e.lines = newLines

		log.Printf("[EDITOR-UTF8-DEBUG] After newline split - line %d: %q, line %d: %q", e.cursorY, e.lines[e.cursorY], e.cursorY+1, e.lines[e.cursorY+1])
	} else {
		log.Printf("[EDITOR-UTF8-DEBUG] Adding new line at end")
		e.lines = append(e.lines, "")
	}
	e.cursorY++
	e.cursorX = 0
	e.modified = true
	// Update wrapped lines after content change
	e.updateWrappedLines()
	e.adjustScroll()

	log.Printf("[EDITOR-UTF8-DEBUG] After newline - new position: cursorY=%d, cursorX=%d", e.cursorY, e.cursorX)
}

// handleInsertChar inserts a character at cursor position
func (e *Editor) handleInsertChar(ch rune) {
	log.Printf("[EDITOR-UTF8-DEBUG] handleInsertChar called with rune: %q (unicode: U+%04X, bytes: %v)", ch, ch, []byte(string(ch)))

	// Ensure there are enough lines
	for len(e.lines) <= e.cursorY {
		e.lines = append(e.lines, "")
	}

	line := e.lines[e.cursorY]
	lineRunes := []rune(line)
	log.Printf("[EDITOR-UTF8-DEBUG] Before insert - line: %q, lineRunes length: %d, cursorX: %d", line, len(lineRunes), e.cursorX)

	// Fix cursor position if it's beyond the line length
	if e.cursorX > len(lineRunes) {
		log.Printf("[EDITOR-UTF8-DEBUG] ERROR: cursorX (%d) > line length (%d), correcting to %d", e.cursorX, len(lineRunes), len(lineRunes))
		e.cursorX = len(lineRunes)
	}
	// Convert line to runes for proper Unicode handling
	leftPart := string(lineRunes[:e.cursorX])
	rightPart := string(lineRunes[e.cursorX:])
	newLine := leftPart + string(ch) + rightPart
	e.lines[e.cursorY] = newLine
	e.cursorX++
	e.modified = true
	// Update wrapped lines after content change
	e.updateWrappedLines()

	log.Printf("[EDITOR-UTF8-DEBUG] After insert - newLine: %q, new cursorX: %d", newLine, e.cursorX)
	// Note: Line wrapping is now handled by display layer (updateWrappedLines)
	// No need to break lines in the actual content
}

// insertText inserts text at the current cursor position
func (e *Editor) insertText(text string) {
	if e.cursorY >= len(e.lines) {
		// Add lines if needed
		for len(e.lines) <= e.cursorY {
			e.lines = append(e.lines, "")
		}
	}

	line := e.lines[e.cursorY]
	lineRunes := []rune(line)
	textRunes := []rune(text)

	// Insert text at cursor position (rune-based)
	if e.cursorX >= len(lineRunes) {
		// Append to end of line
		newLineRunes := append(lineRunes, textRunes...)
		e.lines[e.cursorY] = string(newLineRunes)
	} else {
		// Insert in middle of line
		newLineRunes := make([]rune, 0, len(lineRunes)+len(textRunes))
		newLineRunes = append(newLineRunes, lineRunes[:e.cursorX]...)
		newLineRunes = append(newLineRunes, textRunes...)
		newLineRunes = append(newLineRunes, lineRunes[e.cursorX:]...)
		e.lines[e.cursorY] = string(newLineRunes)
	}
	e.cursorX += len(textRunes)
	e.modified = true
	// Update wrapped lines after content change
	e.updateWrappedLines()
}

// handleEnter handles enter key (create new line)
func (e *Editor) handleEnter() {
	log.Printf("[EDITOR-UTF8-DEBUG] handleEnter called at cursorY: %d, cursorX: %d", e.cursorY, e.cursorX)

	// Check line limit before adding new lines
	maxLines := configuration.GetInt("Editor", "max_lines", 5000)
	if len(e.lines) >= maxLines {
		e.sendStatusMessage(fmt.Sprintf("Maximum line limit reached (%d lines)", maxLines))
		return
	}

	if e.cursorY >= len(e.lines) {
		// Add new line at end
		e.lines = append(e.lines, "")
		e.cursorY = len(e.lines) - 1
		e.cursorX = 0
		log.Printf("[EDITOR-UTF8-DEBUG] Added new line at end, new cursorY: %d", e.cursorY)
	} else {
		// Split current line at cursor (using runes for Unicode support)
		line := e.lines[e.cursorY]
		runes := []rune(line)

		log.Printf("[EDITOR-UTF8-DEBUG] Splitting line: %q (runes: %d), cursor at: %d", line, len(runes), e.cursorX)
		log.Printf("[EDITOR-UTF8-DEBUG] Line bytes: %v", []byte(line))

		var leftPart, rightPart string
		if e.cursorX <= len(runes) {
			leftPart = string(runes[:e.cursorX])
			rightPart = string(runes[e.cursorX:])
			log.Printf("[EDITOR-UTF8-DEBUG] Split successful - leftPart: %q, rightPart: %q", leftPart, rightPart)
		} else {
			leftPart = line
			rightPart = ""
			log.Printf("[EDITOR-UTF8-DEBUG] Cursor beyond line, leftPart: %q, rightPart: %q", leftPart, rightPart)
		}

		e.lines[e.cursorY] = leftPart

		// Insert new line after current
		e.lines = append(e.lines[:e.cursorY+1], append([]string{rightPart}, e.lines[e.cursorY+1:]...)...)

		log.Printf("[EDITOR-UTF8-DEBUG] After split - line %d: %q, line %d: %q", e.cursorY, e.lines[e.cursorY], e.cursorY+1, e.lines[e.cursorY+1])

		e.cursorY++
		e.cursorX = 0
	}
	e.modified = true
	// Update wrapped lines after content change
	e.updateWrappedLines()
}

// handleForceExit exits the editor without saving
func (e *Editor) handleForceExit() bool {
	log.Printf("[EDITOR-BACKEND] handleForceExit called - forcing editor exit")

	// Send explicit "stop" command to frontend
	e.sendEditorMessage("stop", nil)

	// Exit editor without warning
	e.sendMessage(shared.MessageTypeInputControl, "", map[string]interface{}{
		"inputEnabled": true,
	})
	e.active = false
	return false
}

// handleOpen opens a new file (optional)
func (e *Editor) handleOpen(filename string) bool {
	if filename == "" {
		e.sendStatusMessage("No filename specified")
		return true
	}

	if e.modified {
		e.sendStatusMessage("File modified! Save first or use force exit")
		return true
	}

	err := e.LoadFile(filename)
	if err != nil {
		e.sendStatusMessage("Error loading file: " + err.Error())
	} else {
		e.sendStatusMessage("File loaded: " + filename)
	}

	e.Render()
	return true
}

// clearExitWarning clears the exit warning state if it's currently showing
func (e *Editor) clearExitWarning() {
	if e.showingExitWarning {
		e.showingExitWarning = false
	}
}

// GetContent returns the current editor content as a string
func (e *Editor) GetContent() string {
	return strings.Join(e.lines, "\n")
}

// SetContent sets the editor content from a string
func (e *Editor) SetContent(content string) {
	e.lines = strings.Split(content, "\n")
	if len(e.lines) == 0 {
		e.lines = []string{""}
	}

	e.cursorX = 0
	e.cursorY = 0
	e.scrollY = 0
	e.modified = true

	// Update wrapped lines immediately after setting content
	e.updateWrappedLines()

	e.draw()
}

// GetFilename returns the current filename
func (e *Editor) GetFilename() string {
	return e.filename
}

// SetFilename sets the filename for the editor
func (e *Editor) SetFilename(filename string) {
	e.filename = filename
}

// IsModified returns whether the content has been modified
func (e *Editor) IsModified() bool {
	return e.modified
}

// IsActive returns whether the editor is currently active
func (e *Editor) IsActive() bool {
	return e.active
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
