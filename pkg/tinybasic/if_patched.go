//go:build ignore

// Package tinybasic implements a simple BASIC interpreter.
package tinybasic

import (
	"strings"
)

// In dieser Datei wird eine alternative Version für die Verarbeitung von IF-Anweisungen
// mit komplexen logischen Ausdrücken implementiert.

// evalComplexCondition ist eine vereinfachte Funktion, um komplexe logische Ausdrücke auszuwerten,
// die AND oder OR enthalten könnten
func (b *TinyBASIC) evalComplexCondition(condExpr string) (BASICValue, error) {
	// Fall 1: Der Ausdruck enthält ein AND
	if strings.Contains(strings.ToUpper(condExpr), "AND") {
		parts := strings.Split(strings.ToUpper(condExpr), "AND")
		if len(parts) != 2 {
			return BASICValue{}, NewBASICError(ErrCategorySyntax, "COMPLEX_EXPRESSION_ERROR", b.currentLine == 0, b.currentLine)
		}

		leftExpr := strings.TrimSpace(condExpr[:strings.ToUpper(condExpr).Index("AND")])
		rightExpr := strings.TrimSpace(condExpr[strings.ToUpper(condExpr).Index("AND")+3:])

		leftResult, err := b.evalExpression(leftExpr)
		if err != nil {
			return BASICValue{}, err
		}

		// Kurzschlussauswertung - wenn links bereits falsch ist, ist das Gesamtergebnis falsch
		if leftResult.IsNumeric && leftResult.NumValue == 0 {
			return BASICValue{NumValue: 0, IsNumeric: true}, nil
		}

		rightResult, err := b.evalExpression(rightExpr)
		if err != nil {
			return BASICValue{}, err
		}

		// AND-Logik
		leftBool := leftResult.IsNumeric && leftResult.NumValue != 0
		rightBool := rightResult.IsNumeric && rightResult.NumValue != 0
		resultBool := leftBool && rightBool

		resultVal := 0.0
		if resultBool {
			resultVal = -1.0 // BASIC-Stil: true = -1
		}

		return BASICValue{NumValue: resultVal, IsNumeric: true}, nil
	}

	// Fall 2: Der Ausdruck enthält ein OR
	if strings.Contains(strings.ToUpper(condExpr), "OR") {
		parts := strings.Split(strings.ToUpper(condExpr), "OR")
		if len(parts) != 2 {
			return BASICValue{}, NewBASICError(ErrCategorySyntax, "COMPLEX_EXPRESSION_ERROR", b.currentLine == 0, b.currentLine)
		}

		leftExpr := strings.TrimSpace(condExpr[:strings.ToUpper(condExpr).Index("OR")])
		rightExpr := strings.TrimSpace(condExpr[strings.ToUpper(condExpr).Index("OR")+2:])

		leftResult, err := b.evalExpression(leftExpr)
		if err != nil {
			return BASICValue{}, err
		}

		// Kurzschlussauswertung - wenn links bereits wahr ist, ist das Gesamtergebnis wahr
		if leftResult.IsNumeric && leftResult.NumValue != 0 {
			return BASICValue{NumValue: -1.0, IsNumeric: true}, nil
		}

		rightResult, err := b.evalExpression(rightExpr)
		if err != nil {
			return BASICValue{}, err
		}

		// OR-Logik
		leftBool := leftResult.IsNumeric && leftResult.NumValue != 0
		rightBool := rightResult.IsNumeric && rightResult.NumValue != 0
		resultBool := leftBool || rightBool

		resultVal := 0.0
		if resultBool {
			resultVal = -1.0 // BASIC-Stil: true = -1
		}

		return BASICValue{NumValue: resultVal, IsNumeric: true}, nil
	}

	// Fall 3: Einfacher Ausdruck ohne AND/OR
	return b.evalExpression(condExpr)
}

// Alternative zur normalen evalIfCondition, die komplexe Ausdrücke besser verarbeiten kann
func (b *TinyBASIC) evalIfConditionPatched(args string) (ConditionResult, error) {
	// Die ursprüngliche Funktionalität zum Extrahieren von THEN/ELSE bleibt erhalten
	upperArgs := strings.ToUpper(args)
	thenPos := strings.Index(upperArgs, "THEN")

	if thenPos == -1 {
		return ConditionResult{}, NewBASICError(ErrCategorySyntax, "EXPECTED_THEN", b.currentLine == 0, b.currentLine).WithCommand("IF")
	}

	condExpr := strings.TrimSpace(args[:thenPos])
	afterThen := ""
	if thenPos+4 < len(args) {
		afterThen = args[thenPos+4:]
	}

	// Standard-Parsing von ELSE wie in der ursprünglichen Funktion
	thenStmt := ""
	elseStmt := ""
	hasElse := false

	if afterThen != "" {
		inString := false
		elsePos := -1
		i := 0

		for i < len(afterThen) {
			if afterThen[i] == ' ' || afterThen[i] == '\t' {
				i++
				continue
			}
			if afterThen[i] == '"' {
				inString = !inString
				i++
				continue
			}

			if !inString && i+4 <= len(afterThen) && strings.ToUpper(afterThen[i:i+4]) == "ELSE" {
				isFullToken := true
				if i+4 < len(afterThen) {
					nextChar := afterThen[i+4]
					if (nextChar >= 'a' && nextChar <= 'z') || (nextChar >= 'A' && nextChar <= 'Z') ||
						(nextChar >= '0' && nextChar <= '9') || nextChar == '_' {
						isFullToken = false
					}
				}
				if isFullToken {
					elsePos = i
					break
				}
			}
			i++
		}

		if elsePos >= 0 {
			thenStmt = strings.TrimSpace(afterThen[:elsePos])
			elseStmt = strings.TrimSpace(afterThen[elsePos+4:])
			hasElse = true
		} else {
			thenStmt = strings.TrimSpace(afterThen)
		}
	}

	if condExpr == "" {
		return ConditionResult{}, NewBASICError(ErrCategorySyntax, "EXPECTED_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("IF")
	}
	if thenStmt == "" {
		return ConditionResult{}, NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("IF")
	}

	// Die verbesserte Auswertung von komplexen Ausdrücken
	condValue, err := b.evalComplexCondition(condExpr)
	if err != nil {
		return ConditionResult{}, WrapError(err, "IF", b.currentLine == 0, b.currentLine)
	}

	// Bestimme den Wahrheitswert wie in der ursprünglichen Funktion
	isTrue := false
	if condValue.IsNumeric {
		isTrue = (condValue.NumValue != 0)
	} else {
		isTrue = (condValue.StrValue != "")
	}

	return ConditionResult{isTrue: isTrue, thenStmt: thenStmt, elseStmt: elseStmt, hasElse: hasElse}, nil
}
