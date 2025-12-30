package tinybasic

import (
	"strings"
)

// SplitInputList splits a comma-separated string into a list of strings,
// respecting quoted strings (double quotes).
func SplitInputList(input string) []string {
	var result []string
	var current strings.Builder
	inQuote := false

	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	for i := 0; i < len(input); i++ {
		char := input[i]

		if char == '"' {
			inQuote = !inQuote
			current.WriteByte(char)
		} else if char == ',' && !inQuote {
			// Found a separator
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset()
		} else {
			current.WriteByte(char)
		}
	}

	// Add the last item
	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	} else if len(input) > 0 && input[len(input)-1] == ',' {
        // If the string ends with a comma, we might want to append an empty string or ignore it.
        // Standard BASIC usually expects a value.
        // For "A," -> ["A", ""]
        result = append(result, "")
    }

	return result
}
