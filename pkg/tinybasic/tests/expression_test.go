package tinybasic_test

import (
	"strings"
	"testing"
	"time"

	"github.com/antibyte/retroterm/pkg/shared"
	"github.com/antibyte/retroterm/pkg/tinybasic"
)

// NewTestBasic creates a TinyBASIC instance for testing without external dependencies
func NewTestBasic() *tinybasic.TinyBASIC {
	// Create a TinyBASIC instance with nil OS (for testing only)
	basic := tinybasic.NewTinyBASIC(nil)

	// Set session ID for testing
	basic.SetSessionID("test-session")

	return basic
}

// Helper function to execute a command and get the first message as text
func executeAndGetOutput(basic *tinybasic.TinyBASIC, command string) string {
	messages := basic.Execute(command)
	if len(messages) > 0 && messages[0].Type == shared.MessageTypeText {
		return messages[0].Content
	}
	return ""
}

// Helper function to collect all text output from a command
func executeAndGetAllOutput(basic *tinybasic.TinyBASIC, command string) string {
	messages := basic.Execute(command)
	var result strings.Builder
	for _, msg := range messages {
		if msg.Type == shared.MessageTypeText {
			result.WriteString(msg.Content)
			result.WriteString(" ")
		}
	}
	return strings.TrimSpace(result.String())
}

// Helper function to run a program and collect output from OutputChan
func runProgramAndGetOutput(basic *tinybasic.TinyBASIC) string {
	// Start the program
	basic.Execute("RUN")

	// Collect output for a reasonable amount of time
	var result strings.Builder
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case msg, ok := <-basic.GetOutputChannel():
			if !ok {
				// Channel closed
				return strings.TrimSpace(result.String())
			}
			if msg.Type == shared.MessageTypeText {
				result.WriteString(msg.Content)
				result.WriteString(" ")
			}
			// If program ended, break
			if !basic.IsRunning() && result.Len() > 0 {
				// Give a small buffer to catch final messages
				finalTimeout := time.NewTimer(200 * time.Millisecond)
			finalLoop:
				for {
					select {
					case msg2, ok2 := <-basic.GetOutputChannel():
						if !ok2 {
							break finalLoop
						}
						if msg2.Type == shared.MessageTypeText {
							result.WriteString(msg2.Content)
							result.WriteString(" ")
						}
					case <-finalTimeout.C:
						break finalLoop
					}
				}
				finalTimeout.Stop()
				return strings.TrimSpace(result.String())
			}
		case <-timeout.C:
			// Timeout reached
			return strings.TrimSpace(result.String())
		}
	}
}

func TestBasicArithmetic(t *testing.T) {
	basic := NewTestBasic()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "Simple PRINT number",
			command:  "PRINT 5",
			expected: "5",
		},
		{
			name:     "Addition",
			command:  "PRINT 2 + 3",
			expected: "5",
		},
		{
			name:     "Subtraction",
			command:  "PRINT 10 - 3",
			expected: "7",
		},
		{
			name:     "Multiplication",
			command:  "PRINT 4 * 3",
			expected: "12",
		},
		{
			name:     "Division",
			command:  "PRINT 15 / 3",
			expected: "5",
		},
		{
			name:     "Parentheses",
			command:  "PRINT (2 + 3) * 4",
			expected: "20",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeAndGetOutput(basic, tt.command)
			if strings.TrimSpace(output) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, strings.TrimSpace(output))
			}
		})
	}
}

func TestStringOperations(t *testing.T) {
	basic := NewTestBasic()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "Simple string",
			command:  "PRINT \"hello\"",
			expected: "hello",
		},
		{
			name:     "Empty string",
			command:  "PRINT \"\"",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeAndGetOutput(basic, tt.command)
			if strings.TrimSpace(output) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, strings.TrimSpace(output))
			}
		})
	}
}

func TestVariableOperations(t *testing.T) {
	basic := NewTestBasic()

	// Set a numeric variable
	basic.Execute("LET X = 42")
	output := executeAndGetOutput(basic, "PRINT X")
	if strings.TrimSpace(output) != "42" {
		t.Errorf("Expected X=42, got %q", strings.TrimSpace(output))
	}

	// Set a string variable
	basic.Execute("LET NAME$ = \"test\"")
	output = executeAndGetOutput(basic, "PRINT NAME$")
	if strings.TrimSpace(output) != "test" {
		t.Errorf("Expected NAME$=\"test\", got %q", strings.TrimSpace(output))
	}

	// Test expression with variables
	basic.Execute("LET A = 10")
	basic.Execute("LET B = 5")
	output = executeAndGetOutput(basic, "PRINT A + B")
	if strings.TrimSpace(output) != "15" {
		t.Errorf("Expected A + B = 15, got %q", strings.TrimSpace(output))
	}
}

func TestErrorHandling(t *testing.T) {
	basic := NewTestBasic()

	// Test division by zero
	messages := basic.Execute("PRINT 5 / 0")
	if len(messages) == 0 {
		t.Error("Expected error message for division by zero")
		return
	}
	// Check if any message contains error information
	hasError := false
	for _, msg := range messages {
		if strings.Contains(strings.ToLower(msg.Content), "error") || strings.Contains(strings.ToLower(msg.Content), "division") {
			hasError = true
			break
		}
	}

	if !hasError {
		t.Error("Expected error message for division by zero, but got none")
	}
}

func TestProgramExecution(t *testing.T) {
	basic := NewTestBasic()

	// Load a simple program
	basic.Execute("10 PRINT \"Line 10\"")
	basic.Execute("20 PRINT \"Line 20\"")
	basic.Execute("30 END")

	// Check if the program was loaded
	messages := basic.Execute("LIST")
	if len(messages) == 0 {
		t.Error("Expected program listing, but got no output")
		return
	}
	// The LIST command should show our program lines
	listOutput := ""
	for _, msg := range messages {
		if msg.Type == shared.MessageTypeText {
			listOutput += msg.Content
		}
	}

	if !strings.Contains(listOutput, "10") || !strings.Contains(listOutput, "20") || !strings.Contains(listOutput, "30") {
		t.Errorf("Expected program lines in LIST output, got: %q", listOutput)
	}
}

// Additional test functions for more comprehensive coverage

func TestBasicFunctions(t *testing.T) {
	basic := NewTestBasic()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "ABS function - positive",
			command:  "PRINT ABS(5)",
			expected: "5",
		},
		{
			name:     "ABS function - negative",
			command:  "PRINT ABS(-5)",
			expected: "5",
		},
		{
			name:     "RND function returns numeric",
			command:  "PRINT RND(1)",
			expected: "", // We can't predict random, but should not error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeAndGetOutput(basic, tt.command)
			if tt.expected != "" && strings.TrimSpace(output) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, strings.TrimSpace(output))
			}
			// For RND, just check that we got some numeric output
			if tt.name == "RND function returns numeric" && output == "" {
				t.Error("Expected numeric output from RND function")
			}
		})
	}
}

func TestLoopCommands(t *testing.T) {
	basic := NewTestBasic()

	// Load a simple FOR loop program
	basic.Execute("NEW") // Clear any existing program
	basic.Execute("10 FOR I = 1 TO 3")
	basic.Execute("20 PRINT I")
	basic.Execute("30 NEXT I")
	basic.Execute("40 END")

	// Run the program and collect output from OutputChan
	textOutput := runProgramAndGetOutput(basic)

	// Should contain the numbers 1, 2, 3 somewhere in the output
	if !strings.Contains(textOutput, "1") || !strings.Contains(textOutput, "2") || !strings.Contains(textOutput, "3") {
		t.Errorf("Expected FOR loop to print 1, 2, 3, got output: %q", textOutput)
	}
}

func TestConditionalCommands(t *testing.T) {
	basic := NewTestBasic()

	// Test IF-THEN
	basic.Execute("NEW")
	basic.Execute("10 LET X = 5")
	basic.Execute("20 IF X > 3 THEN PRINT \"Greater\"")
	basic.Execute("30 IF X < 3 THEN PRINT \"Lesser\"")
	basic.Execute("40 END")

	textOutput := runProgramAndGetOutput(basic)

	// Should print "Greater" but not "Lesser"
	if !strings.Contains(textOutput, "Greater") {
		t.Errorf("Expected IF-THEN to print 'Greater', got output: %q", textOutput)
	}
	if strings.Contains(textOutput, "Lesser") {
		t.Errorf("Expected IF-THEN not to print 'Lesser', got output: %q", textOutput)
	}
}

func TestInputOutput(t *testing.T) {
	basic := NewTestBasic()

	// Test basic PRINT with semicolon (should stay on same line)
	output1 := executeAndGetOutput(basic, "PRINT \"Hello\";")
	output2 := executeAndGetOutput(basic, "PRINT \"World\"")

	// Test that both commands executed without error
	if output1 == "" && output2 == "" {
		t.Error("Expected output from PRINT commands")
	}
}

func TestProgramManagement(t *testing.T) {
	basic := NewTestBasic()

	// Test NEW command
	basic.Execute("10 PRINT \"Test\"")
	basic.Execute("NEW")

	// After NEW, LIST should show empty program
	listOutput := executeAndGetAllOutput(basic, "LIST")

	// Should not contain line 10 anymore
	if strings.Contains(listOutput, "10") {
		t.Errorf("Expected empty program after NEW, but LIST shows: %q", listOutput)
	}
}
