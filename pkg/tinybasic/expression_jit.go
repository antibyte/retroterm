package tinybasic

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ExpressionJIT provides safe, non-invasive JIT compilation for mathematical expressions
type ExpressionJIT struct {
	// Core JIT state
	enabled         bool
	compiledExprs   map[string]*CompiledExpression
	executionCounts map[string]int64
	threshold       int64
	mutex           sync.RWMutex
	
	// Pattern recognition
	patterns []ExpressionPattern
	
	// Performance metrics
	hits            int64
	misses          int64
	compilations    int64
	totalSavedTime  time.Duration
}

// CompiledExpression represents a JIT-compiled expression
type CompiledExpression struct {
	signature   string
	originalExpr string
	evaluator   func(vars map[string]BASICValue) BASICValue
	pattern     string
	hitCount    int64
	createTime  time.Time
	totalTime   time.Duration
	avgTime     time.Duration
}

// ExpressionPattern defines a recognizable pattern for JIT compilation
type ExpressionPattern struct {
	name        string
	regex       *regexp.Regexp
	complexity  int
	compiler    func(expr string, matches []string) func(vars map[string]BASICValue) BASICValue
	description string
}

// NewExpressionJIT creates a new expression-level JIT compiler
func NewExpressionJIT() *ExpressionJIT {
	ejit := &ExpressionJIT{
		enabled:         true, // Enabled by default for performance
		compiledExprs:   make(map[string]*CompiledExpression),
		executionCounts: make(map[string]int64),
		threshold:       5, // Lower threshold for faster activation
	}
	
	ejit.initializePatterns()
	return ejit
}

// initializePatterns sets up the pattern recognition system
func (ejit *ExpressionJIT) initializePatterns() {
	ejit.patterns = []ExpressionPattern{
		{
			name:        "SimpleMultiplication",
			regex:       regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)\s*\*\s*([0-9]+\.?[0-9]*)$`),
			complexity:  1,
			compiler:    ejit.compileSimpleMultiplication,
			description: "Variable * Constant (e.g., I * 2.5)",
		},
		{
			name:        "SimpleAddition", 
			regex:       regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)\s*\+\s*([0-9]+\.?[0-9]*)$`),
			complexity:  1,
			compiler:    ejit.compileSimpleAddition,
			description: "Variable + Constant (e.g., I + 10)",
		},
		{
			name:        "MathExpression",
			regex:       regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)\s*\*\s*([0-9]+\.?[0-9]*)\s*\+\s*([A-Z_][A-Z0-9_]*)\s*/\s*([0-9]+\.?[0-9]*)$`),
			complexity:  3,
			compiler:    ejit.compileMathExpression,
			description: "Var * Const + Var / Const (e.g., I * 2.5 + I / 3.14)",
		},
		{
			name:        "SinFunction",
			regex:       regexp.MustCompile(`^SIN\s*\(\s*([A-Z_][A-Z0-9_]*)\s*\*\s*([0-9]+\.?[0-9]*)\s*\)$`),
			complexity:  4,
			compiler:    ejit.compileSinFunction,
			description: "SIN(Var * Const) (e.g., SIN(I * 0.01))",
		},
		{
			name:        "VariableAccess",
			regex:       regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)$`),
			complexity:  1,
			compiler:    ejit.compileVariableAccess,
			description: "Simple variable access (e.g., X)",
		},
		{
			name:        "LoopIncrement",
			regex:       regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)\\s*\\+\\s*1$`),
			complexity:  1,
			compiler:    ejit.compileLoopIncrement,
			description: "Variable increment (e.g., I + 1)",
		},
		{
			name:        "VariableSubtraction",
			regex:       regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)\\s*-\\s*([0-9]+\\.?[0-9]*)$`),
			complexity:  1,
			compiler:    ejit.compileVariableSubtraction,
			description: "Variable - Constant (e.g., X - 5)",
		},
		{
			name:        "ArrayIndex",
			regex:       regexp.MustCompile(`^([A-Z_][A-Z0-9_]*)\\s*\\*\\s*([0-9]+)\\s*\\+\\s*([A-Z_][A-Z0-9_]*)$`),
			complexity:  2,
			compiler:    ejit.compileArrayIndex,
			description: "Array indexing (e.g., I * 80 + J)",
		},
	}
}

// TryEvaluate attempts to evaluate an expression using JIT, returns (result, success)
func (ejit *ExpressionJIT) TryEvaluate(expr string, vars map[string]BASICValue) (BASICValue, bool) {
	if !ejit.enabled {
		return BASICValue{}, false
	}
	
	// Ultra-fast pre-filtering: Skip expressions that are guaranteed not to benefit
	if ejit.shouldSkipExpression(expr) {
		return BASICValue{}, false
	}
	
	// Fast normalization (avoid regex overhead for hot path)
	normalizedExpr := ejit.fastNormalizeExpression(expr)
	
	// Fast path: Check if compiled (using RLock for read-only access)
	ejit.mutex.RLock()
	compiled, exists := ejit.compiledExprs[normalizedExpr]
	ejit.mutex.RUnlock()
	
	if exists {
		// Execute compiled expression directly (minimal overhead)
		result := compiled.evaluator(vars)
		
		// Minimal statistics update (avoid atomic operations in hot path)
		compiled.hitCount++
		ejit.hits++
		
		return result, true
	}
	
	// Only increment misses for expressions we actually considered
	ejit.misses++
	return BASICValue{}, false
}

// RecordExecution records an expression execution for JIT compilation consideration
func (ejit *ExpressionJIT) RecordExecution(expr string) {
	if !ejit.enabled {
		return
	}
	
	// Ultra-fast pre-filtering: Skip expressions that won't benefit
	if ejit.shouldSkipExpression(expr) {
		return
	}
	
	normalizedExpr := ejit.fastNormalizeExpression(expr)
	
	// Optimized atomic increment with minimal mutex usage
	ejit.mutex.Lock()
	ejit.executionCounts[normalizedExpr]++
	count := ejit.executionCounts[normalizedExpr]
	ejit.mutex.Unlock()
	
	// Check if we should compile this expression
	if count == ejit.threshold {
		go ejit.compileExpression(normalizedExpr)
	}
}

// compileExpression attempts to compile an expression to native Go code
func (ejit *ExpressionJIT) compileExpression(expr string) {
	start := time.Now()
	
	// Use the full normalization for pattern matching (only during compilation)
	normalizedExpr := ejit.normalizeExpression(expr)
	
	// Try each pattern to see if it matches
	for _, pattern := range ejit.patterns {
		matches := pattern.regex.FindStringSubmatch(normalizedExpr)
		if matches != nil {
			// Pattern matched - compile it
			evaluator := pattern.compiler(normalizedExpr, matches)
			if evaluator != nil {
				compiled := &CompiledExpression{
					signature:    fmt.Sprintf("%s_%d", pattern.name, time.Now().UnixNano()),
					originalExpr: expr,
					evaluator:    evaluator,
					pattern:      pattern.name,
					hitCount:     0,
					createTime:   time.Now(),
					totalTime:    0,
					avgTime:      0,
				}
				
				ejit.mutex.Lock()
				ejit.compiledExprs[expr] = compiled // Use original key for lookups
				ejit.compilations++
				ejit.mutex.Unlock()
				
				compilationTime := time.Since(start)
				tinyBasicDebugLog("[ExpressionJIT] Compiled '%s' using pattern '%s' in %v", 
					expr, pattern.name, compilationTime)
				return
			}
		}
	}
	
	// No pattern matched
	tinyBasicDebugLog("[ExpressionJIT] No pattern matched for expression: %s", expr)
}

// shouldSkipExpression provides smart filtering - skip only expressions that truly won't benefit from JIT
func (ejit *ExpressionJIT) shouldSkipExpression(expr string) bool {
	// Very fast length check
	if len(expr) < 3 {
		return true
	}
	
	trimmed := strings.TrimSpace(expr)
	if len(trimmed) < 3 {
		return true
	}
	
	// Skip pure numeric literals (no variables or operators)
	hasVariable := false
	hasOperator := false
	hasComplexity := false
	
	for _, c := range trimmed {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			hasVariable = true
		}
		if c == '+' || c == '-' || c == '*' || c == '/' || c == '(' || c == ')' {
			hasOperator = true
		}
		if c == '.' || c == 'E' || c == 'e' { // Scientific notation, function calls
			hasComplexity = true
		}
	}
	
	// Skip if it's just a pure number (no variables, no operators)
	if !hasVariable && !hasOperator && !hasComplexity {
		return true
	}
	
	// Skip if it's just a single variable with no operators
	if hasVariable && !hasOperator && len(trimmed) <= 5 {
		// Check if it's really just a variable name
		isSimpleVar := true
		for _, c := range trimmed {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
				isSimpleVar = false
				break
			}
		}
		if isSimpleVar {
			return true
		}
	}
	
	// Skip string literals
	if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
		return true
	}
	
	// Everything else is a potential candidate for JIT optimization
	// This includes: variable + constant, mathematical expressions, function calls
	return false
}

// fastNormalizeExpression provides fast normalization for hot path
func (ejit *ExpressionJIT) fastNormalizeExpression(expr string) string {
	// Simple uppercase conversion without regex (faster)
	return strings.ToUpper(strings.TrimSpace(expr))
}

// normalizeExpression standardizes an expression for consistent matching (used for compilation)
func (ejit *ExpressionJIT) normalizeExpression(expr string) string {
	// Remove extra whitespace and convert to uppercase
	normalized := strings.ToUpper(strings.TrimSpace(expr))
	// Normalize spacing around operators
	normalized = regexp.MustCompile(`\s*\*\s*`).ReplaceAllString(normalized, " * ")
	normalized = regexp.MustCompile(`\s*\+\s*`).ReplaceAllString(normalized, " + ")
	normalized = regexp.MustCompile(`\s*-\s*`).ReplaceAllString(normalized, " - ")
	normalized = regexp.MustCompile(`\s*/\s*`).ReplaceAllString(normalized, " / ")
	normalized = regexp.MustCompile(`\s*\(\s*`).ReplaceAllString(normalized, "(")
	normalized = regexp.MustCompile(`\s*\)\s*`).ReplaceAllString(normalized, ")")
	return normalized
}

// Pattern compiler functions

func (ejit *ExpressionJIT) compileSimpleMultiplication(expr string, matches []string) func(vars map[string]BASICValue) BASICValue {
	if len(matches) != 3 {
		return nil
	}
	
	varName := matches[1]
	constant, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return nil
	}
	
	return func(vars map[string]BASICValue) BASICValue {
		if val, exists := vars[varName]; exists && val.IsNumeric {
			return BASICValue{NumValue: val.NumValue * constant, IsNumeric: true}
		}
		return BASICValue{NumValue: 0, IsNumeric: true}
	}
}

func (ejit *ExpressionJIT) compileSimpleAddition(expr string, matches []string) func(vars map[string]BASICValue) BASICValue {
	if len(matches) != 3 {
		return nil
	}
	
	varName := matches[1]
	constant, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return nil
	}
	
	return func(vars map[string]BASICValue) BASICValue {
		if val, exists := vars[varName]; exists && val.IsNumeric {
			return BASICValue{NumValue: val.NumValue + constant, IsNumeric: true}
		}
		return BASICValue{NumValue: constant, IsNumeric: true}
	}
}

func (ejit *ExpressionJIT) compileMathExpression(expr string, matches []string) func(vars map[string]BASICValue) BASICValue {
	if len(matches) != 5 {
		return nil
	}
	
	var1 := matches[1]
	const1, err1 := strconv.ParseFloat(matches[2], 64)
	var2 := matches[3] 
	const2, err2 := strconv.ParseFloat(matches[4], 64)
	
	if err1 != nil || err2 != nil {
		return nil
	}
	
	return func(vars map[string]BASICValue) BASICValue {
		val1 := 0.0
		val2 := 0.0
		
		if v, exists := vars[var1]; exists && v.IsNumeric {
			val1 = v.NumValue
		}
		if v, exists := vars[var2]; exists && v.IsNumeric {
			val2 = v.NumValue
		}
		
		result := val1*const1 + val2/const2
		return BASICValue{NumValue: result, IsNumeric: true}
	}
}

func (ejit *ExpressionJIT) compileSinFunction(expr string, matches []string) func(vars map[string]BASICValue) BASICValue {
	if len(matches) != 3 {
		return nil
	}
	
	varName := matches[1]
	constant, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return nil
	}
	
	return func(vars map[string]BASICValue) BASICValue {
		if val, exists := vars[varName]; exists && val.IsNumeric {
			result := math.Sin(val.NumValue * constant)
			return BASICValue{NumValue: result, IsNumeric: true}
		}
		return BASICValue{NumValue: 0, IsNumeric: true}
	}
}

func (ejit *ExpressionJIT) compileVariableAccess(expr string, matches []string) func(vars map[string]BASICValue) BASICValue {
	if len(matches) != 2 {
		return nil
	}
	
	varName := matches[1]
	
	return func(vars map[string]BASICValue) BASICValue {
		if val, exists := vars[varName]; exists {
			return val
		}
		return BASICValue{NumValue: 0, IsNumeric: true}
	}
}

func (ejit *ExpressionJIT) compileLoopIncrement(expr string, matches []string) func(vars map[string]BASICValue) BASICValue {
	if len(matches) != 2 {
		return nil
	}
	
	varName := matches[1]
	
	return func(vars map[string]BASICValue) BASICValue {
		if val, exists := vars[varName]; exists && val.IsNumeric {
			return BASICValue{NumValue: val.NumValue + 1, IsNumeric: true}
		}
		return BASICValue{NumValue: 1, IsNumeric: true}
	}
}

func (ejit *ExpressionJIT) compileVariableSubtraction(expr string, matches []string) func(vars map[string]BASICValue) BASICValue {
	if len(matches) != 3 {
		return nil
	}
	
	varName := matches[1]
	constant, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return nil
	}
	
	return func(vars map[string]BASICValue) BASICValue {
		if val, exists := vars[varName]; exists && val.IsNumeric {
			return BASICValue{NumValue: val.NumValue - constant, IsNumeric: true}
		}
		return BASICValue{NumValue: -constant, IsNumeric: true}
	}
}

func (ejit *ExpressionJIT) compileArrayIndex(expr string, matches []string) func(vars map[string]BASICValue) BASICValue {
	if len(matches) != 4 {
		return nil
	}
	
	var1 := matches[1]
	multiplier, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return nil
	}
	var2 := matches[3]
	
	return func(vars map[string]BASICValue) BASICValue {
		val1 := 0.0
		val2 := 0.0
		
		if v, exists := vars[var1]; exists && v.IsNumeric {
			val1 = v.NumValue
		}
		if v, exists := vars[var2]; exists && v.IsNumeric {
			val2 = v.NumValue
		}
		
		result := val1*multiplier + val2
		return BASICValue{NumValue: result, IsNumeric: true}
	}
}

// GetStats returns JIT statistics
func (ejit *ExpressionJIT) GetStats() map[string]interface{} {
	ejit.mutex.RLock()
	defer ejit.mutex.RUnlock()
	
	hitRate := 0.0
	totalRequests := ejit.hits + ejit.misses
	if totalRequests > 0 {
		hitRate = float64(ejit.hits) / float64(totalRequests) * 100
	}
	
	stats := map[string]interface{}{
		"enabled":          ejit.enabled,
		"compiled_exprs":   len(ejit.compiledExprs),
		"execution_counts": len(ejit.executionCounts),
		"threshold":        ejit.threshold,
		"hits":            ejit.hits,
		"misses":          ejit.misses,
		"hit_rate":        fmt.Sprintf("%.2f%%", hitRate),
		"compilations":    ejit.compilations,
		"total_saved_time": ejit.totalSavedTime.String(),
		"patterns":        len(ejit.patterns),
	}
	
	// Add per-expression statistics
	exprStats := make(map[string]interface{})
	for expr, compiled := range ejit.compiledExprs {
		exprStats[expr] = map[string]interface{}{
			"pattern":     compiled.pattern,
			"hit_count":   compiled.hitCount,
			"avg_time":    compiled.avgTime.String(),
			"created_at":  compiled.createTime.Format(time.RFC3339),
		}
	}
	stats["expressions"] = exprStats
	
	return stats
}

// SetEnabled enables or disables the expression JIT
func (ejit *ExpressionJIT) SetEnabled(enabled bool) {
	ejit.enabled = enabled
}

// Clear clears all compiled expressions and statistics
func (ejit *ExpressionJIT) Clear() {
	ejit.mutex.Lock()
	defer ejit.mutex.Unlock()
	
	ejit.compiledExprs = make(map[string]*CompiledExpression)
	ejit.executionCounts = make(map[string]int64)
	ejit.hits = 0
	ejit.misses = 0
	ejit.compilations = 0
	ejit.totalSavedTime = 0
}