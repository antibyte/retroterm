package tinybasic

import (
	"strings"

	"github.com/antibyte/retroterm/pkg/logger"
)

// Parser repräsentiert einen Parser für TinyBASIC
type Parser struct {
	text     string // Der zu parsende Text
	position int    // Aktuelle Position im Text
	current  byte   // Aktuelles Zeichen
}

// NewParser erstellt einen neuen Parser für einen Text
func NewParser(text string) *Parser {
	p := &Parser{
		text:     text,
		position: 0,
	}
	if len(text) > 0 {
		p.current = text[0]
	}
	return p
}

// advance bewegt den Parser um ein Zeichen nach vorne
func (p *Parser) advance() {
	p.position++
	if p.position < len(p.text) {
		p.current = p.text[p.position]
	}
}

// nextNonSpaceIdx returns the index of the first non-space character
// in s at or after start. If no such character is found, it returns len(s).
func nextNonSpaceIdx(s string, start int) int {
	for i := start; i < len(s); i++ {
		// Consider other whitespace? For now, space and tab.
		if s[i] != ' ' && s[i] != '\t' {
			return i
		}
	}
	return len(s)
}

// parseString parses a double-quoted string from the beginning of s.
// It returns the unquoted string content, the number of characters consumed (including quotes),
// and an error if parsing fails (e.g., unterminated string).
// Handles escaped double quotes ("") inside the string.
func parseString(s string) (string, int, error) {
	logger.Debug(logger.AreaTinyBasic, "[PARSESTRING] Input: '%s'", s)

	if len(s) == 0 || s[0] != '"' {
		return "", 0, NewBASICError(ErrCategorySyntax, "MISSING_QUOTES", true, 0)
	}

	var content strings.Builder
	i := 1 // Start after the opening quote
	for i < len(s) {
		char := s[i]
		if char == '"' {
			// Check for escaped quote ""
			if i+1 < len(s) && s[i+1] == '"' {
				content.WriteByte('"')
				i += 2 // Consume both quotes
			} else {
				i++ // Consume closing quote
				result := content.String()
				logger.Debug(logger.AreaTinyBasic, "[PARSESTRING] Output: '%s' (length: %d)", result, len(result))
				return result, i, nil
			}
		} else {
			content.WriteByte(char)
			i++
		}
	}
	return "", 0, NewBASICError(ErrCategorySyntax, "MISSING_QUOTES", true, 0)
}
