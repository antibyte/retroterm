package editor

// This file contains text manipulation functions for the editor

// handleBackspace processes the backspace key
func (e *Editor) handleBackspace() {
	if e.cursorX > 0 {
		// Delete character in current line
		if e.cursorY < len(e.lines) {
			line := e.lines[e.cursorY]
			lineRunes := []rune(line)
			if e.cursorX <= len(lineRunes) {
				newLineRunes := make([]rune, 0, len(lineRunes)-1)
				newLineRunes = append(newLineRunes, lineRunes[:e.cursorX-1]...)
				newLineRunes = append(newLineRunes, lineRunes[e.cursorX:]...)
				e.lines[e.cursorY] = string(newLineRunes)
				e.cursorX--
				e.modified = true
				// Update wrapped lines after content change
				e.updateWrappedLines()
			}
		}
	} else if e.cursorY > 0 {
		// Join with previous line
		if e.cursorY < len(e.lines) {
			currentLine := e.lines[e.cursorY]
			e.lines = append(e.lines[:e.cursorY], e.lines[e.cursorY+1:]...)

			if e.cursorY-1 < len(e.lines) {
				prevLineRunes := []rune(e.lines[e.cursorY-1])
				e.cursorX = len(prevLineRunes)
				e.lines[e.cursorY-1] += currentLine
				e.cursorY--
				e.modified = true
				// Update wrapped lines after content change
				e.updateWrappedLines()
			}
		}
	}
}

// handleDelete processes the Delete key
func (e *Editor) handleDelete() {
	if e.cursorY < len(e.lines) {
		line := e.lines[e.cursorY]
		lineRunes := []rune(line)
		if e.cursorX < len(lineRunes) {
			// Delete character (rune-based)
			newLineRunes := make([]rune, 0, len(lineRunes)-1)
			newLineRunes = append(newLineRunes, lineRunes[:e.cursorX]...)
			newLineRunes = append(newLineRunes, lineRunes[e.cursorX+1:]...)
			e.lines[e.cursorY] = string(newLineRunes)
			e.modified = true
			// Update wrapped lines after content change
			e.updateWrappedLines()
		} else if e.cursorY+1 < len(e.lines) {
			// Join with next line
			e.lines[e.cursorY] += e.lines[e.cursorY+1]
			e.lines = append(e.lines[:e.cursorY+1], e.lines[e.cursorY+2:]...)
			e.modified = true
			// Update wrapped lines after content change
			e.updateWrappedLines()
		}
	}
}
