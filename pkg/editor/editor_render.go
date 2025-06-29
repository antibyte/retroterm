package editor

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
)

// This file contains rendering and display functions for the editor

// GetRenderParams returns parameters needed for rendering the editor
func (e *Editor) GetRenderParams() map[string]interface{} {
	if !e.active {
		return nil
	}

	// CRITICAL: Ensure wrapped lines are up to date
	if len(e.wrappedLines) == 0 || e.cols <= 0 {
		logger.Info(logger.AreaEditor, "GetRenderParams: Updating wrapped lines - wrappedLines=%d, cols=%d", len(e.wrappedLines), e.cols)
		e.updateWrappedLines()
	}
	// Use wrapped lines for display (same logic as in Render())
	const maxVisibleLines = 23 // Must match Render() constant
	// Always send exactly 23 lines (pad with empty lines if needed)
	visibleLines := make([]string, maxVisibleLines)
	for i := 0; i < maxVisibleLines; i++ {
		wrappedLineIdx := e.scrollY + i
		if wrappedLineIdx < len(e.wrappedLines) {
			visibleLines[i] = e.wrappedLines[wrappedLineIdx]
		} else {
			visibleLines[i] = "" // Empty line for padding
		}
	}
	// Calculate visible cursor position based on wrapped lines
	visibleCursorLine := -1
	visibleCursorCol := -1

	if e.readOnly {
		// In ReadOnly mode, position cursor in status line (line 23) at the far right
		visibleCursorLine = 23        // Status line (0-indexed, so 23 is the status line)
		visibleCursorCol = e.cols - 1 // Far right of the terminal
		logger.Info(logger.AreaEditor, "GetRenderParams: ReadOnly cursor positioned in status line at col %d", visibleCursorCol)
	} else {
		// Get visual (wrapped) position from logical position
		wrappedLineIdx, wrappedCol := e.mapLogicalToWrappedCursor()

		// Calculate visible position (adjust for scroll)
		visibleCursorLine = wrappedLineIdx - e.scrollY
		visibleCursorCol = wrappedCol

		logger.Info(logger.AreaEditor, "GetRenderParams: Mapped logical=(%d,%d) to wrapped=(%d,%d), visible=(%d,%d), scrollY=%d",
			e.cursorX, e.cursorY, wrappedCol, wrappedLineIdx, visibleCursorCol, visibleCursorLine, e.scrollY)

		// Clamp to visible area
		if visibleCursorLine < 0 {
			visibleCursorLine = 0
			logger.Debug(logger.AreaEditor, "GetRenderParams: Cursor line clamped to visible area (min)")
		}
		if visibleCursorLine >= maxVisibleLines {
			visibleCursorLine = maxVisibleLines - 1
			logger.Debug(logger.AreaEditor, "GetRenderParams: Cursor line clamped to visible area (max)")
		}

		if visibleCursorCol < 0 {
			visibleCursorCol = 0
			logger.Debug(logger.AreaEditor, "GetRenderParams: Cursor column clamped to visible area (min)")
		}
		if visibleCursorCol >= e.cols {
			visibleCursorCol = e.cols - 1
			logger.Debug(logger.AreaEditor, "GetRenderParams: Cursor column clamped to visible area (max)")
		}

		logger.Info(logger.AreaEditor, "GetRenderParams: FINAL - visible position=(%d,%d)",
			visibleCursorCol, visibleCursorLine)
	}

	// Create status line using buildStatusLine for consistency
	status := e.buildStatusLine()

	// Debug: Log the content of lines that contain non-ASCII characters
	for i, line := range visibleLines {
		if len(line) > 0 {
			hasNonASCII := false
			for _, r := range line {
				if r > 127 {
					hasNonASCII = true
					break
				}
			}
			if hasNonASCII {
				log.Printf("[EDITOR-UTF8-DEBUG] Line %d contains non-ASCII: %q (bytes: %v)", i, line, []byte(line))
			}
		}
	}
	// Detailed debug logs for troubleshooting
	logger.Info(logger.AreaEditor, "GetRenderParams: FINAL MAPPING - visibleCursorLine=%d, visibleCursorCol=%d, logicalLine=%d, logicalCol=%d, scrollY=%d, success=%v, status=%s", visibleCursorLine, visibleCursorCol, e.cursorY, e.cursorX, e.scrollY, visibleCursorLine >= 0, status)

	// Create render parameter map with all necessary information for frontend
	renderParams := map[string]interface{}{
		"lines":              visibleLines,        // Exactly 23 prepared lines
		"cursorX":            visibleCursorCol,    // Use visible cursor position for frontend (X = column)
		"cursorY":            visibleCursorLine,   // Use visible cursor position for frontend (Y = row)
		"cursorLine":         visibleCursorLine,   // Cursor line for Message struct
		"cursorCol":          visibleCursorCol,    // Cursor column for Message struct
		"scrollY":            e.scrollY,           // Current scroll position
		"totalLines":         len(e.wrappedLines), // Total number of wrapped display lines
		"originalLines":      len(e.lines),        // Total number of original lines
		"visibleLineCount":   maxVisibleLines,     // Always 23
		"textCols":           e.cols,              // Terminal width
		"textRows":           e.textRows,          // Terminal height (should be 23)
		"filename":           e.filename,
		"modified":           e.modified,
		"status":             status,     // Status line
		"readOnly":           e.readOnly, // Read-only mode flag
		"hideCursor":         e.readOnly, // Hide cursor in read-only mode
		"editorActive":       e.active,
		"showingExitWarning": e.showingExitWarning,
		"renderTimestamp":    time.Now().UnixMilli(), // For debugging
		"mappingSuccess":     visibleCursorLine >= 0, // Flag for frontend
	}
	return renderParams
}

// buildStatusLine creates the status line displayed at the bottom of the editor
func (e *Editor) buildStatusLine() string {
	// If showing exit warning, return the warning message
	if e.showingExitWarning {
		warning := "Modified! ^S:Save ^C:Exit ^ESC:Cancel"
		// Ensure warning fits in status line
		if len(warning) > e.cols {
			warning = warning[:e.cols]
		} else if len(warning) < e.cols {
			warning += strings.Repeat(" ", e.cols-len(warning))
		}
		return warning
	}
	// Normal status line
	filename := e.filename
	if filename == "" {
		filename = "<new file>"
	}

	modifiedFlag := ""
	if e.modified {
		modifiedFlag = " *"
	}

	// Get visual (wrapped) position for status line display
	wrappedLineIdx, wrappedCol := e.mapLogicalToWrappedCursor()

	// Display wrapped line position (1-based for user display)
	lineInfo := fmt.Sprintf("L%d C%d", wrappedLineIdx+1, wrappedCol+1)
	// Key commands
	commands := "^S:Save ^X:Exit"
	if e.readOnly {
		commands = "^X:Exit (READ-ONLY)"
	}

	// Build status line, padding to full width
	left := fmt.Sprintf("%s%s | %s", filename, modifiedFlag, lineInfo)
	right := commands

	spaces := e.cols - len(left) - len(right)
	if spaces < 1 {
		spaces = 1
	}

	status := left + strings.Repeat(" ", spaces) + right

	// Ensure status line is exactly the right length
	if len(status) > e.cols {
		status = status[:e.cols]
	} else if len(status) < e.cols {
		status += strings.Repeat(" ", e.cols-len(status))
	}

	return status
}

// Render sends the current editor state to the frontend for display
func (e *Editor) Render() {
	logger.Info(logger.AreaEditor, "==== RENDER CALLED ==== active: %v, readOnly: %v, scrollY: %d, filename: %s", e.active, e.readOnly, e.scrollY, e.filename)
	if !e.active {
		logger.Debug(logger.AreaEditor, "Editor not active, skipping render")
		return
	}
	// Debug: Check editor state before wrapping
	logger.Debug(logger.AreaEditor, "Render: cols=%d, lines=%d, wrappedLines=%d", e.cols, len(e.lines), len(e.wrappedLines))

	// Only update wrapped lines if they haven't been initialized or if dimensions changed
	if len(e.wrappedLines) == 0 || e.cols <= 0 {
		logger.Info(logger.AreaEditor, "Render: Updating wrapped lines - wrappedLines=%d, cols=%d", len(e.wrappedLines), e.cols)
		e.updateWrappedLines()
		logger.Debug(logger.AreaEditor, "Render: After updateWrappedLines, wrappedLines count: %d", len(e.wrappedLines))
	}

	// Debug: Show first few wrapped lines
	if len(e.wrappedLines) > 0 {
		logger.Debug(logger.AreaEditor, "Render: First 3 wrapped lines:")
		for i := 0; i < 3 && i < len(e.wrappedLines); i++ {
			logger.Debug(logger.AreaEditor, "Render: wrappedLine[%d]: '%s' (len=%d)", i, e.wrappedLines[i], len(e.wrappedLines[i]))
		}
	}

	// Prepare visible lines based on scroll position and available wrapped lines
	// Don't limit to a fixed number - send all relevant lines for the current view
	const maxVisibleLines = 23 // Maximum lines that can fit on screen

	// Calculate how many lines we actually need to send
	totalWrappedLines := len(e.wrappedLines)
	startLine := e.scrollY
	endLine := startLine + maxVisibleLines

	// Ensure we don't go beyond available wrapped lines
	if endLine > totalWrappedLines {
		endLine = totalWrappedLines
	}

	// Calculate actual visible line count
	actualVisibleCount := endLine - startLine
	if actualVisibleCount < 0 {
		actualVisibleCount = 0
	}

	// Log the calculation for debugging
	logger.Info(logger.AreaEditor, "Render: Calculating visible lines - totalWrapped=%d, scrollY=%d, startLine=%d, endLine=%d, actualVisible=%d",
		totalWrappedLines, e.scrollY, startLine, endLine, actualVisibleCount)

	// Always send exactly 23 lines (pad with empty lines if needed)
	visibleLines := make([]string, maxVisibleLines)

	// Fill visible lines with wrapped content
	for i := 0; i < maxVisibleLines; i++ {
		wrappedLineIdx := e.scrollY + i
		if wrappedLineIdx < len(e.wrappedLines) {
			visibleLines[i] = e.wrappedLines[wrappedLineIdx]
		} else {
			visibleLines[i] = "" // Empty line for padding
		}
	}

	logger.Debug(logger.AreaEditor, "Render: Prepared %d visible lines for frontend", len(visibleLines))
	if len(visibleLines) > 0 {
		logger.Debug(logger.AreaEditor, "Render: First 3 visible lines to send:")
		for i := 0; i < 3 && i < len(visibleLines); i++ {
			logger.Debug(logger.AreaEditor, "Render: visibleLine[%d]: '%s' (len=%d)", i, visibleLines[i], len(visibleLines[i]))
		}
	}
	// Calculate visible cursor position based on wrapped lines
	visibleCursorLine := -1
	visibleCursorCol := -1

	if e.readOnly {
		// In ReadOnly mode, position cursor in status line (line 23) at the far right
		// This hides it from the content area but keeps it visible in a non-intrusive way
		visibleCursorLine = 23        // Status line (0-indexed, so 23 is the status line)
		visibleCursorCol = e.cols - 1 // Far right of the terminal
		logger.Info(logger.AreaEditor, "ReadOnly: Cursor positioned in status line at col %d", visibleCursorCol)
	} else {
		// Use the same corrected cursor mapping function as GetRenderParams
		cursorWrappedLineIdx, cursorWrappedCol := e.mapLogicalToWrappedCursor()
		logger.Debug(logger.AreaEditor, "Render: Cursor mapped to wrapped line %d, col %d", cursorWrappedLineIdx, cursorWrappedCol)

		// If cursor wrapped line is visible, set visible cursor position
		if cursorWrappedLineIdx >= e.scrollY && cursorWrappedLineIdx < e.scrollY+maxVisibleLines {
			visibleCursorLine = cursorWrappedLineIdx - e.scrollY
			visibleCursorCol = cursorWrappedCol

			// Ensure cursor column doesn't exceed bounds
			if visibleCursorCol > e.cols-1 {
				visibleCursorCol = e.cols - 1
			}
			if visibleCursorCol < 0 {
				visibleCursorCol = 0
			}
			logger.Debug(logger.AreaEditor, "Render: Visible cursor set to line %d, col %d", visibleCursorLine, visibleCursorCol)
		} else {
			logger.Warn(logger.AreaEditor, "Render: Cursor not visible - wrapped line %d not in scroll range %d-%d",
				cursorWrappedLineIdx, e.scrollY, e.scrollY+maxVisibleLines-1)
		}
	}

	// Comprehensive editor data with all render information
	editorData := map[string]interface{}{
		"lines":              visibleLines,           // Exactly 23 prepared lines
		"cursorX":            visibleCursorCol,       // Use visible cursor position for frontend (X = column)
		"cursorY":            visibleCursorLine,      // Use visible cursor position for frontend (Y = row)
		"cursorLine":         visibleCursorLine,      // Cursor line for Message struct
		"cursorCol":          visibleCursorCol,       // Cursor column for Message struct
		"scrollY":            e.scrollY,              // Current scroll position
		"totalLines":         len(e.wrappedLines),    // Total number of wrapped display lines
		"originalLines":      len(e.lines),           // Total number of original lines
		"visibleLineCount":   maxVisibleLines,        // Always 23
		"textCols":           e.cols,                 // Terminal width
		"textRows":           e.textRows,             // Terminal height (should be 23)
		"status":             e.buildStatusLine(),    // Status line content
		"filename":           e.filename,             // Filename
		"modified":           e.modified,             // Modification status
		"readOnly":           e.readOnly,             // Read-only flag
		"hideCursor":         e.readOnly,             // Hide cursor in read-only mode
		"editorActive":       e.active,               // Is editor active?
		"showingExitWarning": e.showingExitWarning,   // Is exit warning displayed?
		"renderTimestamp":    time.Now().UnixMilli(), // For debugging
	}

	// Reduced but informative logging output for render function
	logger.Info(logger.AreaEditor, "Render: cursor=(%d,%d), readOnly=%v, file=%s, lines=%d/%d",
		visibleCursorCol, visibleCursorLine, e.readOnly, e.filename, len(e.wrappedLines), len(e.lines))
	e.sendEditorMessage("render", editorData)

	// Send a single consolidated cursor state message after render
	// No need for multiple delayed messages - one is enough
	go func() {
		time.Sleep(50 * time.Millisecond)
		if e.active {
			if e.readOnly {
				e.sendCursorState(false)
				logger.Debug(logger.AreaEditor, "Sent cursor hide message for ReadOnly mode")
			} else {
				e.sendCursorState(true)
				logger.Debug(logger.AreaEditor, "Sent cursor show message for normal editor mode")
			}
		}
	}()

	// The status line is already included in the editorData["status"] field
	// and will be rendered by the frontend editor, not by the terminal system.
	logger.Debug(logger.AreaEditor, "Status line included in render data: %s", e.buildStatusLine())
}
