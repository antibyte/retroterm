package tinybasic

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func basicValueToInt(val BASICValue) (int, error) {
	if val.IsNumeric {
		// Runden statt abschneiden!
		return int(math.Round(val.NumValue)), nil
	}
	return strconv.Atoi(strings.TrimSpace(val.StrValue))
}

func basicValueToString(val BASICValue) (string, error) {
	if !val.IsNumeric {
		return val.StrValue, nil
	}
	return fmt.Sprintf("%g", val.NumValue), nil
}

// ConditionResult holds the outcome of evaluating an IF condition.
type ConditionResult struct {
	isTrue   bool   // Whether the condition is true
	thenStmt string // The part after THEN
	elseStmt string // The part after ELSE (if present)
	hasElse  bool   // Whether there is an ELSE part
}

// --- Helper functions and types for the interpreter (from tinybasic.old) ---

func isAlpha(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}

func isAlphaNum(b byte) bool {
	return isAlpha(b) || (b >= '0' && b <= '9') || b == '_'
}

func formatBasicFloat(f float64) string {
	// BASIC-typical formatting: no unnecessary decimal places
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%g", f)
}

// normalizeLogicalExpression prepares an expression for the recognition
// of logical operators by ensuring that AND and OR
// are recognized as standalone tokens with spaces.
func normalizeLogicalExpression(expr string) string {
	// Create temporary markers for strings
	var sb strings.Builder
	inString := false

	// Protect strings in the input
	for i := 0; i < len(expr); i++ {
		ch := expr[i]
		if ch == '"' {
			// Recognize string literals
			inString = !inString
			sb.WriteByte(ch)
		} else if inString {
			// Keep characters in strings unchanged
			sb.WriteByte(ch)
		} else {
			// Normalize outside of strings
			sb.WriteByte(ch)
		}
	}

	// Result of the first phase
	result := sb.String()

	// Dictionary of logical operators
	logicalOps := []string{"AND", "OR"}

	// Convert the expression to uppercase for searching
	upperExpr := strings.ToUpper(result)

	// First identify and mark the operators
	for _, op := range logicalOps {
		var processedResult string
		pos := 0

		for {
			// Find the next operator in the uppercase version
			opPos := strings.Index(upperExpr[pos:], op)
			if opPos == -1 {
				// No further operator found
				processedResult += result[pos:]
				break
			}

			// Calculate absolute position
			opPos += pos
			opEnd := opPos + len(op)

			// Check if it's a standalone operator
			isStandalone := true

			// Check if the character before is not part of an identifier
			if opPos > 0 {
				prevChar := upperExpr[opPos-1]
				if (prevChar >= 'A' && prevChar <= 'Z') ||
					(prevChar >= '0' && prevChar <= '9') ||
					prevChar == '_' || prevChar == '$' {
					isStandalone = false
				}
			}

			// Check if the character after is not part of an identifier
			if opEnd < len(upperExpr) {
				nextChar := upperExpr[opEnd]
				if (nextChar >= 'A' && nextChar <= 'Z') ||
					(nextChar >= '0' && nextChar <= '9') ||
					nextChar == '_' || nextChar == '$' {
					isStandalone = false
				}
			}

			if isStandalone {
				// Copy unmodified part
				processedResult += result[pos:opPos]

				// Extract original case of the operator and surround with spaces
				processedResult += " " + op + " "
			} else {
				// Copy everything until the end of the operator unmodified
				processedResult += result[pos:opEnd]
			}

			// Update position
			pos = opEnd
		}

		// Prepare result for next iteration
		result = processedResult
		upperExpr = strings.ToUpper(result)
	}

	// Convert to uppercase for easier checking
	// (only for recognition, not for processing)
	upperResult := " " + strings.ToUpper(result) + " "
	// Removed: Debug log output
	return upperResult
}

// replaceLogicalToken replaces a logical token (AND/OR) with a token surrounded by spaces
// This function is no longer used in the new normalizeLogicalExpression implementation,
// but is kept for backward compatibility.
func replaceLogicalToken(expr string, token string) string {
	// The logic has been implemented directly in normalizeLogicalExpression
	// This function is just a wrapper for backward compatibility
	upperExpr := strings.ToUpper(expr)
	result := ""
	pos := 0

	for {
		// Find the next possible occurrence of the token
		tokenPos := strings.Index(upperExpr[pos:], token)
		if tokenPos == -1 {
			// No further occurrence found
			result += expr[pos:]
			break
		}

		// Calculate absolute position
		tokenPos += pos
		tokenEnd := tokenPos + len(token)

		// Check if it's a standalone token
		isStandalone := true

		// Check if the character before is not part of an identifier
		if tokenPos > 0 {
			prevChar := upperExpr[tokenPos-1]
			if (prevChar >= 'A' && prevChar <= 'Z') ||
				(prevChar >= '0' && prevChar <= '9') ||
				prevChar == '_' || prevChar == '$' {
				isStandalone = false
			}
		}

		// Check if the character after is not part of an identifier
		if tokenEnd < len(expr) {
			nextChar := upperExpr[tokenEnd]
			if (nextChar >= 'A' && nextChar <= 'Z') ||
				(nextChar >= '0' && nextChar <= '9') ||
				nextChar == '_' || nextChar == '$' {
				isStandalone = false
			}
		}

		if isStandalone {
			// Standalone token found, surrounded with spaces
			result += expr[pos:tokenPos] + " " + token + " "
		} else {
			// Part of another token, kept unchanged
			result += expr[pos:tokenEnd]
		}

		// Update position
		pos = tokenEnd
	}

	return result
}

func parseBasicVal(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

func splitRespectingQuotes(s string) []string {
	var res []string
	inQuote := false
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '"' {
			inQuote = !inQuote
		}
		if !inQuote && (s[i] == ' ' || s[i] == '\t') {
			if start < i {
				res = append(res, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		res = append(res, s[start:])
	}
	return res
}

func upperOutsideQuotes(s string) string {
	inQuote := false
	var b strings.Builder
	for _, r := range s {
		if r == '"' {
			inQuote = !inQuote
		}
		if !inQuote {
			b.WriteRune(unicode.ToUpper(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// --- Expression Parser Helpers ---
func isComparisonOperator(op string) bool {
	switch op {
	case "=", "<>", "<", ">", "<=", ">=":
		return true
	default:
		return false
	}
}

func isLogicalOperator(op string) bool {
	// Remove whitespace and convert to uppercase
	upperOp := strings.TrimSpace(strings.ToUpper(op))

	// Ensure we have exactly AND or OR (case-insensitive)
	// Important: The token must be exactly AND or OR, not part of a longer word
	if upperOp == "AND" || upperOp == "OR" {
		return true
	}

	// Extra Debug if AND/OR is found in the string but does not match exactly
	if strings.Contains(upperOp, "AND") || strings.Contains(upperOp, "OR") {
	}

	return false
}

func compareValues(left, right BASICValue, op string) (bool, error) {
	if left.IsNumeric != right.IsNumeric {
		return false, fmt.Errorf("%w: cannot compare number with string using '%s'", ErrTypeMismatch, op)
	}
	if left.IsNumeric {
		l, r := left.NumValue, right.NumValue
		switch op {
		case "=":
			return l == r, nil
		case "<>":
			return l != r, nil
		case "<":
			return l < r, nil
		case ">":
			return l > r, nil
		case "<=":
			return l <= r, nil
		case ">=":
			return l >= r, nil
		}
	} else {
		l, r := left.StrValue, right.StrValue
		switch op {
		case "=":
			return l == r, nil
		case "<>":
			return l != r, nil
		case "<":
			return l < r, nil
		case ">":
			return l > r, nil
		case "<=":
			return l <= r, nil
		case ">=":
			return l >= r, nil
		}
	}
	return false, fmt.Errorf("internal error: unknown comparison operator '%s'", op)
}

func tokenTypeToString(tokType int) string {
	switch tokType {
	case tokEOF:
		return "end of expression"
	case tokNumber:
		return "number"
	case tokString:
		return "string"
	case tokIdent:
		return "identifier"
	case tokOp:
		return "operator"
	case tokLParen:
		return "'('"
	case tokRParen:
		return "')'"
	case tokComma:
		return "','"
	case tokHash:
		return "'#'"
	default:
		return fmt.Sprintf("unknown token (%d)", tokType)
	}
}

func tokenizePrintArgs(s string) ([]string, []rune) {
	var items []string
	var seps []rune
	inQuote := false
	start := 0
	for i, r := range s {
		if r == '"' {
			inQuote = !inQuote
		}
		if !inQuote && (r == ',' || r == ';') {
			items = append(items, strings.TrimSpace(s[start:i]))
			seps = append(seps, r)
			start = i + 1
		}
	}
	if start < len(s) {
		items = append(items, strings.TrimSpace(s[start:]))
	}
	return items, seps
}

// isValidVarName checks if a variable name is valid according to TinyBASIC rules.
// Rules:
// 1. Must start with an uppercase letter (A-Z).
// 2. Subsequent characters can be uppercase letters (A-Z), digits (0-9), or underscores (_).
// / 3. The base name (before an optional '$' or array parenthesis) must NOT end with an underscore.
//  4. String variables must end with '$' (e.g., A$, MY_VAR$).
//  5. Array variables are identified by parenthesis, e.g., A( or A$(.
//     The part before '(' must be a valid variable name (simple or string).
func isValidVarName(name string) bool {
	if name == "" {
		return false
	}

	// If it ends with '(', it's an array name. Validate the part before '('.
	// This check is primarily for base names before indexing, e.g. "A" in "A(10)".
	// The full "A(" is not usually passed to isValidVarName directly, but "A" or "A$" would be.
	// However, if a name like "ARR(" is passed, we validate "ARR".
	if strings.HasSuffix(name, "(") {
		if len(name) == 1 { // Just "(" is invalid
			return false
		}
		return isValidVarName(name[:len(name)-1])
	}

	// Regex for general character validation: Starts with an uppercase letter,
	// followed by uppercase letters, numbers, or underscores. Optional $ at the end.
	// This regex allows names ending with an underscore, which is handled by a subsequent check.
	// The $ for string variables needs to be escaped as \$ in Go strings for regexp.
	allowedCharsRegex := `^[A-Z][A-Z0-9_]*(\$)?$`
	match, _ := regexp.MatchString(allowedCharsRegex, name)
	if !match {
		return false // Does not match basic pattern (e.g., starts with number, invalid char)
	}

	// Check if the "base" part of the name (before an optional $) ends with an underscore.
	nameToCheckSuffix := name
	hasDollarSign := strings.HasSuffix(name, "$")

	if hasDollarSign {
		if len(name) == 1 { // Just "$" is invalid
			return false // Should have been caught by regex not starting with A-Z
		}
		nameToCheckSuffix = name[:len(name)-1]
	}

	// After stripping a potential trailing '$', if nameToCheckSuffix is empty, it means original was just '$'.
	if nameToCheckSuffix == "" { // Should not happen if regex passed and name is not just "$"
		return false
	}

	if strings.HasSuffix(nameToCheckSuffix, "_") {
		// An underscore is only allowed if it's not the last character of the base name part.
		// E.g., "A_" is invalid. "A_B" is valid. "A_$" is invalid (base "A_").
		return false
	}

	return true // Passed all checks
}

// calculateTabWidth calculates the number of spaces needed
// to reach the next tab stop.
func calculateTabWidth(currentPosition int, tabSize int) int {
	if tabSize <= 0 {
		// Fallback to a default tab size if an invalid value is provided.
		// 8 is a common default.
		tabSize = 8
	}
	return tabSize - (currentPosition % tabSize)
}

// getStringDisplayLength calculates the display length of a string.
// For simplicity, this version counts runes (characters).
func getStringDisplayLength(s string) int {
	return utf8.RuneCountInString(s)
}

// splitStatement splits a line of BASIC code into the command and its arguments.
// It handles simple cases and does not perform full parsing.
func splitStatement(line string) (string, string) {
	line = strings.TrimSpace(line)
	parts := strings.SplitN(line, " ", 2)
	command := ""
	if len(parts) > 0 {
		command = strings.ToUpper(parts[0])
	}
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}
	return command, args
}
