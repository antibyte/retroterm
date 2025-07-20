package tinybasic

import (
	"fmt"
	"testing"
)

// Benchmark tests for Expression Token Caching

func BenchmarkExpressionTokenCache_Simple(b *testing.B) {
	tb := NewTinyBASIC(nil)
	expr := "A + B * 2"
	
	// Set some variables for the expression
	tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tb.evalExpression(expr)
		if err != nil {
			b.Fatalf("Expression evaluation failed: %v", err)
		}
	}
}

func BenchmarkExpressionTokenCache_Complex(b *testing.B) {
	tb := NewTinyBASIC(nil)
	expr := "(A * B + C) / (D - E) * SIN(F) + SQRT(G)"
	
	// Set variables
	tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}
	tb.variables["C"] = BASICValue{NumValue: 3, IsNumeric: true}
	tb.variables["D"] = BASICValue{NumValue: 8, IsNumeric: true}
	tb.variables["E"] = BASICValue{NumValue: 2, IsNumeric: true}
	tb.variables["F"] = BASICValue{NumValue: 1.5, IsNumeric: true}
	tb.variables["G"] = BASICValue{NumValue: 16, IsNumeric: true}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tb.evalExpression(expr)
		if err != nil {
			b.Fatalf("Expression evaluation failed: %v", err)
		}
	}
}

func BenchmarkExpressionTokenCache_WithoutCache(b *testing.B) {
	tb := NewTinyBASIC(nil)
	// Disable cache by setting size to 0
	tb.exprTokenCache.SetMaxSize(0)
	
	expr := "A + B * C - D / E"
	
	// Set variables
	tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}
	tb.variables["C"] = BASICValue{NumValue: 3, IsNumeric: true}
	tb.variables["D"] = BASICValue{NumValue: 8, IsNumeric: true}
	tb.variables["E"] = BASICValue{NumValue: 2, IsNumeric: true}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tb.evalExpression(expr)
		if err != nil {
			b.Fatalf("Expression evaluation failed: %v", err)
		}
	}
	
	// Reset cache size
	tb.exprTokenCache.SetMaxSize(1000)
}

func BenchmarkExpressionTokenCache_MixedExpressions(b *testing.B) {
	tb := NewTinyBASIC(nil)
	
	// Prepare a set of different expressions
	expressions := []string{
		"A + B",
		"A * B + C",
		"SIN(X) + COS(Y)",
		"SQRT(A * A + B * B)",
		"(A + B) * (C - D)",
		"A * A + B * B",
		"ABS(A - B)",
		"INT(A / B) * B",
		"A MOD B",
		"NOT (A > B)",
	}
	
	// Set variables
	tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}
	tb.variables["C"] = BASICValue{NumValue: 3, IsNumeric: true}
	tb.variables["D"] = BASICValue{NumValue: 8, IsNumeric: true}
	tb.variables["X"] = BASICValue{NumValue: 1.57, IsNumeric: true}
	tb.variables["Y"] = BASICValue{NumValue: 0.78, IsNumeric: true}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expr := expressions[i%len(expressions)]
		_, err := tb.evalExpression(expr)
		if err != nil {
			b.Fatalf("Expression evaluation failed for %s: %v", expr, err)
		}
	}
}

func BenchmarkExpressionTokenCache_CacheHitRatio(b *testing.B) {
	tb := NewTinyBASIC(nil)
	
	// Create a limited set of expressions that will be repeated
	baseExpressions := []string{
		"A + B",
		"A * B",
		"A - B",
		"A / B",
		"A MOD B",
	}
	
	// Set variables
	tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use modulo to ensure we repeat expressions and get cache hits
		expr := baseExpressions[i%len(baseExpressions)]
		_, err := tb.evalExpression(expr)
		if err != nil {
			b.Fatalf("Expression evaluation failed: %v", err)
		}
	}
	
	// Report cache statistics
	stats := tb.exprTokenCache.GetStats()
	b.Logf("Cache hit ratio: %.2f%% (%d hits, %d misses)", 
		stats.HitRatio*100, stats.Hits, stats.Misses)
}

func BenchmarkExpressionTokenCache_LargeProgram(b *testing.B) {
	tb := NewTinyBASIC(nil)
	
	// Simulate a program with repeated expression evaluations (like in loops)
	expressions := make([]string, 100)
	
	// Generate expressions that would appear in a typical BASIC program
	for i := 0; i < 100; i++ {
		switch i % 10 {
		case 0:
			expressions[i] = fmt.Sprintf("I + %d", i%10)
		case 1:
			expressions[i] = fmt.Sprintf("J * %d", i%5+1)
		case 2:
			expressions[i] = "SIN(ANGLE)"
		case 3:
			expressions[i] = "COS(ANGLE)"
		case 4:
			expressions[i] = "X * X + Y * Y"
		case 5:
			expressions[i] = "SQRT(DISTANCE)"
		case 6:
			expressions[i] = fmt.Sprintf("COUNT + %d", i%3)
		case 7:
			expressions[i] = "TOTAL / ITEMS"
		case 8:
			expressions[i] = "INDEX MOD SIZE"
		case 9:
			expressions[i] = "NOT DONE"
		}
	}
	
	// Set variables
	tb.variables["I"] = BASICValue{NumValue: 1, IsNumeric: true}
	tb.variables["J"] = BASICValue{NumValue: 2, IsNumeric: true}
	tb.variables["ANGLE"] = BASICValue{NumValue: 1.57, IsNumeric: true}
	tb.variables["X"] = BASICValue{NumValue: 3, IsNumeric: true}
	tb.variables["Y"] = BASICValue{NumValue: 4, IsNumeric: true}
	tb.variables["DISTANCE"] = BASICValue{NumValue: 25, IsNumeric: true}
	tb.variables["COUNT"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["TOTAL"] = BASICValue{NumValue: 100, IsNumeric: true}
	tb.variables["ITEMS"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["INDEX"] = BASICValue{NumValue: 7, IsNumeric: true}
	tb.variables["SIZE"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["DONE"] = BASICValue{NumValue: 0, IsNumeric: true}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, expr := range expressions {
			_, err := tb.evalExpression(expr)
			if err != nil {
				b.Fatalf("Expression evaluation failed for %s: %v", expr, err)
			}
		}
	}
}

// Test cache functionality
func TestExpressionTokenCache_Basic(t *testing.T) {
	tb := NewTinyBASIC(nil)
	
	// Test that cache is working
	tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	tb.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}
	
	expr := "A + B * 2"
	
	// First evaluation should be a cache miss
	result1, err := tb.evalExpression(expr)
	if err != nil {
		t.Fatalf("First evaluation failed: %v", err)
	}
	
	// Second evaluation should be a cache hit
	result2, err := tb.evalExpression(expr)
	if err != nil {
		t.Fatalf("Second evaluation failed: %v", err)
	}
	
	// Results should be the same
	if result1.NumValue != result2.NumValue {
		t.Errorf("Results differ: %f vs %f", result1.NumValue, result2.NumValue)
	}
	
	// Check cache statistics
	stats := tb.exprTokenCache.GetStats()
	if stats.Hits == 0 {
		t.Errorf("Expected cache hits, got %d", stats.Hits)
	}
	if stats.Misses == 0 {
		t.Errorf("Expected cache misses, got %d", stats.Misses)
	}
}

func TestExpressionTokenCache_ClearCache(t *testing.T) {
	tb := NewTinyBASIC(nil)
	
	// Add some expressions to cache
	tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	for i := 0; i < 5; i++ {
		expr := fmt.Sprintf("A + %d", i)
		_, err := tb.evalExpression(expr)
		if err != nil {
			t.Fatalf("Expression evaluation failed: %v", err)
		}
	}
	
	// Verify cache has entries
	stats := tb.exprTokenCache.GetStats()
	if stats.Size == 0 {
		t.Errorf("Expected cache entries, got %d", stats.Size)
	}
	
	// Clear cache
	tb.exprTokenCache.Clear()
	
	// Verify cache is empty
	stats = tb.exprTokenCache.GetStats()
	if stats.Size != 0 {
		t.Errorf("Expected empty cache, got %d entries", stats.Size)
	}
	if stats.Hits != 0 {
		t.Errorf("Expected zero hits after clear, got %d", stats.Hits)
	}
}

func TestExpressionTokenCache_MaxSize(t *testing.T) {
	tb := NewTinyBASIC(nil)
	
	// Set small cache size
	tb.exprTokenCache.SetMaxSize(3)
	
	// Add more expressions than cache size
	tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	for i := 0; i < 5; i++ {
		expr := fmt.Sprintf("A + %d", i)
		_, err := tb.evalExpression(expr)
		if err != nil {
			t.Fatalf("Expression evaluation failed: %v", err)
		}
	}
	
	// Cache should not exceed max size
	stats := tb.exprTokenCache.GetStats()
	if stats.Size > 3 {
		t.Errorf("Cache size exceeded limit: %d > 3", stats.Size)
	}
	
	// Should have evictions
	if stats.Evictions == 0 {
		t.Errorf("Expected evictions when cache is full, got %d", stats.Evictions)
	}
}

// Benchmark comparison: with vs without caching
func BenchmarkComparison_WithVsWithoutCache(b *testing.B) {
	b.Run("WithCache", func(b *testing.B) {
		tb := NewTinyBASIC(nil)
		tb.exprTokenCache.SetMaxSize(1000) // Enable cache
		
		expr := "A + B * C - D / E + SIN(F)"
		tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
		tb.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}
		tb.variables["C"] = BASICValue{NumValue: 3, IsNumeric: true}
		tb.variables["D"] = BASICValue{NumValue: 8, IsNumeric: true}
		tb.variables["E"] = BASICValue{NumValue: 2, IsNumeric: true}
		tb.variables["F"] = BASICValue{NumValue: 1.57, IsNumeric: true}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := tb.evalExpression(expr)
			if err != nil {
				b.Fatalf("Expression evaluation failed: %v", err)
			}
		}
	})
	
	b.Run("WithoutCache", func(b *testing.B) {
		tb := NewTinyBASIC(nil)
		tb.exprTokenCache.SetMaxSize(0) // Disable cache
		
		expr := "A + B * C - D / E + SIN(F)"
		tb.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
		tb.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}
		tb.variables["C"] = BASICValue{NumValue: 3, IsNumeric: true}
		tb.variables["D"] = BASICValue{NumValue: 8, IsNumeric: true}
		tb.variables["E"] = BASICValue{NumValue: 2, IsNumeric: true}
		tb.variables["F"] = BASICValue{NumValue: 1.57, IsNumeric: true}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := tb.evalExpression(expr)
			if err != nil {
				b.Fatalf("Expression evaluation failed: %v", err)
			}
		}
		
		// Reset cache size for other tests
		tb.exprTokenCache.SetMaxSize(1000)
	})
}