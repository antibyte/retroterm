package tinybasic

// isIdChar returns true if the character can be part of an identifier (after the first character).
// This allows uppercase letters, numbers, and now also underscores.
func isIdChar(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// isIdStart returns true if the character can be the first character of an identifier.
// Only uppercase letters are allowed as first character.
func isIdStart(ch byte) bool {
	return ch >= 'A' && ch <= 'Z'
}
