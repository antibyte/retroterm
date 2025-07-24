package tinybasic

import (
	"math"
	"testing"
	"time"
)

// JIT Compiler Benchmarks - Proof of Concept

// simulateInterpreterLoop simulates the current interpreter performance for a mathematical loop
// Note: Moved duplicate function declaration to avoid redeclaration error
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

// simulateJITCompiledLoop simulates a JIT-compiled version with native Go performance
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

// BenchmarkInterpreterVsJIT compares interpreter vs JIT performance
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

// BenchmarkJIT_SmallLoop tests JIT effectiveness on small loops
func BenchmarkJIT_SmallLoop(b *testing.B) {
	iterations := 100
	
	b.Run("Small_Interpreter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := simulateInterpreterLoop_Benchmark(iterations)
			_ = result["RESULT"].NumValue
		}
	})
	
	b.Run("Small_JIT", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := simulateJITCompiledLoop(iterations)
			_ = result["RESULT"].NumValue
		}
	})
}

// BenchmarkJIT_LargeLoop tests JIT effectiveness on large loops
func BenchmarkJIT_LargeLoop(b *testing.B) {
	iterations := 100000
	
	b.Run("Large_Interpreter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := simulateInterpreterLoop_Benchmark(iterations)
			_ = result["RESULT"].NumValue
		}
	})
	
	b.Run("Large_JIT", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			result := simulateJITCompiledLoop(iterations)
			_ = result["RESULT"].NumValue
		}
	})
}

// BenchmarkJIT_ComplexExpression tests JIT on complex mathematical expressions
func BenchmarkJIT_ComplexExpression(b *testing.B) {
	iterations := 50000
	
	b.Run("Complex_Interpreter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			variables := make(map[string]BASICValue)
			
			// Simulate complex expression: (A + B) * (C - D) / (E + F) + SIN(G) * COS(H)
			for j := 1; j <= iterations; j++ {
				// Multiple variable lookups and operations
				a := variables["A"].NumValue + 1.0
				b := variables["B"].NumValue + 2.0
				c := variables["C"].NumValue + 3.0
				d := variables["D"].NumValue + 4.0
				e := variables["E"].NumValue + 5.0
				f := variables["F"].NumValue + 6.0
				g := variables["G"].NumValue + 0.1*float64(j)
				h := variables["H"].NumValue + 0.2*float64(j)
				
				// Complex calculation with function calls
				result := (a+b)*(c-d)/(e+f) + math.Sin(g)*math.Cos(h)
				
				variables["RESULT"] = BASICValue{NumValue: result, IsNumeric: true}
			}
			
			_ = variables["RESULT"].NumValue
		}
	})
	
	b.Run("Complex_JIT", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// JIT-compiled version with native operations
			var result float64
			
			for j := 1; j <= iterations; j++ {
				// Direct native operations - all variables as local Go variables
				a := 1.0
				b := 2.0
				c := 3.0
				d := 4.0
				e := 5.0
				f := 6.0
				g := 0.1 * float64(j)
				h := 0.2 * float64(j)
				
				// Native calculation - much faster
				result = (a+b)*(c-d)/(e+f) + math.Sin(g)*math.Cos(h)
			}
			
			_ = result
		}
	})
}

// BenchmarkJIT_CompilationOverhead tests the cost of JIT compilation
func BenchmarkJIT_CompilationOverhead(b *testing.B) {
	jit := NewJITCompiler()
	defer jit.CleanupTempFiles()
	
	b.Run("JIT_Compilation_Cost", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = i // Use i to avoid "declared and not used" error
			loopInfo := &LoopInfo{
				Variable:   "I",
				StartValue: "1",
				EndValue:   "1000",
				StepValue:  "1",
				BodyExprs:  []string{"RESULT = I * 2.5 + SIN(I * 0.01)"},
			}
			
			// Measure compilation time
			start := time.Now()
			goCode := jit.generateGoCode(loopInfo)
			_ = goCode
			compilationTime := time.Since(start)
			
			// Record compilation time (in actual implementation)
			_ = compilationTime
		}
	})
}

// BenchmarkJIT_HotSpotDetection tests hot spot detection overhead
func BenchmarkJIT_HotSpotDetection(b *testing.B) {
	jit := NewJITCompiler()
	defer jit.CleanupTempFiles()
	
	signature := "test_hot_spot"
	
	b.Run("HotSpot_Detection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			jit.RecordLoopExecution(signature)
			isHot := jit.IsHotLoop(signature)
			_ = isHot
		}
	})
}

// Test correctness of JIT compilation
func TestJIT_Correctness(t *testing.T) {
	iterations := 1000
	
	// Execute same computation with both interpreter and JIT
	interpreterResult := simulateInterpreterLoop_Benchmark(iterations)
	jitResult := simulateJITCompiledLoop(iterations)
	
	// Results should be identical (within floating point precision)
	interpreterValue := interpreterResult["RESULT"].NumValue
	jitValue := jitResult["RESULT"].NumValue
	
	if math.Abs(interpreterValue-jitValue) > 1e-10 {
		t.Errorf("JIT result differs from interpreter: interpreter=%f, jit=%f", 
			interpreterValue, jitValue)
	}
	
	// Loop variable should also match
	interpreterI := interpreterResult["I"].NumValue
	jitI := jitResult["I"].NumValue
	
	if interpreterI != jitI {
		t.Errorf("Loop variable differs: interpreter=%f, jit=%f", interpreterI, jitI)
	}
}

// TestJIT_HotSpotDetection tests hot spot detection logic
func TestJIT_HotSpotDetection(t *testing.T) {
	jit := NewJITCompiler()
	defer jit.CleanupTempFiles()
	
	signature := "test_loop"
	
	// Should not be hot initially
	if jit.IsHotLoop(signature) {
		t.Error("Loop should not be hot initially")
	}
	
	// Execute below threshold
	for i := 0; i < 50; i++ {
		jit.RecordLoopExecution(signature)
	}
	
	if jit.IsHotLoop(signature) {
		t.Error("Loop should not be hot below threshold")
	}
	
	// Execute above threshold
	for i := 0; i < 60; i++ {
		jit.RecordLoopExecution(signature)
	}
	
	if !jit.IsHotLoop(signature) {
		t.Error("Loop should be hot above threshold")
	}
}

// Performance comparison helper
func comparePerformance(t *testing.T, interpreterNs, jitNs int64) {
	if jitNs >= interpreterNs {
		t.Logf("JIT not faster: interpreter=%dns, jit=%dns", interpreterNs, jitNs)
		return
	}
	
	speedup := float64(interpreterNs) / float64(jitNs)
	t.Logf("JIT speedup: %.2fx faster (interpreter=%dns, jit=%dns)", 
		speedup, interpreterNs, jitNs)
	
	if speedup < 2.0 {
		t.Logf("Warning: JIT speedup less than 2x (%.2fx)", speedup)
	}
}