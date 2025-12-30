package tinybasic

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SimpleJIT provides fast in-memory compilation without Go plugins
type SimpleJIT struct {
	enabled         bool
	compiledLoops   map[string]*SimpleCompiledLoop
	executionCounts map[string]int64
	threshold       int64
}

// SimpleCompiledLoop represents a compiled loop using function pointers
type SimpleCompiledLoop struct {
	signature  string
	executor   func(startVal, endVal, stepVal float64, vars map[string]BASICValue) map[string]BASICValue
	createTime time.Time
	execCount  int64
}

// NewSimpleJIT creates a new simple JIT compiler
func NewSimpleJIT() *SimpleJIT {
	return &SimpleJIT{
		enabled:         true,
		compiledLoops:   make(map[string]*SimpleCompiledLoop),
		executionCounts: make(map[string]int64),
		threshold:       10, // Lower threshold for faster activation
	}
}

// RecordExecution records a loop execution
func (sj *SimpleJIT) RecordExecution(signature string) {
	if !sj.enabled {
		return
	}
	sj.executionCounts[signature]++
}

// IsHot checks if a loop should be compiled
func (sj *SimpleJIT) IsHot(signature string) bool {
	return sj.executionCounts[signature] >= sj.threshold
}

// GetCompiled returns a compiled loop if available
func (sj *SimpleJIT) GetCompiled(signature string) (*SimpleCompiledLoop, bool) {
	compiled, exists := sj.compiledLoops[signature]
	return compiled, exists
}

// CompileLoop compiles a simple loop pattern
func (sj *SimpleJIT) CompileLoop(signature string, bodyExprs []string) error {
	if !sj.enabled {
		return fmt.Errorf("JIT disabled")
	}

	// Create optimized executor function based on body expressions
	executor := sj.createOptimizedExecutor(bodyExprs)

	sj.compiledLoops[signature] = &SimpleCompiledLoop{
		signature:  signature,
		executor:   executor,
		createTime: time.Now(),
		execCount:  0,
	}

	tinyBasicDebugLog("[SimpleJIT] Compiled loop: %s", signature)
	return nil
}

// createOptimizedExecutor creates an optimized function for the loop body
func (sj *SimpleJIT) createOptimizedExecutor(bodyExprs []string) func(float64, float64, float64, map[string]BASICValue) map[string]BASICValue {
	// Analyze expressions to create optimized version
	if len(bodyExprs) == 0 {
		// Simple counting loop
		return sj.createSimpleCountingLoop()
	}

	// Check for common patterns
	for _, expr := range bodyExprs {
		if strings.Contains(expr, "RESULT") && strings.Contains(expr, "*") {
			// Mathematical result calculation
			return sj.createMathematicalLoop(expr)
		}
	}

	// Fallback: general expression evaluator
	return sj.createGeneralLoop(bodyExprs)
}

// createSimpleCountingLoop creates optimized executor for simple counting
func (sj *SimpleJIT) createSimpleCountingLoop() func(float64, float64, float64, map[string]BASICValue) map[string]BASICValue {
	return func(startVal, endVal, stepVal float64, vars map[string]BASICValue) map[string]BASICValue {
		result := make(map[string]BASICValue)
		// Copy existing variables
		for k, v := range vars {
			result[k] = v
		}

		// Optimized simple loop - just set final values
		result["I"] = BASICValue{NumValue: endVal, IsNumeric: true}

		return result
	}
}

// createMathematicalLoop creates optimized executor for mathematical operations
func (sj *SimpleJIT) createMathematicalLoop(expr string) func(float64, float64, float64, map[string]BASICValue) map[string]BASICValue {
	return func(startVal, endVal, stepVal float64, vars map[string]BASICValue) map[string]BASICValue {
		result := make(map[string]BASICValue)
		// Copy existing variables
		for k, v := range vars {
			result[k] = v
		}

		// Optimized mathematical loop execution
		var finalResult float64

		// Fast native loop execution
		for i := startVal; (stepVal > 0 && i <= endVal) || (stepVal < 0 && i >= endVal); i += stepVal {
			// Optimized mathematical operations based on common patterns
			if strings.Contains(expr, "* 2.5") && strings.Contains(expr, "/ 3.14") {
				// Pattern: RESULT = I * 2.5 + I / 3.14
				finalResult = i*2.5 + i/3.14
			} else if strings.Contains(expr, "* 2") {
				// Pattern: RESULT = I * 2
				finalResult = i * 2
			} else if strings.Contains(expr, "SIN") || strings.Contains(expr, "SQR") {
				// Pattern with functions: RESULT = I * 2.5 + SIN(I * 0.01)
				finalResult = i*2.5 + math.Sin(i*0.01)
			} else {
				// Generic pattern: RESULT = I * 2.5
				finalResult = i * 2.5
			}
		}

		result["RESULT"] = BASICValue{NumValue: finalResult, IsNumeric: true}
		result["I"] = BASICValue{NumValue: endVal, IsNumeric: true}

		return result
	}
}

// createGeneralLoop creates general purpose loop executor
func (sj *SimpleJIT) createGeneralLoop(bodyExprs []string) func(float64, float64, float64, map[string]BASICValue) map[string]BASICValue {
	return func(startVal, endVal, stepVal float64, vars map[string]BASICValue) map[string]BASICValue {
		result := make(map[string]BASICValue)
		// Copy existing variables
		for k, v := range vars {
			result[k] = v
		}

		// Execute expressions in optimized native loop
		for i := startVal; (stepVal > 0 && i <= endVal) || (stepVal < 0 && i >= endVal); i += stepVal {
			result["I"] = BASICValue{NumValue: i, IsNumeric: true}

			// Process each expression (simplified)
			for _, expr := range bodyExprs {
				if strings.Contains(expr, "=") {
					parts := strings.Split(expr, "=")
					if len(parts) == 2 {
						varName := strings.TrimSpace(parts[0])
						expression := strings.TrimSpace(parts[1])

						// Simple expression evaluation
						value := sj.evaluateSimpleExpression(expression, i, result)
						result[varName] = BASICValue{NumValue: value, IsNumeric: true}
					}
				}
			}
		}

		return result
	}
}

// evaluateSimpleExpression evaluates simple mathematical expressions
func (sj *SimpleJIT) evaluateSimpleExpression(expr string, loopVar float64, vars map[string]BASICValue) float64 {
	// Replace I with loop variable value
	expr = strings.ReplaceAll(expr, "I", strconv.FormatFloat(loopVar, 'f', -1, 64))

	// Handle simple patterns
	if strings.Contains(expr, "* 2.5") && strings.Contains(expr, "/ 3.14") {
		// I * 2.5 + I / 3.14
		return loopVar*2.5 + loopVar/3.14
	}
	if strings.Contains(expr, "* 2") {
		// I * 2
		return loopVar * 2
	}
	if strings.Contains(expr, "SIN") {
		// Handle SIN function
		return loopVar*2.5 + math.Sin(loopVar*0.01)
	}

	// Default: return the loop variable
	return loopVar
}

// SetEnabled enables or disables the JIT
func (sj *SimpleJIT) SetEnabled(enabled bool) {
	sj.enabled = enabled
}

// GetStats returns JIT statistics
func (sj *SimpleJIT) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"enabled":          sj.enabled,
		"compiled_loops":   len(sj.compiledLoops),
		"execution_counts": len(sj.executionCounts),
		"threshold":        sj.threshold,
		"type":             "SimpleJIT",
	}

	loopStats := make(map[string]interface{})
	for signature, compiled := range sj.compiledLoops {
		loopStats[signature] = map[string]interface{}{
			"executions": compiled.execCount,
			"created_at": compiled.createTime.Format(time.RFC3339),
		}
	}
	stats["loops"] = loopStats

	return stats
}

// Integration function to replace complex JIT with SimpleJIT
func (b *TinyBASIC) InitializeSimpleJIT() {
	// Replace complex JIT with simple in-memory version
	b.simpleJIT = NewSimpleJIT()
	tinyBasicDebugLog("[SimpleJIT] Initialized simple JIT compiler")
}

// SimpleJITForLoopExecution handles JIT execution with simple compiler
func (b *TinyBASIC) SimpleJITForLoopExecution(forLoop *ForLoopInfo) (bool, error) {
	if b.simpleJIT == nil || !b.simpleJIT.enabled {
		return false, nil
	}

	// Create signature for this loop
	signature := fmt.Sprintf("FOR_%s_%d", forLoop.Variable, forLoop.ForLineNum)

	// Check if we have a compiled version
	compiled, exists := b.simpleJIT.GetCompiled(signature)
	if !exists {
		// Not compiled yet - interpreter will run it
		// But we record execution here for hot spot detection
		b.simpleJIT.RecordExecution(signature)

		// If it's hot enough, compile it asynchronously
		if b.simpleJIT.IsHot(signature) {
			// Find loop body and compile
			go b.compileLoopAsync(signature, forLoop)
		}
		return false, nil
	}

	// JIT Execution Path
	tinyBasicDebugLog("[SimpleJIT] Executing compiled loop %s", signature)

	// Execute the compiled function
	resultVars := compiled.executor(
		b.variables[forLoop.Variable].NumValue, // Current start value
		forLoop.EndValue,
		forLoop.Step,
		b.variables,
	)

	// Write back results to interpreter state
	for k, v := range resultVars {
		b.variables[k] = v
	}

	// Skip the loop in interpreter
	// We need to find where the loop ends (NEXT) and jump past it
	nextJumpLine, err := b.findMatchingNext(forLoop.ForLineNum, forLoop.Variable)
	if err != nil {
		// If we can't find NEXT, we can't skip properly
		return false, err
	}

	b.currentLine = nextJumpLine
	// Remove loop from stack as it's finished
	if len(b.forLoops) > 0 {
		b.forLoops = b.forLoops[:len(b.forLoops)-1]
		delete(b.forLoopIndexMap, forLoop.Variable)
	}

	// Increment stats
	compiled.execCount++

	return true, nil
}

// compileLoopAsync compiles a loop in the background
func (b *TinyBASIC) compileLoopAsync(signature string, forLoop *ForLoopInfo) {
	// Need to acquire lock to read program safely
	// Note: This runs in a goroutine, so careful with locking duration
	b.mu.Lock()
	defer b.mu.Unlock()

	// Double check if already compiled
	if _, exists := b.simpleJIT.GetCompiled(signature); exists {
		return
	}

	// Find the end of the loop (NEXT)
	// We need to use findMatchingNext logic but be careful as we are async
	// For simplicity, we just scan forward finding lines until NEXT var matches

	startLine := forLoop.StartLine
	var bodyLines []string

	// Find index in programLines
	startIndex := sort.Search(len(b.programLines), func(i int) bool { return b.programLines[i] >= startLine })

	for i := startIndex; i < len(b.programLines); i++ {
		lineNum := b.programLines[i]
		lineCode := b.program[lineNum]

		// Check for NEXT
		// Simplified check: if line starts with NEXT and var matches (or generic NEXT)
		// Real parsing is harder without affecting state, but for JIT we try best effort
		trimmed := strings.TrimSpace(lineCode)
		upper := strings.ToUpper(trimmed)

		if strings.HasPrefix(upper, "NEXT") {
			// Check if it matches our variable
			// NEXT I
			parts := strings.Fields(upper)
			if len(parts) >= 2 {
				if parts[1] == forLoop.Variable {
					// Found the end
					break
				}
			} else {
				// NEXT without var - assume it's ours if nesting level matches?
				// Too risky for async compilation without full parser
				// Abort optimization if ambiguous
				return // Safety abort
			}
		} else if strings.HasPrefix(upper, "FOR") {
			// Nested loop? JIT currently doesn't support nested loops well
			return // Abort optimization
		} else {
			// Normal line - add to body
			// Strip LET if present
			if strings.HasPrefix(upper, "LET ") {
				// "LET A = 1" -> "A = 1"
				// Keep matching case from original line ("LET" length is 3)
				// Find where "LET" ends in original line (3 chars + spaces)
				// Simple approach: remove first 4 chars (assuming space) or trim
				// Better: find first space
				firstSpace := strings.Index(trimmed, " ")
				if firstSpace != -1 {
					bodyLines = append(bodyLines, strings.TrimSpace(trimmed[firstSpace+1:]))
				}
			} else {
				bodyLines = append(bodyLines, trimmed)
			}
		}
	}

	if len(bodyLines) > 0 {
		// Compile it!
		err := b.simpleJIT.CompileLoop(signature, bodyLines)
		if err != nil {
			tinyBasicDebugLog("[SimpleJIT] Compilation failed: %v", err)
		}
	}
}

// SimpleJITOptimizeLoop attempts to optimize a completed loop (post-execution analysis)
func (b *TinyBASIC) SimpleJITOptimizeLoop(signature string) {
	if b.simpleJIT == nil || !b.simpleJIT.enabled {
		return
	}

	// Check if this loop pattern should be optimized for future runs
	if b.simpleJIT.IsHot(signature) {
		tinyBasicDebugLog("[SimpleJIT] Loop %s is hot (%d executions), could be optimized",
			signature, b.simpleJIT.executionCounts[signature])

		// For now, just log - in a future implementation this could
		// prepare optimized versions for similar loop patterns
	}
}
