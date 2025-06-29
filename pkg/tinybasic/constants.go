// Package tinybasic implements a simple BASIC interpreter.
package tinybasic

// Constants for default values and configuration.
const (
	// DefaultTermCols defines the default terminal width.
	DefaultTermCols = 80
	// DefaultTermRows defines the default terminal height.
	DefaultTermRows = 24 // OutputChannelBufferSize defines the buffer size for the output channel.
	// A larger buffer reduces the chance of dropping messages under load.
	OutputChannelBufferSize = 10000 // Erhöht von 5000 auf 10000 für bessere Endlosschleifen-Performance
	// MaxGosubDepth defines the maximum nesting level for GOSUB calls.
	MaxGosubDepth   = 100 // MaxForLoopDepth defines the maximum nesting level for FOR loops.
	MaxForLoopDepth = 200
)

type token struct {
	typ int
	val string
	pos int
}

// Token-Typen für den Ausdrucksparser
const (
	tokEOF = iota
	tokNumber
	tokString
	tokIdent
	tokOp
	tokLParen
	tokRParen
	tokComma
	tokHash
)

// Tastenkonstanten für INKEY$ Vergleiche
const (
	KeyEsc      = "\x1B"   // ESC-Taste
	KeyCurUp    = "\x1B[A" // Pfeil nach oben
	KeyCurDown  = "\x1B[B" // Pfeil nach unten
	KeyCurRight = "\x1B[C" // Pfeil nach rechts
	KeyCurLeft  = "\x1B[D" // Pfeil nach links
)
