// Package tinybasic implements a simple BASIC interpreter.
package tinybasic

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/antibyte/retroterm/pkg/logger"
)

type exprParser struct {
	src    string
	tokens []token
	pos    int
	tb     *TinyBASIC
}

// evalExpression ist die zentrale Auswertungsfunktion für Ausdrücke
func (b *TinyBASIC) evalExpression(expr string) (BASICValue, error) {
	if strings.TrimSpace(expr) == "" {
		return BASICValue{}, NewBASICError(ErrCategorySyntax, "EXPECTED_EXPRESSION", true, 0)
	}
	p := &exprParser{src: expr, tb: b}
	err := p.tokenize()
	if err != nil {
		return BASICValue{}, WrapError(err, "EXPRESSION", true, 0)
	}

	if len(p.tokens) == 0 {
		return BASICValue{}, NewBASICError(ErrCategorySyntax, "EXPECTED_EXPRESSION", true, 0)
	}

	// Wir beginnen mit der höchsten Ebene der Auswertung
	var val BASICValue

	// Für alle Ausdrücke mit parseComparison beginnen
	// Dies wird automatisch die logischen Operatoren AND/OR behandeln,
	// da parseComparison bei Bedarf parseLogical aufruft
	val, err = p.parseComparison()

	if err != nil {
		return BASICValue{}, WrapError(err, "EXPRESSION", true, 0)
	}

	// Prüfen, ob wir am Ende des Ausdrucks sind
	if p.peek().typ != tokEOF {
		return BASICValue{}, NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", true, 0)
	}

	return val, nil
}

// tokenize breaks the expression string into tokens.
func (p *exprParser) tokenize() error {
	p.tokens = make([]token, 0, len(p.src)/2+1) // Pre-allocate estimate.
	s := p.src
	i := 0
	for i < len(s) {
		startPos := i
		r, size := utf8.DecodeRuneInString(s[i:]) // Use runes for safety.

		switch {
		case r == ' ' || r == '\t':
			i += size // Skip whitespace.
		case r >= '0' && r <= '9' || r == '.': // Start of number.
			numStart := i
			foundDecimal := (r == '.')
			i += size
			for i < len(s) {
				r2, size2 := utf8.DecodeRuneInString(s[i:])
				if r2 >= '0' && r2 <= '9' {
					i += size2
				} else if r2 == '.' && !foundDecimal {
					foundDecimal = true
					i += size2
				} else {
					break
				} // End of number part.
			}
			// Handle case like "." or "1."
			numStr := s[numStart:i]
			if numStr == "." {
				return fmt.Errorf("invalid number '.' at position %d", numStart)
			}
			p.tokens = append(p.tokens, token{typ: tokNumber, val: numStr, pos: startPos})
		case r == '"': // String literal.
			//strStart := i + size // Position after opening quote.
			i += size
			content := &strings.Builder{}
			foundEndQuote := false
			for i < len(s) {
				r2, size2 := utf8.DecodeRuneInString(s[i:])
				if r2 == '"' {
					// Check for escaped quote "".
					if i+size2 < len(s) && s[i+size2] == '"' {
						content.WriteRune('"') // Add single quote.
						i += size2 * 2         // Skip both quotes.
					} else {
						foundEndQuote = true
						i += size2 // Consume closing quote.
						break
					}
				} else {
					content.WriteRune(r2)
					i += size2
				}
			}
			if !foundEndQuote {
				return NewBASICError(ErrCategorySyntax, "MISSING_QUOTES", true, 0)
			}
			p.tokens = append(p.tokens, token{typ: tokString, val: content.String(), pos: startPos})
		case r == '(':
			p.tokens = append(p.tokens, token{typ: tokLParen, val: "(", pos: startPos})
			i += size
		case r == ')':
			p.tokens = append(p.tokens, token{typ: tokRParen, val: ")", pos: startPos})
			i += size
		case r == ',':
			p.tokens = append(p.tokens, token{typ: tokComma, val: ",", pos: startPos})
			i += size
		case r == '#':
			p.tokens = append(p.tokens, token{typ: tokHash, val: "#", pos: startPos})
			i += size
		case r == '+' || r == '-' || r == '*' || r == '/' || r == '^': // Arithmetic operators.
			p.tokens = append(p.tokens, token{typ: tokOp, val: string(r), pos: startPos})
			i += size
		case r == '=' || r == '<' || r == '>': // Comparison operators.
			opStart := i
			op := string(r)
			i += size
			if i < len(s) {
				nextChar := s[i] // Peek next byte.
				if (r == '<' && (nextChar == '>' || nextChar == '=')) || (r == '>' && nextChar == '=') {
					op += string(nextChar)
					i++ // Consume second char of operator.
				}
			}
			p.tokens = append(p.tokens, token{typ: tokOp, val: op, pos: opStart})
		case isAlpha(byte(r)): // Identifier (variable or function).
			identStart := i
			i += size
			for i < len(s) {
				r2, size2 := utf8.DecodeRuneInString(s[i:])
				if isAlphaNum(byte(r2)) || byte(r2) == '_' {
					i += size2
				} else {
					break
				}
			}
			// Check for optional trailing '$'.
			if i < len(s) && s[i] == '$' {
				// Ensure $ is last or followed by non-alphanum.
				isLast := (i+1 == len(s))
				var nextRune rune = 0
				if !isLast {
					nextRune, _ = utf8.DecodeRuneInString(s[i+1:])
				}
				if isLast || !isAlphaNum(byte(nextRune)) {
					i++ // Include '$'.
				}
			} // Prüfe, ob es sich um den MOD-Operator oder logische Operatoren handelt
			identVal := s[identStart:i]
			upperVal := strings.ToUpper(identVal)

			// Verbesserte Erkennung von Operatoren - verwende immer konsistente Großschreibung
			switch upperVal {
			case "MOD":
				p.tokens = append(p.tokens, token{typ: tokOp, val: "MOD", pos: identStart})
			case "AND":
				p.tokens = append(p.tokens, token{typ: tokOp, val: "AND", pos: identStart})
			case "OR":
				p.tokens = append(p.tokens, token{typ: tokOp, val: "OR", pos: identStart})
			default:
				// Normaler Bezeichner
				p.tokens = append(p.tokens, token{typ: tokIdent, val: identVal, pos: identStart})
			}
		default:
			return fmt.Errorf("unexpected character '%c' at position %d", r, startPos)
		}
	}
	p.tokens = append(p.tokens, token{typ: tokEOF, pos: i})
	return nil
}

// Parser helpers: next, peek, expect.
func (p *exprParser) next() token {
	if p.pos >= len(p.tokens) {
		panic(fmt.Sprintf("Token-Index out of bounds: pos=%d, len=%d, src=%q", p.pos, len(p.tokens), p.src))
	}
	tok := p.tokens[p.pos]
	p.pos++
	return tok
}
func (p *exprParser) peek() token {
	if p.pos >= len(p.tokens) {
		panic(fmt.Sprintf("Token-Index out of bounds: pos=%d, len=%d, src=%q", p.pos, len(p.tokens), p.src))
	}
	return p.tokens[p.pos]
}
func (p *exprParser) expect(expectedType int) (token, error) {
	tok := p.next()
	if tok.typ != expectedType {
		return tok, fmt.Errorf("%w: expected %s but got %s ('%s') at position %d", ErrSyntaxError, tokenTypeToString(expectedType), tokenTypeToString(tok.typ), tok.val, tok.pos)
	}
	return tok, nil
}

// Parsing rules (recursive descent).
// parseComparison: comparison = expr [compOp expr]
func (p *exprParser) parseComparison() (BASICValue, error) {
	left, err := p.parseExpr()
	if err != nil {
		return BASICValue{}, err
	}

	// Prüfen, ob ein Vergleichsoperator folgt
	tok := p.peek()
	if tok.typ == tokOp && isComparisonOperator(tok.val) {
		op := tok.val
		p.next() // Konsumiere den Operator

		right, err := p.parseExpr()
		if err != nil {
			return BASICValue{}, err
		}

		result, err := compareValues(left, right, op)
		if err != nil {
			return BASICValue{}, err
		}

		// Konvertiere das Ergebnis zu einem booleschen BASIC-Wert
		numResult := 0.0 // BASIC false = 0.
		if result {
			numResult = -1.0 // BASIC true = -1.
		}

		left = BASICValue{NumValue: numResult, IsNumeric: true}
	} // Prüfe auf logische Operatoren (AND/OR)
	nextTok := p.peek()

	// Strikte Erkennung von logischen Operatoren
	upperVal := strings.ToUpper(strings.TrimSpace(nextTok.val))
	isLogical := nextTok.typ == tokOp && (upperVal == "AND" || upperVal == "OR")

	if isLogical {
		// Gefunden, leite weiter an parseLogical
		return p.parseLogical(left)
	}

	// Kein logischer Operator, gib das Ergebnis zurück
	return left, nil
}

// parseLogical verarbeitet logische Operatoren wie AND und OR
// Diese Methode ist jetzt in pkg/tinybasic/logical_operators.go definiert

// parseExpr: expr = term {(+|-) term}
func (p *exprParser) parseExpr() (BASICValue, error) {
	left, err := p.parseTerm()
	if err != nil {
		return left, err
	}
	for {
		tok := p.peek()
		if tok.typ == tokOp && (tok.val == "+" || tok.val == "-") {
			op := tok.val
			p.next()
			right, err := p.parseTerm()
			if err != nil {
				return left, err
			}
			if op == "+" {
				// Korrektur: Wenn einer der Operanden ein String ist, immer Stringverkettung
				var lstr, rstr string
				if left.IsNumeric {
					lstr = fmt.Sprintf("%g", left.NumValue)
				} else {
					lstr = left.StrValue
				}
				if right.IsNumeric {
					rstr = fmt.Sprintf("%g", right.NumValue)
				} else {
					rstr = right.StrValue
				}
				if !left.IsNumeric || !right.IsNumeric {
					left = BASICValue{StrValue: lstr + rstr, IsNumeric: false}
				} else {
					left.NumValue += right.NumValue
				}
			} else {
				if !left.IsNumeric || !right.IsNumeric {
					return BASICValue{}, fmt.Errorf("%w: subtraction requires numeric operands near '%s'", ErrTypeMismatch, op)
				}
				left.NumValue -= right.NumValue
			}
		} else {
			break
		}
	}
	return left, nil
}

// parseTerm: term = factor {(*|/) factor}
func (p *exprParser) parseTerm() (BASICValue, error) {
	left, err := p.parseUnary()
	if err != nil {
		return left, err
	}
	for {
		tok := p.peek()
		if tok.typ == tokOp && (tok.val == "*" || tok.val == "/" || tok.val == "MOD") {
			op := tok.val
			p.next()
			right, err := p.parseUnary()
			if err != nil {
				return left, err
			}
			if !left.IsNumeric || !right.IsNumeric {
				return BASICValue{}, fmt.Errorf("%w: multiplication/division/modulo requires numeric operands near '%s'", ErrTypeMismatch, op)
			}
			if op == "*" {
				left.NumValue *= right.NumValue
			} else if op == "/" { // Division.
				if right.NumValue == 0 {
					return BASICValue{}, ErrDivisionByZero
				}
				left.NumValue /= right.NumValue
			} else if op == "MOD" { // Modulo
				if right.NumValue == 0 {
					return BASICValue{}, ErrDivisionByZero
				}
				// Implementierung des Modulo-Operators
				left.NumValue = math.Mod(left.NumValue, right.NumValue)
				// BASIC-typisch: Stelle sicher, dass das Ergebnis positiv ist
				if left.NumValue < 0 {
					left.NumValue += right.NumValue
				}
			}
		} else {
			break
		}
	}
	return left, nil
}

// parseUnary: unary = [+| -] factor
func (p *exprParser) parseUnary() (BASICValue, error) {
	tok := p.peek()
	if tok.typ == tokOp && (tok.val == "+" || tok.val == "-") {
		op := tok.val
		p.next()
		operand, err := p.parseFactor() // Parse the operand.
		if err != nil {
			return operand, err
		}
		if !operand.IsNumeric {
			return BASICValue{}, fmt.Errorf("%w: unary '%s' requires a numeric operand", ErrTypeMismatch, op)
		}
		if op == "-" {
			operand.NumValue = -operand.NumValue
		}
		// Unary '+' has no effect.
		return operand, nil
	}
	return p.parseFactor() // No unary op, parse factor directly.
}

// parseFactor: factor = primary [^ factor] (Exponentiation NYI)
func (p *exprParser) parseFactor() (BASICValue, error) {
	// TODO: Implement exponentiation (^) if needed. Requires right-associativity handling.
	return p.parsePrimary()
}

// parsePrimary: number | string | ident | functionCall | (comparison) | #number
func (p *exprParser) parsePrimary() (BASICValue, error) {
	tok := p.peek()
	switch tok.typ {
	case tokNumber:
		p.next()
		n, err := strconv.ParseFloat(tok.val, 64)
		if err != nil {
			return BASICValue{}, fmt.Errorf("internal error: invalid number literal '%s': %w", tok.val, err)
		}
		return BASICValue{NumValue: n, IsNumeric: true}, nil
	case tokString:
		p.next()
		return BASICValue{StrValue: tok.val, IsNumeric: false}, nil
	case tokIdent:
		p.next()
		identName := tok.val                         // Keep original case for errors if needed.
		identNameUpper := strings.ToUpper(identName) // Check if we have an array reference with parentheses
		if p.peek().typ == tokLParen {               // Hier liegt ein Ausdruck mit Klammern vor - entweder ein Funktionsaufruf oder ein Array-Zugriff
			knownFunctions := []string{"ABS", "ATN", "COS", "EXP", "INT", "LOG", "RND", "SGN", "SIN", "SQR", "TAN",
				"CHR$", "LEFT$", "MID$", "RIGHT$", "STR$", "LEN", "ASC", "VAL", "EOF", "KEYSTATE", "KEYPRESSED", "COLLISION"}

			// Bessere Erkennung für String-Funktionen
			isFunction := false

			// 1. Direkte Übereinstimmung prüfen (für Fälle ohne $-Probleme)
			for _, funcName := range knownFunctions {
				if identNameUpper == funcName {
					isFunction = true
					break
				}
			}

			// 2. Wenn keine direkte Übereinstimmung, prüfen ob es eine String-Funktion sein könnte
			if !isFunction && strings.HasSuffix(identNameUpper, "$") {
				// Namen ohne $ vergleichen für STRING$-Funktionen
				baseName := strings.TrimSuffix(identNameUpper, "$")
				for _, funcName := range knownFunctions {
					if strings.HasSuffix(funcName, "$") &&
						strings.TrimSuffix(funcName, "$") == baseName {
						isFunction = true
						// Korrigiere den Funktionsnamen für den Aufruf
						identNameUpper = funcName
						identName = funcName
						break
					}
				}
				// Wenn es keine bekannte Funktion ist, dann ist es ein String-Array
				if !isFunction {
					// String array reference like W$(I)
					return p.parseArrayReference(identNameUpper)
				}
			}

			// 3. Wenn keine Funktion erkannt wurde und kein $-suffix, muss es ein numerisches Array sein
			if !isFunction {
				// Treat as array reference like A(I)
				return p.parseArrayReference(identNameUpper)
			}

			// Regular function call
			return p.parseFunctionCall(identName, tok.pos)
		}
		// Variable or constant like PI.
		if identNameUpper == "PI" {
			return BASICValue{NumValue: math.Pi, IsNumeric: true}, nil
		}
		// Look up variable (case-insensitive). Assumes lock is held by caller.
		// Spezielle Behandlung für INKEY$ - lock-free Zugriff
		if identNameUpper == "INKEY$" {
			// Direkter Zugriff ohne Locks - String-Zugriffe sind in Go atomisch
			return BASICValue{StrValue: p.tb.currentKey, IsNumeric: false}, nil
		}
		// 1. Versuchen wir zuerst mit dem ursprünglichen Namen (mit Unterstrichen)
		if v, ok := p.tb.variables[identName]; ok {
			return v, nil
		}
		// 2. Dann mit dem Namen in Großbuchstaben
		if v, ok := p.tb.variables[identNameUpper]; ok {
			return v, nil
		}
		// DEBUG: Variable nicht gefunden
		logger.Debug(logger.AreaTinyBasic, "Variable '%s' (upper: '%s') not found", identName, identNameUpper)
		fmt.Printf("DEBUG: Available variables: ")
		for k := range p.tb.variables {
			fmt.Printf("'%s' ", k)
		}
		fmt.Printf("\n")
		return BASICValue{}, NewBASICError(ErrCategoryEvaluation, "UNKNOWN_VARIABLE", p.tb.currentLine == 0, p.tb.currentLine).WithCommand("PRINT")

	case tokLParen:
		p.next()
		val, err := p.parseComparison() // Parse expression inside parens.
		if err != nil {
			return val, err
		}
		_, err = p.expect(tokRParen) // Expect closing paren.
		return val, err
	case tokHash: // Handle file handle number like #1 used in EOF(1).
		p.next() // Consume '#'.
		numTok, err := p.expect(tokNumber)
		if err != nil {
			return BASICValue{}, fmt.Errorf("expected number after '#' for file handle: %w", err)
		}
		handleNum, _ := strconv.ParseFloat(numTok.val, 64)
		// Return the handle number itself. Function like EOF will use it.
		return BASICValue{NumValue: handleNum, IsNumeric: true}, nil
	default:
		// Fehlerbehandlung für unbekannte Token
		return BASICValue{}, fmt.Errorf("parsePrimary: Unexpected token type %v", tok.typ)
	}
}

// parseFunctionCall handles function calls. Assumes identifier consumed.
func (p *exprParser) parseFunctionCall(funcName string, namePos int) (BASICValue, error) {
	funcNameUpper := strings.ToUpper(funcName)
	_, err := p.expect(tokLParen) // Consume '('.
	if err != nil {
		return BASICValue{}, err
	}

	args := []BASICValue{}
	if p.peek().typ != tokRParen { // Check if there are arguments.
		for {
			argVal, err := p.parseComparison() // Parse argument expression.
			if err != nil {
				return BASICValue{}, fmt.Errorf("parsing argument for function %s: %w", funcName, err)
			}
			args = append(args, argVal)
			nextToken := p.peek()
			if nextToken.typ == tokComma {
				p.next()
			} else {
				break
			} // Consume comma or break loop.
		}
	}

	_, err = p.expect(tokRParen) // Expect closing ')'.
	if err != nil {
		return BASICValue{}, err
	}

	// Call TinyBASIC method to evaluate the function. Assumes lock is held by caller.
	return p.tb.evalBuiltinFunction(funcNameUpper, args, namePos)
}

// evalBuiltinFunction evaluates built-in functions. Assumes lock is held.
func (b *TinyBASIC) evalBuiltinFunction(funcNameUpper string, args []BASICValue, namePos int) (BASICValue, error) {
	argCount := len(args)
	errArgs := func(expected string) error { // Helper for argument errors.
		return fmt.Errorf("%w: %s requires %s at pos %d", ErrSyntaxError, funcNameUpper, expected, namePos)
	}
	errNumArg := func(n int) error { return errArgs(fmt.Sprintf("%d numeric argument(s)", n)) }
	errStrArg := func(n int) error { return errArgs(fmt.Sprintf("%d string argument(s)", n)) }
	errArgsRange := func(min, max int, typ string) error {
		return errArgs(fmt.Sprintf("%d-%d %s argument(s)", min, max, typ))
	}

	switch funcNameUpper {
	// Math Functions
	case "ABS":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		return BASICValue{NumValue: math.Abs(args[0].NumValue), IsNumeric: true}, nil
	case "SGN":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		var sgn float64 = 0
		if args[0].NumValue > 0 {
			sgn = 1
		} else if args[0].NumValue < 0 {
			sgn = -1
		}
		return BASICValue{NumValue: sgn, IsNumeric: true}, nil
	case "INT":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		return BASICValue{NumValue: math.Floor(args[0].NumValue), IsNumeric: true}, nil
	case "SIN":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		return BASICValue{NumValue: math.Sin(args[0].NumValue), IsNumeric: true}, nil
	case "COS":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		return BASICValue{NumValue: math.Cos(args[0].NumValue), IsNumeric: true}, nil
	case "TAN":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		return BASICValue{NumValue: math.Tan(args[0].NumValue), IsNumeric: true}, nil
	case "ATN":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		return BASICValue{NumValue: math.Atan(args[0].NumValue), IsNumeric: true}, nil
	case "EXP":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		return BASICValue{NumValue: math.Exp(args[0].NumValue), IsNumeric: true}, nil
	case "LOG":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		if args[0].NumValue <= 0 {
			return BASICValue{}, fmt.Errorf("%w: LOG argument must be > 0 at pos %d", ErrInvalidExpression, namePos)
		}
		return BASICValue{NumValue: math.Log(args[0].NumValue), IsNumeric: true}, nil
	case "SQR":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		if args[0].NumValue < 0 {
			return BASICValue{}, fmt.Errorf("%w: SQR argument must be >= 0 at pos %d", ErrInvalidExpression, namePos)
		}
		return BASICValue{NumValue: math.Sqrt(args[0].NumValue), IsNumeric: true}, nil

	case "RND":
		if argCount > 1 {
			return BASICValue{}, errArgs("0 or 1 arguments")
		}
		if argCount == 0 {
			// RND ohne Argument: Zufallszahl 0..1
			return BASICValue{NumValue: rand.Float64(), IsNumeric: true}, nil
		}
		if !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		param := args[0].NumValue
		if param > 0 {
			// Zufallszahl 0..param-1 (ganzzahlig)
			return BASICValue{NumValue: float64(rand.Intn(int(param))), IsNumeric: true}, nil
		} else if param < 0 {
			// Negativer Parameter - setze neuen Seed und gib Zufallszahl zurück
			rand.Seed(int64(math.Abs(param)))
			return BASICValue{NumValue: rand.Float64(), IsNumeric: true}, nil
		} else {
			// param == 0: Zufallszahl 0..1
			return BASICValue{NumValue: rand.Float64(), IsNumeric: true}, nil
		}

	// String Functions
	case "LEN":
		if argCount != 1 || args[0].IsNumeric {
			return BASICValue{}, errStrArg(1)
		}
		strLength := float64(utf8.RuneCountInString(args[0].StrValue))
		return BASICValue{NumValue: strLength, IsNumeric: true}, nil

	// Erweiterte Tastaturabfrage-Funktionen
	case "KEYSTATE":
		// KEYSTATE(key) - gibt 1 zurück wenn Taste gedrückt, 0 wenn losgelassen
		if argCount != 1 || args[0].IsNumeric {
			return BASICValue{}, errStrArg(1)
		}
		keyName := args[0].StrValue
		isPressed := b.GetKeyState(keyName)
		return BASICValue{NumValue: isPressed, IsNumeric: true}, nil

	case "KEYPRESSED":
		// KEYPRESSED(key) - alias für KEYSTATE für bessere Lesbarkeit
		if argCount != 1 || args[0].IsNumeric {
			return BASICValue{}, errStrArg(1)
		}
		keyName := args[0].StrValue
		isPressed := b.GetKeyState(keyName)
		return BASICValue{NumValue: isPressed, IsNumeric: true}, nil

	case "CHR$":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		charCode := int(math.Round(args[0].NumValue)) // Round to nearest int.
		return BASICValue{StrValue: string(rune(charCode)), IsNumeric: false}, nil
	case "LEFT$":
		if argCount != 2 || args[0].IsNumeric || !args[1].IsNumeric {
			return BASICValue{}, errArgs("string, number")
		}
		str := args[0].StrValue
		n := int(math.Round(args[1].NumValue))
		if n < 0 {
			n = 0
		}
		runes := []rune(str)
		if n > len(runes) {
			n = len(runes)
		}
		return BASICValue{StrValue: string(runes[:n]), IsNumeric: false}, nil
	case "RIGHT$":
		if argCount != 2 || args[0].IsNumeric || !args[1].IsNumeric {
			return BASICValue{}, errArgs("string, number")
		}
		str := args[0].StrValue
		n := int(math.Round(args[1].NumValue))
		if n < 0 {
			n = 0
		}
		runes := []rune(str)
		if n > len(runes) {
			n = len(runes)
		}
		start := len(runes) - n
		if start < 0 {
			start = 0
		}
		return BASICValue{StrValue: string(runes[start:]), IsNumeric: false}, nil
	case "MID$":
		if !(argCount == 2 || argCount == 3) || args[0].IsNumeric || !args[1].IsNumeric || (argCount == 3 && !args[2].IsNumeric) {
			return BASICValue{}, errArgsRange(2, 3, "string, number[, number]")
		}
		str := args[0].StrValue
		runes := []rune(str)
		startPos := int(math.Round(args[1].NumValue)) // 1-based index.
		length := len(runes)
		if argCount == 3 {
			length = int(math.Round(args[2].NumValue))
		}
		if startPos < 1 {
			startPos = 1
		}
		if length < 0 {
			length = 0
		}
		startIndex := startPos - 1 // 0-based.
		endIndex := startIndex + length
		if startIndex < 0 {
			startIndex = 0
		}
		if startIndex >= len(runes) {
			return BASICValue{StrValue: "", IsNumeric: false}, nil
		}
		if endIndex > len(runes) {
			endIndex = len(runes)
		}
		if endIndex < startIndex {
			endIndex = startIndex
		}
		return BASICValue{StrValue: string(runes[startIndex:endIndex]), IsNumeric: false}, nil
	case "STR$":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}
		return BASICValue{StrValue: formatBasicFloat(args[0].NumValue), IsNumeric: false}, nil
	case "VAL":
		if argCount != 1 || args[0].IsNumeric {
			return BASICValue{}, errStrArg(1)
		}
		numVal, _ := parseBasicVal(args[0].StrValue) // Use helper, ignore error (returns 0 on failure).
		return BASICValue{NumValue: numVal, IsNumeric: true}, nil

	// File I/O Function
	case "EOF":
		if argCount != 1 || !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		} // Expects handle number.
		return b.evalEOF(args[0]) // Call specific handler.

	case "COLLISION":
		// COLLISION(spriteID) - gibt Anzahl der Kollisionen zurück
		// COLLISION(spriteID, index) - gibt ID des kollidierenden Sprites zurück
		if argCount < 1 || argCount > 2 {
			return BASICValue{}, errArgs("1 or 2 arguments")
		}
		if !args[0].IsNumeric {
			return BASICValue{}, errNumArg(1)
		}

		spriteID := int(math.Round(args[0].NumValue))

		if argCount == 2 {
			// Zwei Parameter: Index für spezifischen kollidierenden Sprite
			if !args[1].IsNumeric {
				return BASICValue{}, errNumArg(2)
			}
			index := int(math.Round(args[1].NumValue))

			// Konvertiere zu String-Argumenten für cmdCollision
			argsStr := fmt.Sprintf("%d %d", spriteID, index)
			result, err := b.cmdCollision(argsStr)
			if err != nil {
				return BASICValue{}, fmt.Errorf("COLLISION error: %v", err)
			}
			return BASICValue{NumValue: float64(result), IsNumeric: true}, nil
		} else {
			// Ein Parameter: Anzahl der Kollisionen
			argsStr := fmt.Sprintf("%d", spriteID)
			result, err := b.cmdCollision(argsStr)
			if err != nil {
				return BASICValue{}, fmt.Errorf("COLLISION error: %v", err)
			}
			return BASICValue{NumValue: float64(result), IsNumeric: true}, nil
		}

	default:
		return BASICValue{}, fmt.Errorf("%w: unknown function '%s' at position %d", ErrUnknownCommand, funcNameUpper, namePos)
	}
}

// evalEOF checks EOF status for a file handle. Assumes lock is held.
func (b *TinyBASIC) evalEOF(handleArg BASICValue) (BASICValue, error) {
	handle := int(math.Round(handleArg.NumValue)) // Expect handle number.
	if handle <= 0 {
		return BASICValue{}, fmt.Errorf("%w: %d", ErrInvalidFileHandle, handle)
	}

	of, ok := b.openFiles[handle]
	if !ok {
		return BASICValue{}, fmt.Errorf("%w: %d", ErrFileNotOpen, handle)
	}
	if of.Mode != "INPUT" {
		return BASICValue{}, fmt.Errorf("%w: Handle #%d", ErrFileNotInInputMode, handle)
	}

	// Check if the *next* potential read would be past the end, skipping whitespace lines.
	tempPos := of.Pos
	eofReached := true
	for tempPos < len(of.Lines) {
		if strings.TrimSpace(of.Lines[tempPos]) != "" {
			eofReached = false // Found a readable line.
			break
		}
		tempPos++
	}
	result := 0.0 // BASIC false = 0.
	if eofReached {
		result = -1.0
	} // BASIC true = -1.
	return BASICValue{NumValue: result, IsNumeric: true}, nil
}

// parseCoords parses a comma-separated list of coordinate expressions.
// expectedCount: number of coordinates expected (-1 for any even number >= 4). Assumes lock is held.
func (b *TinyBASIC) parseCoords(args string, expectedCount int) ([]int, error) {
	parts := strings.Split(args, ",")
	var exprParts []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			exprParts = append(exprParts, trimmed)
		}
	}
	actualCount := len(exprParts)
	if expectedCount > 0 && actualCount != expectedCount {
		return nil, fmt.Errorf("%w: expected %d coordinates, got %d", ErrSyntaxError, expectedCount, actualCount)
	}
	if expectedCount == -1 && (actualCount < 4 || actualCount%2 != 0) {
		return nil, fmt.Errorf("%w: expected an even number of coordinates (>= 4), got %d", ErrSyntaxError, actualCount)
	}
	if actualCount == 0 {
		if expectedCount > 0 {
			return nil, fmt.Errorf("%w: expected %d coordinates, got 0", ErrSyntaxError, expectedCount)
		}
		if expectedCount == -1 {
			return nil, fmt.Errorf("%w: expected an even number of coordinates (>= 4), got 0", ErrSyntaxError)
		}
	}
	coords := make([]int, actualCount)
	for i, exprPart := range exprParts {
		val, err := b.evalExpression(exprPart)
		if err != nil || !val.IsNumeric {
			return nil, fmt.Errorf("%w near '%s': %v", ErrInvalidCoordinates, exprPart, err)
		}
		coords[i] = int(math.Round(val.NumValue))
	}
	return coords, nil
}

// parseGraphicsCommand validates and structures graphics commands. Assumes lock is held.
func (b *TinyBASIC) parseGraphicsCommand(command, args string) (map[string]interface{}, error) {
	switch command {
	case "PLOT":
		coords, err := b.parseCoords(args, 2)
		if err != nil {
			return nil, fmt.Errorf("PLOT: %w", err)
		}
		return map[string]interface{}{"action": "plot", "x": coords[0], "y": coords[1]}, nil
	case "DRAW":
		coords, err := b.parseCoords(args, 4)
		if err != nil {
			return nil, fmt.Errorf("DRAW: %w", err)
		}
		return map[string]interface{}{"action": "line", "x1": coords[0], "y1": coords[1], "x2": coords[2], "y2": coords[3]}, nil
	case "CIRCLE":
		coords, err := b.parseCoords(args, 3)
		if err != nil {
			return nil, fmt.Errorf("CIRCLE: %w", err)
		}
		return map[string]interface{}{"action": "circle", "x": coords[0], "y": coords[1], "radius": coords[2]}, nil
	case "RECT":
		coords, err := b.parseCoords(args, 4)
		if err != nil {
			return nil, fmt.Errorf("RECT: %w", err)
		}
		return map[string]interface{}{"action": "rect", "x": coords[0], "y": coords[1], "width": coords[2], "height": coords[3]}, nil
	case "FILL":
		coords, err := b.parseCoords(args, 2)
		if err != nil {
			return nil, fmt.Errorf("FILL: %w", err)
		}
		return map[string]interface{}{"action": "fill", "x": coords[0], "y": coords[1]}, nil
	case "INK":
		val, err := b.evalExpression(args)
		if err != nil || !val.IsNumeric {
			return nil, fmt.Errorf("INK: invalid color expression '%s': %w", args, err)
		}
		color := int(math.Round(val.NumValue))
		if color < 1 || color > 16 {
			return nil, ErrInvalidColor
		}
		return map[string]interface{}{"action": "ink", "color": color}, nil
	case "POLY":
		coords, err := b.parseCoords(args, -1)
		if err != nil {
			return nil, fmt.Errorf("POLY: %w", err)
		}
		if len(coords) < 4 || len(coords)%2 != 0 {
			return nil, fmt.Errorf("%w: POLY: Requires an even number of coordinates (>= 4)", ErrSyntaxError)
		}
		return map[string]interface{}{"action": "poly", "points": coords}, nil
	default:
		return nil, fmt.Errorf("internal error: unknown graphics command '%s'", command)
	}
}

// parseListRange parses optional start/end line numbers for LIST.
func parseListRange(args string) (int, int, error) {
	startLine := 0
	endLine := math.MaxInt32
	args = strings.TrimSpace(args)
	if args == "" {
		return startLine, endLine, nil
	}
	var parseErr error
	separator := ""
	if strings.Contains(args, "-") {
		separator = "-"
	} else if strings.Contains(args, ",") {
		separator = ","
	}
	if separator != "" {
		parts := strings.SplitN(args, separator, 2)
		p1 := strings.TrimSpace(parts[0])
		p2 := strings.TrimSpace(parts[1])
		if p1 != "" {
			startLine, parseErr = strconv.Atoi(p1)
			if parseErr != nil || startLine <= 0 {
				return 0, 0, fmt.Errorf("%w: LIST: Invalid start line '%s'", ErrInvalidLineNumber, p1)
			}
		}
		if p2 != "" {
			endLine, parseErr = strconv.Atoi(p2)
			if parseErr != nil || endLine <= 0 {
				return 0, 0, fmt.Errorf("%w: LIST: Invalid end line '%s'", ErrInvalidLineNumber, p2)
			}
		}
	} else {
		startLine, parseErr = strconv.Atoi(args)
		if parseErr != nil || startLine <= 0 {
			return 0, 0, fmt.Errorf("%w: LIST: Invalid line number '%s'", ErrInvalidLineNumber, args)
		}
		endLine = startLine
	}
	if endLine < startLine {
		return 0, 0, fmt.Errorf("%w: LIST: End line (%d) cannot be before start line (%d)", ErrSyntaxError, endLine, startLine)
	}
	return startLine, endLine, nil
}
