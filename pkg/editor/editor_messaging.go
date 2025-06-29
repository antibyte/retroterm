package editor

import (
	"log"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// This file contains messaging functions for communication with the frontend

// sendEditorMessage sends an editor-specific message to the frontend
func (e *Editor) sendEditorMessage(command string, data map[string]interface{}) {
	log.Printf("[EDITOR-BACKEND] sendEditorMessage called with command: %s", command)
	msg := shared.Message{
		Type:          20, // shared.MessageTypeEditor = 20
		EditorCommand: command,
		SessionID:     e.sessionID,
	}

	if data != nil {
		msg.Params = data
		log.Printf("[EDITOR-BACKEND] Message params: %+v", data)
		// Also populate specific fields for easier access
		if filename, ok := data["filename"].(string); ok {
			msg.EditorFile = filename
		}
		if rows, ok := data["rows"].(int); ok {
			msg.EditorRows = rows
		}
		if cols, ok := data["cols"].(int); ok {
			msg.EditorCols = cols
		}
		if cursorLine, ok := data["cursorLine"].(int); ok {
			msg.CursorLine = cursorLine
		}
		if cursorCol, ok := data["cursorCol"].(int); ok {
			msg.CursorCol = cursorCol
		}
		if status, ok := data["status"].(string); ok {
			msg.EditorStatus = status
		}
		if modified, ok := data["modified"].(bool); ok {
			msg.EditorMod = modified
		}
	}

	select {
	case e.outputChan <- msg:
		log.Printf("[EDITOR-BACKEND] Message sent to outputChan successfully")
	default:
		log.Printf("[EDITOR-BACKEND] WARNING: outputChan full, message dropped!")
	}
}

// sendStatusMessage sends a status message that will be displayed in the status line
func (e *Editor) sendStatusMessage(status string) {
	e.sendEditorMessage("status", map[string]interface{}{
		"status": status,
	})
}

// sendCursorState sends a message to show or hide the cursor
func (e *Editor) sendCursorState(visible bool) {
	if !e.active {
		return
	}

	action := "show"
	if !visible {
		action = "hide"
	}

	logger.Info(logger.AreaEditor, "Sending cursor %s message", action)
	e.sendMessage(22, action, nil) // 22 = shared.MessageTypeCursor
}

// sendMessage sends a message to the client
func (e *Editor) sendMessage(msgType shared.MessageType, content string, params ...map[string]interface{}) {
	msg := shared.Message{
		Type:      msgType,
		Content:   content,
		SessionID: e.sessionID,
	}

	if len(params) > 0 {
		msg.Params = params[0]
	}

	select {
	case e.outputChan <- msg:
		// Message sent successfully
	default:
		// Channel full, drop message
	}
}

// GetOutputChannel returns the output channel for reading editor messages
func (e *Editor) GetOutputChannel() <-chan shared.Message {
	return e.outputChan
}
