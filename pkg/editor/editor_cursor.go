package editor

import (
	"github.com/antibyte/retroterm/pkg/logger"
)

// This file contains cursor movement and scrolling functions for the editor

// moveCursor moves the cursor by the specified offset
// This function handles all cursor movement in a logical, consistent way
func (e *Editor) moveCursor(deltaX, deltaY int) {
	// Log cursor movement operation for debugging
	logger.Debug(logger.AreaEditor, "moveCursor: delta=(%d,%d), before=(%d,%d)",
		deltaX, deltaY, e.cursorX, e.cursorY)
	// Ensure wrapped lines are updated
	e.updateWrappedLines()

	// Handle special case: no wrapped lines available
	if len(e.wrappedLines) == 0 || len(e.lineMapping) == 0 {
		logger.Warn(logger.AreaEditor, "moveCursor: No wrapped lines available, using standard movement")
		e.cursorY += deltaY
		e.cursorX += deltaX
		return
	} // Important: Handle horizontal movement separately from vertical movement
	if deltaY == 0 && deltaX != 0 {
		// Handle horizontal cursor movement with proper line wrapping
		newCursorX := e.cursorX + deltaX

		// Clamp cursor and handle line wrapping for horizontal movement
		if e.cursorY >= 0 && e.cursorY < len(e.lines) {
			lineLen := len([]rune(e.lines[e.cursorY])) // Use runes for proper UTF-8 handling

			// Moving right: if cursor goes past end of line, move to next line
			if deltaX > 0 && newCursorX > lineLen {
				if e.cursorY < len(e.lines)-1 {
					e.cursorY++
					e.cursorX = 0
					logger.Debug(logger.AreaEditor, "moveCursor: Wrapped to next line start")
				} else {
					e.cursorX = lineLen // Stay at end of last line
					logger.Debug(logger.AreaEditor, "moveCursor: Clamped to end of last line")
				}
			} else if deltaX < 0 && newCursorX < 0 {
				// Moving left: if cursor goes before start of line, move to previous line
				if e.cursorY > 0 {
					e.cursorY--
					if e.cursorY < len(e.lines) {
						e.cursorX = len([]rune(e.lines[e.cursorY]))
						logger.Debug(logger.AreaEditor, "moveCursor: Wrapped to end of previous line")
					}
				} else {
					e.cursorX = 0 // Stay at start of first line
					logger.Debug(logger.AreaEditor, "moveCursor: Clamped to start of first line")
				}
			} else {
				// Normal horizontal movement within line bounds
				e.cursorX = newCursorX
				// Ensure cursor doesn't go beyond line end
				if e.cursorX > lineLen {
					e.cursorX = lineLen
				}
				if e.cursorX < 0 {
					e.cursorX = 0
				}
			}
		} else {
			// Fallback for invalid line position
			e.cursorX = newCursorX
		}
	} else if deltaY != 0 {
		// VERTICAL MOVEMENT - Simplified using new mapping functions

		// Step 1: Get current visual position from logical position
		currentWrappedLine, currentWrappedCol := e.mapLogicalToWrappedCursor()
		logger.Debug(logger.AreaEditor, "moveCursor: Current logical (%d,%d) -> wrapped (%d,%d)",
			e.cursorX, e.cursorY, currentWrappedCol, currentWrappedLine)

		// Step 2: Calculate target wrapped line
		targetWrappedLine := currentWrappedLine + deltaY

		// Clamp to valid range
		if targetWrappedLine < 0 {
			targetWrappedLine = 0
		}
		if targetWrappedLine >= len(e.wrappedLines) {
			targetWrappedLine = len(e.wrappedLines) - 1
		}

		// Step 3: Keep same column in visual world
		targetWrappedCol := currentWrappedCol

		// Step 4: Convert back to logical position
		newLogicalY, newLogicalX := e.mapVisualToLogical(targetWrappedLine, targetWrappedCol)

		// Set new cursor position
		e.cursorY = newLogicalY
		e.cursorX = newLogicalX

		logger.Debug(logger.AreaEditor, "moveCursor: Target wrapped (%d,%d) -> logical (%d,%d)",
			targetWrappedCol, targetWrappedLine, newLogicalX, newLogicalY)
	}

	// Final clamping to ensure cursor is in valid document range
	if e.cursorY < 0 {
		e.cursorY = 0
		e.cursorX = 0
	}
	if e.cursorY >= len(e.lines) {
		e.cursorY = len(e.lines) - 1
		if e.cursorY >= 0 && e.cursorY < len(e.lines) {
			e.cursorX = len([]rune(e.lines[e.cursorY]))
		}
	}

	logger.Debug(logger.AreaEditor, "moveCursor: after=(%d,%d)", e.cursorX, e.cursorY)

	// Ensure cursor remains visible by adjusting scroll position
	e.ensureCursorVisible()
}

// adjustScroll adjusts the scroll position to keep the cursor visible
func (e *Editor) adjustScroll() {
	// Use the new wrapped line aware scrolling logic
	e.ensureCursorVisible()
}
