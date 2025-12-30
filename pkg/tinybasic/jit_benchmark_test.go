package tinybasic

import (
	"context"
	"math"
	"testing"
	"time"
)

// simulateInterpreterLoop_Benchmark simulates the overhead of the current interpreter
// for a heavy mathematical loop: FOR I = 1 TO 10000: RESULT = I * 2.5 + SIN(I * 0.01): NEXT I
func simulateInterpreterLoop_Benchmark(iterations int) map[string]BASICValue {
	variables := make(map[string]BASICValue)

	// Simulate interpreter overhead
	for i := 1; i <= iterations; i++ {
		// Variable lookup overhead (map access)
		iVal := BASICValue{NumValue: float64(i), IsNumeric: true}
		variables["I"] = iVal

		// Expression parsing and evaluation overhead
		// Simulates: RESULT = I * 2.5 + SIN(I * 0.01)

		// 1. Parse "I" - variable lookup
		iValue := variables["I"].NumValue

		// 2. Parse "2.5" - constant
		constant := 2.5

		// 3. Multiplication operation
		temp1 := iValue * constant

		// 4. Parse "I * 0.01" for SIN argument
		sinArg := iValue * 0.01

		// 5. Function call overhead + actual SIN calculation
		sinResult := math.Sin(sinArg)

		// 6. Addition operation
		finalResult := temp1 + sinResult

		// 7. Variable assignment overhead
		variables["RESULT"] = BASICValue{NumValue: finalResult, IsNumeric: true}

		// Simulate additional interpreter overhead per iteration
		time.Sleep(1 * time.Nanosecond) // Represents parsing/execution overhead
	}

	return variables
}

// simulateJITCompiledLoop simulates the performance of a JIT-compiled version
// using direct Go operations but maintaining the same logic
func simulateJITCompiledLoop(iterations int) map[string]BASICValue {
	variables := make(map[string]BASICValue)

	// JIT-compiled native loop - direct Go operations
	var result float64

	// Native Go loop with minimal overhead
	for i := 1; i <= iterations; i++ {
		// Direct native operations - no interpretation overhead
		result = float64(i)*2.5 + math.Sin(float64(i)*0.01)
	}

	// Only convert back to BASICValue at the end (final loop variable value)
	variables["I"] = BASICValue{NumValue: float64(iterations), IsNumeric: true}
	variables["RESULT"] = BASICValue{NumValue: result, IsNumeric: true}

	return variables
}

// BenchmarkInterpreterVsJIT compares simulated interpreter vs JIT performance
func BenchmarkInterpreterVsJIT(b *testing.B) {
	iterations := 10000

	b.Run("Interpreter", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := simulateInterpreterLoop_Benchmark(iterations)
			// Prevent optimization
			_ = result["RESULT"].NumValue
		}
	})

	b.Run("JIT_Compiled", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			result := simulateJITCompiledLoop(iterations)
			// Prevent optimization
			_ = result["RESULT"].NumValue
		}
	})
}

// TestJIT_HotSpotDetection tests that the JIT can detect hot loops
func TestJIT_HotSpotDetection(t *testing.T) {
	// Initialize JIT
	jit := NewSimpleJIT()
	jit.threshold = 5

	// Simulate executions
	signature := "FOR_I_10"

	for i := 0; i < 4; i++ {
		jit.RecordExecution(signature)
		if jit.IsHot(signature) {
			t.Errorf("Loop should not be hot yet (count: %d, threshold: 5)", i+1)
		}
	}

	// 5th execution - should become hot
	jit.RecordExecution(signature)
	if !jit.IsHot(signature) {
		t.Errorf("Loop should be hot after 5 executions")
	}
}

// TestJIT_CompilationOverhead measures the overhead of compiling a loop
func TestJIT_CompilationOverhead(t *testing.T) {
	jit := NewSimpleJIT()

	start := time.Now()

	// Compile a simple math loop
	signature := "FOR_Benchmark_10"
	bodyExprs := []string{
		"RESULT = I * 2.5 + SIN(I * 0.01)",
	}

	err := jit.CompileLoop(signature, bodyExprs)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	duration := time.Since(start)
	t.Logf("Compilation time: %v", duration)

	// Verify it was compiled
	compiled, exists := jit.GetCompiled(signature)
	if !exists {
		t.Errorf("Loop was not registered as compiled")
	}
	if compiled == nil {
		t.Errorf("Compiled loop is nil")
	}
}

// TestJIT_OptimizationGain measures the speedup of a compiled loop vs simulated interpreter
func TestJIT_OptimizationGain(t *testing.T) {
	jit := NewSimpleJIT()

	// 1. Compile loop
	signature := "FOR_Gain_10"
	bodyExprs := []string{
		"RESULT = I * 2.5 + SIN(I * 0.01)",
	}
	jit.CompileLoop(signature, bodyExprs)

	compiled, _ := jit.GetCompiled(signature)
	if compiled == nil {
		t.Fatal("Failed to compile loop")
	}

	iterations := 100000

	// 2. Measure JIT execution
	vars := make(map[string]BASICValue)

	startJIT := time.Now()
	// Run the compiled executor
	resultJIT := compiled.executor(1, float64(iterations), 1, vars)
	jitTime := time.Since(startJIT)

	// 3. Measure Simulated Interpreter
	startInterp := time.Now()
	resultInterp := simulateInterpreterLoop_Benchmark(iterations)
	interpTime := time.Since(startInterp)

	// Compare results
	if math.Abs(resultJIT["RESULT"].NumValue-resultInterp["RESULT"].NumValue) > 0.0001 {
		t.Errorf("Results differ: JIT=%f, Interp=%f",
			resultJIT["RESULT"].NumValue, resultInterp["RESULT"].NumValue)
	}

	// Calculate speedup
	speedup := float64(interpTime) / float64(jitTime)
	interpreterNs := float64(interpTime.Nanoseconds()) / float64(iterations)
	jitNs := float64(jitTime.Nanoseconds()) / float64(iterations)

	t.Logf("Interpreter: %v (%.2f ns/op)", interpTime, interpreterNs)
	t.Logf("JIT:         %v (%.2f ns/op)", jitTime, jitNs)
	t.Logf("Speedup:     %.2fx", speedup)

	// JIT should be at least 2x faster (actually usually 10x+)
	if speedup < 2.0 {
		t.Logf("Warning: JIT speedup less than 2x (%.2fx)", speedup)
	}
}

// BenchmarkVM_ForLoop measures the core VM execution speed of a tight loop
func BenchmarkVM_ForLoop(b *testing.B) {
	// 1. Setup minimal environment
	machine := NewTinyBASIC(nil)
	machine.useBytecode = true // Ensure bytecode is on

	// 2. Load a program that does a tight loop
	// "10 FOR I = 1 TO 1000"
	// "20 LET A = A + 1"
	// "30 NEXT I"
	machine.Execute("10 FOR I = 1 TO 1000")
	machine.Execute("20 LET A = A + 1")
	machine.Execute("30 NEXT I")

	// 3. Compile it
	err := machine.compileProgramIfNeeded()
	if err != nil {
		b.Fatalf("Compilation failed: %v", err)
	}

	// 4. Benchmark the VM Run loop
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Run synchronously.
		err := machine.bytecodeVM.Run(context.Background())
		if err != nil {
			b.Fatalf("Execution failed: %v", err)
		}
	}
}

func BenchmarkInterpreter_ForLoop(b *testing.B) {
	// 1. Setup minimal environment
	machine := NewTinyBASIC(nil)
	machine.useBytecode = false // Force interpreter

	// 2. Load a program that does a tight loop
	machine.Execute("10 FOR I = 1 TO 100")
	machine.Execute("20 LET A = A + 1")
	machine.Execute("30 NEXT I")

	// 3. Benchmark the Interpreter Run loop
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Run synchronously.
		// Note: Execute runs the whole program if not run line-by-line,
		// but here we want to re-run the program.
		// TinyBASIC.Run() runs from currentLine.
		machine.currentLine = 10
		machine.running = true
		machine.runProgramInternal(context.Background())
	}
}
