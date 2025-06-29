package editor

import (
	"log"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// This file contains input handling functions for the editor

// handleControlKey processes control key combinations
func (e *Editor) handleControlKey(key string) bool {
	logger.Info(logger.AreaEditor, "handleControlKey: Processing key: %s", key)

	switch strings.ToLower(key) {
	case "s":
		// Save file (only if not read-only)
		if !e.readOnly {
			return e.handleSave()
		} else {
			e.sendStatusMessage("Read-only mode - cannot save")
			return true
		}
	case "x":
		// Exit editor
		logger.Info(logger.AreaEditor, "handleControlKey: Exit requested (CTRL+X)")
		return e.handleExit()
	case "c":
		// Force exit if exit warning is showing
		logger.Info(logger.AreaEditor, "handleControlKey: CTRL+C received, exitWarning=%v", e.showingExitWarning)
		if e.showingExitWarning {
			// Send stop command to frontend
			e.sendEditorMessage("stop", nil)
			e.sendMessage(shared.MessageTypeInputControl, "", map[string]interface{}{
				"inputEnabled": true,
			})
			e.active = false
			return false
		}
		return true
		// Note: Original Ctrl+C only worked when exit warning is showing in ProcessInput
	}
	return true
}

// handleSave saves the current file
func (e *Editor) handleSave() bool {
	log.Printf("[EDITOR-BACKEND] handleSave called, filename='%s', len=%d", e.filename, len(e.filename))
	if e.filename == "" || e.filename == "<current program>" || e.filename == "<new file>" {
		// No filename specified or special placeholder filename, request filename input
		log.Printf("[EDITOR-BACKEND] No real filename specified (filename='%s'), requesting filename input", e.filename)
		e.requestingFilename = true
		e.filenameInput = ""
		e.exitAfterSave = e.showingExitWarning // Remember if we should exit after save
		e.showingExitWarning = false           // Clear exit warning during filename input

		// Send filename input command to frontend
		e.sendEditorMessage("filename_input", map[string]interface{}{
			"prompt": "Save as: ",
		})
		return true
	}

	log.Printf("[EDITOR-BACKEND] Filename exists, proceeding with save: %s", e.filename)
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
		} else {
			log.Printf("[EDITOR-BACKEND] File saved successfully: %s", e.filename)
			e.showingExitWarning = false
			e.sendStatusMessage("File saved: " + e.filename)
			// Re-render to update status line
			e.Render()
		}
	case <-time.After(5 * time.Second):
		log.Printf("[EDITOR-BACKEND] Save operation timed out for: %s", e.filename)
		e.sendStatusMessage("Save timeout - try again")
	}

	return true
}

// handleExit exits the editor with unsaved changes warning
func (e *Editor) handleExit() bool {
	log.Printf("[EDITOR-BACKEND] handleExit called - modified: %v", e.modified)
	logger.Info(logger.AreaEditor, "handleExit called - modified: %v", e.modified)

	if e.modified {
		// Set exit warning state and re-render to show warning
		e.showingExitWarning = true
		logger.Info(logger.AreaEditor, "handleExit: File modified, showing exit warning")
		e.Render()
		return true
	}

	// WICHTIG: Editor sofort deaktivieren, bevor weitere Nachrichten gesendet werden
	e.active = false

	// Sende explizites "stop"-Kommando an das Frontend
	e.sendEditorMessage("stop", nil)

	// Exit editor - send proper message to re-enable input
	e.sendMessage(shared.MessageTypeInputControl, "", map[string]interface{}{
		"inputEnabled": true,
	})
	return false
}

// handleCancelExit cancels the exit warning and returns to normal editor mode
func (e *Editor) handleCancelExit() bool {
	log.Printf("[EDITOR-BACKEND] handleCancelExit called - canceling exit warning")
	e.showingExitWarning = false
	// Redraw with normal status line
	e.Render()
	return true
}
