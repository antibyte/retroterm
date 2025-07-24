package tinybasic

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// JITLoopAnalyzer analyzes BASIC FOR loops for JIT compilation potential
type JITLoopAnalyzer struct {
	basic *TinyBASIC
}

// NewJITLoopAnalyzer creates a new loop analyzer
func NewJITLoopAnalyzer(basic *TinyBASIC) *JITLoopAnalyzer {
	return &JITLoopAnalyzer{basic: basic}
}

// AnalyzeForLoop extracts information from a FOR loop for JIT compilation
func (jla *JITLoopAnalyzer) AnalyzeForLoop(forLineNum int, variable string, startValue, endValue, stepValue float64) (*LoopInfo, error) {
	// Extract the loop body by finding all statements between FOR and NEXT
	bodyLines, err := jla.extractLoopBody(forLineNum, variable)
	if err != nil {
		return nil, err
	}
	
	// Analyze the expressions in the loop body
	bodyExprs := jla.extractExpressions(bodyLines)
	
	return &LoopInfo{
		Variable:    variable,
		StartValue:  fmt.Sprintf("%.0f", startValue),
		EndValue:    fmt.Sprintf("%.0f", endValue), 
		StepValue:   fmt.Sprintf("%.0f", stepValue),
		BodyExprs:   bodyExprs,
	}, nil
}

// extractLoopBody finds all lines between FOR and corresponding NEXT
func (jla *JITLoopAnalyzer) extractLoopBody(forLineNum int, variable string) ([]string, error) {
	var bodyLines []string
	nestingLevel := 0
	searchLine := forLineNum
	
	// Find the next line after FOR
	nextLine, found := jla.basic.findNextLine(searchLine)
	if !found {
		return nil, fmt.Errorf("no lines after FOR statement")
	}
	
	for {
		lineCode, exists := jla.basic.program[nextLine]
		if !exists {
			return nil, fmt.Errorf("NEXT not found for FOR %s at line %d", variable, forLineNum)
		}
		
		// Check if this line contains FOR or NEXT
		upperLine := strings.ToUpper(strings.TrimSpace(lineCode))
		
		if strings.HasPrefix(upperLine, "FOR ") {
			nestingLevel++
			bodyLines = append(bodyLines, lineCode)
		} else if strings.HasPrefix(upperLine, "NEXT") {
			if nestingLevel == 0 {
				// Found our matching NEXT
				break
			}
			nestingLevel--
			bodyLines = append(bodyLines, lineCode)
		} else {
			// Regular loop body line
			bodyLines = append(bodyLines, lineCode)
		}
		
		// Move to next line
		nextLine, found = jla.basic.findNextLine(nextLine)
		if !found {
			return nil, fmt.Errorf("NEXT not found for FOR %s at line %d", variable, forLineNum)
		}
	}
	
	return bodyLines, nil
}

// extractExpressions analyzes loop body lines to extract expressions for compilation
func (jla *JITLoopAnalyzer) extractExpressions(bodyLines []string) []string {
	var expressions []string
	
	for _, line := range bodyLines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(strings.ToUpper(trimmed), "REM") {
			continue
		}
		
		// Split by colons to handle multiple statements per line
		statements := strings.Split(trimmed, ":")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			
			// Look for assignment statements (var = expression)
			if strings.Contains(stmt, "=") && !strings.Contains(stmt, "==") {
				expressions = append(expressions, stmt)
			}
		}
	}
	
	return expressions
}

// shouldJITCompile determines if a loop is worth JIT compiling
func (jla *JITLoopAnalyzer) shouldJITCompile(info *LoopInfo) bool {
	// Calculate estimated iterations
	start, _ := strconv.ParseFloat(info.StartValue, 64)
	end, _ := strconv.ParseFloat(info.EndValue, 64)
	step, _ := strconv.ParseFloat(info.StepValue, 64)
	
	if step == 0 {
		return false // Infinite loop, don't JIT
	}
	
	var iterations float64
	if step > 0 {
		iterations = (end - start) / step + 1
	} else {
		iterations = (start - end) / (-step) + 1
	}
	
	// Only JIT if we expect significant iterations and have mathematical expressions
	return iterations >= 100 && len(info.BodyExprs) > 0
}

// JITForLoopExecution integrates JIT compilation into FOR loop execution
func (b *TinyBASIC) JITForLoopExecution(forLoop *ForLoopInfo) (bool, error) {
	// Check if JIT is enabled
	if b.jitCompiler == nil || !b.jitCompiler.enabled {
		return false, nil // JIT disabled, use interpreter
	}
	
	// Generate signature for this loop
	signature := b.generateForLoopSignature(forLoop)
	
	// Record execution for hot spot detection
	b.jitCompiler.RecordLoopExecution(signature)
	
	// Check if we have a compiled version
	if compiled, exists := b.jitCompiler.GetCompiledLoop(signature); exists {
		// Execute JIT compiled version
		return b.executeJITCompiledLoop(compiled, forLoop)
	}
	
	// Check if this loop should be JIT compiled
	if b.jitCompiler.IsHotLoop(signature) {
		// Analyze the loop for compilation potential
		analyzer := NewJITLoopAnalyzer(b)
		loopInfo, err := analyzer.AnalyzeForLoop(
			forLoop.ForLineNum, 
			forLoop.Variable,
			b.variables[forLoop.Variable].NumValue, // Current start value
			forLoop.EndValue,
			forLoop.Step,
		)
		if err != nil {
			return false, nil // Fallback to interpreter
		}
		
		if analyzer.shouldJITCompile(loopInfo) {
			// Try immediate compilation for simple loops
			err := b.compileForLoopImmediate(signature, loopInfo)
			if err == nil {
				// Compilation successful, try to execute with JIT
				if compiled, exists := b.jitCompiler.GetCompiledLoop(signature); exists {
					return b.executeJITCompiledLoop(compiled, forLoop)
				}
			}
		}
	}
	
	return false, nil // Use interpreter for this iteration
}

// generateForLoopSignature creates a unique signature for a FOR loop
func (b *TinyBASIC) generateForLoopSignature(forLoop *ForLoopInfo) string {
	// Extract body expressions for signature
	analyzer := NewJITLoopAnalyzer(b)
	bodyLines, err := analyzer.extractLoopBody(forLoop.ForLineNum, forLoop.Variable)
	if err != nil {
		// Fallback signature based on line numbers
		return fmt.Sprintf("FOR_%s_%d", forLoop.Variable, forLoop.ForLineNum)
	}
	
	bodyExprs := analyzer.extractExpressions(bodyLines)
	return b.jitCompiler.generateLoopSignature(
		forLoop.ForLineNum,
		forLoop.StartLine,
		forLoop.Variable,
		bodyExprs,
	)
}

// executeJITCompiledLoop executes a JIT-compiled loop
func (b *TinyBASIC) executeJITCompiledLoop(compiled *CompiledLoop, forLoop *ForLoopInfo) (bool, error) {
	start := time.Now()
	
	// Prepare variables for JIT execution
	jitVars := make(map[string]BASICValue)
	for k, v := range b.variables {
		jitVars[k] = v
	}
	
	// Execute the compiled loop
	result := compiled.function(jitVars)
	
	// Update interpreter state with results
	for k, v := range result {
		b.variables[k] = v
	}
	
	// Update statistics
	duration := time.Since(start)
	compiled.execCount++
	compiled.totalTime += duration
	
	// Complete the loop (skip to after NEXT)
	nextLine, err := b.findMatchingNext(forLoop.ForLineNum, forLoop.Variable)
	if err != nil {
		return false, err
	}
	
	// Remove the loop from the stack
	if len(b.forLoops) > 0 && b.forLoops[len(b.forLoops)-1].Variable == forLoop.Variable {
		delete(b.forLoopIndexMap, forLoop.Variable)
		b.forLoops = b.forLoops[:len(b.forLoops)-1]
	}
	
	// Set the current line to after the NEXT
	b.currentLine = nextLine
	
	return true, nil // Successfully executed with JIT
}

// compileForLoop compiles a FOR loop using the JIT compiler (background)
func (b *TinyBASIC) compileForLoop(signature string, loopInfo *LoopInfo) {
	// This will be called by the JIT compiler's compileLoop method
	// The actual compilation logic is in jit_poc.go
	b.jitCompiler.compileLoop(signature)
}

// compileForLoopImmediate compiles a FOR loop immediately (blocking)
func (b *TinyBASIC) compileForLoopImmediate(signature string, loopInfo *LoopInfo) error {
	start := time.Now()
	
	// Generate Go code for the loop
	goCode := b.jitCompiler.generateGoCodeForBasic(loopInfo)
	if goCode == "" {
		return fmt.Errorf("failed to generate Go code")
	}
	
	// Compile the Go code to a plugin
	pluginPath, err := b.jitCompiler.compileGoCodeToPlugin(signature, goCode)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}
	
	// Load the compiled plugin
	compiledFunc, err := b.jitCompiler.loadCompiledPlugin(pluginPath)
	if err != nil {
		return fmt.Errorf("plugin load failed: %v", err)
	}
	
	// Store the compiled loop
	b.jitCompiler.mutex.Lock()
	b.jitCompiler.compiledCode[signature] = &CompiledLoop{
		signature:  signature,
		function:   compiledFunc,
		createTime: time.Now(),
		execCount:  0,
		totalTime:  0,
	}
	b.jitCompiler.mutex.Unlock()
	
	// Log compilation success
	compilationTime := time.Since(start)
	tinyBasicDebugLog("[JIT] Compiled loop %s in %v", signature, compilationTime)
	
	return nil
}

// Enhanced generateGoCode for real BASIC expressions
func (jit *JITCompiler) generateGoCodeForBasic(info *LoopInfo) string {
	if info == nil || len(info.BodyExprs) == 0 {
		return ""
	}
	
	// Generate more sophisticated Go code for BASIC expressions
	template := `package main

import (
	"math"
)

// BASICValue represents a BASIC value
type BASICValue struct {
	NumValue  float64
	StrValue  string
	IsNumeric bool
}

// CompiledLoop_%s executes the JIT-compiled BASIC loop
func CompiledLoop_%s(variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy input variables
	for k, v := range variables {
		result[k] = v
	}
	
	// Extract loop parameters
	start := 1.0
	end := 1000.0
	step := 1.0
	
	if val, ok := result["_START"]; ok {
		if fval, ok := val.(float64); ok {
			start = fval
		}
	}
	if val, ok := result["_END"]; ok {
		if fval, ok := val.(float64); ok {
			end = fval
		}
	}
	if val, ok := result["_STEP"]; ok {
		if fval, ok := val.(float64); ok {
			step = fval
		}
	}
	
	// JIT-compiled loop with native Go performance
	for i := start; (step > 0 && i <= end) || (step < 0 && i >= end); i += step {
		// Set loop variable
		result["%s"] = i
		
		// Compile loop body expressions
%s
	}
	
	return result
}

// Plugin interface function
func Execute(variables map[string]interface{}) map[string]interface{} {
	return CompiledLoop_%s(variables)
}
`
	
	// Generate code for each expression in the loop body
	var bodyCode strings.Builder
	for _, expr := range info.BodyExprs {
		compiledExpr := jit.compileBasicExpression(expr)
		if compiledExpr != "" {
			bodyCode.WriteString("\t\t")
			bodyCode.WriteString(compiledExpr)
			bodyCode.WriteString("\n")
		}
	}
	
	loopName := strings.ReplaceAll(info.Variable, " ", "_")
	return fmt.Sprintf(template, loopName, loopName, info.Variable, bodyCode.String(), loopName)
}

// compileBasicExpression converts a BASIC expression to Go code
func (jit *JITCompiler) compileBasicExpression(expr string) string {
	// Simple expression compiler for common BASIC patterns
	expr = strings.TrimSpace(expr)
	
	// Handle assignment: VAR = expression
	if eqPos := strings.Index(expr, "="); eqPos != -1 {
		varName := strings.TrimSpace(expr[:eqPos])
		expression := strings.TrimSpace(expr[eqPos+1:])
		
		// Convert BASIC functions to Go equivalents
		goExpr := jit.convertBasicFunctionsToGo(expression)
		
		return fmt.Sprintf("result[\"%s\"] = %s", varName, goExpr)
	}
	
	return ""
}

// convertBasicFunctionsToGo converts BASIC functions to Go math functions
func (jit *JITCompiler) convertBasicFunctionsToGo(expr string) string {
	// Replace BASIC functions with Go equivalents
	replacements := map[string]string{
		"SIN(":   "math.Sin(",
		"COS(":   "math.Cos(",
		"TAN(":   "math.Tan(",
		"SQRT(":  "math.Sqrt(",
		"ABS(":   "math.Abs(",
		"INT(":   "math.Floor(",
		"RND(":   "rand.Float64() * ",
	}
	
	result := expr
	for basic, goFunc := range replacements {
		result = strings.ReplaceAll(result, basic, goFunc)
	}
	
	// Convert variable references to map lookups
	// This is a simplified version - a real implementation would need proper parsing
	result = jit.convertVariableReferences(result)
	
	return result
}

// convertVariableReferences converts BASIC variable references to Go map lookups
func (jit *JITCompiler) convertVariableReferences(expr string) string {
	// This is a simplified implementation
	// A production version would need proper expression parsing
	
	// For now, assume simple variable patterns
	words := strings.Fields(expr)
	for i, word := range words {
		// Check if this looks like a variable (letters only, not a function)
		if len(word) > 0 && word[0] >= 'A' && word[0] <= 'Z' && !strings.Contains(word, "(") {
			// Convert to map lookup
			words[i] = fmt.Sprintf("result[\"%s\"].(float64)", word)
		}
	}
	
	return strings.Join(words, " ")
}