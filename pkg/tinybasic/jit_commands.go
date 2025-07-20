package tinybasic

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/shared"
)

// cmdJITOn enables JIT compilation
func (b *TinyBASIC) cmdJITOn(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("JITON")
	}
	
	if b.expressionJIT == nil {
		b.expressionJIT = NewExpressionJIT()
	}
	
	b.expressionJIT.SetEnabled(true)
	
	b.OutputChan <- shared.Message{
		Type:    shared.MessageTypeText,
		Content: "ExpressionJIT compilation enabled (safe, non-invasive)",
	}
	
	return nil
}

// cmdJITOff disables JIT compilation
func (b *TinyBASIC) cmdJITOff(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("JITOFF")
	}
	
	if b.expressionJIT != nil {
		b.expressionJIT.SetEnabled(false)
	}
	
	b.OutputChan <- shared.Message{
		Type:    shared.MessageTypeText,
		Content: "ExpressionJIT compilation disabled",
	}
	
	return nil
}

// cmdJITStats displays JIT compiler statistics
func (b *TinyBASIC) cmdJITStats(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("JITSTATS")
	}
	
	var output strings.Builder
	output.WriteString("=== JIT Compiler Statistics ===\n\n")
	
	// ExpressionJIT Stats (Active System)
	if b.expressionJIT != nil {
		stats := b.expressionJIT.GetStats()
		output.WriteString("--- ExpressionJIT (ACTIVE) ---\n")
		output.WriteString(fmt.Sprintf("Status: %s\n", map[bool]string{true: "ENABLED", false: "DISABLED"}[stats["enabled"].(bool)]))
		output.WriteString(fmt.Sprintf("Compiled Expressions: %d\n", stats["compiled_exprs"].(int)))
		output.WriteString(fmt.Sprintf("Tracked Expressions: %d\n", stats["execution_counts"].(int)))
		output.WriteString(fmt.Sprintf("Hit Rate: %s\n", stats["hit_rate"].(string)))
		output.WriteString(fmt.Sprintf("Total Compilations: %d\n", stats["compilations"].(int64)))
		output.WriteString(fmt.Sprintf("Patterns Available: %d\n", stats["patterns"].(int)))
		output.WriteString(fmt.Sprintf("Total Saved Time: %s\n", stats["total_saved_time"].(string)))
		
		// Display per-expression statistics
		if exprStats, ok := stats["expressions"].(map[string]interface{}); ok && len(exprStats) > 0 {
			output.WriteString("\n  Compiled Expressions:\n")
			for expr, data := range exprStats {
				if exprInfo, ok := data.(map[string]interface{}); ok {
					output.WriteString(fmt.Sprintf("    %s:\n", expr))
					output.WriteString(fmt.Sprintf("      Pattern: %s\n", exprInfo["pattern"].(string)))
					output.WriteString(fmt.Sprintf("      Hits: %d\n", exprInfo["hit_count"].(int64)))
					output.WriteString(fmt.Sprintf("      Avg Time: %s\n", exprInfo["avg_time"].(string)))
				}
			}
		}
		output.WriteString("\n")
	}
	
	// SimpleJIT Stats (Disabled)
	if b.simpleJIT != nil {
		stats := b.simpleJIT.GetStats()
		output.WriteString("--- SimpleJIT (DISABLED) ---\n")
		output.WriteString(fmt.Sprintf("Status: %s\n", map[bool]string{true: "ENABLED", false: "DISABLED"}[stats["enabled"].(bool)]))
		output.WriteString(fmt.Sprintf("Type: %s\n", stats["type"].(string)))
		output.WriteString(fmt.Sprintf("Note: Disabled for loop execution safety\n\n"))
	}
	
	output.WriteString("--- System Notes ---\n")
	output.WriteString("• ExpressionJIT: Safe, non-invasive expression optimization\n")
	output.WriteString("• SimpleJIT: Disabled due to loop interference issues\n")
	output.WriteString("• Token Cache: Active and working alongside ExpressionJIT\n")
	
	b.OutputChan <- shared.Message{
		Type:    shared.MessageTypeText,
		Content: output.String(),
	}
	
	return nil
}

// cmdJITClear clears JIT compilation cache
func (b *TinyBASIC) cmdJITClear(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("JITCLEAR")
	}
	
	if b.jitCompiler == nil {
		b.OutputChan <- shared.Message{
			Type:    shared.MessageTypeText,
			Content: "JIT compiler not initialized",
		}
		return nil
	}
	
	// Clean up temporary files
	err := b.jitCompiler.CleanupTempFiles()
	if err != nil {
		return NewBASICError(ErrCategorySystem, "FILE_ERROR", b.currentLine == 0, b.currentLine).WithCommand("JITCLEAR")
	}
	
	// Reinitialize the JIT compiler to clear cache
	enabled := b.jitCompiler.enabled
	b.jitCompiler = NewJITCompiler()
	b.jitCompiler.SetEnabled(enabled)
	
	b.OutputChan <- shared.Message{
		Type:    shared.MessageTypeText,
		Content: "JIT compilation cache cleared",
	}
	
	return nil
}

// cmdJITConfig configures JIT compiler settings
func (b *TinyBASIC) cmdJITConfig(args string) error {
	if b.jitCompiler == nil {
		b.jitCompiler = NewJITCompiler()
	}
	
	if args == "" {
		// Display current configuration
		var output strings.Builder
		output.WriteString("=== JIT Compiler Configuration ===\n")
		output.WriteString(fmt.Sprintf("Enabled: %t\n", b.jitCompiler.enabled))
		output.WriteString(fmt.Sprintf("Compilation Threshold: %d\n", b.jitCompiler.detector.threshold))
		output.WriteString(fmt.Sprintf("Temp Directory: %s\n", b.jitCompiler.tempDir))
		
		b.OutputChan <- shared.Message{
			Type:    shared.MessageTypeText,
			Content: output.String(),
		}
		return nil
	}
	
	// Parse configuration command
	parts := strings.Fields(strings.ToUpper(args))
	if len(parts) != 2 {
		return NewBASICError(ErrCategorySyntax, "INVALID_ARGUMENTS", b.currentLine == 0, b.currentLine).WithCommand("JITCONFIG")
	}
	
	setting := parts[0]
	value := parts[1]
	
	switch setting {
	case "THRESHOLD":
		threshold, err := strconv.ParseInt(value, 10, 64)
		if err != nil || threshold < 1 {
			return NewBASICError(ErrCategorySyntax, "INVALID_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("JITCONFIG")
		}
		b.jitCompiler.detector.threshold = threshold
		b.OutputChan <- shared.Message{
			Type:    shared.MessageTypeText,
			Content: fmt.Sprintf("JIT compilation threshold set to %d", threshold),
		}
		
	case "ENABLED":
		switch value {
		case "TRUE", "ON", "1":
			b.jitCompiler.SetEnabled(true)
			b.OutputChan <- shared.Message{
				Type:    shared.MessageTypeText,
				Content: "JIT compilation enabled",
			}
		case "FALSE", "OFF", "0":
			b.jitCompiler.SetEnabled(false)
			b.OutputChan <- shared.Message{
				Type:    shared.MessageTypeText,
				Content: "JIT compilation disabled",
			}
		default:
			return NewBASICError(ErrCategorySyntax, "INVALID_BOOLEAN", b.currentLine == 0, b.currentLine).WithCommand("JITCONFIG")
		}
		
	default:
		return NewBASICError(ErrCategorySyntax, "UNKNOWN_SETTING", b.currentLine == 0, b.currentLine).WithCommand("JITCONFIG")
	}
	
	return nil
}

// cmdJITBench runs a real JIT benchmark using actual ExpressionJIT
func (b *TinyBASIC) cmdJITBench(args string) error {
	// Parse iterations argument (default 1000)
	iterations := 1000
	if args != "" {
		var err error
		iterations, err = strconv.Atoi(strings.TrimSpace(args))
		if err != nil || iterations < 1 {
			return NewBASICError(ErrCategorySyntax, "INVALID_NUMBER", b.currentLine == 0, b.currentLine).WithCommand("JITBENCH")
		}
	}
	
	b.OutputChan <- shared.Message{
		Type:    shared.MessageTypeText,
		Content: fmt.Sprintf("Running REAL JIT benchmark with %d iterations...", iterations),
	}
	
	// Test expression that should benefit from JIT
	testExpr := "I * 2.5 + I / 3.14"
	
	// Benchmark WITHOUT JIT (using normal expression evaluation)
	start := time.Now()
	noJitResult := 0.0
	jitWasEnabled := false
	if b.expressionJIT != nil {
		jitWasEnabled = b.expressionJIT.enabled
		b.expressionJIT.SetEnabled(false) // Disable JIT for baseline
	}
	
	for i := 1; i <= iterations; i++ {
		b.variables["I"] = BASICValue{NumValue: float64(i), IsNumeric: true}
		result, err := b.evalExpression(testExpr)
		if err == nil && result.IsNumeric {
			noJitResult = result.NumValue
		}
	}
	noJitTime := time.Since(start)
	
	// Benchmark WITH JIT (enable and warm up)
	if b.expressionJIT != nil {
		b.expressionJIT.SetEnabled(true)
		b.expressionJIT.Clear() // Clear any existing compilations
		
		// Warm up JIT - trigger compilation
		for i := 1; i <= 25; i++ { // More than threshold of 20
			b.variables["I"] = BASICValue{NumValue: float64(i), IsNumeric: true}
			b.evalExpression(testExpr)
		}
	}
	
	start = time.Now()
	jitResult := 0.0
	for i := 1; i <= iterations; i++ {
		b.variables["I"] = BASICValue{NumValue: float64(i), IsNumeric: true}
		result, err := b.evalExpression(testExpr)
		if err == nil && result.IsNumeric {
			jitResult = result.NumValue
		}
	}
	jitTime := time.Since(start)
	
	// Restore original JIT state
	if b.expressionJIT != nil {
		b.expressionJIT.SetEnabled(jitWasEnabled)
	}
	
	// Calculate speedup
	speedup := float64(noJitTime.Nanoseconds()) / float64(jitTime.Nanoseconds())
	
	// Format results
	var output strings.Builder
	output.WriteString("=== REAL JIT Benchmark Results ===\n")
	output.WriteString(fmt.Sprintf("Expression: %s\n", testExpr))
	output.WriteString(fmt.Sprintf("Iterations: %d\n", iterations))
	
	// Show times with appropriate precision
	noJitMicros := float64(noJitTime.Nanoseconds()) / 1000.0
	jitMicros := float64(jitTime.Nanoseconds()) / 1000.0
	noJitMillis := noJitMicros / 1000.0
	jitMillis := jitMicros / 1000.0
	
	if noJitMillis >= 1.0 {
		output.WriteString(fmt.Sprintf("Without JIT: %.3f ms (%.0f µs)\n", noJitMillis, noJitMicros))
	} else if noJitMicros >= 1.0 {
		output.WriteString(fmt.Sprintf("Without JIT: %.1f µs\n", noJitMicros))
	} else {
		output.WriteString(fmt.Sprintf("Without JIT: %.0f ns\n", float64(noJitTime.Nanoseconds())))
	}
	
	if jitMillis >= 1.0 {
		output.WriteString(fmt.Sprintf("With JIT: %.3f ms (%.0f µs)\n", jitMillis, jitMicros))
	} else if jitMicros >= 1.0 {
		output.WriteString(fmt.Sprintf("With JIT: %.1f µs\n", jitMicros))
	} else {
		output.WriteString(fmt.Sprintf("With JIT: %.0f ns\n", float64(jitTime.Nanoseconds())))
	}
	
	if speedup > 1.0 {
		output.WriteString(fmt.Sprintf("Speedup: %.2fx FASTER with JIT\n", speedup))
	} else if speedup < 1.0 {
		output.WriteString(fmt.Sprintf("Slowdown: %.2fx SLOWER with JIT\n", 1.0/speedup))
	} else {
		output.WriteString("Performance: Same speed\n")
	}
	
	output.WriteString(fmt.Sprintf("Per-iteration time:\n"))
	output.WriteString(fmt.Sprintf("  Without JIT: %.2f ns/iteration\n", float64(noJitTime.Nanoseconds())/float64(iterations)))
	output.WriteString(fmt.Sprintf("  With JIT: %.2f ns/iteration\n", float64(jitTime.Nanoseconds())/float64(iterations)))
	
	// Check correctness
	diff := noJitResult - jitResult
	if diff < 1e-10 && diff > -1e-10 {
		output.WriteString("Correctness: PASSED\n")
	} else {
		output.WriteString(fmt.Sprintf("Correctness: FAILED (diff: %.10f)\n", diff))
	}
	
	// Show JIT stats
	if b.expressionJIT != nil {
		stats := b.expressionJIT.GetStats()
		output.WriteString(fmt.Sprintf("\nJIT Statistics:\n"))
		output.WriteString(fmt.Sprintf("  Compiled expressions: %d\n", stats["compiled_exprs"].(int)))
		output.WriteString(fmt.Sprintf("  Hit rate: %s\n", stats["hit_rate"].(string)))
		output.WriteString(fmt.Sprintf("  Total compilations: %d\n", stats["compilations"].(int64)))
	}
	
	b.OutputChan <- shared.Message{
		Type:    shared.MessageTypeText,
		Content: output.String(),
	}
	
	return nil
}

// simulateInterpreterLoop_Simple is defined in jit_poc.go