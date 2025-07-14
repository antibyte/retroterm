package tinybasic

import (
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
)

// EnableBytecode enables or disables bytecode execution
func (b *TinyBASIC) EnableBytecode(enabled bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.useBytecode = enabled
	
	// If disabling, clear compiled program
	if !enabled {
		b.compiledProgram = nil
		b.compiledHash = ""
	}
}

// IsBytecodeEnabled returns whether bytecode execution is enabled
func (b *TinyBASIC) IsBytecodeEnabled() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.useBytecode
}

// compileProgramIfNeeded compiles the current program to bytecode if needed
func (b *TinyBASIC) compileProgramIfNeeded() error {
	if !b.useBytecode {
		return nil
	}
	
	// Calculate hash of current program
	currentHash := b.calculateProgramHash()
	
	// Check if we need to recompile
	if b.compiledProgram != nil && b.compiledHash == currentHash {
		// Program hasn't changed, use existing compilation
		return nil
	}
	
	// Need to compile or recompile
	tinyBasicDebugLog("Compiling program to bytecode (hash: %s)", currentHash)
	
	compiler := NewBytecodeCompiler()
	compiledProgram, err := compiler.CompileProgram(b.program, b.programLines)
	if err != nil {
		tinyBasicDebugLog("Bytecode compilation failed: %v", err)
		// Don't disable bytecode permanently, just return error for this attempt
		return fmt.Errorf("bytecode compilation failed: %v", err)
	}
	
	// Update compiled program and hash
	b.compiledProgram = compiledProgram
	b.compiledHash = currentHash
	
	// Load program into VM
	b.bytecodeVM.LoadProgram(compiledProgram)
	
	tinyBasicDebugLog("Program compiled successfully: %d instructions", len(compiledProgram.Instructions))
	return nil
}

// calculateProgramHash calculates a hash of the current program for caching
func (b *TinyBASIC) calculateProgramHash() string {
	// Sort line numbers for consistent hashing
	lines := make([]int, 0, len(b.programLines))
	for _, lineNum := range b.programLines {
		lines = append(lines, lineNum)
	}
	sort.Ints(lines)
	
	// Build concatenated program text
	var programText strings.Builder
	for _, lineNum := range lines {
		if code, exists := b.program[lineNum]; exists {
			programText.WriteString(fmt.Sprintf("%d:%s\n", lineNum, code))
		}
	}
	
	// Calculate MD5 hash
	hash := md5.Sum([]byte(programText.String()))
	return fmt.Sprintf("%x", hash)
}

// runBytecodeInternal runs the compiled bytecode program internally
func (b *TinyBASIC) runBytecodeInternal() error {
	if b.compiledProgram == nil {
		return fmt.Errorf("no compiled program available")
	}
	
	// Reset VM state
	b.bytecodeVM.Reset()
	
	// Copy current variables to VM
	for varName, value := range b.variables {
		b.bytecodeVM.variables[varName] = value
	}
	
	// Run the bytecode program
	err := b.bytecodeVM.Run(b.ctx)
	
	// Copy variables back from VM
	for varName, value := range b.bytecodeVM.GetVariables() {
		b.variables[varName] = value
	}
	
	return err
}

// invalidateCompiledProgram marks the compiled program as invalid
func (b *TinyBASIC) invalidateCompiledProgram() {
	b.compiledProgram = nil
	b.compiledHash = ""
}

// GetBytecodeStats returns statistics about bytecode compilation and execution
func (b *TinyBASIC) GetBytecodeStats() map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	stats := map[string]interface{}{
		"bytecode_enabled": b.useBytecode,
		"program_compiled": b.compiledProgram != nil,
		"program_hash":     b.compiledHash,
	}
	
	if b.compiledProgram != nil {
		stats["instruction_count"] = len(b.compiledProgram.Instructions)
		stats["constant_count"] = len(b.compiledProgram.Constants)
		stats["label_count"] = len(b.compiledProgram.Labels)
	}
	
	if b.bytecodeVM != nil {
		stats["vm_running"] = b.bytecodeVM.IsRunning()
		stats["vm_pc"] = b.bytecodeVM.GetPC()
	}
	
	return stats
}

// debugPrintBytecode prints the compiled bytecode for debugging
func (b *TinyBASIC) debugPrintBytecode() {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if b.compiledProgram == nil {
		tinyBasicDebugLog("No compiled program available")
		return
	}
	
	tinyBasicDebugLog("=== COMPILED BYTECODE ===")
	tinyBasicDebugLog("Constants: %v", b.compiledProgram.Constants)
	tinyBasicDebugLog("Labels: %v", b.compiledProgram.Labels)
	tinyBasicDebugLog("Instructions:")
	
	for i, inst := range b.compiledProgram.Instructions {
		label := ""
		for lineNum, addr := range b.compiledProgram.Labels {
			if addr == i {
				label = fmt.Sprintf("L%d: ", lineNum)
				break
			}
		}
		
		tinyBasicDebugLog("%s%04d: %s", label, i, inst.String())
	}
	
	tinyBasicDebugLog("=== END BYTECODE ===")
}

// CompileToBytecode manually compiles the current program (for debugging/testing)
func (b *TinyBASIC) CompileToBytecode() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if !b.useBytecode {
		return fmt.Errorf("bytecode execution is disabled")
	}
	
	// Force recompilation
	b.invalidateCompiledProgram()
	return b.compileProgramIfNeeded()
}

// SetBytecodeDebug enables/disables bytecode debugging output
func (b *TinyBASIC) SetBytecodeDebug(enabled bool) {
	// This could be used to control debug output level
	// For now, just print current bytecode if enabled
	if enabled {
		b.debugPrintBytecode()
	}
}