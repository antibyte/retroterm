// Package tinybasic implements a simple BASIC interpreter.
package tinybasic

import (
	"fmt"
	"math"
	"strings"
)

// splitRespectingParentheses teilt einen String durch Kommas auf,
// ignoriert aber Kommas innerhalb von Klammern
func splitRespectingParentheses(input string) []string {
	var result []string
	var current strings.Builder
	parenDepth := 0

	for _, char := range input {
		switch char {
		case '(':
			parenDepth++
			current.WriteRune(char)
		case ')':
			parenDepth--
			current.WriteRune(char)
		case ',':
			if parenDepth == 0 {
				// Komma außerhalb von Klammern - neues Element
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				// Komma innerhalb von Klammern - zu aktueller Definition hinzufügen
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	// Letztes Element hinzufügen
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}

	return result
}

// cmdDim implementiert den DIM-Befehl zum Erstellen von Arrays
func (b *TinyBASIC) cmdDim(args string) error {
	// Prüfe, ob Argumente vorhanden sind
	if args == "" {
		return NewBASICError(ErrCategorySyntax, "INVALID_DIM_STATEMENT", b.currentLine == 0, b.currentLine).WithCommand("DIM")
	}
	// Teile die Argumente durch Kommata (können mehrere Array-Definitionen sein)
	// Aber ignoriere Kommas innerhalb von Klammern für 2D-Arrays
	arrayDefs := splitRespectingParentheses(args)

	for _, def := range arrayDefs {
		def = strings.TrimSpace(def)

		// Finde die öffnende Klammer
		openParenPos := strings.IndexByte(def, '(')
		if openParenPos <= 0 {
			return NewBASICError(ErrCategorySyntax, "INVALID_DIM_STATEMENT", b.currentLine == 0, b.currentLine).WithCommand("DIM")
		}

		// Finde die schließende Klammer
		closeParenPos := strings.IndexByte(def, ')')
		if closeParenPos <= openParenPos || closeParenPos != len(def)-1 {
			return NewBASICError(ErrCategorySyntax, "INVALID_DIM_STATEMENT", b.currentLine == 0, b.currentLine).WithCommand("DIM")
		}

		// Extrahiere den Variablennamen
		varName := strings.TrimSpace(def[:openParenPos])
		varName = strings.ToUpper(varName) // Variablennamen in Großbuchstabenin Großbuchstaben

		// Validiere den Variablennamen
		if !isValidVarName(varName) {
			return NewBASICError(ErrCategorySyntax, "INVALID_VARIABLE_NAME", b.currentLine == 0, b.currentLine).WithCommand("DIM")
		}
		// Extrahiere die Array-Dimensionen
		dimensionsStr := strings.TrimSpace(def[openParenPos+1 : closeParenPos])

		// Parse Dimensionen (kann 1D oder 2D sein)
		dimensions := strings.Split(dimensionsStr, ",")
		if len(dimensions) > 2 {
			return NewBASICError(ErrCategorySyntax, "TOO_MANY_DIMENSIONS", b.currentLine == 0, b.currentLine).WithCommand("DIM").WithUsageHint("Maximum 2 dimensions supported")
		}

		// Erste Dimension auswerten
		dimValue1, err := b.evalExpression(strings.TrimSpace(dimensions[0]))
		if err != nil {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_ARRAY_DIMENSION_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("DIM")
		}
		if !dimValue1.IsNumeric {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_ARRAY_DIMENSION_TYPE", b.currentLine == 0, b.currentLine).WithCommand("DIM")
		}

		size1 := int(math.Round(dimValue1.NumValue))
		if size1 < 0 {
			return NewBASICError(ErrCategoryEvaluation, "INVALID_ARRAY_INDEX", b.currentLine == 0, b.currentLine).WithCommand("DIM")
		}

		// Prüfe auf 2D-Array
		var size2 int = -1 // -1 bedeutet 1D-Array
		if len(dimensions) == 2 {
			dimValue2, err := b.evalExpression(strings.TrimSpace(dimensions[1]))
			if err != nil {
				return NewBASICError(ErrCategoryEvaluation, "INVALID_ARRAY_DIMENSION_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("DIM")
			}
			if !dimValue2.IsNumeric {
				return NewBASICError(ErrCategoryEvaluation, "INVALID_ARRAY_DIMENSION_TYPE", b.currentLine == 0, b.currentLine).WithCommand("DIM")
			}

			size2 = int(math.Round(dimValue2.NumValue))
			if size2 < 0 {
				return NewBASICError(ErrCategoryEvaluation, "INVALID_ARRAY_INDEX", b.currentLine == 0, b.currentLine).WithCommand("DIM")
			}
		}
		// Bestimme, ob es sich um ein String- oder numerisches Array handelt
		isString := strings.HasSuffix(varName, "$")

		// Speichere die Array-Definition
		arrayKey := varName + "("

		if size2 == -1 {
			// 1D-Array
			if isString {
				// Für String-Arrays setzen wir einen leeren String als Standardwert
				b.variables[arrayKey+"SIZE"] = BASICValue{NumValue: float64(size1), IsNumeric: true}
				for i := 0; i <= size1; i++ {
					b.variables[fmt.Sprintf("%s%d)", varName, i)] = BASICValue{StrValue: "", IsNumeric: false}
				}
			} else {
				// Für numerische Arrays setzen wir 0 als Standardwert
				b.variables[arrayKey+"SIZE"] = BASICValue{NumValue: float64(size1), IsNumeric: true}
				for i := 0; i <= size1; i++ {
					b.variables[fmt.Sprintf("%s%d)", varName, i)] = BASICValue{NumValue: 0, IsNumeric: true}
				}
			}
		} else {
			// 2D-Array
			b.variables[arrayKey+"DIMS"] = BASICValue{NumValue: 2, IsNumeric: true}
			b.variables[arrayKey+"SIZE1"] = BASICValue{NumValue: float64(size1), IsNumeric: true}
			b.variables[arrayKey+"SIZE2"] = BASICValue{NumValue: float64(size2), IsNumeric: true}

			if isString {
				// Für String-Arrays setzen wir einen leeren String als Standardwert
				for i := 0; i <= size1; i++ {
					for j := 0; j <= size2; j++ {
						b.variables[fmt.Sprintf("%s%d,%d)", varName, i, j)] = BASICValue{StrValue: "", IsNumeric: false}
					}
				}
			} else {
				// Für numerische Arrays setzen wir 0 als Standardwert
				for i := 0; i <= size1; i++ {
					for j := 0; j <= size2; j++ {
						b.variables[fmt.Sprintf("%s%d,%d)", varName, i, j)] = BASICValue{NumValue: 0, IsNumeric: true}
					}
				}
			}
		}
	}

	return nil
}

// parseStringArrayReference handles string array references like W$(I) for 1D arrays
// and W$(I,J) for 2D arrays
// It assumes the identifier token has already been consumed
func (p *exprParser) parseStringArrayReference(arrayName string) (BASICValue, error) {
	b := p.tb

	// Expect opening parenthesis
	_, err := p.expect(tokLParen)
	if err != nil {
		return BASICValue{}, err
	}

	// Parse the first index expression inside the parentheses
	index1Val, err := p.parseComparison()
	if err != nil {
		return BASICValue{}, fmt.Errorf("invalid array index: %w", err)
	}

	// Check that the index is numeric
	if !index1Val.IsNumeric {
		return BASICValue{}, NewBASICError(ErrCategoryEvaluation, "ARRAY_INDEX_NOT_NUMERIC", b.currentLine == 0, b.currentLine)
	}

	// Check if this is a 2D array (comma follows)
	if p.peek().typ == tokComma {
		p.next() // Consume comma

		// Parse the second index expression
		index2Val, err := p.parseComparison()
		if err != nil {
			return BASICValue{}, fmt.Errorf("invalid second array index: %w", err)
		}

		// Check that the second index is numeric
		if !index2Val.IsNumeric {
			return BASICValue{}, NewBASICError(ErrCategoryEvaluation, "ARRAY_INDEX_NOT_NUMERIC", b.currentLine == 0, b.currentLine)
		}

		// Expect closing parenthesis
		_, err = p.expect(tokRParen)
		if err != nil {
			return BASICValue{}, err
		}

		// Convert indices to int (rounding as needed)
		index1 := int(math.Round(index1Val.NumValue))
		index2 := int(math.Round(index2Val.NumValue))

		// Create the 2D variable name in the form "NAME(INDEX1,INDEX2)"
		indexedName := fmt.Sprintf("%s(%d,%d)", arrayName, index1, index2)

		// Look up the indexed variable
		value, ok := p.tb.variables[indexedName]
		if !ok {
			// If not found, return empty string (default value for string variables)
			return BASICValue{StrValue: "", IsNumeric: false}, nil
		}

		return value, nil
	}

	// 1D array - expect closing parenthesis
	_, err = p.expect(tokRParen)
	if err != nil {
		return BASICValue{}, err
	}

	// Convert index to int (rounding as needed)
	index := int(math.Round(index1Val.NumValue))

	// Create the variable name in the form "NAME(INDEX)"
	// This is the format used internally to store array elements
	indexedName := fmt.Sprintf("%s(%d)", arrayName, index)

	// Look up the indexed variable
	value, ok := p.tb.variables[indexedName]
	if !ok {
		// If not found, return empty string (default value for string variables)
		return BASICValue{StrValue: "", IsNumeric: false}, nil
	}

	return value, nil
}

// parseVariableWithIndex parses a variable name with an index like "W$(I)" or "A(I,J)" and returns
// the variable name and index values. If it's not an array expression, it returns
// the original expression as baseName, empty indices slice, and no error.
func (b *TinyBASIC) parseVariableWithIndex(varExpr string) (string, []int, error) {
	// Find the opening parenthesis
	openParenIndex := strings.Index(varExpr, "(")
	// Find the closing parenthesis
	closeParenIndex := strings.LastIndex(varExpr, ")")

	// Check if it looks like an array (both parentheses found and in correct order)
	if openParenIndex != -1 && closeParenIndex != -1 && closeParenIndex > openParenIndex {
		// Extract the base variable name and the index expression
		baseName := strings.TrimSpace(varExpr[:openParenIndex])
		indexExpr := strings.TrimSpace(varExpr[openParenIndex+1 : closeParenIndex])

		// Validate the base name
		if !isValidVarName(baseName) {
			return "", nil, fmt.Errorf("%w: invalid variable name '%s' for array", ErrSyntaxError, baseName)
		}

		// Check if this is a 2D array (contains comma)
		if strings.Contains(indexExpr, ",") {
			// Parse 2D array indices
			indexParts := strings.Split(indexExpr, ",")
			if len(indexParts) != 2 {
				return "", nil, fmt.Errorf("invalid 2D array syntax: expected exactly 2 indices, got %d", len(indexParts))
			}

			indices := make([]int, 2)
			for i, part := range indexParts {
				indexVal, err := b.evalExpression(strings.TrimSpace(part))
				if err != nil {
					return "", nil, fmt.Errorf("invalid array index expression '%s': %w", strings.TrimSpace(part), err)
				}

				// Check that the index is numeric
				if !indexVal.IsNumeric {
					return "", nil, fmt.Errorf("%w: array index must be numeric, got '%v'", ErrTypeMismatch, indexVal)
				}

				// Convert to int and store
				indices[i] = int(math.Round(indexVal.NumValue))
			}
			return strings.ToUpper(baseName), indices, nil
		} else {
			// Parse 1D array index (original logic)
			indexVal, err := b.evalExpression(indexExpr)
			if err != nil {
				return "", nil, fmt.Errorf("invalid array index expression '%s': %w", indexExpr, err)
			}

			// Check that the index is numeric
			if !indexVal.IsNumeric {
				return "", nil, fmt.Errorf("%w: array index must be numeric, got '%v'", ErrTypeMismatch, indexVal)
			}

			// Convert to int and return as single-element slice
			index := int(math.Round(indexVal.NumValue))
			return strings.ToUpper(baseName), []int{index}, nil
		}
	}

	// Not an array expression, return original (trimmed) expression as baseName and empty indices
	// This indicates it should be treated as a simple variable.
	return strings.TrimSpace(varExpr), []int{}, nil
}

// parseArrayReference handles array references like A(I) or A$(I) for 1D arrays,
// and A(I,J) or A$(I,J) for 2D arrays
// It determines whether it's a string or numeric array and calls the appropriate handler
// It assumes the identifier token has already been consumed
func (p *exprParser) parseArrayReference(arrayName string) (BASICValue, error) {
	// Check if this is a string array (name ending with $)
	isStringArray := strings.HasSuffix(arrayName, "$")

	// For string arrays, handle as W$(I) or W$(I,J)
	if isStringArray {
		return p.parseStringArrayReference(arrayName)
	}

	// For numeric arrays, handle as A(I) or A(I,J)
	_, err := p.expect(tokLParen) // Consume '('
	if err != nil {
		return BASICValue{}, err
	}

	// Parse the first index expression inside the parentheses
	index1Val, err := p.parseComparison()
	if err != nil {
		return BASICValue{}, fmt.Errorf("invalid array index: %w", err)
	}

	// Check that the index is numeric
	if !index1Val.IsNumeric {
		return BASICValue{}, NewBASICError(ErrCategoryEvaluation, "ARRAY_INDEX_NOT_NUMERIC", p.tb.currentLine == 0, p.tb.currentLine)
	}

	// Check if this is a 2D array (comma follows)
	if p.peek().typ == tokComma {
		p.next() // Consume comma

		// Parse the second index expression
		index2Val, err := p.parseComparison()
		if err != nil {
			return BASICValue{}, fmt.Errorf("invalid second array index: %w", err)
		}

		// Check that the second index is numeric
		if !index2Val.IsNumeric {
			return BASICValue{}, NewBASICError(ErrCategoryEvaluation, "ARRAY_INDEX_NOT_NUMERIC", p.tb.currentLine == 0, p.tb.currentLine)
		}

		// Expect closing parenthesis
		_, err = p.expect(tokRParen)
		if err != nil {
			return BASICValue{}, err
		}

		// Convert indices to int (rounding as needed)
		index1 := int(math.Round(index1Val.NumValue))
		index2 := int(math.Round(index2Val.NumValue))

		// Create the 2D variable name in the form "NAME(INDEX1,INDEX2)"
		indexedName := fmt.Sprintf("%s(%d,%d)", arrayName, index1, index2)

		// Look up the indexed variable
		value, ok := p.tb.variables[indexedName]
		if !ok {
			// If not found, return 0 (default value for numeric variables)
			return BASICValue{NumValue: 0, IsNumeric: true}, nil
		}

		return value, nil
	}

	// 1D array - expect closing parenthesis
	_, err = p.expect(tokRParen)
	if err != nil {
		return BASICValue{}, err
	}

	// Convert index to int (rounding as needed)
	index := int(math.Round(index1Val.NumValue))

	// Create the variable name in the form "NAME(INDEX)"
	// This is the format used internally to store array elements
	indexedName := fmt.Sprintf("%s(%d)", arrayName, index)

	// Look up the indexed variable
	value, ok := p.tb.variables[indexedName]
	if !ok {
		// If not found, return 0 (default value for numeric variables)
		return BASICValue{NumValue: 0, IsNumeric: true}, nil
	}

	return value, nil
}
