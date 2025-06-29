// Package tinybasic implements a simple BASIC interpreter.
package tinybasic

import (
	"strings"
)

// DirectLogicalEvaluator bietet eine einfache Möglichkeit, logische Ausdrücke direkt auszuwerten
// ohne den komplexen Parser-Stack zu verwenden. Diese Implementierung ist speziell
// für den Umgang mit AND und OR in IF-Anweisungen gedacht.

// EvaluateLogicalExpression wertet einen logischen Ausdruck wie "A <= B AND C > D" aus
func (b *TinyBASIC) EvaluateLogicalExpression(expr string) (BASICValue, error) {
	// Normalisieren des Ausdrucks für eine zuverlässigere Erkennung
	normalizedExpr := normalizeLogicalExpression(expr)

	// Fall 1: Ausdruck enthält AND
	if strings.Contains(normalizedExpr, " AND ") {
		// Extrahiere die linke und rechte Seite des AND-Ausdrucks
		// Verwende den ursprünglichen Ausdruck, um den AND-Operator zu finden
		upperExpr := strings.ToUpper(expr)
		leftExprPos := -1
		andToken := ""

		// Suche nach dem AND-Token mit verschiedenen Schreibweisen
		for _, token := range []string{" AND ", "AND ", " AND", "AND"} {
			if idx := strings.Index(upperExpr, token); idx >= 0 {
				leftExprPos = idx
				andToken = token
				break
			}
		}
		if leftExprPos == -1 {
			return BASICValue{}, NewBASICError(ErrCategorySyntax, "INVALID_LOGICAL_EXPRESSION", b.currentLine == 0, b.currentLine)
		}

		// Extrahiere die linke und rechte Seite des AND-Ausdrucks
		leftExpr := strings.TrimSpace(expr[:leftExprPos])
		rightExpr := strings.TrimSpace(expr[leftExprPos+len(andToken):]) // Berücksichtige die tatsächliche Länge des AND-Tokens

		// Werte linke Seite aus
		leftResult, err := b.evalExpression(leftExpr)
		if err != nil {
			return BASICValue{}, err
		}

		// Kurzschlussauswertung: Wenn links bereits falsch ist, ist das Gesamtergebnis falsch
		if leftResult.IsNumeric && leftResult.NumValue == 0 {
			return BASICValue{NumValue: 0, IsNumeric: true}, nil
		}

		// Werte rechte Seite aus
		rightResult, err := b.evalExpression(rightExpr)
		if err != nil {
			return BASICValue{}, err
		}

		// Bestimme Wahrheitswerte
		leftBool := isTruthy(leftResult)
		rightBool := isTruthy(rightResult)
		resultBool := leftBool && rightBool

		// BASIC-Stil: Falsch=0, Wahr=-1
		resultVal := 0.0
		if resultBool {
			resultVal = -1.0
		}

		return BASICValue{NumValue: resultVal, IsNumeric: true}, nil
	}

	// Fall 2: Ausdruck enthält OR
	if strings.Contains(normalizedExpr, " OR ") {
		// Extrahiere die linke und rechte Seite des OR-Ausdrucks
		upperExpr := strings.ToUpper(expr)
		leftExprPos := -1
		orToken := ""

		// Suche nach dem OR-Token mit verschiedenen Schreibweisen
		for _, token := range []string{" OR ", "OR ", " OR", "OR"} {
			if idx := strings.Index(upperExpr, token); idx >= 0 {
				leftExprPos = idx
				orToken = token
				break
			}
		}
		if leftExprPos == -1 {
			return BASICValue{}, NewBASICError(ErrCategorySyntax, "INVALID_LOGICAL_EXPRESSION", b.currentLine == 0, b.currentLine)
		}

		// Extrahiere die linke und rechte Seite des OR-Ausdrucks
		leftExpr := strings.TrimSpace(expr[:leftExprPos])
		rightExpr := strings.TrimSpace(expr[leftExprPos+len(orToken):]) // Berücksichtige die tatsächliche Länge des OR-Tokens

		// Werte linke Seite aus
		leftResult, err := b.evalExpression(leftExpr)
		if err != nil {
			return BASICValue{}, err
		}

		// Kurzschlussauswertung: Wenn links bereits wahr ist, ist das Gesamtergebnis wahr
		if isTruthy(leftResult) {
			return BASICValue{NumValue: -1.0, IsNumeric: true}, nil
		}

		// Werte rechte Seite aus
		rightResult, err := b.evalExpression(rightExpr)
		if err != nil {
			return BASICValue{}, err
		}

		// Bestimme Wahrheitswerte
		leftBool := isTruthy(leftResult)
		rightBool := isTruthy(rightResult)
		resultBool := leftBool || rightBool

		// BASIC-Stil: Falsch=0, Wahr=-1
		resultVal := 0.0
		if resultBool {
			resultVal = -1.0
		}

		return BASICValue{NumValue: resultVal, IsNumeric: true}, nil
	}

	// Fall 3: Einfache Auswertung ohne logische Operatoren
	return b.evalExpression(expr)
}
