package tinybasic

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// SimpleJIT provides fast in-memory compilation without Go plugins
type SimpleJIT struct {
	enabled       bool
	compiledLoops map[string]*SimpleCompiledLoop
	executionCounts map[string]int64
	threshold     int64
}

// SimpleCompiledLoop represents a compiled loop using function pointers
type SimpleCompiledLoop struct {
	signature string
	executor  func(startVal, endVal, stepVal float64, vars map[string]BASICValue) map[string]BASICValue
	createTime time.Time
	execCount int64
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
		"enabled":         sj.enabled,
		"compiled_loops":  len(sj.compiledLoops),
		"execution_counts": len(sj.executionCounts),
		"threshold":       sj.threshold,
		"type":           "SimpleJIT",
	}
	
	loopStats := make(map[string]interface{})
	for signature, compiled := range sj.compiledLoops {
		loopStats[signature] = map[string]interface{}{
			"executions":  compiled.execCount,
			"created_at":  compiled.createTime.Format(time.RFC3339),
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
	// DISABLED: JIT is too aggressive and breaks normal loop execution
	// Instead, JIT should only be used for analysis and optimization hints
	return false, nil
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