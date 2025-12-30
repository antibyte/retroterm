package tinybasic

import (
	"context"
	"testing"
)

func TestVM_ForLoop_Debug_WithLET(t *testing.T) {
	// 1. Setup minimal environment
	machine := NewTinyBASIC(nil)
	machine.useBytecode = true

	// 2. Load a program that does a tight loop
	machine.Execute("10 FOR I = 1 TO 1000")
	machine.Execute("20 LET A = A + 1")
	machine.Execute("30 NEXT I")

	// 3. Compile it
	err := machine.compileProgramIfNeeded()
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	// 4. Run loop once
	err = machine.bytecodeVM.Run(context.Background())
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}
}
