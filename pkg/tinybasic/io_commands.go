package tinybasic

import (
	"fmt"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// cmdWait pausiert die Programmausführung für die angegebene Zeit in Millisekunden.
func (b *TinyBASIC) cmdWait(args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("WAIT")
	}

	val, err := b.evalExpression(args)
	if err != nil || !val.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("WAIT")
	}

	millis := int(val.NumValue)
	if millis < 0 || millis > 60000 {
		return NewBASICError(ErrCategorySyntax, "INVALID_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("WAIT")
	}

	// Release the lock before sleeping to avoid blocking the interpreter.
	b.mu.Unlock()
	time.Sleep(time.Duration(millis) * time.Millisecond)
	// Re-acquire the lock after sleeping.
	b.mu.Lock()

	return nil
}

// PrintItem represents a single item in a PRINT statement with its separator
type PrintItem struct {
	Value     string
	Separator rune // ';' for no space, ',' for tab spacing, 0 for end of statement
}

// cmdPrint prints values and strings.
func (b *TinyBASIC) cmdPrint(args string) error {
	// Debug-Log für PRINT-Befehl
	logger.Debug(logger.AreaTinyBasic, "[PRINT] cmdPrint called with args: '%s'", args)

	// PRINT ohne Argumente gibt nur eine leere Zeile aus
	if strings.TrimSpace(args) == "" {
		if b.printCursorOnSameLine {
			b.sendTextToClient("\n", true) // Send just newline if cursor was on same line
		} else {
			b.sendTextToClient("", false) // Send newline for empty PRINT
		}
		b.printCursorOnSameLine = false // Reset cursor state
		return nil
	}
	// Prüfe ob der gesamte PRINT-Befehl mit einem Trennzeichen endet
	args = strings.TrimSpace(args)
	endsWithSeparator := false

	if strings.HasSuffix(args, ";") {
		endsWithSeparator = true
		args = args[:len(args)-1]
		args = strings.TrimSpace(args)
	} else if strings.HasSuffix(args, ",") {
		endsWithSeparator = true
		args = args[:len(args)-1]
		args = strings.TrimSpace(args)
	}

	// Parse alle Items getrennt durch Semicolons oder Kommas
	var items []PrintItem
	pos := 0
	for pos < len(args) {
		// Überspringe Leerzeichen
		for pos < len(args) && (args[pos] == ' ' || args[pos] == '\t') {
			pos++
		}
		if pos >= len(args) {
			break
		}

		var item string
		var err error
		if args[pos] == '"' {
			// String literal parsen
			var advance int
			item, advance, err = parseString(args[pos:])
			if err != nil {
				return err
			}
			logger.Debug(logger.AreaTinyBasic, "[PRINT] Parsed string literal: '%s'", item)
			pos += advance
		} else {
			// Expression parsen bis zum nächsten Trennzeichen oder Ende
			// Respektiere dabei Klammern für Array-Zugriffe wie G(0,0)
			start := pos
			parenDepth := 0

			for pos < len(args) {
				char := args[pos]

				if char == '(' {
					parenDepth++
				} else if char == ')' {
					parenDepth--
				} else if parenDepth == 0 && (char == ';' || char == ',') {
					// Nur abbrechen bei Trennzeichen außerhalb von Klammern
					break
				}

				pos++
			}

			expr := strings.TrimSpace(args[start:pos])
			if expr == "" {
				break
			}

			// Expression auswerten
			val, err := b.evalExpression(expr)
			if err != nil {
				return err
			}

			item, _ = basicValueToString(val)
		}

		// Bestimme das Trennzeichen
		separator := rune(0) // Standard: Ende der Anweisung
		if pos < len(args) {
			if args[pos] == ';' || args[pos] == ',' {
				separator = rune(args[pos])
				pos++
			}
		}

		items = append(items, PrintItem{Value: item, Separator: separator})
	}
	// Baue die Ausgabe basierend auf den Trennzeichen auf
	var output strings.Builder
	for _, item := range items {
		output.WriteString(item.Value)

		// Füge entsprechende Abstände hinzu
		if item.Separator == ',' {
			// Komma: Tab-Abstand (14 Zeichen in klassischem BASIC)
			currentLen := len(item.Value)
			tabSize := 14
			spacesToAdd := tabSize - (currentLen % tabSize)
			if spacesToAdd == 0 {
				spacesToAdd = tabSize
			}
			output.WriteString(strings.Repeat(" ", spacesToAdd))
		} else if item.Separator == ';' {
			// Semikolon: kein zusätzlicher Abstand
		}
		// Kein Separator (Ende): kein zusätzlicher Abstand
	}
	outputText := output.String()
	logger.Debug(logger.AreaTinyBasic, "[PRINT] Final output text: '%s' (length: %d)", outputText, len(outputText))

	// Bestimme das noNewline Flag basierend auf Cursor-Status und Trennzeichen
	var sendNoNewline bool
	if b.printCursorOnSameLine {
		// Wenn der Cursor bereits auf derselben Zeile steht, sende immer mit noNewline=true
		sendNoNewline = true
	} else {
		// Normale PRINT-Ausgabe: noNewline nur wenn PRINT mit einem Trennzeichen endet
		sendNoNewline = endsWithSeparator
	}

	// Verwende die neue Textumbruch-Funktion
	b.sendTextToClientWrapped(outputText, sendNoNewline)

	// Set cursor state for next PRINT statement
	b.printCursorOnSameLine = endsWithSeparator

	return nil
}

// cmdInput handles console input. Assumes lock is held.
func (b *TinyBASIC) cmdInput(args string) error {
	// Syntax: INPUT [prompt_string;] var1 [, var2...]
	prompt := "? "
	varListStr := strings.TrimSpace(args)

	// Check for optional prompt string.
	if strings.HasPrefix(varListStr, "\"") {
		endQuote := strings.Index(varListStr[1:], "\"")
		if endQuote != -1 {
			sep := endQuote + 2 // Position after closing quote.
			if sep < len(varListStr) && varListStr[sep] == ';' {
				prompt = varListStr[1 : endQuote+1]                // Extract prompt.
				varListStr = strings.TrimSpace(varListStr[sep+1:]) // Get variables after ';'.
			}
			// If no semicolon, the whole thing might be just a prompt? No, standard needs var.
		}
	}

	if varListStr == "" {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("INPUT")
	}

	// For simplicity, handle only one variable per INPUT for now.
	// TODO: Extend INPUT to handle multiple comma-separated variables if needed.
	varName := strings.ToUpper(strings.TrimSpace(varListStr))
	if strings.Contains(varName, ",") {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("INPUT")
	}
	if !isValidVarName(varName) {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("INPUT")
	}

	b.inputVar = varName                                 // Set flag indicating interpreter is waiting.
	b.sendInputControl("disable")                        // Signal frontend to disable normal input handling.
	b.sendMessageWrapped(shared.MessageTypeText, prompt) // Send prompt.

	// Execution pauses here; runProgramInternal returns because inputVar is set.
	return nil
}

// sendTextToClient ersetzt durch eine sicherere Version, die mit verschiedenen Typen umgehen kann
func (b *TinyBASIC) sendTextToClient(text interface{}, noNewline bool) {
	var textString string

	// Typ-Konvertierung
	switch v := text.(type) {
	case string:
		textString = v
	case rune:
		textString = string(v)
	case int:
		textString = string(rune(v))
	default:
		textString = fmt.Sprintf("%v", v)
	}

	// Umbrochenen Text senden
	wrappedText := b.wrapText(textString)
	b.sendMessage(shared.MessageTypeText, wrappedText)

	// Zeilenumbruch hinzufügen, wenn erforderlich
	if !noNewline {
		b.sendMessage(shared.MessageTypeText, "\n")
	}
}

// sendEmptyLine sends an empty line to create a line break, even if the content is empty
func (b *TinyBASIC) sendEmptyLine() {
	message := shared.Message{
		Type:      shared.MessageTypeText,
		Content:   "",    // Empty content but will still create a new line
		NoNewline: false, // Explicitly request a newline
		SessionID: b.sessionID,
	}
	b.sendMessageObject(message)

}

// wrapText bricht Text an den Grenzen der Terminalbreite um
func (b *TinyBASIC) wrapText(text string) string {
	// Debug-Ausgabe begrenzen, um Log-Spam zu vermeiden

	// Prüfe, ob Text leer ist
	if text == "" {
		return ""
	} // Standard-Terminalbreite verwenden
	width := 80

	// Wenn der Text kürzer als die Breite ist, gib ihn unverändert zurück
	if len(text) <= width {
		return text
	}

	// Text in Zeilen aufspalten und jede Zeile umbrechen
	var result strings.Builder
	lines := strings.Split(text, "\n")

	for i, line := range lines {
		if len(line) <= width {
			result.WriteString(line)
		} else {
			// Längere Zeilen umbrechen
			currentLine := ""
			words := strings.Fields(line)

			for _, word := range words {
				// Bei zu langen Wörtern
				if len(word) > width {
					if currentLine != "" {
						result.WriteString(currentLine)
						result.WriteString("\n")
						currentLine = ""
					}

					// Langes Wort aufteilen
					for len(word) > width {
						result.WriteString(word[:width])
						result.WriteString("\n")
						word = word[width:]
					}
					currentLine = word
				} else if len(currentLine)+len(word)+1 > width {
					// Zeilenumbruch bei Überlauf
					result.WriteString(currentLine)
					result.WriteString("\n")
					currentLine = word
				} else {
					// Wort anhängen
					if currentLine == "" {
						currentLine = word
					} else {
						currentLine += " " + word
					}
				}
			}

			// Letzte Zeile hinzufügen
			if currentLine != "" {
				result.WriteString(currentLine)
			}
		}

		// Zeilenumbruch hinzufügen, außer bei der letzten Zeile
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	wrapped := result.String()

	return wrapped
}

// sendTextToClientDirect sendet Text direkt ohne zusätzlichen Text-Umbruch
func (b *TinyBASIC) sendTextToClientDirect(text string, noNewline bool) {

	// Erstelle die Nachricht direkt ohne Text-Wrapping
	message := shared.Message{
		Type:      shared.MessageTypeText,
		Content:   text,
		NoNewline: noNewline, // Verwende das NoNewline-Feld der Message-Struktur
		SessionID: b.sessionID,
		Inverse:   b.inverseTextMode, // NEU: Inverser Modus
	}

	// Sende die Nachricht
	b.sendMessageObject(message)

}

// sendTextToClientWrapped sendet Text mit automatischem Umbruch
func (b *TinyBASIC) sendTextToClientWrapped(text string, noNewline ...bool) {
	shouldNotAddNewline := len(noNewline) > 0 && noNewline[0]

	if text == "" {
		b.sendTextToClient(text, shouldNotAddNewline)
		return
	}

	// Text umbrechen
	wrappedText := b.wrapText(text)

	// Umgebrochenen Text in Zeilen aufteilen
	wrappedLines := strings.Split(wrappedText, "\n")

	for i, line := range wrappedLines {
		isLastLine := i == len(wrappedLines)-1
		if isLastLine && shouldNotAddNewline {
			// Letzte Zeile ohne Zeilenumbruch senden - verwende sendTextToClientDirect
			b.sendTextToClientDirect(line, true)
		} else {
			// Normale Zeile mit Zeilenumbruch senden - verwende sendTextToClientDirect
			b.sendTextToClientDirect(line, false)
		}
	}
}
