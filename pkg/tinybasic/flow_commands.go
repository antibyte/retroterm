package tinybasic

import (
	"fmt" // Added for debugFP
	"strconv"
	"strings"
	"time" // Re-added for debug logging
)

const Epsilon = 1e-9 // A small tolerance for floating point comparisons

// ForLoopInfo stores the state of an active FOR loop.
type ForLoopInfo struct {
	Variable              string  // Loop control variable name (uppercase).
	EndValue              float64 // Target value for the loop variable.
	Step                  float64 // Increment/decrement value per iteration.
	StartLine             int     // The line number *after* the FOR statement (first line of the loop body in multi-line loops).
	ForLineNum            int     // The line number of the FOR statement itself.
	BodySubStatementIndex int     // Index (0-based) of the sub-statement AFTER the FOR on the SAME line. -1 if FOR is the last statement or single statement on the line.
	GosubDepth            int     // GOSUB stack depth at the time of FOR loop creation.
}

// cmdGoto performs an unconditional jump. Assumes lock is held.
func (b *TinyBASIC) cmdGoto(args string) error {
	targetLine, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil || targetLine <= 0 {
		return NewBASICError(ErrCategorySyntax, "INVALID_LINE_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("GOTO")
	}
	if _, exists := b.program[targetLine]; !exists {
		return NewBASICError(ErrCategoryExecution, "LINE_NOT_FOUND", b.currentLine == 0, b.currentLine).WithCommand("GOTO")
	}
	// Clean up FOR loops that would be skipped by this GOTO
	// This prevents stack overflow when GOTO jumps backwards in main loops
	if err := b.cleanupForLoopsOnGoto(b.currentLine, targetLine); err != nil {
		return err
	}

	b.currentLine = targetLine // Set program counter for next iteration.
	return nil
}

// cmdGosub calls a subroutine. Assumes lock is held.
func (b *TinyBASIC) cmdGosub(args string) error {
	targetLine, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil || targetLine <= 0 {
		return NewBASICError(ErrCategorySyntax, "INVALID_LINE_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("GOSUB")
	}
	if _, exists := b.program[targetLine]; !exists {
		return NewBASICError(ErrCategoryExecution, "LINE_NOT_FOUND", b.currentLine == 0, b.currentLine).WithCommand("GOSUB")
	}
	if len(b.gosubStack) >= MaxGosubDepth {
		return ErrGosubDepthExceeded
	}
	// Find the line number *after* the GOSUB to return to.
	returnLine, found := b.findNextLine(b.currentLine)
	if !found {
		returnLine = 0 // Use 0 to signify return should end program.
	}
	b.gosubStack = append(b.gosubStack, returnLine)

	b.currentLine = targetLine // Jump to subroutine.

	return nil
}

// cmdReturn returns from a subroutine. Assumes lock is held.
func (b *TinyBASIC) cmdReturn(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("RETURN")
	}
	if len(b.gosubStack) == 0 {
		return NewBASICError(ErrCategoryExecution, "RETURN_WITHOUT_GOSUB", b.currentLine == 0, b.currentLine).WithCommand("RETURN")
	}
	lastIndex := len(b.gosubStack) - 1
	returnLine := b.gosubStack[lastIndex]
	currentGosubDepth := len(b.gosubStack) // Capture GOSUB depth *before* popping

	b.gosubStack = b.gosubStack[:lastIndex] // Pop from stack.

	// Bereinige FOR-Schleifen, die nicht mehr zur aktuellen Ausführungsebene gehören
	b.cleanupForLoopsOnReturn(currentGosubDepth)

	b.currentLine = returnLine // Set program counter for next iteration (might be 0).
	return nil
}

// evalIfCondition evaluates the condition part of an IF statement. Assumes lock is held.
func (b *TinyBASIC) evalIfCondition(args string) (ConditionResult, error) {
	// Find the position of "THEN" keyword
	upperArgs := strings.ToUpper(args)
	thenPos := strings.Index(upperArgs, "THEN")

	if thenPos == -1 {
		return ConditionResult{}, NewBASICError(ErrCategorySyntax, "EXPECTED_THEN", b.currentLine == 0, b.currentLine).WithCommand("IF")
	}

	// Extract the condition expression before "THEN"
	condExpr := strings.TrimSpace(args[:thenPos])

	// Extract everything after "THEN"
	afterThen := ""
	if thenPos+4 < len(args) { // "THEN" has 4 characters
		afterThen = args[thenPos+4:]
	}

	// Parse ELSE: Wir müssen aufpassen, dass wir nicht ELSE in Strings oder innerhalb anderer Befehle erwischen
	thenStmt := ""
	elseStmt := ""
	hasElse := false

	// Verarbeitung des Teils nach THEN
	if afterThen != "" {
		// Suche nach ELSE, das nicht Teil eines Strings oder anderen Tokens ist
		inString := false
		elsePos := -1
		i := 0

		for i < len(afterThen) {
			// Ignoriere Leerzeichen
			if afterThen[i] == ' ' || afterThen[i] == '\t' {
				i++
				continue
			}

			// String-Literal erkennen
			if afterThen[i] == '"' {
				inString = !inString
				i++
				continue
			}

			// ELSE nur außerhalb von Strings erkennen
			if !inString && i+4 <= len(afterThen) && strings.ToUpper(afterThen[i:i+4]) == "ELSE" {
				// Prüfen, ob es sich um ein vollständiges Token handelt
				isFullToken := true
				if i+4 < len(afterThen) {
					nextChar := afterThen[i+4]
					if (nextChar >= 'a' && nextChar <= 'z') || (nextChar >= 'A' && nextChar <= 'Z') ||
						(nextChar >= '0' && nextChar <= '9') || nextChar == '_' {
						isFullToken = false // Teil eines längeren Tokens
					}
				}

				if isFullToken {
					elsePos = i
					break
				}
			}
			i++
		}

		// ELSE gefunden
		if elsePos >= 0 {
			thenStmt = strings.TrimSpace(afterThen[:elsePos])
			elseStmt = strings.TrimSpace(afterThen[elsePos+4:]) // "ELSE" hat 4 Zeichen
			hasElse = true
		} else {
			thenStmt = strings.TrimSpace(afterThen)
		}
	}

	if condExpr == "" {
		return ConditionResult{}, NewBASICError(ErrCategorySyntax, "EXPECTED_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("IF")
	}
	// Eine leere THEN-Anweisung ist erlaubt (NOP/noop)
	// In BASIC ist "IF x THEN" gültig und tut nichts, wenn x wahr ist

	// ===== Behandlung für komplexe logische Ausdrücke (AND/OR) =====
	// Verbesserte Erkennung von logischen Operatoren
	// Wichtig: Wir erkennen AND und OR nur als eigenständige Tokens zwischen Ausdrücken

	// Normalisieren und prüfen auf logische Operatoren
	normExpr := normalizeLogicalExpression(condExpr)
	hasAnd := strings.Contains(normExpr, " AND ")
	hasOr := strings.Contains(normExpr, " OR ")

	// Wenn logische Operatoren vorhanden sind, verwende die direkte Auswertungsmethode
	if hasAnd || hasOr {
		// Auswertung mit der neuen direkten Implementierung
		result, err := b.evalDirectLogical(condExpr)
		if err != nil {
			// Bei Fehlern in der direkten Auswertung, versuche den Standard-Parser
			// Standardparser verwenden
			condValue, err := b.evalExpression(condExpr)
			if err != nil {
				return ConditionResult{}, WrapError(err, "IF", b.currentLine == 0, b.currentLine)
			}

			isTrue := false
			if condValue.IsNumeric {
				isTrue = (condValue.NumValue != 0)
			} else {
				isTrue = (condValue.StrValue != "")
			}
			return ConditionResult{
				isTrue:   isTrue,
				thenStmt: thenStmt,
				elseStmt: elseStmt,
				hasElse:  hasElse,
			}, nil
		}

		// Ergebnis in boolean umwandeln (BASIC-konvention: 0=falsch, nicht-0=wahr)
		isTrue := result.IsNumeric && result.NumValue != 0
		return ConditionResult{
			isTrue:   isTrue,
			thenStmt: thenStmt,
			elseStmt: elseStmt,
			hasElse:  hasElse,
		}, nil
	}

	// ===== Standardauswertung ohne logische Operatoren =====
	condValue, err := b.evalExpression(condExpr)
	if err != nil {
		return ConditionResult{}, WrapError(err, "IF", b.currentLine == 0, b.currentLine)
	}

	// Determine truthiness (BASIC style: 0 is false, non-zero is true; empty string false, non-empty true).
	isTrue := false
	if condValue.IsNumeric {
		isTrue = (condValue.NumValue != 0)
	} else {
		isTrue = (condValue.StrValue != "")
	}
	return ConditionResult{isTrue: isTrue, thenStmt: thenStmt, elseStmt: elseStmt, hasElse: hasElse}, nil
}

// evalDirectLogical verwendet die neue EvaluateLogicalExpression-Funktion zur Auswertung logischer Ausdrücke
func (b *TinyBASIC) evalDirectLogical(expr string) (BASICValue, error) {
	return b.EvaluateLogicalExpression(expr)
}

// cmdFor initiates a FOR loop. Assumes lock is held.
func (b *TinyBASIC) cmdFor(args string, nextTokenIndex int) error {
	// CRITICAL: Context cancellation check to prevent deadlocks
	select {
	case <-b.ctx.Done():
		return NewBASICError(ErrCategorySystem, "EXECUTION_CANCELLED", b.currentLine == 0, b.currentLine)
	default:
	}

	// Wir müssen den Befehl richtig parsen - auch wenn er ein Teil einer Multi-Command-Zeile ist
	// Wir suchen nach einem Doppelpunkt, der NICHT in einem String ist
	var processedArgs string
	inString := false
	colonPos := -1

	for i := 0; i < len(args); i++ {
		char := args[i]
		// Handle string literals correctly
		if char == '"' {
			// Check for escaped quotes
			if inString && i+1 < len(args) && args[i+1] == '"' {
				i++ // Skip the next quote
				continue
			}
			inString = !inString
			continue
		}

		// Nur Doppelpunkte außerhalb von Strings zählen als Trennzeichen
		if char == ':' && !inString {
			colonPos = i
			break
		}
	}

	// Extract just the FOR part if we found a colon outside a string
	if colonPos != -1 {
		processedArgs = strings.TrimSpace(args[:colonPos])
	} else {
		processedArgs = args
	}

	// Verbesserte Tokenerkennung für FOR-Schleife
	// Format: FOR var=start TO end [STEP step]

	// 1. Finde das "=" Zeichen für die Variablenzuweisung
	eqPos := -1
	inStr := false
	parenLevel := 0

	for i := 0; i < len(processedArgs); i++ {
		c := processedArgs[i]
		if c == '"' {
			inStr = !inStr
		} else if !inStr {
			if c == '(' {
				parenLevel++
			} else if c == ')' {
				parenLevel--
			} else if c == '=' && parenLevel == 0 {
				eqPos = i
				break
			}
		}
	}
	if eqPos == -1 {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_EQUALS", b.currentLine == 0, b.currentLine).WithCommand("FOR")
	}

	// Variablenname extrahieren (Links vom =)
	varName := strings.TrimSpace(processedArgs[:eqPos])

	if !isValidVarName(varName) || strings.HasSuffix(varName, "$") {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("FOR")
	}
	varNameUpper := strings.ToUpper(varName)

	// 2. Finde das "TO" Keyword (berücksichtige nicht Vorkommen in Strings oder Variablennamen)
	toPos := -1
	inStr = false
	parenLevel = 0
	i := eqPos + 1

	for i < len(processedArgs) {
		// Suche nach "TO" als eigenständiges Token
		if !inStr && parenLevel == 0 && i+2 <= len(processedArgs) {
			if strings.ToUpper(processedArgs[i:i+2]) == "TO" {
				// Prüfe, ob es ein eigenständiges Token ist
				isToken := true

				// Prüfe davor (muss entweder am Anfang stehen oder ein Trennzeichen haben)
				if i > 0 && !strings.ContainsAny(string(processedArgs[i-1]), " \t,;()") {
					isToken = false
				}

				// Prüfe danach (muss entweder am Ende stehen oder ein Trennzeichen haben)
				if i+2 < len(processedArgs) && !strings.ContainsAny(string(processedArgs[i+2]), " \t,;()") {
					isToken = false
				}

				if isToken {
					toPos = i
					break
				}
			}
		}

		// String und Klammerung verfolgen
		c := processedArgs[i]
		if c == '"' {
			inStr = !inStr
		} else if !inStr {
			if c == '(' {
				parenLevel++
			} else if c == ')' {
				parenLevel--
			}
		}
		i++
	}

	if toPos == -1 {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_TO", b.currentLine == 0, b.currentLine).WithCommand("FOR")
	}

	// Startausdruck extrahieren (zwischen = und TO)
	startExpr := strings.TrimSpace(processedArgs[eqPos+1 : toPos])

	// 3. Ende des End-Ausdrucks finden (entweder bis STEP oder bis zum Ende)
	stepPos := -1
	inStr = false
	parenLevel = 0
	i = toPos + 2 // Position nach "TO"

	for i < len(processedArgs) {
		// Suche nach "STEP" als eigenständiges Token
		if !inStr && parenLevel == 0 && i+4 <= len(processedArgs) {
			if strings.ToUpper(processedArgs[i:i+4]) == "STEP" {
				// Prüfe, ob es ein eigenständiges Token ist
				isToken := true

				// Prüfe davor (muss entweder am Anfang stehen oder ein Trennzeichen haben)
				if i > 0 && !strings.ContainsAny(string(processedArgs[i-1]), " \t,;()") {
					isToken = false
				}

				// Prüfe danach (muss entweder am Ende stehen oder ein Trennzeichen haben)
				if i+4 < len(processedArgs) && !strings.ContainsAny(string(processedArgs[i+4]), " \t,;()") {
					isToken = false
				}

				if isToken {
					stepPos = i
					break
				}
			}
		}

		// String und Klammerung verfolgen
		c := processedArgs[i]
		if c == '"' {
			inStr = !inStr
		} else if !inStr {
			if c == '(' {
				parenLevel++
			} else if c == ')' {
				parenLevel--
			}
		}
		i++
	}

	// End-Expression extrahieren (zwischen TO und STEP oder Ende)
	var endExpr string
	if stepPos == -1 {
		endExpr = strings.TrimSpace(processedArgs[toPos+2:])
	} else {
		endExpr = strings.TrimSpace(processedArgs[toPos+2 : stepPos])
	}
	// 4. Step-Expression extrahieren (falls vorhanden)
	stepExpr := "1" // Standardwert
	if stepPos != -1 {
		stepExpr = strings.TrimSpace(processedArgs[stepPos+4:])
	}

	// Ausdrücke auswerten
	startVal, err := b.evalExpression(startExpr)
	if err != nil || !startVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("FOR")
	}

	endVal, err := b.evalExpression(endExpr)
	if err != nil || !endVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("FOR")
	}

	stepVal := BASICValue{NumValue: 1.0, IsNumeric: true} // Default STEP 1
	if stepExpr != "1" {                                  // Nur auswerten, wenn explizit angegeben
		stepVal, err = b.evalExpression(stepExpr)
		if err != nil || !stepVal.IsNumeric {
			return NewBASICError(ErrCategorySyntax, "INVALID_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("FOR")
		}
		if stepVal.NumValue == 0 {
			return NewBASICError(ErrCategorySyntax, "INVALID_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("FOR")
		}
	}

	// --- Loop Logic ---
	if len(b.forLoops) >= MaxForLoopDepth {
		// Kritischer Fehler: Stack Overflow
		b.appendToDebugLog(fmt.Sprintf("FOR STACK OVERFLOW: %d Schleifen aktiv, Limit: %d", len(b.forLoops), MaxForLoopDepth))
		for i, loop := range b.forLoops {
			b.appendToDebugLog(fmt.Sprintf("  Aktive Schleife %d: Variable=%s, GosubDepth=%d, ForLine=%d", i, loop.Variable, loop.GosubDepth, loop.ForLineNum))
		}
		return NewBASICError(ErrCategoryRuntime, "FOR_DEPTH", b.currentLine == 0, b.currentLine).WithCommand("FOR")
	}

	// Initialize loop variable.
	b.variables[varNameUpper] = startVal
	// Determine line number *after* FOR statement for loop body start.
	firstLoopLine, found := b.findNextLine(b.currentLine)
	if !found {
		firstLoopLine = 0 // Loop at end of program.
	}

	// Push loop info onto stack.
	loopInfo := ForLoopInfo{
		Variable:              varNameUpper,
		EndValue:              endVal.NumValue,
		Step:                  stepVal.NumValue,
		StartLine:             firstLoopLine,     // Line *after* FOR.
		ForLineNum:            b.currentLine,     // Line *of* the FOR statement.
		BodySubStatementIndex: nextTokenIndex,    // Index des Statements *nach* FOR auf dieser Zeile
		GosubDepth:            len(b.gosubStack), // Store current GOSUB depth
	}

	b.forLoops = append(b.forLoops, loopInfo)

	// Check if loop should be skipped entirely from start.
	currentValue := startVal.NumValue
	endValue := loopInfo.EndValue
	step := loopInfo.Step
	skipLoop := (step > 0 && currentValue > endValue) || (step < 0 && currentValue < endValue)

	if skipLoop {
		// Find matching NEXT and jump program counter past it.
		nextJumpLine, err := b.findMatchingNext(b.currentLine, varNameUpper)
		if err != nil {
			b.forLoops = b.forLoops[:len(b.forLoops)-1] // Pop invalid loop info.
			return WrapError(err, "FOR", b.currentLine == 0, b.currentLine)
		}
		b.currentLine = nextJumpLine                // Jump past NEXT.
		b.forLoops = b.forLoops[:len(b.forLoops)-1] // Pop loop info as es wurde übersprungen.
	} else {
		// Set currentLine to the first line of the loop body
		b.currentLine = firstLoopLine
	}
	// If not skipping, run loop proceeds naturally to StartLine.

	return nil
}

// cmdNext processes the end of a FOR loop iteration. Assumes lock is held.
func (b *TinyBASIC) cmdNext(args string) error {
	// CRITICAL: Context cancellation check to prevent deadlocks
	select {
	case <-b.ctx.Done():
		return NewBASICError(ErrCategorySystem, "EXECUTION_CANCELLED", b.currentLine == 0, b.currentLine)
	default:
	}

	// Verbesserte Erkennung von Variablennamen bei NEXT
	// Extrahiert den Variablennamen aus dem Argument, unabhängig vom Format:
	// "NEXT I", "NEXTI", "NEXT  I" etc.

	varName := strings.TrimSpace(args)

	// Wenn keine Argumente vorhanden, verwenden wir die innerste Schleife (klassisches BASIC-Verhalten)
	if varName == "" {
		if len(b.forLoops) == 0 {
			return NewBASICError(ErrCategoryExecution, "NEXT_WITHOUT_FOR", b.currentLine == 0, b.currentLine).WithCommand("NEXT")
		}
		varName = b.forLoops[len(b.forLoops)-1].Variable
	} else {
		// Variablennamen in Großbuchstaben umwandeln
		varName = strings.ToUpper(varName)
	}

	// Suche die passende Schleife im Stack (von innen nach außen)
	found := false
	forLoopStackIndex := -1
	for i := len(b.forLoops) - 1; i >= 0; i-- {
		if b.forLoops[i].Variable == varName {
			found = true
			forLoopStackIndex = i
			break
		}
	}

	if !found {
		return NewBASICError(ErrCategoryExecution, "FOR_NEXT_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("NEXT")
	}

	if forLoopStackIndex >= len(b.forLoops) || forLoopStackIndex < 0 {
		return NewBASICError(ErrCategorySystem, "INTERNAL_LOOP_STACK_ERROR", b.currentLine == 0, b.currentLine).WithCommand("NEXT")
	}

	if len(b.forLoops) > forLoopStackIndex+1 {
		b.forLoops = b.forLoops[:forLoopStackIndex+1]
	}

	currentNextLine := b.currentLine       // Line number of the NEXT statement itself
	loop := &b.forLoops[len(b.forLoops)-1] // Get the current loop (should be the one we matched or the innermost if varName was empty)

	// Defensive check: ensure the loop variable matches, especially if varName was specified.
	if loop.Variable != varName {
		// This could happen if varName was specified, but the stack manipulation above was incorrect.
		// Search again for the specific varName if it doesn't match the top.
		// This indicates a more complex scenario or a bug in stack handling.
		// For now, we'll proceed assuming the top loop is the correct one after pruning.
		// However, a more robust find and prune might be needed if this becomes an issue.
		// Re-find the correct loop to be absolutely sure, this is safer.
		correctLoopIdx := -1
		for i := len(b.forLoops) - 1; i >= 0; i-- {
			if b.forLoops[i].Variable == varName {
				correctLoopIdx = i
				break
			}
		}
		if correctLoopIdx == -1 {
			return NewBASICError(ErrCategoryExecution, "FOR_NEXT_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("NEXT").WithUsageHint("Could not re-find loop var after stack adjustment.")
		}
		loop = &b.forLoops[correctLoopIdx]
	}

	val, varExists := b.variables[loop.Variable]
	if !varExists {
		// This should not happen if FOR initialized the variable
		return NewBASICError(ErrCategoryRuntime, "UNDEFINED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("NEXT").WithUsageHint("Loop variable " + loop.Variable + " not found.")
	}
	val.NumValue += loop.Step
	b.variables[loop.Variable] = val
	continueLoop := (loop.Step > 0 && val.NumValue <= loop.EndValue+Epsilon) || (loop.Step < 0 && val.NumValue >= loop.EndValue-Epsilon) // Added Epsilon for float comparisons

	if continueLoop {

		// NEUE LOGIK: Prüfen, ob FOR auf derselben Zeile war
		if loop.ForLineNum == currentNextLine && loop.BodySubStatementIndex >= 0 { // Ja, FOR war auf derselben Zeile. Setze den Resume-Index.
			// WICHTIG: b.currentLine NICHT ändern, da executeStatement die originalCurrentLine bewahrt!
			b.resumeSubStatementIndex = loop.BodySubStatementIndex
			// b.appendToDebugLog(fmt.Sprintf("cmdNext (continue, same line FOR): Resuming at sub-statement index %d on line %d", b.resumeSubStatementIndex, b.currentLine))
		} else {
			// Nein, FOR war auf anderer Zeile ODER es gab kein Statement danach auf der Zeile. Springe zur StartLine.
			b.currentLine = loop.StartLine
			b.resumeSubStatementIndex = 0 // Sicherstellen, dass kein alter Index übrigbleibt
			if b.currentLine == 0 {
				// Fehlermeldung ausgeben und Programm beenden
				b.appendToDebugLog(fmt.Sprintf("cmdNext Error: loop.StartLine is 0 for Var=%s on NEXT at line %d", loop.Variable, currentNextLine))
				return NewBASICError(ErrCategorySystem, "INTERNAL_ERROR", true, currentNextLine).WithCommand("NEXT").WithUsageHint("Loop start line is 0.")
			}
		}
	} else {
		// Schleife ist fertig: Pop und setze Variable auf Exit-Wert (klassisches BASIC-Verhalten)
		// Die Variable sollte den Wert haben, der die Schleifenbedingung nicht mehr erfüllt
		// Das ist der aktuelle Wert in val.NumValue (bereits um Step erhöht)

		b.forLoops = b.forLoops[:len(b.forLoops)-1]
		b.resumeSubStatementIndex = 0 // Sicherstellen, dass kein alter Index übrigbleibt

		// b.appendToDebugLog(fmt.Sprintf("cmdNext (loop finished): Var=%s. Popped loop. Remaining loops: %d. currentLine %d not changed by NEXT.", loop.Variable, len(b.forLoops), b.currentLine))
		// Die Ausführung geht normal mit dem nächsten Sub-Statement weiter (falls vorhanden)
		// oder zur nächsten Zeile, da currentLine nicht geändert wurde.
		// Die Variable behält bereits den korrekten Exit-Wert (val.NumValue)
	}
	return nil
}

// findMatchingNext searches forward to find the corresponding NEXT for a FOR loop.
// Used when skipping a loop entirely. Returns the line number *after* the NEXT. Assumes lock is held.
func (b *TinyBASIC) findMatchingNext(forLineNum int, forVarName string) (int, error) {
	nestingLevel := 0        // Start at level 0 for the FOR we are matching.
	searchLine := forLineNum // Start searching from the line *after* FOR.
	checkCounter := 0        // Counter für periodische Context-Checks

	for {
		// Periodische Context-Checks alle 100 Iterationen für Performance
		checkCounter++
		if checkCounter >= 100 {
			checkCounter = 0
			// CRITICAL: Context cancellation check to prevent deadlocks
			select {
			case <-b.ctx.Done():
				return 0, NewBASICError(ErrCategorySystem, "EXECUTION_CANCELLED", b.currentLine == 0, b.currentLine)
			default:
			}
		}

		nextLine, found := b.findNextLine(searchLine)
		if !found {
			return 0, NewBASICError(ErrCategoryRuntime, "MISSING_NEXT", false, searchLine).
				WithCommand("FOR").
				WithUsageHint(fmt.Sprintf("Matching NEXT for variable %s from line %d not found (EOF reached)", forVarName, forLineNum))
		}
		searchLine = nextLine
		lineCode := b.program[searchLine]

		// Verbesserte Analyse der Zeile
		i := 0

		// Überspringe führende Leerzeichen
		for i < len(lineCode) && (lineCode[i] == ' ' || lineCode[i] == '\t') {
			i++
		}

		// Prüfe, ob wir einen FOR- oder NEXT-Befehl haben
		if i+3 < len(lineCode) && strings.ToUpper(lineCode[i:i+3]) == "FOR" {
			// Prüfe, ob das ein vollständiges Token ist
			if i+3 >= len(lineCode) || !isAlphaNum(lineCode[i+3]) {
				// FOR-Befehl gefunden
				nestingLevel++
				continue
			}
		} else if i+4 < len(lineCode) && strings.ToUpper(lineCode[i:i+4]) == "NEXT" {
			// Finde den Variablennamen nach NEXT
			varStart := i + 4

			// Überspringe Leerzeichen nach NEXT
			for varStart < len(lineCode) && (lineCode[varStart] == ' ' || lineCode[varStart] == '\t') {
				varStart++
			}

			// Extrahiere den Variablennamen
			varEnd := varStart
			for varEnd < len(lineCode) && isAlphaNum(lineCode[varEnd]) {
				varEnd++
			}

			// Kein Variablenname gefunden oder es ist nichts nach "NEXT"
			if varStart >= len(lineCode) || varStart == varEnd {
				// NEXT ohne Variablennamen - in klassischem BASIC erlaubt
				// Wenn wir auf Ebene 0 sind, dann ist dies unser gesuchtes NEXT
				if nestingLevel == 0 {
					// Gefunden! Zurück zur nächsten Zeile nach dem NEXT
					nextAfterLine, nextAfterFound := b.findNextLine(searchLine)
					if !nextAfterFound {
						nextAfterLine = 0 // Ende des Programms
					}
					return nextAfterLine, nil
				} else {
					// NEXT für eine innere Schleife
					nestingLevel--
					continue
				}
			}

			// Variablenname extrahiert
			nextVarName := strings.ToUpper(lineCode[varStart:varEnd])

			if nestingLevel == 0 {
				// Sind wir auf der gesuchten Ebene?
				if nextVarName == forVarName {
					// Gefunden! Zurück zur nächsten Zeile nach dem NEXT
					nextAfterLine, nextAfterFound := b.findNextLine(searchLine)
					if !nextAfterFound {
						nextAfterLine = 0 // Ende des Programms
					}
					return nextAfterLine, nil
				} else {
					// NEXT für eine andere Variable auf der gesuchten Ebene
					return 0, NewBASICError(ErrCategoryRuntime, "NEXT_VARIABLE_MISMATCH", false, searchLine).
						WithCommand("NEXT").
						WithUsageHint(fmt.Sprintf("Encountered NEXT %s at line %d while searching for NEXT %s from FOR at line %d", nextVarName, searchLine, forVarName, forLineNum))
				}
			} else {
				// NEXT für eine innere Schleife
				nestingLevel--
			}
		}
	}
	// Unreachable if loop always finds next line or EOF.
}

// cmdIf handles the IF statement execution.
// If the condition is false, it now advances b.currentLine to the next
// physical line number, causing the multi-statement loop to skip the rest of the current line.
// Assumes lock is held.
func (b *TinyBASIC) cmdIf(args string) error {
	// Capture the line number where the IF statement resides *before* potentially changing it.
	originalLine := b.currentLine
	// Evaluate the condition and get the THEN statement
	result, err := b.evalIfCondition(args)
	if err != nil {
		return err // Propagate error (e.g., Syntax Error in condition)
	}
	// Wenn die Bedingung wahr ist, den THEN-Teil ausführen
	if result.isTrue {
		// FIXED: Verwende die korrekte splitStatementsByColon-Funktion anstatt strings.Split
		// Diese berücksichtigt String-Literale und andere Syntaxelemente korrekt
		thenStatements := b.splitStatementsByColon(result.thenStmt)
		for _, subStatement := range thenStatements {
			subStatement = strings.TrimSpace(subStatement)
			if subStatement == "" {
				continue
			}
			_, err := b.executeSingleStatementInternal(subStatement, b.ctx)
			if err != nil {
				return WrapError(err, "IF", b.currentLine == 0, originalLine)
			}
		}
	} else if result.hasElse {
		// FIXED: Gleiches Problem beim ELSE-Teil - verwende splitStatementsByColon
		elseStatements := b.splitStatementsByColon(result.elseStmt)
		for _, subStatement := range elseStatements {
			subStatement = strings.TrimSpace(subStatement)
			if subStatement == "" {
				continue
			}

			_, err := b.executeSingleStatementInternal(subStatement, b.ctx)
			if err != nil {
				return WrapError(err, "IF", b.currentLine == 0, originalLine)
			}
		}
	}
	// Bemerkung: Wenn die Bedingung falsch ist und es KEINEN ELSE-Teil gibt,
	// wird NICHTS gemacht - die Ausführung geht normal zur nächsten Zeile weiter.
	// Das IF-Statement ändert NICHT b.currentLine, das macht die Hauptschleife.

	return nil
}

// cleanupForLoopsOnReturn bereinigt FOR-Schleifen, die innerhalb der Subroutine liegen, die gerade verlassen wird.
// Diese Funktion wird aufgerufen, wenn RETURN ausgeführt wird.
// currentGosubDepth ist die Tiefe des GOSUB-Stacks *bevor* die aktuelle Subroutine verlassen wurde.
func (b *TinyBASIC) cleanupForLoopsOnReturn(currentGosubDepth int) {
	if len(b.forLoops) == 0 {
		return
	}

	// Finde den Index der letzten Schleife, die beibehalten werden soll.
	// Schleifen werden beibehalten, wenn ihre GosubDepth kleiner ist als die Tiefe der gerade verlassenen Routine.
	keepUpTo := -1
	for i := len(b.forLoops) - 1; i >= 0; i-- {
		if b.forLoops[i].GosubDepth < currentGosubDepth {
			keepUpTo = i
			break
		}
	}
	if keepUpTo == -1 {
		// keepUpTo == -1 bedeutet: KEINE Schleife hat eine GosubDepth < currentGosubDepth
		// Das heißt, alle Schleifen haben GosubDepth >= currentGosubDepth

		// Wir müssen nur die Schleifen entfernen, die GosubDepth >= currentGosubDepth haben.
		// ABER: Schleifen mit GosubDepth == currentGosubDepth wurden in der GLEICHEN Ebene erstellt,
		// die wir gerade verlassen. Diese müssen entfernt werden.
		// Schleifen mit GosubDepth > currentGosubDepth sollten theoretisch nicht existieren,
		// da sie bereits bei früheren RETURN-Aufrufen bereinigt worden sein sollten.
		// Korrekte Logik: Entferne nur Schleifen mit GosubDepth >= currentGosubDepth
		// Aber da keepUpTo == -1, bedeutet das, dass ALLE Schleifen GosubDepth >= currentGosubDepth haben.
		// Also müssen wir die Schleifen einzeln prüfen und nur die mit GosubDepth >= currentGosubDepth entfernen.
		newForLoops := make([]ForLoopInfo, 0, len(b.forLoops))
		for _, loop := range b.forLoops {
			if loop.GosubDepth < currentGosubDepth {
				newForLoops = append(newForLoops, loop)
			}
		}
		b.forLoops = newForLoops
	} else {
		// Schleifen von Index 0 bis keepUpTo werden beibehalten.
		// Schleifen von Index keepUpTo + 1 bis Ende werden entfernt.
		removedCount := len(b.forLoops) - (keepUpTo + 1)
		if removedCount > 0 {
		}
		b.forLoops = b.forLoops[:keepUpTo+1]
	}
}

// appendToDebugLog appends a string to the debug log file if it's open.
func (b *TinyBASIC) appendToDebugLog(s string) {
	// Re-enabled debug logging to analyze FOR/NEXT cleanup issues with sprite definitions
	if b.debugFP != nil {
		timestamp := time.Now().Format("15:04:05.000")
		// Ensure newline is added if not present, and escape any existing newlines in s
		// to keep the log entry on one line for easier parsing.
		// However, for this specific logging, we assume s does not contain newlines.
		if _, err := fmt.Fprintf(b.debugFP, "[%s] %s\n", timestamp, s); err != nil {
			// Optional: Handle error writing to debug log, e.g., print to console
			// fmt.Printf("Error writing to debug log: %v\n", err)
		}
	}
}

// cleanupForLoopsOnGoto bereinigt FOR-Schleifen, die durch ein GOTO "übersprungen" werden.
// Diese Funktion verhindert Stack-Überläufe in Hauptschleifen, die mit GOTO implementiert sind.
func (b *TinyBASIC) cleanupForLoopsOnGoto(currentLine, targetLine int) error {
	if len(b.forLoops) == 0 {
		return nil
	}

	// Deadlock-Schutz: Nur echte Interpreter-Deadlocks abfangen, nicht legale Hauptschleifen
	// Prüfe nur, wenn es aktive FOR-Schleifen gibt, die durch das GOTO potentiell übersprungen werden
	// Das ist das eigentliche Problem: FOR-Schleifen, die nie ein NEXT erreichen

	// Wenn es keine aktiven FOR-Schleifen gibt, kann es keinen FOR/NEXT-Deadlock geben
	// Normale GOTO-Hauptschleifen ohne FOR-Schleifen sind immer erlaubt
	if len(b.forLoops) == 0 {
		return nil
	}

	// Prüfe, ob das GOTO eine FOR-Schleife überspringt (echte Deadlock-Gefahr)
	hasSkippedForLoop := false
	for _, loop := range b.forLoops {
		// Ein FOR-Deadlock entsteht, wenn ein GOTO eine FOR-Schleife überspringt,
		// so dass das entsprechende NEXT nie erreicht wird
		if targetLine <= loop.ForLineNum && currentLine > loop.ForLineNum {
			hasSkippedForLoop = true
			break
		}
	}

	// Nur wenn FOR-Schleifen übersprungen werden, überwache auf Deadlocks
	if hasSkippedForLoop {
		key := fmt.Sprintf("%d->%d", currentLine, targetLine)
		if b.gotoCleanupCount == nil {
			b.gotoCleanupCount = make(map[string]int)
		}
		b.gotoCleanupCount[key]++

		// Sehr hohe Schwelle: Nur bei eindeutigen FOR/NEXT-Deadlocks eingreifen
		if b.gotoCleanupCount[key] > 10000 {
			// Kritischer Fehler: FOR/NEXT-Deadlock erkannt
			return NewBASICError(ErrCategoryRuntime, "FOR_NEXT_DEADLOCK", false, currentLine).
				WithCommand("GOTO").
				WithUsageHint(fmt.Sprintf("FOR/NEXT deadlock detected: GOTO skips FOR loops %d iterations", b.gotoCleanupCount[key]))
		}
	}

	// Wenn GOTO rückwärts springt (typisch für Hauptschleifen),
	// bereinige alle FOR-Schleifen, die nach dem Ziel liegen
	if targetLine < currentLine {
		var keptLoops []ForLoopInfo
		for _, loop := range b.forLoops { // Behalte nur FOR-Schleifen, die vor dem GOTO-Ziel liegen
			// oder die im gleichen GOSUB-Level wie das GOTO-Ziel sind
			if loop.ForLineNum < targetLine {
				keptLoops = append(keptLoops, loop)
			}
		}
		b.forLoops = keptLoops
	}
	// Für Vorwärts-GOTOs machen wir normalerweise nichts,
	// da diese meist korrekte Sprünge aus Schleifen heraus sind
	return nil
}
