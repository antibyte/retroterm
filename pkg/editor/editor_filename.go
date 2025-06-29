package editor

import (
	"log"
	"time"

	"github.com/antibyte/retroterm/pkg/shared"
)

// This file contains filename input handling functions for the editor

// handleFilenameInput handles input when requesting a filename
func (e *Editor) handleFilenameInput(input string) bool {
	switch input {
	case "Escape":
		// Cancel filename input
		e.requestingFilename = false
		e.filenameInput = ""
		e.exitAfterSave = false
		e.sendStatusMessage("Save cancelled")
		e.Render()
		return true
	case "Enter":
		// Confirm filename and save
		if e.filenameInput == "" {
			e.sendStatusMessage("Save as: (filename cannot be empty)")
			return true
		}
		// Set the filename and save
		e.filename = e.filenameInput
		e.requestingFilename = false

		// Save the file
		e.sendStatusMessage("Saving file...")
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
				e.sendStatusMessage("File saved: " + e.filename)

				// If we should exit after save (user pressed Ctrl+X then Ctrl+S)
				if e.exitAfterSave {
					e.exitAfterSave = false

					// Send stop command to frontend
					e.sendEditorMessage("stop", nil)
					// Re-enable normal input
					e.sendMessage(shared.MessageTypeInputControl, "", map[string]interface{}{
						"inputEnabled": true,
					})
					e.active = false
					return false
				}
				// Re-render to update status line
				e.Render()
			}
		case <-time.After(5 * time.Second):
			log.Printf("[EDITOR-BACKEND] Save operation timed out for: %s", e.filename)
			e.sendStatusMessage("Save timeout - try again")
		}
		return true
	case "Backspace":
		// Remove last character from filename input
		if len(e.filenameInput) > 0 {
			e.filenameInput = e.filenameInput[:len(e.filenameInput)-1]
		}
		e.sendStatusMessage("Save as: " + e.filenameInput)
		return true
	default:
		// Add character to filename (only if it's a single printable character)
		if len(input) == 1 && input[0] >= 32 && input[0] <= 126 {
			e.filenameInput += input
			e.sendStatusMessage("Save as: " + e.filenameInput)
		}
		return true
	}
}

// handleFilenameSubmit handles submission of filename
func (e *Editor) handleFilenameSubmit(data string) bool {
	filename := data // The filename is passed directly as data
	log.Printf("[EDITOR-BACKEND] handleFilenameSubmit called with filename: %q", filename)

	if filename == "" {
		e.sendStatusMessage("Save as: (filename cannot be empty)")
		return true
	}

	// Set the filename and clear filename input mode
	e.filename = filename
	e.requestingFilename = false
	e.filenameInput = ""

	// Send command to frontend to exit filename input mode
	e.sendEditorMessage("filename_input_complete", nil)

	// Now actually save the file
	if err := e.SaveFile(); err != nil {
		log.Printf("[EDITOR-BACKEND] Error saving file: %v", err)
		e.sendStatusMessage("Error saving: " + err.Error())
		return true
	}

	log.Printf("[EDITOR-BACKEND] File saved successfully: %s", e.filename)
	e.sendStatusMessage("File saved: " + e.filename)

	// If we were exiting before the save, exit now
	if e.exitAfterSave {

		e.sendEditorMessage("stop", nil)
		e.sendMessage(shared.MessageTypeInputControl, "", map[string]interface{}{
			"inputEnabled": true,
		})
		e.active = false
		return false
	}

	// Re-render to update status line
	e.Render()
	return true
}

// handleFilenameCancel handles cancellation of filename input
func (e *Editor) handleFilenameCancel() bool {
	log.Printf("[EDITOR-BACKEND] handleFilenameCancel called")

	// Clear filename input mode
	e.requestingFilename = false
	e.filenameInput = ""
	e.exitAfterSave = false

	// Send command to frontend to exit filename input mode
	e.sendEditorMessage("filename_input_complete", nil)

	e.sendStatusMessage("Save cancelled")
	e.Render()
	return true
}
