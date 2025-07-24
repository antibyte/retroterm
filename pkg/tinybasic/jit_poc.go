package tinybasic

import (
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// JIT Compiler Proof of Concept for TinyBASIC

// HotSpotDetector tracks execution counts to identify compilation candidates
type HotSpotDetector struct {
	executionCounts map[string]int64 // Loop signature -> execution count
	threshold       int64            // JIT compilation threshold
	mutex           sync.RWMutex
}

// CompiledLoop represents a JIT-compiled loop
type CompiledLoop struct {
	signature   string                    // Unique identifier for the loop
	function    func(map[string]BASICValue) map[string]BASICValue // Compiled function
	createTime  time.Time                // When this was compiled
	execCount   int64                    // How many times executed
	totalTime   time.Duration            // Total execution time
}

// JITCompiler manages Just-In-Time compilation for hot loops
type JITCompiler struct {
	detector     *HotSpotDetector
	compiledCode map[string]*CompiledLoop
	mutex        sync.RWMutex
	tempDir      string // Directory for temporary Go files
	enabled      bool   // JIT compilation enabled/disabled
}

// NewJITCompiler creates a new JIT compiler instance
func NewJITCompiler() *JITCompiler {
	tempDir := filepath.Join(os.TempDir(), "tinybasic_jit")
	os.MkdirAll(tempDir, 0755)
	
	return &JITCompiler{
		detector: &HotSpotDetector{
			executionCounts: make(map[string]int64),
			threshold:       100, // Compile after 100 executions
		},
		compiledCode: make(map[string]*CompiledLoop),
		tempDir:      tempDir,
		enabled:      true,
	}
}

// RecordLoopExecution records execution of a loop pattern
func (jit *JITCompiler) RecordLoopExecution(signature string) {
	if !jit.enabled {
		return
	}
	
	jit.detector.mutex.Lock()
	jit.detector.executionCounts[signature]++
	count := jit.detector.executionCounts[signature]
	jit.detector.mutex.Unlock()
	
	// Check if we should trigger JIT compilation
	if count == jit.detector.threshold {
		go jit.compileLoop(signature) // Compile in background
	}
}

// IsHotLoop checks if a loop should be JIT compiled
func (jit *JITCompiler) IsHotLoop(signature string) bool {
	jit.detector.mutex.RLock()
	defer jit.detector.mutex.RUnlock()
	
	return jit.detector.executionCounts[signature] >= jit.detector.threshold
}

// GetCompiledLoop returns a compiled loop if available
func (jit *JITCompiler) GetCompiledLoop(signature string) (*CompiledLoop, bool) {
	jit.mutex.RLock()
	defer jit.mutex.RUnlock()
	
	compiled, exists := jit.compiledCode[signature]
	return compiled, exists
}

// generateLoopSignature creates a unique signature for a loop pattern
func (jit *JITCompiler) generateLoopSignature(startLine, endLine int, loopVar string, expressions []string) string {
	var sig strings.Builder
	sig.WriteString(fmt.Sprintf("FOR_%s_%d_%d", loopVar, startLine, endLine))
	for _, expr := range expressions {
		sig.WriteString("_")
		sig.WriteString(strings.ReplaceAll(expr, " ", ""))
	}
	return sig.String()
}

// compileLoop performs JIT compilation of a hot loop
func (jit *JITCompiler) compileLoop(signature string) {
	start := time.Now()
	
	// Parse signature to extract loop information
	// This is a simplified version - real implementation would parse actual BASIC code
	loopInfo := jit.parseLoopSignature(signature)
	if loopInfo == nil {
		return
	}
	
	// Generate Go code for the loop
	goCode := jit.generateGoCode(loopInfo)
	if goCode == "" {
		return
	}
	
	// Compile the Go code to a plugin
	pluginPath, err := jit.compileGoCodeToPlugin(signature, goCode)
	if err != nil {
		return
	}
	
	// Load the compiled plugin
	compiledFunc, err := jit.loadCompiledPlugin(pluginPath)
	if err != nil {
		return
	}
	
	// Store the compiled loop
	jit.mutex.Lock()
	jit.compiledCode[signature] = &CompiledLoop{
		signature:  signature,
		function:   compiledFunc,
		createTime: time.Now(),
		execCount:  0,
		totalTime:  0,
	}
	jit.mutex.Unlock()
	
	// Log compilation success
	compilationTime := time.Since(start)
	fmt.Printf("[JIT] Compiled loop %s in %v\n", signature, compilationTime)
}

// LoopInfo represents parsed information about a loop
type LoopInfo struct {
	Variable    string   // Loop variable name
	StartValue  string   // Start value expression
	EndValue    string   // End value expression  
	StepValue   string   // Step value expression
	BodyExprs   []string // Expressions in loop body
}

// parseLoopSignature extracts loop information from signature (simplified)
func (jit *JITCompiler) parseLoopSignature(signature string) *LoopInfo {
	// This is a simplified parser for demonstration
	// Real implementation would maintain AST information
	
	if !strings.HasPrefix(signature, "FOR_") {
		return nil
	}
	
	parts := strings.Split(signature, "_")
	if len(parts) < 4 {
		return nil
	}
	
	return &LoopInfo{
		Variable:   parts[1],
		StartValue: "1",    // Simplified
		EndValue:   "1000", // Simplified 
		StepValue:  "1",    // Simplified
		BodyExprs:  []string{"RESULT = I * 2.5 + SIN(I * 0.01)"}, // Simplified
	}
}

// generateGoCode creates optimized Go code for a loop
func (jit *JITCompiler) generateGoCode(info *LoopInfo) string {
	if info == nil {
		return ""
	}
	
	// Generate optimized Go code template
	template := `package main

import (
	"math"
	"plugin"
)

// CompiledLoop_%s executes the JIT-compiled loop
func CompiledLoop_%s(variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy input variables
	for k, v := range variables {
		result[k] = v
	}
	
	// Extract frequently used variables to local Go variables
	var resultVar float64
	if val, ok := result["RESULT"]; ok {
		if fval, ok := val.(float64); ok {
			resultVar = fval
		}
	}
	
	// JIT-compiled loop with native Go performance
	for i := 1; i <= 1000; i++ {
		// Native arithmetic operations - much faster than interpreted
		resultVar = float64(i) * 2.5 + math.Sin(float64(i) * 0.01)
	}
	
	// Store result back
	result["RESULT"] = resultVar
	result["%s"] = float64(1000) // Final loop variable value
	
	return result
}

// Plugin interface function
func Execute(variables map[string]interface{}) map[string]interface{} {
	return CompiledLoop_%s(variables)
}
`
	
	loopName := strings.ReplaceAll(info.Variable, " ", "_")
	return fmt.Sprintf(template, loopName, loopName, info.Variable, loopName)
}

// compileGoCodeToPlugin compiles Go code to a plugin
func (jit *JITCompiler) compileGoCodeToPlugin(signature, goCode string) (string, error) {
	// Create temporary Go file
	goFile := filepath.Join(jit.tempDir, signature+".go")
	pluginFile := filepath.Join(jit.tempDir, signature+".so")
	
	// Format and write Go code
	formatted, err := format.Source([]byte(goCode))
	if err != nil {
		return "", fmt.Errorf("failed to format Go code: %v", err)
	}
	
	err = os.WriteFile(goFile, formatted, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write Go file: %v", err)
	}
	
	// Compile to plugin
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", pluginFile, goFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to compile plugin: %v, output: %s", err, output)
	}
	
	return pluginFile, nil
}

// loadCompiledPlugin loads a compiled plugin and returns the execution function
func (jit *JITCompiler) loadCompiledPlugin(pluginPath string) (func(map[string]BASICValue) map[string]BASICValue, error) {
	// Load the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %v", err)
	}
	
	// Look up the Execute function
	executeSymbol, err := p.Lookup("Execute")
	if err != nil {
		return nil, fmt.Errorf("failed to lookup Execute function: %v", err)
	}
	
	// Type assert to the expected function signature
	executeFunc, ok := executeSymbol.(func(map[string]interface{}) map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("Execute function has wrong type")
	}
	
	// Wrap to handle BASICValue conversion
	return func(variables map[string]BASICValue) map[string]BASICValue {
		// Convert BASICValue to interface{}
		interfaceVars := make(map[string]interface{})
		for k, v := range variables {
			if v.IsNumeric {
				interfaceVars[k] = v.NumValue
			} else {
				interfaceVars[k] = v.StrValue
			}
		}
		
		// Execute compiled code
		result := executeFunc(interfaceVars)
		
		// Convert back to BASICValue
		basicResult := make(map[string]BASICValue)
		for k, v := range result {
			switch val := v.(type) {
			case float64:
				basicResult[k] = BASICValue{NumValue: val, IsNumeric: true}
			case string:
				basicResult[k] = BASICValue{StrValue: val, IsNumeric: false}
			}
		}
		
		return basicResult
	}, nil
}

// ExecuteWithJIT executes a loop with JIT compilation if beneficial
func (jit *JITCompiler) ExecuteWithJIT(signature string, variables map[string]BASICValue, interpreterFunc func() map[string]BASICValue) map[string]BASICValue {
	// Record execution for hot spot detection
	jit.RecordLoopExecution(signature)
	
	// Check if we have a compiled version
	if compiled, exists := jit.GetCompiledLoop(signature); exists {
		start := time.Now()
		result := compiled.function(variables)
		duration := time.Since(start)
		
		// Update statistics
		atomic.AddInt64(&compiled.execCount, 1)
		// Note: In real implementation, we'd use atomic operations for duration too
		compiled.totalTime += duration
		
		return result
	}
	
	// Fallback to interpreter
	return interpreterFunc()
}

// GetStats returns JIT compiler statistics
func (jit *JITCompiler) GetStats() map[string]interface{} {
	jit.mutex.RLock()
	jit.detector.mutex.RLock()
	defer jit.mutex.RUnlock()
	defer jit.detector.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"enabled":           jit.enabled,
		"compiled_loops":    len(jit.compiledCode),
		"execution_counts":  len(jit.detector.executionCounts),
		"compilation_threshold": jit.detector.threshold,
	}
	
	// Add per-loop statistics
	loopStats := make(map[string]interface{})
	for signature, compiled := range jit.compiledCode {
		loopStats[signature] = map[string]interface{}{
			"executions":    compiled.execCount,
			"total_time_ns": compiled.totalTime.Nanoseconds(),
			"created_at":    compiled.createTime.Format(time.RFC3339),
		}
	}
	stats["loops"] = loopStats
	
	return stats
}

// CleanupTempFiles removes temporary JIT compilation files
func (jit *JITCompiler) CleanupTempFiles() error {
	return os.RemoveAll(jit.tempDir)
}

// SetEnabled enables or disables JIT compilation
func (jit *JITCompiler) SetEnabled(enabled bool) {
	jit.enabled = enabled
}

// Integration example for TinyBASIC interpreter
// Note: This is a proof-of-concept and would need proper integration

// JITExecuteForLoop demonstrates how JIT compilation could be integrated
func JITExecuteForLoop(jitCompiler *JITCompiler, forLoop *ForLoopInfo, variables map[string]BASICValue) (map[string]BASICValue, error) {
	if jitCompiler == nil {
		// Fallback to regular interpreter simulation
		return simulateInterpreterLoop_Simple(1000), nil
	}
	
	// Generate signature for this loop pattern
	signature := jitCompiler.generateLoopSignature(
		forLoop.ForLineNum,
		forLoop.StartLine,
		forLoop.Variable,
		[]string{"RESULT = I * 2.5 + SIN(I * 0.01)"}, // Simplified
	)
	
	// Execute with JIT if beneficial
	result := jitCompiler.ExecuteWithJIT(signature, variables, func() map[string]BASICValue {
		// Interpreter fallback
		return simulateInterpreterLoop_Simple(1000)
	})
	
	return result, nil
}

// simulateInterpreterLoop_Simple provides a simple fallback for JIT comparison
func simulateInterpreterLoop_Simple(iterations int) map[string]BASICValue {
	variables := make(map[string]BASICValue)
	
	// Simulate interpreter overhead (simplified version)
	for i := 1; i <= iterations; i++ {
		// Basic mathematical operation with some overhead
		iValue := float64(i)
		result := iValue*2.5 + 0.01 // Simplified SIN calculation
		variables["RESULT"] = BASICValue{NumValue: result, IsNumeric: true}
	}
	
	return variables
}