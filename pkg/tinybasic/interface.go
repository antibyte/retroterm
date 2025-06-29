package tinybasic

import (
	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// --- Communication Helpers ---
// sendMessage safely sends a message to the OutputChan, logging if blocked/dropped.
// Returns true if sent/queued, false if dropped.
func (b *TinyBASIC) sendMessage(msgType shared.MessageType, content string) bool {
	msg := shared.Message{Type: msgType, Content: content, SessionID: b.sessionID}
	return b.sendMessageObject(msg)
}

// sendMessageObject sends a pre-constructed message. Returns true if sent/queued, false if dropped.
func (b *TinyBASIC) sendMessageObject(msg shared.Message) bool {
	// Ensure the message has the correct SessionID if not already set
	if msg.SessionID == "" {
		msg.SessionID = b.sessionID
	}

	// Debug-Logging für Sprite-Nachrichten (disabled for performance)
	// if msg.Type == shared.MessageTypeSprite {
	//	fmt.Printf("[DEBUG-SPRITE-SEND] Sending sprite message: Type=%d, Command=%s, ID=%d, SessionID=%s\n",
	//		int(msg.Type), msg.Command, msg.ID, msg.SessionID)
	// }

	select {
	case b.OutputChan <- msg:
		return true // sent successfully
	default:
		contentPreview := msg.Content
		if len(contentPreview) > 50 {
			contentPreview = contentPreview[:50] + "..."
		}
		return false // dropped
	}
}

// sendInputControl sends enable/disable signal. Assumes lock is held.
// Returns true if sent/queued, false if dropped.
func (b *TinyBASIC) sendInputControl(state string) bool {
	select {
	case b.OutputChan <- shared.Message{Type: shared.MessageTypeInputControl, Content: state, SessionID: b.sessionID}:
		return true
	default:
		return false
	}
}

// sendMessageWrapped sendet eine text message mit automatischem Zeilenumbruch
// basierend auf der Terminalbreite
func (b *TinyBASIC) sendMessageWrapped(msgType shared.MessageType, content string) {
	logger.Debug(logger.AreaTinyBasic, "[SENDWRAPPED] Called with type=%d, content length=%d", msgType, len(content))

	if b.OutputChan == nil {
		logger.Debug(logger.AreaTinyBasic, "[SENDWRAPPED] Error: OutputChan is nil")
		return
	}

	// Debug: Check channel capacity
	logger.Debug(logger.AreaTinyBasic, "[SENDWRAPPED] OutputChan capacity: %d, current length: %d", cap(b.OutputChan), len(b.OutputChan))

	// Text umbrechen für Textnachrichten
	if msgType == shared.MessageTypeText && content != "" {
		logger.Debug(logger.AreaTinyBasic, "[SENDWRAPPED] Wrapping text")
		wrappedContent := b.wrapText(content)
		logger.Debug(logger.AreaTinyBasic, "[SENDWRAPPED] Text wrapped, sending message")
		// Nachricht senden
		success := b.sendMessage(msgType, wrappedContent)
		logger.Debug(logger.AreaTinyBasic, "[SENDWRAPPED] Message sent, success=%v", success)
	} else {
		logger.Debug(logger.AreaTinyBasic, "[SENDWRAPPED] Sending message directly")
		// Andere Nachrichten direkt senden
		success := b.sendMessage(msgType, content)
		logger.Debug(logger.AreaTinyBasic, "[SENDWRAPPED] Direct message sent, success=%v", success)
	}
}

// sendMessageWrapped2 ist ein temporärer Ersatz für sendMessageWrapped
func (b *TinyBASIC) sendMessageWrapped2(msgType shared.MessageType, content string, noNewline bool) {
	// Einfache Implementierung ohne Wrapping für den Start
	b.sendMessage(msgType, content)

	// Füge Zeilenumbruch hinzu wenn nötig
	if !noNewline && msgType == shared.MessageTypeText {
		b.sendMessage(shared.MessageTypeText, "\n")
	}
}
