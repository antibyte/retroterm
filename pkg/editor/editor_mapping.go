package editor

import (
	"strings"

	"github.com/antibyte/retroterm/pkg/logger"
)

// mapLogicalToWrappedCursor maps logical cursor position to wrapped line position
func (e *Editor) mapLogicalToWrappedCursor() (wrappedLineIdx, wrappedCol int) {
	// Default to valid position
	wrappedLineIdx = 0
	wrappedCol = 0

	// Validate input and wrapped lines data
	if e.cursorY < 0 || e.cursorY >= len(e.lines) {
		logger.Warn(logger.AreaEditor, "mapLogicalToWrappedCursor: Invalid cursor line %d (total lines: %d), using (0,0)",
			e.cursorY, len(e.lines))
		return
	}

	if len(e.wrappedLines) == 0 || len(e.lineMapping) == 0 {
		logger.Warn(logger.AreaEditor, "mapLogicalToWrappedCursor: No wrapped lines available - wrappedLines=%d, lineMapping=%d, using (0,0)",
			len(e.wrappedLines), len(e.lineMapping))
		return
	}

	// First step: Find all wrapped lines that correspond to the current logical line
	wrappedIndices := []int{}
	for idx, logicalLine := range e.lineMapping {
		if logicalLine == e.cursorY {
			wrappedIndices = append(wrappedIndices, idx)
		}
	}

	// If we couldn't find any wrapped indices for this line, return default value
	if len(wrappedIndices) == 0 {
		logger.Warn(logger.AreaEditor, "mapLogicalToWrappedCursor: No wrapped indices found for logical line %d",
			e.cursorY)
		return
	}
	// Get the original line content and check cursor X bounds
	originalLine := strings.ReplaceAll(e.lines[e.cursorY], "\r", "")
	originalRunes := []rune(originalLine)
	originalLength := len(originalRunes)

	// Use cursor X as-is for mapping (don't modify it here)
	// The cursor movement logic should handle boundary checks
	clampedCursorX := e.cursorX
	if clampedCursorX < 0 {
		clampedCursorX = 0
	}
	if clampedCursorX > originalLength {
		clampedCursorX = originalLength
	}

	// Calculate which wrapped segment contains our cursor and what column within that segment
	logicalPosition := 0
	currentSegment := 0
	logger.Debug(logger.AreaEditor, "mapLogicalToWrappedCursor: Mapping logical (%d,%d) to wrapped position",
		clampedCursorX, e.cursorY)

	// Process each wrapped segment to find where cursor is located
	for currentSegment < len(wrappedIndices) {
		wrappedIdx := wrappedIndices[currentSegment]
		if wrappedIdx >= len(e.wrappedLines) {
			logger.Error(logger.AreaEditor, "mapLogicalToWrappedCursor: Wrapped index %d out of bounds (max: %d)",
				wrappedIdx, len(e.wrappedLines)-1)
			break
		}

		segmentText := e.wrappedLines[wrappedIdx]
		segmentRunes := []rune(segmentText)
		segmentLength := len(segmentRunes)

		// Calculate how many characters from original line this segment represents
		segmentEndPosition := logicalPosition + segmentLength

		// Check if we need to account for a space that was removed during wrapping
		spaceRemoved := false
		if currentSegment < len(wrappedIndices)-1 { // Not the last segment
			// Check if there's a space at the boundary position in the original text
			if segmentEndPosition < originalLength &&
				(originalRunes[segmentEndPosition] == ' ' || originalRunes[segmentEndPosition] == '\t') {
				spaceRemoved = true
				segmentEndPosition++ // Account for the space in logical positioning
			}
		}
		// Check if cursor is within this segment
		if clampedCursorX >= logicalPosition && clampedCursorX < segmentEndPosition {
			wrappedLineIdx = wrappedIdx
			wrappedCol = clampedCursorX - logicalPosition

			// If we removed a space and cursor is at the space position,
			// place cursor at end of current segment
			if spaceRemoved && clampedCursorX == segmentEndPosition-1 {
				wrappedCol = segmentLength
			}

			logger.Debug(logger.AreaEditor, "mapLogicalToWrappedCursor: Found cursor in segment %d: logical (%d,%d) -> wrapped (%d,%d)",
				currentSegment, clampedCursorX, e.cursorY, wrappedCol, wrappedLineIdx)
			return
		}

		// Move to next segment
		logicalPosition = segmentEndPosition
		currentSegment++
	}

	// If we get here, cursor is after the last text segment
	// Place it at the end of the last wrapped segment
	if len(wrappedIndices) > 0 {
		lastWrappedIdx := wrappedIndices[len(wrappedIndices)-1]
		wrappedLineIdx = lastWrappedIdx

		if lastWrappedIdx < len(e.wrappedLines) {
			lastSegment := e.wrappedLines[lastWrappedIdx]
			wrappedCol = len([]rune(lastSegment))
		}

		logger.Debug(logger.AreaEditor, "mapLogicalToWrappedCursor: Cursor at end of line - mapped to wrapped position (%d,%d)",
			wrappedCol, wrappedLineIdx)
	}

	return
}

// mapVisualToLogical maps wrapped line position back to logical cursor position
// This is the inverse function of mapLogicalToWrappedCursor
func (e *Editor) mapVisualToLogical(wrappedLineIdx, wrappedCol int) (logicalY, logicalX int) {
	// Default to valid position
	logicalY = 0
	logicalX = 0

	// Validate input
	if wrappedLineIdx < 0 || wrappedLineIdx >= len(e.wrappedLines) {
		logger.Warn(logger.AreaEditor, "mapVisualToLogical: Invalid wrapped line index %d (max: %d), using (0,0)",
			wrappedLineIdx, len(e.wrappedLines)-1)
		return
	}

	if len(e.lineMapping) == 0 || len(e.wrappedLines) == 0 {
		logger.Warn(logger.AreaEditor, "mapVisualToLogical: No wrapped lines available - wrappedLines=%d, lineMapping=%d, using (0,0)",
			len(e.wrappedLines), len(e.lineMapping))
		return
	}

	// Get the logical line from line mapping
	if wrappedLineIdx >= len(e.lineMapping) {
		logger.Warn(logger.AreaEditor, "mapVisualToLogical: Wrapped line index %d exceeds lineMapping length %d, using (0,0)",
			wrappedLineIdx, len(e.lineMapping))
		return
	}

	logicalY = e.lineMapping[wrappedLineIdx]

	// Validate logical line index
	if logicalY < 0 || logicalY >= len(e.lines) {
		logger.Warn(logger.AreaEditor, "mapVisualToLogical: Invalid logical line %d (max: %d), using (0,0)",
			logicalY, len(e.lines)-1)
		logicalY = 0
		return
	}

	// Find all segments that belong to this logical line
	var segmentIndices []int
	for i := 0; i < len(e.lineMapping); i++ {
		if e.lineMapping[i] == logicalY {
			segmentIndices = append(segmentIndices, i)
		}
	}

	// Find which segment number our target wrapped line is
	targetSegmentNumber := -1
	for segNum, segIdx := range segmentIndices {
		if segIdx == wrappedLineIdx {
			targetSegmentNumber = segNum
			break
		}
	}

	if targetSegmentNumber == -1 {
		logger.Error(logger.AreaEditor, "mapVisualToLogical: Could not find segment number for wrapped line %d", wrappedLineIdx)
		return
	}

	// Get original line for reference
	originalLine := strings.ReplaceAll(e.lines[logicalY], "\r", "")
	originalRunes := []rune(originalLine)

	// Calculate logical position: start with column within current segment
	logicalX = wrappedCol

	// Add lengths of all previous segments, accounting for removed spaces
	logicalPosition := 0
	for segNum := 0; segNum < targetSegmentNumber; segNum++ {
		segIdx := segmentIndices[segNum]
		if segIdx < len(e.wrappedLines) {
			segmentText := e.wrappedLines[segIdx]
			segmentLength := len([]rune(segmentText))
			logicalPosition += segmentLength

			// Check if there was a space removed between this segment and the next
			if segNum < len(segmentIndices)-1 { // Not the last segment
				if logicalPosition < len(originalRunes) &&
					(originalRunes[logicalPosition] == ' ' || originalRunes[logicalPosition] == '\t') {
					logicalPosition++ // Account for the removed space
				}
			}
		}
	}

	// Add the column position within the target segment
	logicalX = logicalPosition + wrappedCol

	// Validate the result against original line length
	if logicalX > len(originalRunes) {
		logger.Warn(logger.AreaEditor, "mapVisualToLogical: Calculated logicalX %d exceeds line length %d, clamping",
			logicalX, len(originalRunes))
		logicalX = len(originalRunes)
	}

	logger.Debug(logger.AreaEditor, "mapVisualToLogical: Wrapped (%d,%d) -> logical (%d,%d), segment %d/%d",
		wrappedCol, wrappedLineIdx, logicalX, logicalY, targetSegmentNumber+1, len(segmentIndices))

	return
}
