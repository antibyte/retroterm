package tinybasic

import (
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/shared"
)

// cmdBeep sends a beep signal. No lock needed.
func (b *TinyBASIC) cmdBeep(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("BEEP")
	}

	beepMsg := shared.Message{
		Type:    shared.MessageTypeSound,
		Content: "beep", // Signalisiert einen einfachen Beep
	}
	b.sendMessageObject(beepMsg)
	return nil
}

// cmdSound evaluates arguments for SOUND and sends a sound message. Assumes lock is held.
func (b *TinyBASIC) cmdSound(args string) error {
	// Syntax: SOUND <freq_expr>, <duration_expr>
	parts := strings.SplitN(args, ",", 2)
	if len(parts) != 2 {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("SOUND")
	}
	freqExpr := strings.TrimSpace(parts[0])
	durExpr := strings.TrimSpace(parts[1])

	freqVal, err := b.evalExpression(freqExpr)
	if err != nil || !freqVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("SOUND").WithUsageHint("Frequency expression is invalid.")
	}
	durVal, err := b.evalExpression(durExpr)
	if err != nil || !durVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("SOUND").WithUsageHint("Duration expression is invalid.")
	}

	soundMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"frequency": freqVal.NumValue,
			"duration":  durVal.NumValue,
		},
	}
	if !b.sendMessageObject(soundMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("SOUND")
	}

	return nil
}

// cmdNoise sends a noise signal. No lock needed.
// Syntax: NOISE <pitch_expr>, <attack_expr>, <decay_expr>
func (b *TinyBASIC) cmdNoise(args string) error {
	// Syntax: NOISE <pitch_expr>, <attack_expr>, <decay_expr>
	parts := splitRespectingParentheses(args)
	if len(parts) != 3 {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("NOISE").WithUsageHint("NOISE <pitch>, <attack>, <decay>")
	}

	pitchExpr := strings.TrimSpace(parts[0])
	attackExpr := strings.TrimSpace(parts[1])
	decayExpr := strings.TrimSpace(parts[2])

	pitchVal, err := b.evalExpression(pitchExpr)
	if err != nil || !pitchVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("NOISE").WithUsageHint("Pitch expression is invalid.")
	}
	attackVal, err := b.evalExpression(attackExpr)
	if err != nil || !attackVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("NOISE").WithUsageHint("Attack expression is invalid.")
	}
	decayVal, err := b.evalExpression(decayExpr)
	if err != nil || !decayVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("NOISE").WithUsageHint("Decay expression is invalid.")
	}

	noiseMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"type":   "noise",
			"pitch":  pitchVal.NumValue,
			"attack": attackVal.NumValue,
			"decay":  decayVal.NumValue,
		},
	}
	if !b.sendMessageObject(noiseMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("NOISE")
	}

	return nil
}

// cmdSpeak evaluates argument for SAY/SPEAK. Returns text string. Implements robust SAY_DONE Synchronisation.
func (b *TinyBASIC) cmdSpeak(args string) (string, error) {
	// Versuch, die Mutex zu sperren - führt nur den Lock durch, wenn die Sperre nicht bereits
	// von der aufrufenden Funktion gehalten wird. Dies verhindert eine Deadlock-Situation.
	var hasExternalLock bool
	var lockAttemptSuccessful bool

	// Wenn die Operation tryLock unterstützt würde (Go hat dies nicht standardmäßig),
	// könnten wir tryLock verwenden, um zu prüfen, ob die Sperre bereits gehalten wird
	// Stattdessen müssen wir dies über die Umgebung der aufrufenden Funktion wissen.

	// Die externen Locks werden in executeSingleStatementInternal und executeStatement verwaltet.
	// In tinybasic.go wurden die Mutex-Sperren für SAY bereits entfernt, da executeStatement sie handhabt.
	// Daher nehmen wir an, dass die Sperre bereits von außen gehalten wird.
	hasExternalLock = true

	if !hasExternalLock {
		// Nur wenn keine externe Sperre existiert, sperren wir selbst
		b.mu.Lock()
		lockAttemptSuccessful = true
		defer func() {
			if lockAttemptSuccessful {
				b.mu.Unlock()
			}
		}()
	}

	// --- Teil 1: Rate Limiting, initiale Prüfungen und Argument-Parsing (benötigt Sperre) ---

	// Rate Limiting prüfen
	now := time.Now()
	validTimestamps := make([]time.Time, 0, len(b.sayCommandTimestamps))
	for _, ts := range b.sayCommandTimestamps {
		if now.Sub(ts) <= time.Second {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	b.sayCommandTimestamps = validTimestamps

	if len(b.sayCommandTimestamps) >= b.maxSayRatePerSecond {
		return "", nil // Befehl stillschweigend ignorieren
	}
	b.sayCommandTimestamps = append(b.sayCommandTimestamps, now)

	// Prüfe auf leere Eingabe
	if strings.TrimSpace(args) == "" {
		return "", NewBASICError(ErrCategorySyntax, "INVALID_ARGUMENT", b.currentLine == 0, b.currentLine).
			WithCommand("SAY/SPEAK").
			WithUsageHint("SAY \"text\"[, WAIT]")
	}

	// Standardwert für waitForSpeech setzen
	waitForSpeech := false
	textToEvaluate := args // Dieser Text wird an evalExpression übergeben

	// Suche nach dem WAIT-Parameter (case-insensitive mit flexiblem Whitespace)
	waitPatterns := []string{
		`(?i),\s*wait\s*$`, // ", WAIT" oder ",WAIT" am Ende
		`(?i)\s+wait\s*$`,  // " WAIT" am Ende (ohne Komma)
	}

	for _, pattern := range waitPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(textToEvaluate) {
			waitForSpeech = true
			textToEvaluate = re.ReplaceAllString(textToEvaluate, "")
			textToEvaluate = strings.TrimSpace(textToEvaluate)
			break
		}
	}

	// Erfasse currentLine für mögliche Fehlererstellung nach Freigabe der Sperre
	currentLineForError := b.currentLine
	isDirectMode := b.currentLine == 0

	// --- Teil 2: Ausdrucksauswertung (evalExpression behandelt seine eigene Sperre) ---
	// Sperre während der Auswertung freigeben
	if hasExternalLock {
		b.mu.Unlock() // Bestehende externe Sperre freigeben
	}

	textVal, err := b.evalExpression(textToEvaluate)
	if err != nil {
		// Keine Sperre gehalten, Fehler mit erfasster currentLine erstellen
		if hasExternalLock {
			b.mu.Lock() // Sperre für Aufrufer wiederherstellen
		}
		return "", NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", isDirectMode, currentLineForError).
			WithCommand("SAY/SPEAK")
	}

	// --- Teil 3: Finale Verarbeitung, Nachrichtenversand, Setzen der WAIT-Flags (benötigt Sperre) ---
	// Sperre für die Nachrichtenverarbeitung wiederherstellen
	if hasExternalLock {
		b.mu.Lock() // Bestehende externe Sperre wiederherstellen
	}

	var finalText string
	if textVal.IsNumeric {
		finalText = formatBasicFloat(textVal.NumValue)
	} else {
		finalText = textVal.StrValue
	}

	// Längenbegrenzung prüfen
	const maxSayTextLength = 256
	if len(finalText) > maxSayTextLength {
		return "", NewBASICError(ErrCategoryCommand, "TEXT_TOO_LONG", b.currentLine == 0, b.currentLine).WithCommand("SAY")
	}

	normalizedText := normalizeTextForSpeech(finalText)

	// Unique ID für diese Sprachausgabe generieren
	b.lastSpeechID++
	speechID := b.lastSpeechID
	b.lastSayText = normalizedText // Nützlich für Timeout-Schätzung, falls WAIT verwendet wird

	speechMsg := shared.Message{
		Type:     shared.MessageTypeSpeak, // Korrigiert von shared.MessageTypeSay
		Content:  normalizedText,
		SpeechID: speechID, // Korrigiert von SayID zu SpeechID
	}

	// Nachricht senden. Wenn nicht erfolgreich, Fehler zurückgeben.
	if !b.sendMessageObject(speechMsg) {
		return "", NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("SAY")
	}

	if waitForSpeech {
		// Timeout festlegen (z.B. 10 Sekunden + 1 Sekunde pro 20 Zeichen)
		timeoutMs := 10000 + (len(normalizedText) * 50) // 50ms pro Zeichen
		b.waitForSpeechDone = speechID                  // Die ID, auf deren SAY_DONE wir warten
		b.speechTimeout = time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
		b.waitingForSayDone = true // Signalisiert der Ausführungsschleife zu warten

	}

	return "", nil
}

// normalizeTextForSpeech normalisiert Text für den SAM-Synthesizer
func normalizeTextForSpeech(text string) string {
	// Umlaute ersetzen
	replacements := map[string]string{
		"ä": "ae", "ö": "oe", "ü": "ue",
		"Ä": "Ae", "Ö": "Oe", "Ü": "Ue",
		"ß": "ss",
		// Weitere Sonderzeichen könnten hier hinzugefügt werden
	}

	for k, v := range replacements {
		text = strings.ReplaceAll(text, k, v)
	}

	// Nur ASCII-Zeichen behalten und Sonderzeichen filtern, die SAM verwirren könnten
	var result strings.Builder
	for _, r := range text {
		if r >= 32 && r < 127 {
			result.WriteRune(r)
		} else {
			// Ersetze Nicht-ASCII durch Leerzeichen
			result.WriteRune(' ')
		}
	}

	return strings.TrimSpace(result.String())
}

// truncateString kürzt einen String für Log-Ausgaben
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// cmdMusic handles all MUSIC commands for SID file playback
// Syntax: MUSIC OPEN "filename.sid"
//
//	MUSIC PLAY
//	MUSIC STOP
//	MUSIC PAUSE
func (b *TinyBASIC) cmdMusic(args string) error {
	args = strings.TrimSpace(args)
	if args == "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("MUSIC").WithUsageHint("MUSIC OPEN \"filename.sid\" | MUSIC PLAY | MUSIC STOP | MUSIC PAUSE")
	}

	// Parse subcommand
	parts := strings.SplitN(args, " ", 2)
	subcommand := strings.ToUpper(strings.TrimSpace(parts[0]))

	switch subcommand {
	case "OPEN":
		return b.cmdMusicOpen(strings.TrimSpace(parts[1]))
	case "PLAY":
		return b.cmdMusicPlay()
	case "STOP":
		return b.cmdMusicStop()
	case "PAUSE":
		return b.cmdMusicPause()
	default:
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("MUSIC").WithUsageHint("Valid subcommands: OPEN, PLAY, STOP, PAUSE")
	}
}

// cmdMusicOpen loads a SID file for playback
func (b *TinyBASIC) cmdMusicOpen(args string) error {
	if args == "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("MUSIC OPEN").WithUsageHint("MUSIC OPEN \"filename.sid\"")
	}

	// Evaluate the filename expression
	filenameVal, err := b.evalExpression(args)
	if err != nil || filenameVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("MUSIC OPEN").WithUsageHint("Filename must be a string expression.")
	}
	filename := filenameVal.StrValue
	if filename == "" {
		return NewBASICError(ErrCategorySyntax, "INVALID_ARGUMENT", b.currentLine == 0, b.currentLine).WithCommand("MUSIC OPEN").WithUsageHint("Filename cannot be empty.")
	}
	// Always stop any previous music before opening a new file
	// This ensures clean state for the frontend sound system
	stopMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"action": "music_stop",
		},
	}
	b.sendMessageObject(stopMsg) // Don't check return value, continue even if stop fails

	// Release the lock before sleeping to avoid blocking the interpreter.
	b.mu.Unlock()
	// Small delay to allow frontend to process the stop command
	// This works around potential frontend sound system state issues
	time.Sleep(200 * time.Millisecond)
	// Re-acquire the lock after sleeping.
	b.mu.Lock()

	musicMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"action":   "music_open",
			"filename": filename,
		},
	}
	log.Printf("[DEBUG-MUSIC] Sending MUSIC OPEN command: %s", filename)
	if !b.sendMessageObject(musicMsg) {
		log.Printf("[DEBUG-MUSIC] MUSIC OPEN failed to send message")
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("MUSIC OPEN")
	}
	log.Printf("[DEBUG-MUSIC] MUSIC OPEN message sent successfully")

	return nil
}

// cmdMusicPlay starts or resumes SID music playback
func (b *TinyBASIC) cmdMusicPlay() error {
	musicMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"action": "music_play",
		},
	}
	log.Printf("[DEBUG-MUSIC] Sending MUSIC PLAY command")
	if !b.sendMessageObject(musicMsg) {
		log.Printf("[DEBUG-MUSIC] MUSIC PLAY failed to send message")
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("MUSIC PLAY")
	}
	log.Printf("[DEBUG-MUSIC] MUSIC PLAY message sent successfully")

	return nil
}

// cmdMusicStop stops SID music playback
func (b *TinyBASIC) cmdMusicStop() error {
	musicMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"action": "music_stop",
		},
	}
	if !b.sendMessageObject(musicMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("MUSIC STOP")
	}

	return nil
}

// cmdMusicPause pauses SID music playback
func (b *TinyBASIC) cmdMusicPause() error {
	musicMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"action": "music_pause",
		},
	}
	if !b.sendMessageObject(musicMsg) {
		return NewBASICError(ErrCategorySystem, "MESSAGE_SEND_FAILED", b.currentLine == 0, b.currentLine).WithCommand("MUSIC PAUSE")
	}

	return nil
}
