// Package tinybasic implements a simple BASIC interpreter.
package tinybasic

import (
	"log"
	"math/rand"
	"strings"
	"time"
)

// cmdRandomize initializes the random number generator with a new seed.
// With no argument, uses the current time as seed.
// With an argument, uses the provided value as seed.
// Assumes lock is held.
func (b *TinyBASIC) cmdRandomize(args string) error {
	args = strings.TrimSpace(args)

	var seed int64

	if args == "" {
		// Wenn kein Argument angegeben ist, verwende aktuelle Zeit als Seed
		seed = time.Now().UnixNano()
		log.Printf("[INFO] RANDOMIZE using current time as seed: %d", seed)
	} else {
		// Wenn ein Argument angegeben ist, versuche es als Zahl zu interpretieren
		val, err := b.evalExpression(args)
		if err != nil {
			return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("RANDOMIZE")
		}

		if !val.IsNumeric {
			return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("RANDOMIZE")
		}

		seed = int64(val.NumValue)
		log.Printf("[INFO] RANDOMIZE using provided seed: %d", seed)
	}
	// Setze den neuen Seed für den Zufallszahlengenerator
	rand.Seed(seed)

	return nil
}
