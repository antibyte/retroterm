package tinybasic

import (
	"context"
	"fmt"
	"time"

	"github.com/antibyte/retroterm/pkg/shared"
)

// runBytecodeProgram executes the compiled bytecode program
func (b *TinyBASIC) runBytecodeProgram() {
	defer func() {
		b.mu.Lock()
		b.running = false
		wasEnableSent := b.inputControlEnableSent
		callback := b.onProgramEnd // Get callback reference before unlocking
		b.mu.Unlock()

		// Stop any playing SID music when program execution ends
		musicStopMsg := shared.Message{
			Type: shared.MessageTypeSound,
			Params: map[string]interface{}{
				"action": "music_stop",
			},
		}
		b.sendMessageObject(musicStopMsg)

		// Reaktiviere Eingabe nach Programmende - nur wenn noch nicht gesendet
		if !wasEnableSent {
			b.sendInputControl("enable")
		}
		// Send OK when program execution is complete
		b.sendMessageWrapped(shared.MessageTypeText, "OK")
		// Call the callback if set (for autorun mode to return to TinyOS)
		if callback != nil {
			callback()
			// Clear callback after execution to prevent multiple calls
			b.mu.Lock()
			b.onProgramEnd = nil
			b.mu.Unlock()
		}
	}()

	// Ensure we have a compiled program
	b.mu.Lock()
	if b.compiledProgram == nil {
		b.mu.Unlock()
		tinyBasicDebugLog("[BYTECODE-RUNNER] No compiled program available")
		b.sendMessageWrapped(shared.MessageTypeText, "BYTECODE COMPILATION ERROR")
		return
	}

	tinyBasicDebugLog("[BYTECODE-RUNNER] Loading compiled program with %d instructions", len(b.compiledProgram.Instructions))

	// Debug: Print first 10 instructions to see what was compiled
	tinyBasicDebugLog("[BYTECODE-RUNNER] First 10 instructions:")
	for i := 0; i < 10 && i < len(b.compiledProgram.Instructions); i++ {
		inst := b.compiledProgram.Instructions[i]
		tinyBasicDebugLog("[BYTECODE-RUNNER] %d: OpCode=%d (%s)", i, int(inst.OpCode), inst.String())
	}

	b.bytecodeVM.LoadProgram(b.compiledProgram)

	// Copy variables from interpreter to VM
	varCount := 0
	for varName, value := range b.variables {
		b.bytecodeVM.variables[varName] = value
		varCount++
	}
	tinyBasicDebugLog("[BYTECODE-RUNNER] Copied %d variables to VM", varCount)
	b.mu.Unlock()

	// Execute the bytecode program
	tinyBasicDebugLog("[BYTECODE-RUNNER] Starting bytecode execution")
	err := b.bytecodeVM.Run(b.ctx)

	// Copy variables back from VM to interpreter
	b.mu.Lock()
	vmVariables := b.bytecodeVM.GetVariables()
	for varName, value := range vmVariables {
		b.variables[varName] = value
	}
	b.mu.Unlock()

	if err != nil {
		// Handle bytecode execution errors
		if err == context.Canceled {
			b.sendMessageWrapped(shared.MessageTypeText, "EXECUTION CANCELLED")
		} else {
			tinyBasicDebugLog("Bytecode execution error: %v", err)
			b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("RUNTIME ERROR: %v", err))
		}
	}
}

// runBytecodeWithFallback attempts bytecode execution and falls back to interpreted if needed
func (b *TinyBASIC) runBytecodeWithFallback() {
	// First try to compile to bytecode
	b.mu.Lock()
	compileErr := b.compileProgramIfNeeded()
	canUseBytecode := compileErr == nil && b.compiledProgram != nil
	b.mu.Unlock()

	if canUseBytecode {
		tinyBasicDebugLog("Running program with bytecode")

		// Try bytecode execution
		b.runBytecodeProgram()

		// Check if bytecode execution completed successfully
		b.mu.Lock()
		vmWasRunning := b.bytecodeVM.IsRunning()
		b.mu.Unlock()

		if !vmWasRunning {
			// Bytecode execution completed (successfully or with error)
			return
		}
	}

	// Fall back to interpreted execution
	tinyBasicDebugLog("Falling back to interpreted execution")
	b.runProgramInternal(b.ctx)
}

// Performance comparison utilities

// BenchmarkExecutionMode represents different execution modes for performance testing
type BenchmarkExecutionMode int

const (
	BenchmarkInterpreted BenchmarkExecutionMode = iota
	BenchmarkBytecode
	BenchmarkBoth
)

// BenchmarkResult contains performance measurement results
type BenchmarkResult struct {
	Mode             string
	ExecutionTime    time.Duration
	InstructionCount int64
	Error            error
}

// BenchmarkExecution runs performance comparison between interpreted and bytecode execution
func (b *TinyBASIC) BenchmarkExecution(mode BenchmarkExecutionMode, iterations int) []BenchmarkResult {
	results := make([]BenchmarkResult, 0)

	if mode == BenchmarkInterpreted || mode == BenchmarkBoth {
		result := b.benchmarkInterpreted(iterations)
		results = append(results, result)
	}

	if mode == BenchmarkBytecode || mode == BenchmarkBoth {
		result := b.benchmarkBytecode(iterations)
		results = append(results, result)
	}

	return results
}

// benchmarkInterpreted measures performance of interpreted execution
func (b *TinyBASIC) benchmarkInterpreted(iterations int) BenchmarkResult {
	b.mu.Lock()
	// Temporarily disable bytecode
	originalUseBytecode := b.useBytecode
	b.useBytecode = false
	b.mu.Unlock()

	start := time.Now()
	var lastErr error

	for i := 0; i < iterations; i++ {
		// Reset state for each iteration
		b.ResetExecutionState()

		// Run program synchronously for benchmarking
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := b.runProgramSynchronous(ctx)
		cancel()

		if err != nil {
			lastErr = err
			break
		}
	}

	duration := time.Since(start)

	// Restore original bytecode setting
	b.mu.Lock()
	b.useBytecode = originalUseBytecode
	b.mu.Unlock()

	return BenchmarkResult{
		Mode:          "Interpreted",
		ExecutionTime: duration,
		Error:         lastErr,
	}
}

// benchmarkBytecode measures performance of bytecode execution
func (b *TinyBASIC) benchmarkBytecode(iterations int) BenchmarkResult {
	b.mu.Lock()
	// Ensure bytecode is enabled
	b.useBytecode = true

	// Compile program once
	compileErr := b.compileProgramIfNeeded()
	if compileErr != nil {
		b.mu.Unlock()
		return BenchmarkResult{
			Mode:  "Bytecode",
			Error: fmt.Errorf("compilation failed: %v", compileErr),
		}
	}
	b.mu.Unlock()

	start := time.Now()
	var lastErr error

	for i := 0; i < iterations; i++ {
		// Reset VM state for each iteration
		b.bytecodeVM.Reset()

		// Copy variables to VM
		b.mu.Lock()
		for varName, value := range b.variables {
			b.bytecodeVM.variables[varName] = value
		}
		b.mu.Unlock()

		// Run bytecode synchronously for benchmarking
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := b.bytecodeVM.Run(ctx)
		cancel()

		if err != nil {
			lastErr = err
			break
		}
	}

	duration := time.Since(start)

	return BenchmarkResult{
		Mode:          "Bytecode",
		ExecutionTime: duration,
		Error:         lastErr,
	}
}

// runProgramSynchronous runs the program synchronously (for benchmarking)
func (b *TinyBASIC) runProgramSynchronous(ctx context.Context) error {
	b.mu.Lock()
	if len(b.program) == 0 {
		b.mu.Unlock()
		return fmt.Errorf("no program loaded")
	}

	b.rebuildProgramLines()
	if len(b.programLines) == 0 {
		b.mu.Unlock()
		return fmt.Errorf("no program lines")
	}

	b.rebuildData()
	b.currentLine = b.programLines[0]
	b.running = true
	b.mu.Unlock()

	// Run the main execution loop synchronously
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		b.mu.Lock()
		if !b.running || b.currentLine == 0 {
			b.mu.Unlock()
			break
		}

		currentLine := b.currentLine
		code, ok := b.program[currentLine]
		b.mu.Unlock()

		if !ok {
			break
		}

		nextLine, err := b.executeStatement(code, ctx)
		if err != nil {
			return err
		}

		if nextLine == 0 {
			break
		}

		b.mu.Lock()
		_ = b.currentLine // originalLine was unused

		if b.currentLine != currentLine {
			// Command changed current line (GOTO, etc.)
			// Keep the new line
		} else if nextLine != currentLine {
			// executeStatement returned different line
			b.currentLine = nextLine
		} else {
			// Normal execution: advance to next line
			nextLineFound, found := b.findNextLine(currentLine)
			if found {
				b.currentLine = nextLineFound
			} else {
				b.currentLine = 0 // End of program
			}
		}
		b.mu.Unlock()
	}

	return nil
}

// GetExecutionStats returns current execution statistics with enhanced performance data
func (b *TinyBASIC) GetExecutionStats() map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()

	stats := map[string]interface{}{
		"current_line":   b.currentLine,
		"running":        b.running,
		"use_bytecode":   b.useBytecode,
		"program_lines":  len(b.programLines),
		"variable_count": len(b.variables),
	}

	if b.bytecodeVM != nil {
		stats["vm_running"] = b.bytecodeVM.IsRunning()
		stats["vm_pc"] = b.bytecodeVM.GetPC()
		
		// Add comprehensive performance statistics
		performanceStats := b.bytecodeVM.GetPerformanceStats()
		for k, v := range performanceStats {
			stats["vm_"+k] = v
		}
	}

	if b.compiledProgram != nil {
		stats["bytecode_instructions"] = len(b.compiledProgram.Instructions)
		stats["bytecode_constants"] = len(b.compiledProgram.Constants)
		stats["compiled_hash"] = b.compiledHash
	}

	return stats
}
