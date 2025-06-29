package tinybasic

import (
	"fmt"
	"strings"
)

// Deadlock-sichere und vereinfachte FOR-Loop-Implementierung
// Diese Version ersetzt die komplexe Logik durch einfache, zuverlässige Implementierungen

// cmdForSafe - Vereinfachte FOR-Implementierung ohne Deadlock-Risiko
func (b *TinyBASIC) cmdForSafe(args string, nextSubStatementIndex int) error {
	// Einfaches Parsing: FOR var = start TO end [STEP step]
	parts := strings.Fields(args)
	if len(parts) < 4 {
		return fmt.Errorf("SYNTAX ERROR: FOR requires at least 4 parts")
	}

	// Variable name
	varName := strings.ToUpper(parts[0])

	// "=" check
	if parts[1] != "=" {
		return fmt.Errorf("SYNTAX ERROR: Expected = after variable name")
	}

	// Start value
	startVal, err := b.evalExpression(parts[2])
	if err != nil || !startVal.IsNumeric {
		return fmt.Errorf("SYNTAX ERROR: Invalid start value")
	}

	// "TO" check
	if len(parts) < 4 || strings.ToUpper(parts[3]) != "TO" {
		return fmt.Errorf("SYNTAX ERROR: Expected TO")
	}

	// End value
	if len(parts) < 5 {
		return fmt.Errorf("SYNTAX ERROR: Missing end value")
	}
	endVal, err := b.evalExpression(parts[4])
	if err != nil || !endVal.IsNumeric {
		return fmt.Errorf("SYNTAX ERROR: Invalid end value")
	}

	// Step value (optional)
	stepVal := 1.0
	if len(parts) >= 7 && strings.ToUpper(parts[5]) == "STEP" {
		step, err := b.evalExpression(parts[6])
		if err != nil || !step.IsNumeric {
			return fmt.Errorf("SYNTAX ERROR: Invalid step value")
		}
		stepVal = step.NumValue
	}

	// Prevent infinite loops
	if stepVal == 0 {
		return fmt.Errorf("SYNTAX ERROR: STEP cannot be zero")
	}

	// Check for immediate skip condition
	currentValue := startVal.NumValue
	endValue := endVal.NumValue

	skipLoop := false
	if stepVal > 0 && currentValue > endValue {
		skipLoop = true
	} else if stepVal < 0 && currentValue < endValue {
		skipLoop = true
	}

	if skipLoop {
		// Skip the entire loop - find matching NEXT
		return b.skipToMatchingNext(varName)
	}

	// Initialize the loop
	b.variables[varName] = startVal

	// Add to loop stack with simplified info
	if len(b.forLoops) >= 10 { // Prevent deep nesting
		return fmt.Errorf("ERROR: Too many nested FOR loops")
	}

	loopInfo := ForLoopInfo{
		Variable:   varName,
		EndValue:   endValue,
		Step:       stepVal,
		StartLine:  b.currentLine, // Current line for simple loops
		ForLineNum: b.currentLine,
	}

	b.forLoops = append(b.forLoops, loopInfo)
	return nil
}

// cmdNextSafe - Vereinfachte NEXT-Implementierung
func (b *TinyBASIC) cmdNextSafe(args string) error {
	varName := strings.TrimSpace(strings.ToUpper(args))

	// If no variable specified, use innermost loop
	if varName == "" {
		if len(b.forLoops) == 0 {
			return fmt.Errorf("NEXT without FOR")
		}
		varName = b.forLoops[len(b.forLoops)-1].Variable
	}

	// Find the matching loop
	loopIndex := -1
	for i := len(b.forLoops) - 1; i >= 0; i-- {
		if b.forLoops[i].Variable == varName {
			loopIndex = i
			break
		}
	}

	if loopIndex == -1 {
		return fmt.Errorf("NEXT without matching FOR")
	}

	// Get the loop info
	loop := b.forLoops[loopIndex]

	// Get current variable value and increment
	currentVal, exists := b.variables[varName]
	if !exists {
		return fmt.Errorf("Loop variable not found")
	}

	// Increment the variable
	newValue := currentVal.NumValue + loop.Step
	b.variables[varName] = BASICValue{NumValue: newValue, IsNumeric: true}

	// Check if loop should continue
	continueLoop := false
	if loop.Step > 0 && newValue <= loop.EndValue {
		continueLoop = true
	} else if loop.Step < 0 && newValue >= loop.EndValue {
		continueLoop = true
	}

	if continueLoop {
		// Continue loop - go back to the line after FOR
		nextLine, found := b.findNextLine(loop.ForLineNum)
		if found {
			b.currentLine = nextLine
		}
	} else {
		// Loop finished - remove from stack
		b.forLoops = b.forLoops[:loopIndex]
		// Continue with next statement (currentLine unchanged)
	}

	return nil
}

// skipToMatchingNext - Einfache Implementierung zum Überspringen einer Schleife
func (b *TinyBASIC) skipToMatchingNext(forVarName string) error {
	nestingLevel := 0
	searchLine := b.currentLine

	// Search through program lines
	for {
		nextLine, found := b.findNextLine(searchLine)
		if !found {
			return fmt.Errorf("Matching NEXT not found for variable %s", forVarName)
		}

		searchLine = nextLine
		lineCode, exists := b.program[searchLine]
		if !exists {
			continue
		}

		// Simple keyword detection
		upperCode := strings.ToUpper(strings.TrimSpace(lineCode))
		words := strings.Fields(upperCode)

		if len(words) == 0 {
			continue
		}

		// Check for FOR (increases nesting)
		if words[0] == "FOR" {
			nestingLevel++
			continue
		}

		// Check for NEXT (decreases nesting)
		if words[0] == "NEXT" {
			if nestingLevel == 0 {
				// This is our target NEXT
				nextAfterNext, found := b.findNextLine(searchLine)
				if found {
					b.currentLine = nextAfterNext
				} else {
					b.currentLine = 0 // End of program
				}
				return nil
			} else {
				nestingLevel--
			}
		}
	}
}
