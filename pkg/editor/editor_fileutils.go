package editor

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/antibyte/retroterm/pkg/configuration"
	"github.com/antibyte/retroterm/pkg/logger"
)

// cleanCodeForLoading removes non-printable characters except newlines from code
// This prevents parsing issues caused by invisible characters in editor-created code
func cleanCodeForLoading(content string) string {
	var cleaned strings.Builder

	for _, r := range content {
		// Keep printable characters (including Unicode like üöä), newlines, and essential whitespace
		if r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r) {
			cleaned.WriteRune(r)
		}
		// Skip all other non-printable characters (including zero-width spaces, etc.)
	}

	result := cleaned.String()

	// Normalize line endings
	result = strings.ReplaceAll(result, "\r\n", "\n")
	result = strings.ReplaceAll(result, "\r", "\n")

	// Remove any trailing/leading whitespace from each line but preserve structure
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t") // Remove trailing whitespace but keep leading
		// Normalize BASIC code while preserving strings
		lines[i] = normalizeBASICLine(lines[i])
	}

	return strings.Join(lines, "\n")
}

// normalizeBASICLine converts BASIC commands and variables to uppercase while preserving string literals
func normalizeBASICLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}

	var result strings.Builder
	inString := false
	var quote rune

	for _, r := range line {
		if !inString {
			// Outside of string literals - check for quote start
			if r == '"' || r == '\'' {
				inString = true
				quote = r
				result.WriteRune(r)
			} else {
				// Convert to uppercase outside of strings
				result.WriteRune(unicode.ToUpper(r))
			}
		} else {
			// Inside string literal - preserve case and check for end
			result.WriteRune(r)
			if r == quote {
				// Check if it's an escaped quote (not implemented in TinyBASIC, so simple end)
				inString = false
			}
		}
	}

	return result.String()
}

// LoadFile loads a file from the virtual filesystem into the editor
func (e *Editor) LoadFile(filename string) error {
	if e.vfs == nil {
		return fmt.Errorf("no virtual file system available")
	}

	content, err := e.vfs.ReadFile(filename, e.sessionID)
	if err != nil {
		// File doesn't exist - start with empty content
		e.lines = []string{""}
		e.filename = filename
		e.modified = false
		return nil
	}

	// Split content into lines
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	// Check line limit
	maxLines := configuration.GetInt("Editor", "max_lines", 5000)
	if len(lines) > maxLines {
		return fmt.Errorf("file too large: %d lines (maximum %d lines allowed)", len(lines), maxLines)
	}

	e.lines = lines
	// Remove trailing empty line if content ends with newline
	if len(e.lines) > 1 && e.lines[len(e.lines)-1] == "" {
		e.lines = e.lines[:len(e.lines)-1]
	}

	e.filename = filename
	e.modified = false
	e.cursorX = 0
	e.cursorY = 0
	e.scrollY = 0

	// CRITICAL: Update wrapped lines immediately after loading content
	// This ensures word wrapping is applied to the loaded content
	logger.Info(logger.AreaEditor, "LoadFile: Updating wrapped lines with cols=%d", e.cols)
	e.updateWrappedLines()
	logger.Info(logger.AreaEditor, "LoadFile: Content loaded and wrapped - %d lines -> %d wrapped lines", len(e.lines), len(e.wrappedLines))

	return nil
}

// SaveFile saves the current content to a file
func (e *Editor) SaveFile() error {
	if e.vfs == nil {
		return fmt.Errorf("no virtual file system available")
	}

	if e.filename == "" {
		return fmt.Errorf("no filename specified")
	}
	// Join lines with newlines
	content := strings.Join(e.lines, "\n")

	// Clean the content to remove non-printable characters that could corrupt the BASIC parser
	content = cleanCodeForLoading(content)

	err := e.vfs.WriteFile(e.filename, content, e.sessionID)
	if err != nil {
		return err
	}

	e.modified = false
	return nil
}
