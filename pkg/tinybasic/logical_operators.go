package tinybasic

import (
	"strings"
)

// parseLogical verarbeitet logische Operatoren wie AND und OR
// left ist der bereits ausgewertete linke Operand
func (p *exprParser) parseLogical(left BASICValue) (BASICValue, error) {
	// Das aktuelle Token sollte ein logischer Operator sein
	opToken := p.next() // Konsumiere den logischen Operator
	op := strings.ToUpper(strings.TrimSpace(opToken.val))

	// Rechten Operanden auswerten (eine weitere Vergleichsoperation)
	right, err := p.parseComparison()
	if err != nil {
		return BASICValue{}, err
	}

	// Beide Seiten in Wahrheitswerte umwandeln
	leftBool := isTruthy(left)
	rightBool := isTruthy(right)

	// Logische Operation anwenden
	var resultBool bool
	switch op {
	case "AND":
		resultBool = leftBool && rightBool
	case "OR":
		resultBool = leftBool || rightBool
	default:
		return BASICValue{}, NewBASICError(ErrCategorySyntax, "INVALID_OPERATOR", false, p.tb.currentLine)
	}

	// BASIC-Stil: Falsch=0, Wahr=-1
	resultVal := 0.0
	if resultBool {
		resultVal = -1.0
	}

	// Rekursiv weitere logische Operatoren verarbeiten
	nextToken := p.peek()
	if nextToken.typ == tokOp {
		upperVal := strings.ToUpper(strings.TrimSpace(nextToken.val))
		if upperVal == "AND" || upperVal == "OR" {
			// Weitere logische Operation folgt
			return p.parseLogical(BASICValue{NumValue: resultVal, IsNumeric: true})
		}
	}

	return BASICValue{NumValue: resultVal, IsNumeric: true}, nil
}
