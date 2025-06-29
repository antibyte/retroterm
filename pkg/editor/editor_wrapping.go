package editor

import (
	"strings"

	"github.com/antibyte/retroterm/pkg/logger"
)

// This file contains word wrapping functionality for the editor

// updateWrappedLines creates word-wrapped display lines and mapping
func (e *Editor) updateWrappedLines() {
	logger.Info(logger.AreaEditor, "updateWrappedLines: ENTRY - cols=%d, lines=%d", e.cols, len(e.lines))

	// Skip if we don't have valid dimensions yet
	if e.cols <= 0 {
		logger.Warn(logger.AreaEditor, "updateWrappedLines: INVALID cols=%d, using default 80", e.cols)
		// Use default width of 80 if cols is not set properly
		e.cols = 80
	}

	// Prevent column widths that are too small
	if e.cols < 20 {
		logger.Warn(logger.AreaEditor, "updateWrappedLines: Column width too small, using minimum 20")
		e.cols = 20
	}

	logger.Info(logger.AreaEditor, "updateWrappedLines: PROCESSING - cols=%d, lines=%d", e.cols, len(e.lines))
	// Create completely new arrays instead of clearing existing ones
	wrapCapacity := len(e.lines) * 3 // Higher capacity reserve for complex wrapping scenarios
	e.wrappedLines = make([]string, 0, wrapCapacity)
	e.lineMapping = make([]int, 0, wrapCapacity)

	for originalLineIdx, line := range e.lines {
		// Remove \r characters for frontend display (but keep them in e.lines for saving)
		cleanLine := strings.ReplaceAll(line, "\r", "")

		// Convert to runes for proper Unicode handling - important for non-ASCII characters
		lineRunes := []rune(cleanLine)

		// If line fits within column width, add it as-is
		if len(lineRunes) <= e.cols {
			e.wrappedLines = append(e.wrappedLines, cleanLine)
			e.lineMapping = append(e.lineMapping, originalLineIdx)
			continue
		}

		// Line is too long, wrap it
		logger.Debug(logger.AreaEditor, "Wrapping line %d: length=%d, cols=%d", originalLineIdx, len(lineRunes), e.cols)

		// Improved line wrapping logic with better word boundary handling
		remainingRunes := lineRunes
		for len(remainingRunes) > 0 {
			// Determine how much of the line fits in one row
			chunkSize := e.cols
			if len(remainingRunes) < chunkSize {
				chunkSize = len(remainingRunes)
			}
			// Improved line wrapping logic with better word boundary handling
			if chunkSize < len(remainingRunes) && chunkSize > 0 {
				// First check if the remaining piece is very small and can fit completely in the chunk
				// If only a few characters would remain, put everything in the current line
				if len(remainingRunes) <= e.cols+10 && len(remainingRunes)-chunkSize <= 7 {
					chunkSize = len(remainingRunes)
					logger.Debug(logger.AreaEditor, "Word wrap: Small remaining chunk (%d chars), keeping intact", len(remainingRunes))
				} else {
					// Otherwise look for a space to break at
					breakPoint := -1

					// Search in a larger area (35% of line length) for a good break point
					searchStart := int(float64(chunkSize) * 0.65) // Changed from 0.7 to 0.65
					if searchStart < 0 {
						searchStart = 0
					}

					// Look for a space where we can break
					for i := chunkSize - 1; i >= searchStart; i-- {
						if remainingRunes[i] == ' ' || remainingRunes[i] == '\t' {
							breakPoint = i + 1 // +1 adds the space to the first line
							break
						}
					}

					// If a good break point was found, use it
					if breakPoint > 0 {
						// Check if the new break point is not too small
						if breakPoint >= int(float64(chunkSize)*0.5) {
							chunkSize = breakPoint
							logger.Debug(logger.AreaEditor, "Word wrap: Found break point at %d (%.1f%% of max)",
								breakPoint, float64(breakPoint)/float64(e.cols)*100)
						} else {
							// If the break point is less than 50% of the line, it might be
							// better to move the word to the next line
							logger.Debug(logger.AreaEditor, "Word wrap: Rejected break point at %d as too small", breakPoint)
						}
					} // If the remaining part is very small, put it all in this line anyway
					if len(remainingRunes)-chunkSize <= 5 && len(remainingRunes) <= e.cols+5 {
						chunkSize = len(remainingRunes)
						logger.Debug(logger.AreaEditor, "Word wrap: Small remaining chunk, keeping intact")
					}
				}
			} // Add the chunk as a wrapped line with consistent tracking
			chunk := string(remainingRunes[:chunkSize])
			e.wrappedLines = append(e.wrappedLines, chunk)
			e.lineMapping = append(e.lineMapping, originalLineIdx)

			// Debug info helps trace segments
			if len(e.wrappedLines) > 1 && e.lineMapping[len(e.lineMapping)-2] == originalLineIdx {
				// This is a continuation segment of the same original line
				originalLength := len([]rune(e.lines[originalLineIdx]))
				logger.Debug(logger.AreaEditor, "Wrapped segment added for line %d: '%s' (original line length=%d)",
					originalLineIdx, chunk, originalLength)
			}

			// Move to the next part of the line
			remainingRunes = remainingRunes[chunkSize:]

			// Improved handling of whitespace after a wrapped line segment
			// This ensures consistent results when mapping cursor positions
			if len(remainingRunes) > 0 {
				// Skip exactly one space or tab if at the beginning of the remaining text
				// This prevents odd offsets while keeping the text readable
				if remainingRunes[0] == ' ' || remainingRunes[0] == '\t' {
					remainingRunes = remainingRunes[1:]
				}
			}
		}
	}
	// Detailed information for improved diagnostics
	logger.Info(logger.AreaEditor, "updateWrappedLines: COMPLETED - %d original lines -> %d wrapped lines",
		len(e.lines), len(e.wrappedLines))
	// Output some diagnostic information for the first lines
	if len(e.wrappedLines) > 0 && len(e.lineMapping) > 0 {
		logger.Debug(logger.AreaEditor, "First 3 wrapped lines:")
		for i := 0; i < 3 && i < len(e.wrappedLines); i++ {
			logger.Debug(logger.AreaEditor, "  [%d] -> original line %d: '%s'",
				i, e.lineMapping[i], e.wrappedLines[i])
		}
	}
	// Mapping verification tests are now silent unless errors are found
	// Uncomment the following lines if mapping issues need to be debugged:
	// e.debugVerifyMappings()
	// e.focusedMappingTest()
}

// ensureCursorVisible ensures the cursor is visible by adjusting scroll position
// This function works with wrapped lines to provide correct scrolling behavior
func (e *Editor) ensureCursorVisible() {
	if len(e.wrappedLines) == 0 {
		return
	}

	// Map logical cursor position to wrapped line position using shared mapping function
	// for consistency across the codebase
	wrappedLineIdx, _ := e.mapLogicalToWrappedCursor()

	// Early exit if mapping failed
	if wrappedLineIdx < 0 {
		logger.Warn(logger.AreaEditor, "ensureCursorVisible: Failed to map cursor, not adjusting scroll")
		return
	}

	// Use the wrapped line index for scroll calculations
	cursorWrappedLineIdx := wrappedLineIdx
	logger.Debug(logger.AreaEditor, "ensureCursorVisible: Cursor at logical (%d,%d) mapped to wrapped line %d",
		e.cursorX, e.cursorY, cursorWrappedLineIdx)

	const maxVisibleLines = 23

	// Scroll up if cursor is above visible area
	if cursorWrappedLineIdx < e.scrollY {
		e.scrollY = cursorWrappedLineIdx
		logger.Debug(logger.AreaEditor, "ensureCursorVisible: Scrolled up to %d", e.scrollY)
	}
	// Scroll down if cursor is below visible area
	if cursorWrappedLineIdx >= e.scrollY+maxVisibleLines {
		// Less aggressive scrolling: ensure cursor is at least 2 lines from bottom for better overview
		e.scrollY = cursorWrappedLineIdx - maxVisibleLines + 3
		if e.scrollY < 0 {
			e.scrollY = 0
		}
		logger.Debug(logger.AreaEditor, "ensureCursorVisible: Scrolled down to %d", e.scrollY)

		// Debug: Log cursor state after scrolling
		logger.Debug(logger.AreaEditor, "ensureCursorVisible: After scrolling down, cursorX=%d, cursorY=%d, wrappedIdx=%d, scrollY=%d",
			e.cursorX, e.cursorY, cursorWrappedLineIdx, e.scrollY)
	}

	// Ensure scroll position stays within bounds
	// Calculate maximum scroll position so all lines remain visible without excessive empty space
	maxScroll := len(e.wrappedLines) - maxVisibleLines

	// Small optimization: if document is only a few lines longer than visible area,
	// scroll a bit further to avoid unnecessary empty space
	if len(e.wrappedLines) > maxVisibleLines && len(e.wrappedLines) < maxVisibleLines+5 {
		// For short documents, add a few lines to avoid unnecessary empty space
		maxScroll += 2
		logger.Debug(logger.AreaEditor, "ensureCursorVisible: Adjusted maxScroll to optimize display of short document")
	}

	if maxScroll < 0 {
		maxScroll = 0
	}

	if e.scrollY > maxScroll {
		e.scrollY = maxScroll
		logger.Debug(logger.AreaEditor, "ensureCursorVisible: Adjusted scroll to max %d", e.scrollY)
	}
}
