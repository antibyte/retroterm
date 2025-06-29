package tinybasic

import (
	"context"
	"testing"

	"github.com/antibyte/retroterm/pkg/shared"
)

// NewTestBasic creates a TinyBASIC instance for testing without external dependencies
func NewTestBasic() *TinyBASIC {
	ctx, cancel := context.WithCancel(context.Background())

	b := &TinyBASIC{
		os:           nil, // No OS dependency for basic tests
		fs:           nil, // No file system dependency for basic tests
		program:      make(map[int]string),
		variables:    make(map[string]BASICValue),
		programLines: make([]int, 0),
		openFiles:    make(map[int]*OpenFile),
		forLoops:     make([]ForLoopInfo, 0),
		gosubStack:   make([]int, 0),
		data:         make([]string, 0),
		OutputChan:   make(chan shared.Message, 100), // Buffered channel for tests
		termCols:     80,
		termRows:     24,
		ctx:          ctx,
		cancel:       cancel,
		sessionID:    "test-session",
	}

	return b
}

// TestEvalExpression tests the expression evaluation engine
func TestEvalExpression(t *testing.T) {
	basic := NewTestBasic()

	tests := []struct {
		name     string
		expr     string
		expected BASICValue
		hasError bool
	}{
		// Numeric expressions
		{
			name:     "simple addition",
			expr:     "2 + 3",
			expected: BASICValue{NumValue: 5, IsNumeric: true},
			hasError: false,
		},
		{
			name:     "multiplication with parentheses",
			expr:     "2 * (3 + 4)",
			expected: BASICValue{NumValue: 14, IsNumeric: true},
			hasError: false,
		},
		{
			name:     "division",
			expr:     "10 / 2",
			expected: BASICValue{NumValue: 5, IsNumeric: true},
			hasError: false,
		},
		{
			name:     "negative number",
			expr:     "-5",
			expected: BASICValue{NumValue: -5, IsNumeric: true},
			hasError: false,
		},
		// String expressions
		{
			name:     "simple string",
			expr:     "\"hello\"",
			expected: BASICValue{StrValue: "hello", IsNumeric: false},
			hasError: false,
		},
		{
			name:     "empty string",
			expr:     "\"\"",
			expected: BASICValue{StrValue: "", IsNumeric: false},
			hasError: false,
		},
		// Error cases
		{
			name:     "division by zero",
			expr:     "10 / 0",
			expected: BASICValue{},
			hasError: true,
		},
		{
			name:     "invalid syntax",
			expr:     "2 +",
			expected: BASICValue{},
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := basic.evalExpression(test.expr)

			if test.hasError {
				if err == nil {
					t.Errorf("Expected error for expression '%s', but got none", test.expr)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for expression '%s': %v", test.expr, err)
				return
			}

			if result.IsNumeric != test.expected.IsNumeric {
				t.Errorf("For expression '%s': expected IsNumeric=%v, got %v",
					test.expr, test.expected.IsNumeric, result.IsNumeric)
				return
			}

			if result.IsNumeric {
				if result.NumValue != test.expected.NumValue {
					t.Errorf("For expression '%s': expected NumValue=%v, got %v",
						test.expr, test.expected.NumValue, result.NumValue)
				}
			} else {
				if result.StrValue != test.expected.StrValue {
					t.Errorf("For expression '%s': expected StrValue=%q, got %q",
						test.expr, test.expected.StrValue, result.StrValue)
				}
			}
		})
	}
}

// TestVariables tests variable assignment and retrieval
func TestVariables(t *testing.T) {
	basic := NewTestBasic()

	// Test numeric variable
	basic.variables["X"] = BASICValue{NumValue: 42, IsNumeric: true}
	result, err := basic.evalExpression("X")
	if err != nil {
		t.Errorf("Error evaluating variable X: %v", err)
	}
	if !result.IsNumeric || result.NumValue != 42 {
		t.Errorf("Expected X=42, got %v", result)
	}

	// Test string variable
	basic.variables["NAME"] = BASICValue{StrValue: "test", IsNumeric: false}
	result, err = basic.evalExpression("NAME")
	if err != nil {
		t.Errorf("Error evaluating variable NAME: %v", err)
	}
	if result.IsNumeric || result.StrValue != "test" {
		t.Errorf("Expected NAME=\"test\", got %v", result)
	}

	// Test undefined variable
	_, err = basic.evalExpression("UNDEFINED")
	if err == nil {
		t.Error("Expected error for undefined variable, but got none")
	}
}

// TestExpressionWithVariables tests expressions that include variables
func TestExpressionWithVariables(t *testing.T) {
	basic := NewTestBasic()

	// Set up variables
	basic.variables["A"] = BASICValue{NumValue: 10, IsNumeric: true}
	basic.variables["B"] = BASICValue{NumValue: 5, IsNumeric: true}

	tests := []struct {
		name     string
		expr     string
		expected float64
	}{
		{"variable addition", "A + B", 15},
		{"variable with constant", "A + 2", 12},
		{"complex expression", "A * B + 2", 52},
		{"parentheses with variables", "(A + B) * 2", 30},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := basic.evalExpression(test.expr)
			if err != nil {
				t.Errorf("Error evaluating '%s': %v", test.expr, err)
				return
			}

			if !result.IsNumeric {
				t.Errorf("Expected numeric result for '%s', got string", test.expr)
				return
			}

			if result.NumValue != test.expected {
				t.Errorf("For expression '%s': expected %v, got %v",
					test.expr, test.expected, result.NumValue)
			}
		})
	}
}
