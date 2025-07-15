package tinybasic

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Token types for expression parsing
type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_NUMBER
	TOKEN_STRING
	TOKEN_IDENTIFIER
	TOKEN_PLUS
	TOKEN_MINUS
	TOKEN_MULTIPLY
	TOKEN_DIVIDE
	TOKEN_POWER
	TOKEN_MOD
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_EQ
	TOKEN_NE
	TOKEN_LT
	TOKEN_LE
	TOKEN_GT
	TOKEN_GE
	TOKEN_AND
	TOKEN_OR
	TOKEN_NOT
	TOKEN_COMMA
	TOKEN_SEMICOLON
)

// ExprToken represents a lexical token for expression parsing
type ExprToken struct {
	Type   TokenType
	Value  string
	NumVal float64
	StrVal string
}

// ExpressionLexer tokenizes BASIC expressions
type ExpressionLexer struct {
	input string
	pos   int
	char  byte
}

// NewExpressionLexer creates a new expression lexer
func NewExpressionLexer(input string) *ExpressionLexer {
	l := &ExpressionLexer{input: input}
	l.readChar()
	return l
}

// readChar advances the lexer position
func (l *ExpressionLexer) readChar() {
	if l.pos >= len(l.input) {
		l.char = 0 // EOF
	} else {
		l.char = l.input[l.pos]
	}
	l.pos++
}

// peekChar returns the next character without advancing
func (l *ExpressionLexer) peekChar() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

// skipWhitespace skips whitespace characters
func (l *ExpressionLexer) skipWhitespace() {
	for l.char == ' ' || l.char == '\t' {
		l.readChar()
	}
}

// readString reads a string literal
func (l *ExpressionLexer) readString() string {
	startPos := l.pos
	l.readChar() // Skip opening quote

	for l.char != '"' && l.char != 0 {
		if l.char == '"' && l.peekChar() == '"' {
			// Handle escaped quotes ""
			l.readChar()
			l.readChar()
		} else {
			l.readChar()
		}
	}

	if l.char == '"' {
		result := l.input[startPos : l.pos-1]
		l.readChar() // Skip closing quote
		// Handle escaped quotes in result
		result = strings.ReplaceAll(result, `""`, `"`)
		return result
	}

	// Unterminated string
	return l.input[startPos:]
}

// readNumber reads a numeric literal
func (l *ExpressionLexer) readNumber() (string, float64) {
	startPos := l.pos - 1

	for unicode.IsDigit(rune(l.char)) {
		l.readChar()
	}

	// Handle decimal point
	if l.char == '.' && unicode.IsDigit(rune(l.peekChar())) {
		l.readChar() // Skip dot
		for unicode.IsDigit(rune(l.char)) {
			l.readChar()
		}
	}

	numStr := l.input[startPos : l.pos-1]
	numVal, _ := strconv.ParseFloat(numStr, 64)
	return numStr, numVal
}

// readIdentifier reads an identifier or keyword
func (l *ExpressionLexer) readIdentifier() string {
	startPos := l.pos - 1

	for unicode.IsLetter(rune(l.char)) || unicode.IsDigit(rune(l.char)) || l.char == '$' || l.char == '_' {
		l.readChar()
	}

	return l.input[startPos : l.pos-1]
}

// NextToken returns the next token from the input
func (l *ExpressionLexer) NextToken() ExprToken {
	l.skipWhitespace()

	switch l.char {
	case 0:
		return ExprToken{Type: TOKEN_EOF}
	case '+':
		l.readChar()
		return ExprToken{Type: TOKEN_PLUS, Value: "+"}
	case '-':
		l.readChar()
		return ExprToken{Type: TOKEN_MINUS, Value: "-"}
	case '*':
		l.readChar()
		return ExprToken{Type: TOKEN_MULTIPLY, Value: "*"}
	case '/':
		l.readChar()
		return ExprToken{Type: TOKEN_DIVIDE, Value: "/"}
	case '^':
		l.readChar()
		return ExprToken{Type: TOKEN_POWER, Value: "^"}
	case '(':
		l.readChar()
		return ExprToken{Type: TOKEN_LPAREN, Value: "("}
	case ')':
		l.readChar()
		return ExprToken{Type: TOKEN_RPAREN, Value: ")"}
	case ',':
		l.readChar()
		return ExprToken{Type: TOKEN_COMMA, Value: ","}
	case ';':
		l.readChar()
		return ExprToken{Type: TOKEN_SEMICOLON, Value: ";"}
	case '=':
		l.readChar()
		return ExprToken{Type: TOKEN_EQ, Value: "="}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return ExprToken{Type: TOKEN_LE, Value: "<="}
		} else if l.peekChar() == '>' {
			l.readChar()
			l.readChar()
			return ExprToken{Type: TOKEN_NE, Value: "<>"}
		}
		l.readChar()
		return ExprToken{Type: TOKEN_LT, Value: "<"}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			l.readChar()
			return ExprToken{Type: TOKEN_GE, Value: ">="}
		}
		l.readChar()
		return ExprToken{Type: TOKEN_GT, Value: ">"}
	case '"':
		str := l.readString()
		return ExprToken{Type: TOKEN_STRING, Value: `"` + str + `"`, StrVal: str}
	default:
		if unicode.IsDigit(rune(l.char)) {
			numStr, numVal := l.readNumber()
			return ExprToken{Type: TOKEN_NUMBER, Value: numStr, NumVal: numVal}
		} else if unicode.IsLetter(rune(l.char)) {
			ident := l.readIdentifier()
			identUpper := strings.ToUpper(ident)

			// Check for keywords
			switch identUpper {
			case "MOD":
				return ExprToken{Type: TOKEN_MOD, Value: "MOD"}
			case "AND":
				return ExprToken{Type: TOKEN_AND, Value: "AND"}
			case "OR":
				return ExprToken{Type: TOKEN_OR, Value: "OR"}
			case "NOT":
				return ExprToken{Type: TOKEN_NOT, Value: "NOT"}
			default:
				return ExprToken{Type: TOKEN_IDENTIFIER, Value: ident}
			}
		}

		// Unknown character - skip it
		l.readChar()
		return l.NextToken()
	}
}

// ExpressionParser parses BASIC expressions and generates bytecode
type ExpressionParser struct {
	lexer   *ExpressionLexer
	current ExprToken
	peek    ExprToken

	// Compiler reference for emitting instructions
	compiler *BytecodeCompiler
}

// NewExpressionParser creates a new expression parser
func NewExpressionParser(input string, compiler *BytecodeCompiler) *ExpressionParser {
	p := &ExpressionParser{
		lexer:    NewExpressionLexer(input),
		compiler: compiler,
	}

	// Read two tokens to initialize current and peek
	p.nextToken()
	p.nextToken()

	return p
}

// nextToken advances to the next token
func (p *ExpressionParser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
}

// currentTokenIs checks if current token is of given type
func (p *ExpressionParser) currentTokenIs(t TokenType) bool {
	return p.current.Type == t
}

// peekTokenIs checks if peek token is of given type
func (p *ExpressionParser) peekTokenIs(t TokenType) bool {
	return p.peek.Type == t
}

// expectPeek checks if peek token is of given type and advances
func (p *ExpressionParser) expectPeek(t TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	return false
}

// ParseExpression parses a complete expression and emits bytecode
func (p *ExpressionParser) ParseExpression() error {
	return p.parseOrExpression()
}

// parseOrExpression handles OR operations (lowest precedence)
func (p *ExpressionParser) parseOrExpression() error {
	err := p.parseAndExpression()
	if err != nil {
		return err
	}

	for p.currentTokenIs(TOKEN_OR) {
		p.nextToken()
		err := p.parseAndExpression()
		if err != nil {
			return err
		}
		p.compiler.Emit(OP_OR)
	}

	return nil
}

// parseAndExpression handles AND operations
func (p *ExpressionParser) parseAndExpression() error {
	err := p.parseEqualityExpression()
	if err != nil {
		return err
	}

	for p.currentTokenIs(TOKEN_AND) {
		p.nextToken()
		err := p.parseEqualityExpression()
		if err != nil {
			return err
		}
		p.compiler.Emit(OP_AND)
	}

	return nil
}

// parseEqualityExpression handles =, <>, etc.
func (p *ExpressionParser) parseEqualityExpression() error {
	err := p.parseRelationalExpression()
	if err != nil {
		return err
	}

	for {
		switch p.current.Type {
		case TOKEN_EQ:
			p.nextToken()
			err := p.parseRelationalExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_EQ)
		case TOKEN_NE:
			p.nextToken()
			err := p.parseRelationalExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_NE)
		default:
			return nil
		}
	}
}

// parseRelationalExpression handles <, >, <=, >=
func (p *ExpressionParser) parseRelationalExpression() error {
	err := p.parseAdditiveExpression()
	if err != nil {
		return err
	}

	for {
		switch p.current.Type {
		case TOKEN_LT:
			p.nextToken()
			err := p.parseAdditiveExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_LT)
		case TOKEN_LE:
			p.nextToken()
			err := p.parseAdditiveExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_LE)
		case TOKEN_GT:
			p.nextToken()
			err := p.parseAdditiveExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_GT)
		case TOKEN_GE:
			p.nextToken()
			err := p.parseAdditiveExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_GE)
		default:
			return nil
		}
	}
}

// parseAdditiveExpression handles + and -
func (p *ExpressionParser) parseAdditiveExpression() error {
	err := p.parseMultiplicativeExpression()
	if err != nil {
		return err
	}

	for {
		switch p.current.Type {
		case TOKEN_PLUS:
			p.nextToken()
			err := p.parseMultiplicativeExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_ADD)
		case TOKEN_MINUS:
			p.nextToken()
			err := p.parseMultiplicativeExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_SUB)
		default:
			return nil
		}
	}
}

// parseMultiplicativeExpression handles *, /, MOD
func (p *ExpressionParser) parseMultiplicativeExpression() error {
	err := p.parsePowerExpression()
	if err != nil {
		return err
	}

	for {
		switch p.current.Type {
		case TOKEN_MULTIPLY:
			p.nextToken()
			err := p.parsePowerExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_MUL)
		case TOKEN_DIVIDE:
			p.nextToken()
			err := p.parsePowerExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_DIV)
		case TOKEN_MOD:
			p.nextToken()
			err := p.parsePowerExpression()
			if err != nil {
				return err
			}
			p.compiler.Emit(OP_MOD)
		default:
			return nil
		}
	}
}

// parsePowerExpression handles ^ (exponentiation)
func (p *ExpressionParser) parsePowerExpression() error {
	err := p.parseUnaryExpression()
	if err != nil {
		return err
	}

	// Right-associative
	if p.currentTokenIs(TOKEN_POWER) {
		p.nextToken()
		err := p.parsePowerExpression()
		if err != nil {
			return err
		}
		p.compiler.Emit(OP_POW)
	}

	return nil
}

// parseUnaryExpression handles unary -, +, NOT
func (p *ExpressionParser) parseUnaryExpression() error {
	switch p.current.Type {
	case TOKEN_MINUS:
		p.nextToken()
		err := p.parseUnaryExpression()
		if err != nil {
			return err
		}
		p.compiler.Emit(OP_NEG)
		return nil
	case TOKEN_PLUS:
		p.nextToken()
		return p.parseUnaryExpression()
	case TOKEN_NOT:
		p.nextToken()
		err := p.parseUnaryExpression()
		if err != nil {
			return err
		}
		p.compiler.Emit(OP_NOT)
		return nil
	default:
		return p.parsePrimaryExpression()
	}
}

// parsePrimaryExpression handles literals, variables, parentheses, function calls
func (p *ExpressionParser) parsePrimaryExpression() error {
	switch p.current.Type {
	case TOKEN_NUMBER:
		p.compiler.EmitConstant(p.current.NumVal)
		p.nextToken()
		return nil

	case TOKEN_STRING:
		p.compiler.EmitConstant(p.current.StrVal)
		p.nextToken()
		return nil

	case TOKEN_IDENTIFIER:
		varName := strings.ToUpper(p.current.Value)

		// Check if it's a function call
		if p.peekTokenIs(TOKEN_LPAREN) {
			return p.parseFunctionCall(varName)
		}

		// Simple variable
		p.compiler.Emit(OP_LOAD_VAR, varName)
		p.nextToken()
		return nil

	case TOKEN_LPAREN:
		p.nextToken() // Skip (
		err := p.ParseExpression()
		if err != nil {
			return err
		}
		if !p.expectPeek(TOKEN_RPAREN) {
			return fmt.Errorf("expected ')'")
		}
		return nil

	default:
		return fmt.Errorf("unexpected token: %v", p.current.Value)
	}
}

// parseFunctionCall handles function calls like SIN(X), MID$(S$,1,3)
func (p *ExpressionParser) parseFunctionCall(funcName string) error {
	p.nextToken() // Skip function name
	p.nextToken() // Skip (

	argCount := 0

	// Parse arguments
	if !p.currentTokenIs(TOKEN_RPAREN) {
		err := p.ParseExpression()
		if err != nil {
			return err
		}
		argCount++

		for p.currentTokenIs(TOKEN_COMMA) {
			p.nextToken() // Skip comma
			err := p.ParseExpression()
			if err != nil {
				return err
			}
			argCount++
		}
	}

	if !p.expectPeek(TOKEN_RPAREN) {
		return fmt.Errorf("expected ')' after function arguments")
	}

	// Emit function call instruction
	p.compiler.Emit(OP_CALL_FUNC, funcName, argCount)

	return nil
}

// CompileExpression is the main entry point for expression compilation
func (c *BytecodeCompiler) CompileExpression(expr string) error {
	if strings.TrimSpace(expr) == "" {
		return fmt.Errorf("empty expression")
	}

	// Try constant folding optimization first
	if folded, ok := c.tryConstantFold(expr); ok {
		if folded.IsNumeric {
			c.EmitConstant(folded.NumValue)
		} else {
			c.EmitConstant(folded.StrValue)
		}
		return nil
	}

	parser := NewExpressionParser(expr, c)
	return parser.ParseExpression()
}

// tryConstantFold attempts to evaluate constant expressions at compile time
func (c *BytecodeCompiler) tryConstantFold(expr string) (BASICValue, bool) {
	expr = strings.TrimSpace(expr)

	// Only attempt folding for simple expressions without variables or function calls
	if strings.ContainsAny(expr, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz$") {
		// Contains letters - likely variables or functions
		return BASICValue{}, false
	}

	// Try to evaluate as numeric expression
	if result, err := c.evaluateConstantExpression(expr); err == nil {
		return BASICValue{NumValue: result, IsNumeric: true}, true
	}

	// Try to evaluate as string literal
	if strings.HasPrefix(expr, "\"") && strings.HasSuffix(expr, "\"") {
		content := expr[1 : len(expr)-1]
		return BASICValue{StrValue: content, IsNumeric: false}, true
	}

	return BASICValue{}, false
}

// ConstantEvaluator provides enhanced constant folding capabilities
type ConstantEvaluator struct {
	compiler *BytecodeCompiler
}

// NewConstantEvaluator creates a new constant evaluator
func NewConstantEvaluator(compiler *BytecodeCompiler) *ConstantEvaluator {
	return &ConstantEvaluator{compiler: compiler}
}

// evaluateConstantExpression evaluates a constant numeric expression with enhanced folding
func (c *BytecodeCompiler) evaluateConstantExpression(expr string) (float64, error) {
	evaluator := NewConstantEvaluator(c)
	return evaluator.EvaluateExpression(expr)
}

// EvaluateExpression evaluates a constant expression using recursive descent parsing
func (ce *ConstantEvaluator) EvaluateExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return 0, fmt.Errorf("empty expression")
	}

	lexer := NewExpressionLexer(expr)
	parser := &ConstantExpressionParser{
		lexer: lexer,
		current: lexer.NextToken(),
	}
	parser.peek = lexer.NextToken()
	
	return parser.parseExpression()
}

// ConstantExpressionParser handles constant expression parsing for folding
type ConstantExpressionParser struct {
	lexer   *ExpressionLexer
	current ExprToken
	peek    ExprToken
}

// nextToken advances to the next token
func (p *ConstantExpressionParser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
}

// parseExpression parses OR expressions (lowest precedence)
func (p *ConstantExpressionParser) parseExpression() (float64, error) {
	left, err := p.parseAndExpression()
	if err != nil {
		return 0, err
	}

	for p.current.Type == TOKEN_OR {
		p.nextToken()
		right, err := p.parseAndExpression()
		if err != nil {
			return 0, err
		}
		// Logical OR: return -1 if either is true, 0 if both false
		if left != 0 || right != 0 {
			left = -1
		} else {
			left = 0
		}
	}

	return left, nil
}

// parseAndExpression parses AND expressions
func (p *ConstantExpressionParser) parseAndExpression() (float64, error) {
	left, err := p.parseEqualityExpression()
	if err != nil {
		return 0, err
	}

	for p.current.Type == TOKEN_AND {
		p.nextToken()
		right, err := p.parseEqualityExpression()
		if err != nil {
			return 0, err
		}
		// Logical AND: return -1 if both are true, 0 otherwise
		if left != 0 && right != 0 {
			left = -1
		} else {
			left = 0
		}
	}

	return left, nil
}

// parseEqualityExpression parses equality expressions
func (p *ConstantExpressionParser) parseEqualityExpression() (float64, error) {
	left, err := p.parseRelationalExpression()
	if err != nil {
		return 0, err
	}

	for {
		switch p.current.Type {
		case TOKEN_EQ:
			p.nextToken()
			right, err := p.parseRelationalExpression()
			if err != nil {
				return 0, err
			}
			if left == right {
				left = -1
			} else {
				left = 0
			}
		case TOKEN_NE:
			p.nextToken()
			right, err := p.parseRelationalExpression()
			if err != nil {
				return 0, err
			}
			if left != right {
				left = -1
			} else {
				left = 0
			}
		default:
			return left, nil
		}
	}
}

// parseRelationalExpression parses relational expressions
func (p *ConstantExpressionParser) parseRelationalExpression() (float64, error) {
	left, err := p.parseAdditiveExpression()
	if err != nil {
		return 0, err
	}

	for {
		switch p.current.Type {
		case TOKEN_LT:
			p.nextToken()
			right, err := p.parseAdditiveExpression()
			if err != nil {
				return 0, err
			}
			if left < right {
				left = -1
			} else {
				left = 0
			}
		case TOKEN_LE:
			p.nextToken()
			right, err := p.parseAdditiveExpression()
			if err != nil {
				return 0, err
			}
			if left <= right {
				left = -1
			} else {
				left = 0
			}
		case TOKEN_GT:
			p.nextToken()
			right, err := p.parseAdditiveExpression()
			if err != nil {
				return 0, err
			}
			if left > right {
				left = -1
			} else {
				left = 0
			}
		case TOKEN_GE:
			p.nextToken()
			right, err := p.parseAdditiveExpression()
			if err != nil {
				return 0, err
			}
			if left >= right {
				left = -1
			} else {
				left = 0
			}
		default:
			return left, nil
		}
	}
}

// parseAdditiveExpression parses additive expressions
func (p *ConstantExpressionParser) parseAdditiveExpression() (float64, error) {
	left, err := p.parseMultiplicativeExpression()
	if err != nil {
		return 0, err
	}

	for {
		switch p.current.Type {
		case TOKEN_PLUS:
			p.nextToken()
			right, err := p.parseMultiplicativeExpression()
			if err != nil {
				return 0, err
			}
			left = left + right
		case TOKEN_MINUS:
			p.nextToken()
			right, err := p.parseMultiplicativeExpression()
			if err != nil {
				return 0, err
			}
			left = left - right
		default:
			return left, nil
		}
	}
}

// parseMultiplicativeExpression parses multiplicative expressions
func (p *ConstantExpressionParser) parseMultiplicativeExpression() (float64, error) {
	left, err := p.parsePowerExpression()
	if err != nil {
		return 0, err
	}

	for {
		switch p.current.Type {
		case TOKEN_MULTIPLY:
			p.nextToken()
			right, err := p.parsePowerExpression()
			if err != nil {
				return 0, err
			}
			left = left * right
		case TOKEN_DIVIDE:
			p.nextToken()
			right, err := p.parsePowerExpression()
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, fmt.Errorf("division by zero in constant expression")
			}
			left = left / right
		case TOKEN_MOD:
			p.nextToken()
			right, err := p.parsePowerExpression()
			if err != nil {
				return 0, err
			}
			if right == 0 {
				return 0, fmt.Errorf("division by zero in modulo operation")
			}
			left = math.Mod(left, right)
		default:
			return left, nil
		}
	}
}

// parsePowerExpression parses power expressions (right-associative)
func (p *ConstantExpressionParser) parsePowerExpression() (float64, error) {
	left, err := p.parseUnaryExpression()
	if err != nil {
		return 0, err
	}

	if p.current.Type == TOKEN_POWER {
		p.nextToken()
		right, err := p.parsePowerExpression() // Right-associative
		if err != nil {
			return 0, err
		}
		return math.Pow(left, right), nil
	}

	return left, nil
}

// parseUnaryExpression parses unary expressions
func (p *ConstantExpressionParser) parseUnaryExpression() (float64, error) {
	switch p.current.Type {
	case TOKEN_MINUS:
		p.nextToken()
		val, err := p.parseUnaryExpression()
		if err != nil {
			return 0, err
		}
		return -val, nil
	case TOKEN_PLUS:
		p.nextToken()
		return p.parseUnaryExpression()
	case TOKEN_NOT:
		p.nextToken()
		val, err := p.parseUnaryExpression()
		if err != nil {
			return 0, err
		}
		if val == 0 {
			return -1, nil
		}
		return 0, nil
	default:
		return p.parsePrimaryExpression()
	}
}

// parsePrimaryExpression parses primary expressions
func (p *ConstantExpressionParser) parsePrimaryExpression() (float64, error) {
	switch p.current.Type {
	case TOKEN_NUMBER:
		val := p.current.NumVal
		p.nextToken()
		return val, nil
	case TOKEN_LPAREN:
		p.nextToken() // Skip (
		val, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		if p.current.Type != TOKEN_RPAREN {
			return 0, fmt.Errorf("expected ')' in constant expression")
		}
		p.nextToken() // Skip )
		return val, nil
	case TOKEN_IDENTIFIER:
		// Check for mathematical constants
		switch strings.ToUpper(p.current.Value) {
		case "PI":
			p.nextToken()
			return math.Pi, nil
		case "E":
			p.nextToken()
			return math.E, nil
		default:
			// Check for simple mathematical functions
			if p.peek.Type == TOKEN_LPAREN {
				return p.parseConstantFunction()
			}
			return 0, fmt.Errorf("variable '%s' not allowed in constant expression", p.current.Value)
		}
	default:
		return 0, fmt.Errorf("unexpected token in constant expression: %s", p.current.Value)
	}
}

// parseConstantFunction parses mathematical functions in constant expressions
func (p *ConstantExpressionParser) parseConstantFunction() (float64, error) {
	funcName := strings.ToUpper(p.current.Value)
	p.nextToken() // Skip function name
	p.nextToken() // Skip (

	arg, err := p.parseExpression()
	if err != nil {
		return 0, err
	}

	if p.current.Type != TOKEN_RPAREN {
		return 0, fmt.Errorf("expected ')' after function argument")
	}
	p.nextToken() // Skip )

	switch funcName {
	case "ABS":
		if arg < 0 {
			return -arg, nil
		}
		return arg, nil
	case "INT":
		return float64(int64(arg)), nil
	case "SIN":
		return math.Sin(arg), nil
	case "COS":
		return math.Cos(arg), nil
	case "TAN":
		return math.Tan(arg), nil
	case "ASIN":
		return math.Asin(arg), nil
	case "ACOS":
		return math.Acos(arg), nil
	case "ATAN":
		return math.Atan(arg), nil
	case "LOG":
		if arg <= 0 {
			return 0, fmt.Errorf("logarithm of non-positive number")
		}
		return math.Log(arg), nil
	case "LOG10":
		if arg <= 0 {
			return 0, fmt.Errorf("logarithm of non-positive number")
		}
		return math.Log10(arg), nil
	case "EXP":
		return math.Exp(arg), nil
	case "SQR", "SQRT":
		if arg < 0 {
			return 0, fmt.Errorf("square root of negative number")
		}
		return math.Sqrt(arg), nil
	default:
		return 0, fmt.Errorf("function '%s' not supported in constant expressions", funcName)
	}
}
