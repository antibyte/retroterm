package tinybasic

type Lexer struct {
	input string
	pos   int
}

// NewLexer erstellt einen neuen Lexer
func NewLexer(input string) *Lexer {
	return &Lexer{
		input: input,
		pos:   0,
	}
}

// isSpace überprüft, ob ein Zeichen ein Leerzeichen ist
func isSpace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

// isDigit überprüft, ob ein Zeichen eine Ziffer ist
func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
