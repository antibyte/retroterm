package tinybasic

import (
	"fmt"
	"strconv"
	"strings"
)

// cmdLet assigns a value to a variable. Assumes lock is held.
func (b *TinyBASIC) cmdLet(args string) error {
	cleanedArgs := args
	// Remove "LET " or "LET\t" from the beginning of the string, case-insensitive.
	if len(cleanedArgs) > 4 && strings.EqualFold(cleanedArgs[:3], "LET") && (cleanedArgs[3] == ' ' || cleanedArgs[3] == '\t') {
		cleanedArgs = strings.TrimSpace(cleanedArgs[4:])
	}
	eqIdx := strings.Index(cleanedArgs, "=")
	if eqIdx == -1 {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_EQUALS", b.currentLine == 0, b.currentLine).WithCommand("LET").WithUsageHint("LET var = expr")
	}
	varName := strings.TrimSpace(cleanedArgs[:eqIdx])
	exprStr := strings.TrimSpace(cleanedArgs[eqIdx+1:])
	if varName == "" {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("LET")
	}
	if exprStr == "" {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("LET")
	}

	// Store the original varName before ToUpper for later use with underscores

	originalVarNameWithCase := varName
	varName = strings.ToUpper(varName)
	// parseVariableWithIndex expects the interpreter (b) as receiver
	baseVarName, arrayIndices, errPVWI := b.parseVariableWithIndex(originalVarNameWithCase) // Verwende originalVarNameWithCase

	finalVarNameToValidate := varName
	// If parseVariableWithIndex succeeded and returned a baseVarName (i.e., it was an array)
	// or if no error occurred (simple variable), then validate the corresponding name.
	if errPVWI == nil {
		if baseVarName != "" { // It was an array expression
			finalVarNameToValidate = strings.ToUpper(baseVarName)
		} else { // It was a simple variable, not an array expression
			finalVarNameToValidate = strings.ToUpper(originalVarNameWithCase)
		}
	} else {
		// An error from parseVariableWithIndex that is not nil indicates a syntax problem,
		// e.g. "A(B" or "A(1+)"

		// In this case, you could choose to validate the original varName (already ToUpper)
		// or return the error directly. Here we validate the ToUpper name.
		// return WrapError(errPVWI, "LET", b.currentLine == 0, b.currentLine) // Option: return error directly
	}
	if !isValidVarName(finalVarNameToValidate) {
		return NewBASICError(ErrCategorySyntax, "INVALID_VARIABLE_NAME", b.currentLine == 0, b.currentLine).WithCommand("LET")
	}

	value, err := b.evalExpression(exprStr)
	if err != nil {
		return WrapError(err, "LET", b.currentLine == 0, b.currentLine)
	}

	isStringVar := strings.HasSuffix(finalVarNameToValidate, "$") // Validate against the final name
	if isStringVar {
		var strVal string
		if value.IsNumeric {
			strVal = fmt.Sprintf("%g", value.NumValue)
		} else {
			strVal = value.StrValue
		}
		value = BASICValue{StrValue: strVal, IsNumeric: false}
	} else if !value.IsNumeric {
		numVal, convErr := parseBasicVal(value.StrValue) // Call to utilities.go
		if convErr != nil {
			return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("LET")
		}
		value = BASICValue{NumValue: numVal, IsNumeric: true}
	}
	if len(arrayIndices) > 0 {
		// Array-Zugriff
		var arrayVarName string
		if len(arrayIndices) == 1 {
			// 1D-Array
			arrayVarName = fmt.Sprintf("%s(%d)", finalVarNameToValidate, arrayIndices[0])
		} else if len(arrayIndices) == 2 {
			// 2D-Array
			arrayVarName = fmt.Sprintf("%s(%d,%d)", finalVarNameToValidate, arrayIndices[0], arrayIndices[1])
		} else {
			return NewBASICError(ErrCategorySyntax, "INVALID_ARRAY_DIMENSIONS", b.currentLine == 0, b.currentLine).WithCommand("LET")
		}
		b.variables[arrayVarName] = value
	} else {
		b.variables[finalVarNameToValidate] = value // Store with the validated name (uppercase, no index part)

		// Handle variable names with underscores in the original case (before ToUpper)
		// This is a specific requirement to maintain compatibility with certain BASIC dialects,
		// where \`my_var\` and \`MYVAR\` could be treated differently or \`my_var\` is used directly.
		// Here, \`originalVarNameWithCase\` is used, which contains the name before \`ToUpper\`.
		if strings.Contains(originalVarNameWithCase, "_") && !strings.Contains(originalVarNameWithCase, "(") { // Only store if it differs from the already stored \`finalVarNameToValidate\`
			// and no array brackets were present in the original name (since arrays are treated differently).
			if strings.ToUpper(originalVarNameWithCase) != finalVarNameToValidate { // Ensure we don't store the same thing twice
				b.variables[originalVarNameWithCase] = value
			}
		}
	}

	return nil
}

// cmdData does nothing at runtime. Assumes lock is held.
func (b *TinyBASIC) cmdData(args string) error {
	return nil // Handled by rebuildData.
}

// cmdRead reads items from DATA statements. Assumes lock is held.
func (b *TinyBASIC) cmdRead(args string) error {
	varNamesRaw := splitRespectingParentheses(args)
	if len(varNamesRaw) == 0 || strings.TrimSpace(varNamesRaw[0]) == "" {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("READ")
	}

	for _, varNameRaw := range varNamesRaw {
		varNameRaw = strings.TrimSpace(varNameRaw)
		if varNameRaw == "" {
			return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("READ")
		}

		if b.dataPointer >= len(b.data) {
			return NewBASICError(ErrCategoryExecution, "OUT_OF_DATA", b.currentLine == 0, b.currentLine).WithCommand("READ")
		}

		dataValueStr := b.data[b.dataPointer]
		b.dataPointer++
		isQuoted := strings.HasPrefix(dataValueStr, "\"") && strings.HasSuffix(dataValueStr, "\"")
		if isQuoted {
			dataValueStr = dataValueStr[1 : len(dataValueStr)-1]
			dataValueStr = strings.ReplaceAll(dataValueStr, "\"\"", "\"") // Replace escaped quotes
		}

		varName, arrayIndices, err := b.parseVariableWithIndex(varNameRaw)
		if err != nil {
			b.dataPointer--
			return WrapError(err, "READ", b.currentLine == 0, b.currentLine)
		}
		varName = strings.ToUpper(varName)
		if !isValidVarName(varName) { // Uses isValidVarName from utilities.go
			b.dataPointer--
			return NewBASICError(ErrCategorySyntax, "INVALID_VARIABLE_NAME", b.currentLine == 0, b.currentLine).WithCommand("READ")
		}

		if strings.HasSuffix(varName, "$") {
			if len(arrayIndices) > 0 {
				var arrayVarName string
				if len(arrayIndices) == 1 {
					// 1D-Array
					arrayVarName = fmt.Sprintf("%s(%d)", varName, arrayIndices[0])
				} else if len(arrayIndices) == 2 {
					// 2D-Array
					arrayVarName = fmt.Sprintf("%s(%d,%d)", varName, arrayIndices[0], arrayIndices[1])
				} else {
					b.dataPointer--
					return NewBASICError(ErrCategorySyntax, "INVALID_ARRAY_DIMENSIONS", b.currentLine == 0, b.currentLine).WithCommand("READ")
				}
				b.variables[arrayVarName] = BASICValue{StrValue: dataValueStr, IsNumeric: false}
			} else {
				b.variables[varName] = BASICValue{StrValue: dataValueStr, IsNumeric: false}
			}
		} else {
			f, err := strconv.ParseFloat(dataValueStr, 64)
			if err != nil {
				b.dataPointer--
				return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("READ")
			}
			if len(arrayIndices) > 0 {
				var arrayVarName string
				if len(arrayIndices) == 1 {
					// 1D-Array
					arrayVarName = fmt.Sprintf("%s(%d)", varName, arrayIndices[0])
				} else if len(arrayIndices) == 2 {
					// 2D-Array
					arrayVarName = fmt.Sprintf("%s(%d,%d)", varName, arrayIndices[0], arrayIndices[1])
				} else {
					b.dataPointer--
					return NewBASICError(ErrCategorySyntax, "INVALID_ARRAY_DIMENSIONS", b.currentLine == 0, b.currentLine).WithCommand("READ")
				}
				b.variables[arrayVarName] = BASICValue{NumValue: f, IsNumeric: true}
			} else {
				b.variables[varName] = BASICValue{NumValue: f, IsNumeric: true}
			}
		}
	}
	return nil
}

// cmdRestore resets the DATA pointer. Assumes lock is held.
func (b *TinyBASIC) cmdRestore(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("RESTORE")
	}
	b.dataPointer = 0
	return nil
}

// rebuildData reconstructs the data list from all DATA statements in the program.
// It should be called when the program is loaded or modified.
func (b *TinyBASIC) rebuildData() {
	b.data = make([]string, 0)
	b.dataPointer = 0 // Reset data pointer
	// b.programLines is already sorted, so we can iterate over it directly.
	for _, lineNum := range b.programLines { // Iterate in sorted order
		line := b.program[lineNum]
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToUpper(trimmedLine), "DATA ") {
			dataStr := strings.TrimSpace(trimmedLine[5:]) // Get the part after "DATA "
			items := parseDataItems(dataStr)              // Use the new local helper
			b.data = append(b.data, items...)
		}
	}
}

// parseDataItems splits a string from a DATA statement into individual items.
// It handles comma-separated values and trims whitespace.
// It also handles quoted strings properly.
func parseDataItems(dataStr string) []string {
	if dataStr == "" {
		return []string{}
	}

	var items []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(dataStr); i++ {
		char := dataStr[i]

		if char == '"' {
			if inQuotes && i+1 < len(dataStr) && dataStr[i+1] == '"' {
				// Escaped quote ""
				current.WriteByte('"')
				i++ // Skip next quote
			} else {
				// Toggle quote state
				inQuotes = !inQuotes
				current.WriteByte('"')
			}
		} else if char == ',' && !inQuotes {
			// Comma outside quotes - end of item
			items = append(items, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(char)
		}
	}

	// Add the last item
	if current.Len() > 0 {
		items = append(items, strings.TrimSpace(current.String()))
	}

	return items
}
