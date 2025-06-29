package editor

import (
	"strings"

	"github.com/antibyte/retroterm/pkg/logger"
)

// debugVerifyMappings performs comprehensive round-trip mapping verification
// Tests every possible logical cursor position to ensure mapping consistency
// This function is available for debugging purposes but is not called in normal operation
func (e *Editor) debugVerifyMappings() {
	// Check if mapping verification is enabled in configuration
	// This is disabled by default since mapping issues have been resolved
	if false { // Set to true only when debugging mapping issues
		return
	}

	logger.Info(logger.AreaEditor, "=== STARTING MAPPING VERIFICATION ===")

	if len(e.lines) == 0 || len(e.wrappedLines) == 0 || len(e.lineMapping) == 0 {
		logger.Warn(logger.AreaEditor, "debugVerifyMappings: Skipping verification - insufficient data (lines=%d, wrappedLines=%d, lineMapping=%d)",
			len(e.lines), len(e.wrappedLines), len(e.lineMapping))
		return
	}

	// Store original cursor position to restore later
	originalCursorY := e.cursorY
	originalCursorX := e.cursorX

	errorCount := 0
	totalTests := 0

	// Test every logical position in the document
	for y := 0; y < len(e.lines); y++ {
		lineContent := strings.ReplaceAll(e.lines[y], "\r", "")
		lineLength := len([]rune(lineContent))

		// Test positions from 0 to line length (including position after last character)
		for x := 0; x <= lineLength; x++ {
			totalTests++

			// Set temporary cursor position for mapping
			e.cursorY = y
			e.cursorX = x

			// Step 1: Map logical to visual
			wy, wx := e.mapLogicalToWrappedCursor()

			// Step 2: Map visual back to logical
			yPrime, xPrime := e.mapVisualToLogical(wy, wx)

			// Step 3: Verify round-trip consistency
			if y != yPrime || x != xPrime {
				errorCount++
				logger.Error(logger.AreaEditor, "MAPPING MISMATCH! Original: (%d,%d) -> Visual: (%d,%d) -> Result: (%d,%d) | Line: '%s'",
					y, x, wy, wx, yPrime, xPrime, lineContent)

				// Additional debug information for the failed case
				logger.Error(logger.AreaEditor, "Debug info - Line length: %d, Wrapped segments for line %d:",
					lineLength, y)

				// Show which wrapped lines belong to this logical line
				for idx, logicalLine := range e.lineMapping {
					if logicalLine == y {
						logger.Error(logger.AreaEditor, "  Wrapped[%d] -> Logical[%d]: '%s'",
							idx, logicalLine, e.wrappedLines[idx])
					}
				}
			}
		}
	}

	// Restore original cursor position
	e.cursorY = originalCursorY
	e.cursorX = originalCursorX

	// Report results
	if errorCount == 0 {
		logger.Info(logger.AreaEditor, "=== MAPPING VERIFICATION COMPLETED SUCCESSFULLY ===")
		logger.Info(logger.AreaEditor, "All %d position mappings are consistent", totalTests)
	} else {
		logger.Error(logger.AreaEditor, "=== MAPPING VERIFICATION FAILED ===")
		logger.Error(logger.AreaEditor, "Found %d inconsistencies out of %d tested positions", errorCount, totalTests)
	}
}

// focusedMappingTest performs a specific test for the problematic mapping case
// This function is available for debugging purposes but is not called in normal operation
func (e *Editor) focusedMappingTest() {
	if len(e.lines) <= 49 {
		return
	}

	// Test the specific problematic case: line 49, position 153
	originalY, originalX := 49, 153

	// Save current cursor
	savedY, savedX := e.cursorY, e.cursorX

	// Set test position
	e.cursorY = originalY
	e.cursorX = originalX

	// Map to visual
	visualY, visualX := e.mapLogicalToWrappedCursor()

	// Map back to logical
	resultY, resultX := e.mapVisualToLogical(visualY, visualX)

	// Only log if there's an actual error
	if originalY != resultY || originalX != resultX {
		// Log detailed analysis only on mismatch
		logger.Error(logger.AreaEditor, "=== FOCUSED MAPPING TEST FAILED ===")
		logger.Error(logger.AreaEditor, "Original: (%d,%d) -> Visual: (%d,%d) -> Result: (%d,%d)",
			originalY, originalX, visualY, visualX, resultY, resultX)

		// Analyze the original line
		originalLine := strings.ReplaceAll(e.lines[originalY], "\r", "")
		originalRunes := []rune(originalLine)

		logger.Error(logger.AreaEditor, "Original line length: %d", len(originalRunes))
		if originalX > 0 && originalX-1 < len(originalRunes) {
			logger.Error(logger.AreaEditor, "Character at position %d: '%c' (rune: %d)",
				originalX-1, originalRunes[originalX-1], originalRunes[originalX-1])
		}
		if originalX < len(originalRunes) {
			logger.Error(logger.AreaEditor, "Character at position %d: '%c' (rune: %d)",
				originalX, originalRunes[originalX], originalRunes[originalX])
		}

		// Show segment information
		for idx, logicalLine := range e.lineMapping {
			if logicalLine == originalY {
				logger.Error(logger.AreaEditor, "Segment[%d]: '%s' (length: %d)",
					idx, e.wrappedLines[idx], len([]rune(e.wrappedLines[idx])))
			}
		}
	} else {
		// Mapping is correct - no logging needed for successful tests
	}

	// Restore cursor
	e.cursorY = savedY
	e.cursorX = savedX
}
