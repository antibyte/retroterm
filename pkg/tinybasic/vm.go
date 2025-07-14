package tinybasic

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/shared"
)

// BytecodeVM represents the virtual machine for executing bytecode
type BytecodeVM struct {
	tinybasic    *TinyBASIC          // Reference to TinyBASIC interpreter
	program      *BytecodeProgram    // Compiled bytecode program
	pc           int                 // Program counter (instruction pointer)
	stack        *VMStack            // Execution stack
	callStack    []int               // Call stack for GOSUB/RETURN
	forLoops     []VMForLoop         // FOR loop stack
	variables    map[string]BASICValue // Variable storage
	running      bool                // Execution state
	ctx          context.Context     // Execution context
}

// VMForLoop represents a FOR loop in the virtual machine
type VMForLoop struct {
	Variable   string      // Loop variable name
	Current    BASICValue  // Current value
	End        BASICValue  // End value
	Step       BASICValue  // Step value
	StartPC    int         // Start instruction address
	NextPC     int         // Address after FOR statement
}

// NewBytecodeVM creates a new virtual machine instance
func NewBytecodeVM(tb *TinyBASIC) *BytecodeVM {
	return &BytecodeVM{
		tinybasic: tb,
		pc:        0,
		stack:     NewVMStack(1000), // 1000 element stack
		callStack: make([]int, 0, 100), // 100 deep call stack
		forLoops:  make([]VMForLoop, 0, 50), // 50 deep FOR loops
		variables: make(map[string]BASICValue),
		running:   false,
	}
}

// LoadProgram loads a compiled bytecode program
func (vm *BytecodeVM) LoadProgram(program *BytecodeProgram) {
	vm.program = program
	vm.pc = 0
	vm.running = false
}

// Reset resets the VM state
func (vm *BytecodeVM) Reset() {
	vm.pc = 0
	vm.stack.Clear()
	vm.callStack = vm.callStack[:0]
	vm.forLoops = vm.forLoops[:0]
	vm.variables = make(map[string]BASICValue)
	vm.running = false
}

// Run executes the loaded bytecode program
func (vm *BytecodeVM) Run(ctx context.Context) error {
	if vm.program == nil {
		return fmt.Errorf("no program loaded")
	}
	
	vm.ctx = ctx
	vm.running = true
	vm.pc = 0
	
	for vm.running && vm.pc < len(vm.program.Instructions) {
		// Check for cancellation
		select {
		case <-ctx.Done():
			vm.running = false
			return ctx.Err()
		default:
		}
		
		// Execute current instruction
		err := vm.executeInstruction()
		if err != nil {
			vm.running = false
			return err
		}
		
		// Small yield to prevent blocking
		if vm.pc%100 == 0 {
			time.Sleep(0)
		}
	}
	
	return nil
}

// Stop stops VM execution
func (vm *BytecodeVM) Stop() {
	vm.running = false
}

// executeInstruction executes a single bytecode instruction
func (vm *BytecodeVM) executeInstruction() error {
	if vm.pc >= len(vm.program.Instructions) {
		vm.running = false
		return nil
	}
	
	inst := vm.program.Instructions[vm.pc]
	
	switch inst.OpCode {
	// Arithmetic operations
	case OP_ADD:
		return vm.execBinaryOp(func(a, b BASICValue) (BASICValue, error) {
			if a.IsNumeric && b.IsNumeric {
				return newNumericBASICValue(a.NumValue + b.NumValue), nil
			}
			// String concatenation for ADD with strings
			return newStringBASICValue(vm.toString(a) + vm.toString(b)), nil
		})
		
	case OP_SUB:
		return vm.execBinaryOp(func(a, b BASICValue) (BASICValue, error) {
			if !a.IsNumeric || !b.IsNumeric {
				return BASICValue{}, fmt.Errorf("type mismatch in subtraction")
			}
			return newNumericBASICValue(a.NumValue - b.NumValue), nil
		})
		
	case OP_MUL:
		return vm.execBinaryOp(func(a, b BASICValue) (BASICValue, error) {
			if !a.IsNumeric || !b.IsNumeric {
				return BASICValue{}, fmt.Errorf("type mismatch in multiplication")
			}
			return newNumericBASICValue(a.NumValue * b.NumValue), nil
		})
		
	case OP_DIV:
		return vm.execBinaryOp(func(a, b BASICValue) (BASICValue, error) {
			if !a.IsNumeric || !b.IsNumeric {
				return BASICValue{}, fmt.Errorf("type mismatch in division")
			}
			if b.NumValue == 0 {
				return BASICValue{}, fmt.Errorf("division by zero")
			}
			return newNumericBASICValue(a.NumValue / b.NumValue), nil
		})
		
	case OP_MOD:
		return vm.execBinaryOp(func(a, b BASICValue) (BASICValue, error) {
			if !a.IsNumeric || !b.IsNumeric {
				return BASICValue{}, fmt.Errorf("type mismatch in modulo")
			}
			if b.NumValue == 0 {
				return BASICValue{}, fmt.Errorf("division by zero in modulo")
			}
			return newNumericBASICValue(math.Mod(a.NumValue, b.NumValue)), nil
		})
		
	case OP_POW:
		return vm.execBinaryOp(func(a, b BASICValue) (BASICValue, error) {
			if !a.IsNumeric || !b.IsNumeric {
				return BASICValue{}, fmt.Errorf("type mismatch in power")
			}
			return newNumericBASICValue(math.Pow(a.NumValue, b.NumValue)), nil
		})
		
	case OP_NEG:
		return vm.execUnaryOp(func(a BASICValue) (BASICValue, error) {
			if !a.IsNumeric {
				return BASICValue{}, fmt.Errorf("type mismatch in negation")
			}
			return newNumericBASICValue(-a.NumValue), nil
		})
		
	// Comparison operations
	case OP_EQ:
		return vm.execComparison(func(a, b BASICValue) bool {
			if a.IsNumeric && b.IsNumeric {
				return a.NumValue == b.NumValue
			}
			return vm.toString(a) == vm.toString(b)
		})
		
	case OP_NE:
		return vm.execComparison(func(a, b BASICValue) bool {
			if a.IsNumeric && b.IsNumeric {
				return a.NumValue != b.NumValue
			}
			return vm.toString(a) != vm.toString(b)
		})
		
	case OP_LT:
		return vm.execComparison(func(a, b BASICValue) bool {
			if a.IsNumeric && b.IsNumeric {
				return a.NumValue < b.NumValue
			}
			return vm.toString(a) < vm.toString(b)
		})
		
	case OP_LE:
		return vm.execComparison(func(a, b BASICValue) bool {
			if a.IsNumeric && b.IsNumeric {
				return a.NumValue <= b.NumValue
			}
			return vm.toString(a) <= vm.toString(b)
		})
		
	case OP_GT:
		return vm.execComparison(func(a, b BASICValue) bool {
			if a.IsNumeric && b.IsNumeric {
				return a.NumValue > b.NumValue
			}
			return vm.toString(a) > vm.toString(b)
		})
		
	case OP_GE:
		return vm.execComparison(func(a, b BASICValue) bool {
			if a.IsNumeric && b.IsNumeric {
				return a.NumValue >= b.NumValue
			}
			return vm.toString(a) >= vm.toString(b)
		})
		
	// Logical operations
	case OP_AND:
		return vm.execBinaryOp(func(a, b BASICValue) (BASICValue, error) {
			aTrue := vm.isTrue(a)
			bTrue := vm.isTrue(b)
			return newNumericBASICValue(vm.boolToNum(aTrue && bTrue)), nil
		})
		
	case OP_OR:
		return vm.execBinaryOp(func(a, b BASICValue) (BASICValue, error) {
			aTrue := vm.isTrue(a)
			bTrue := vm.isTrue(b)
			return newNumericBASICValue(vm.boolToNum(aTrue || bTrue)), nil
		})
		
	case OP_NOT:
		return vm.execUnaryOp(func(a BASICValue) (BASICValue, error) {
			return newNumericBASICValue(vm.boolToNum(!vm.isTrue(a))), nil
		})
		
	// Stack operations
	case OP_PUSH_NUM:
		constIndex := inst.Operand1.(int)
		value := vm.program.Constants[constIndex]
		if num, ok := value.(float64); ok {
			vm.stack.Push(newNumericBASICValue(num))
		} else if inum, ok := value.(int); ok {
			vm.stack.Push(newNumericBASICValue(float64(inum)))
		} else {
			return fmt.Errorf("invalid numeric constant")
		}
		vm.pc++
		
	case OP_PUSH_STR:
		constIndex := inst.Operand1.(int)
		value := vm.program.Constants[constIndex]
		if str, ok := value.(string); ok {
			vm.stack.Push(newStringBASICValue(str))
		} else {
			return fmt.Errorf("invalid string constant")
		}
		vm.pc++
		
	case OP_LOAD_VAR:
		varName := strings.ToUpper(inst.Operand1.(string))
		if value, exists := vm.variables[varName]; exists {
			// Validate that the stored type matches the expected type
			isStringVar := strings.HasSuffix(varName, "$")
			if isStringVar && value.IsNumeric {
				// Convert numeric to string if accessing string variable
				convertedValue := newStringBASICValue(vm.toString(value))
				vm.variables[varName] = convertedValue // Update stored value
				vm.stack.Push(convertedValue)
			} else if !isStringVar && !value.IsNumeric {
				// Convert string to numeric if accessing numeric variable
				if numVal, err := strconv.ParseFloat(value.StrValue, 64); err == nil {
					convertedValue := newNumericBASICValue(numVal)
					vm.variables[varName] = convertedValue // Update stored value
					vm.stack.Push(convertedValue)
				} else {
					// Invalid conversion - default to 0
					convertedValue := newNumericBASICValue(0)
					vm.variables[varName] = convertedValue
					vm.stack.Push(convertedValue)
				}
			} else {
				vm.stack.Push(value)
			}
		} else {
			// Uninitialized variables default to 0 or empty string based on type
			if strings.HasSuffix(varName, "$") {
				defaultValue := newStringBASICValue("")
				vm.variables[varName] = defaultValue
				vm.stack.Push(defaultValue)
			} else {
				defaultValue := newNumericBASICValue(0)
				vm.variables[varName] = defaultValue
				vm.stack.Push(defaultValue)
			}
		}
		vm.pc++
		
	case OP_STORE_VAR:
		varName := strings.ToUpper(inst.Operand1.(string))
		value, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		// Validate and convert type if necessary
		isStringVar := strings.HasSuffix(varName, "$")
		if isStringVar && value.IsNumeric {
			// Convert numeric to string for string variable
			value = newStringBASICValue(vm.toString(value))
		} else if !isStringVar && !value.IsNumeric {
			// Convert string to numeric for numeric variable
			if numVal, err := strconv.ParseFloat(value.StrValue, 64); err == nil {
				value = newNumericBASICValue(numVal)
			} else {
				// Invalid conversion - store as 0
				value = newNumericBASICValue(0)
			}
		}
		
		vm.variables[varName] = value
		vm.pc++
		
	case OP_POP:
		_, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		vm.pc++
		
	// Control flow
	case OP_JUMP:
		lineNum := inst.Operand1.(int)
		if addr, exists := vm.program.Labels[lineNum]; exists {
			vm.pc = addr
		} else {
			return fmt.Errorf("undefined line number %d", lineNum)
		}
		
	case OP_JUMP_IF:
		condition, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		if vm.isTrue(condition) {
			lineNum := inst.Operand1.(int)
			if addr, exists := vm.program.Labels[lineNum]; exists {
				vm.pc = addr
			} else {
				return fmt.Errorf("undefined line number %d", lineNum)
			}
		} else {
			vm.pc++
		}
		
	case OP_JUMP_UNLESS:
		condition, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		if !vm.isTrue(condition) {
			addr := inst.Operand1.(int)
			vm.pc = addr
		} else {
			vm.pc++
		}
		
	case OP_CALL:
		lineNum := inst.Operand1.(int)
		// Push return address
		vm.callStack = append(vm.callStack, vm.pc+1)
		// Jump to subroutine
		if addr, exists := vm.program.Labels[lineNum]; exists {
			vm.pc = addr
		} else {
			return fmt.Errorf("undefined line number %d", lineNum)
		}
		
	case OP_RETURN:
		if len(vm.callStack) == 0 {
			return fmt.Errorf("RETURN without GOSUB")
		}
		// Pop return address
		returnAddr := vm.callStack[len(vm.callStack)-1]
		vm.callStack = vm.callStack[:len(vm.callStack)-1]
		vm.pc = returnAddr
		
	// FOR loop operations
	case OP_FOR_INIT:
		varName := strings.ToUpper(inst.Operand1.(string))
		
		// Pop step, end values from stack (in reverse order)
		step, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		end, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		// Validate step value
		if step.NumValue == 0 {
			return fmt.Errorf("FOR loop with zero step value at line %d", inst.LineNum)
		}
		
		// Current value is already in the variable
		current := vm.variables[varName]
		
		// Ensure current, end, and step are numeric
		if !current.IsNumeric || !end.IsNumeric || !step.IsNumeric {
			return fmt.Errorf("FOR loop requires numeric values at line %d", inst.LineNum)
		}
		
		// Check if loop should execute at all
		shouldExecute := false
		if step.NumValue > 0 {
			shouldExecute = current.NumValue <= end.NumValue
		} else if step.NumValue < 0 {
			shouldExecute = current.NumValue >= end.NumValue
		}
		
		if shouldExecute {
			// Create FOR loop entry and continue with loop body
			forLoop := VMForLoop{
				Variable: varName,
				Current:  current,
				End:      end,
				Step:     step,
				StartPC:  vm.pc + 1, // Next instruction after FOR_INIT
				NextPC:   vm.pc + 1,
			}
			vm.forLoops = append(vm.forLoops, forLoop)
		} else {
			// Skip loop body - in a real implementation, we'd need to find the matching NEXT
			// For now, this will be handled by the loop termination check
		}
		// Always continue to next instruction (loop body or past loop)
		vm.pc++
		
	case OP_FOR_CHECK:
		if len(vm.forLoops) == 0 {
			return fmt.Errorf("FOR_CHECK without FOR loop")
		}
		
		// Check current FOR loop condition
		loop := &vm.forLoops[len(vm.forLoops)-1]
		current := vm.variables[loop.Variable]
		
		// Determine if loop should continue
		shouldContinue := false
		if loop.Step.NumValue > 0 {
			shouldContinue = current.NumValue <= loop.End.NumValue
		} else {
			shouldContinue = current.NumValue >= loop.End.NumValue
		}
		
		if shouldContinue {
			vm.pc++ // Continue with loop body
		} else {
			// Exit loop - find matching NEXT
			vm.forLoops = vm.forLoops[:len(vm.forLoops)-1]
			// For now, just continue - NEXT will handle the jump
			vm.pc++
		}
		
	case OP_FOR_NEXT:
		varName := ""
		if inst.Operand1 != nil {
			varName = strings.ToUpper(inst.Operand1.(string))
		}
		
		if len(vm.forLoops) == 0 {
			// No active FOR loops - this is an error in BASIC
			return fmt.Errorf("NEXT without FOR at line %d", inst.LineNum)
		}
		
		// Find matching FOR loop
		loopIndex := len(vm.forLoops) - 1
		if varName != "" {
			found := false
			for i := len(vm.forLoops) - 1; i >= 0; i-- {
				if vm.forLoops[i].Variable == varName {
					loopIndex = i
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("NEXT %s without matching FOR at line %d", varName, inst.LineNum)
			}
		}
		
		loop := &vm.forLoops[loopIndex]
		
		// Increment loop variable
		current := vm.variables[loop.Variable]
		if !current.IsNumeric {
			return fmt.Errorf("FOR loop variable %s must be numeric at line %d", loop.Variable, inst.LineNum)
		}
		
		current.NumValue += loop.Step.NumValue
		vm.variables[loop.Variable] = current
		loop.Current = current
		
		// Check if loop should continue with proper step direction handling
		shouldContinue := false
		if loop.Step.NumValue > 0 {
			shouldContinue = current.NumValue <= loop.End.NumValue
		} else if loop.Step.NumValue < 0 {
			shouldContinue = current.NumValue >= loop.End.NumValue
		}
		
		if shouldContinue {
			// Jump back to loop start
			vm.pc = loop.StartPC
		} else {
			// Exit loop - remove all nested loops up to and including this one
			vm.forLoops = vm.forLoops[:loopIndex]
			vm.pc++
		}
		
	// I/O operations
	case OP_PRINT:
		value, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		text := vm.toString(value)
		vm.tinybasic.sendMessageWrapped(shared.MessageTypeText, text)
		vm.pc++
		
	case OP_PRINT_NL:
		vm.tinybasic.sendMessageWrapped(shared.MessageTypeText, "\n")
		vm.pc++
		
	case OP_INPUT:
		varName := strings.ToUpper(inst.Operand1.(string))
		
		// Pause VM execution and delegate to TinyBASIC input handling
		vm.running = false
		
		// Set up input state in TinyBASIC
		vm.tinybasic.mu.Lock()
		vm.tinybasic.inputVar = varName
		vm.tinybasic.waitingInput = true
		vm.tinybasic.inputPC = vm.pc + 1 // Store where to resume after input
		vm.tinybasic.mu.Unlock()
		
		// Send input prompt
		vm.tinybasic.sendInputControl("enable")
		vm.tinybasic.sendMessageWrapped(shared.MessageTypeText, "? ")
		
		// VM will be resumed by TinyBASIC when input is received
		return nil
		
	// Special operations
	case OP_HALT:
		vm.running = false
		return nil
		
	case OP_NOP:
		vm.pc++
		
	case OP_SOUND:
		// Get duration from stack
		duration, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("SOUND: missing duration argument")
		}
		if !duration.IsNumeric {
			return fmt.Errorf("SOUND: duration must be numeric")
		}
		
		// Get frequency from stack
		frequency, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("SOUND: missing frequency argument")
		}
		if !frequency.IsNumeric {
			return fmt.Errorf("SOUND: frequency must be numeric")
		}
		
		// Execute SOUND command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			// Format arguments and call cmdSound directly
			args := fmt.Sprintf("%g, %g", frequency.NumValue, duration.NumValue)
			err := vm.tinybasic.cmdSound(args)
			if err != nil {
				return fmt.Errorf("SOUND execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_WAIT:
		// Get duration from stack
		duration, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("WAIT: missing duration argument")
		}
		if !duration.IsNumeric {
			return fmt.Errorf("WAIT: duration must be numeric")
		}
		
		// Validate duration range
		millis := int(duration.NumValue)
		if millis < 0 || millis > 60000 {
			return fmt.Errorf("WAIT: duration must be between 0 and 60000 milliseconds")
		}
		
		// Execute WAIT directly without mutex issues
		time.Sleep(time.Duration(millis) * time.Millisecond)
		
		vm.pc++
		
	case OP_NOISE:
		// Get decay from stack
		decay, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("NOISE: missing decay argument")
		}
		if !decay.IsNumeric {
			return fmt.Errorf("NOISE: decay must be numeric")
		}
		
		// Get attack from stack
		attack, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("NOISE: missing attack argument")
		}
		if !attack.IsNumeric {
			return fmt.Errorf("NOISE: attack must be numeric")
		}
		
		// Get pitch from stack
		pitch, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("NOISE: missing pitch argument")
		}
		if !pitch.IsNumeric {
			return fmt.Errorf("NOISE: pitch must be numeric")
		}
		
		// Execute NOISE command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			// Format arguments and call cmdNoise directly
			args := fmt.Sprintf("%g, %g, %g", pitch.NumValue, attack.NumValue, decay.NumValue)
			err := vm.tinybasic.cmdNoise(args)
			if err != nil {
				return fmt.Errorf("NOISE execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_BEEP:
		// Execute BEEP command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			err := vm.tinybasic.cmdBeep("")
			if err != nil {
				return fmt.Errorf("BEEP execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_CLS:
		// Execute CLS command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			err := vm.tinybasic.cmdCls("")
			if err != nil {
				return fmt.Errorf("CLS execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_MUSIC:
		// Get filename from stack
		filename, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("MUSIC: missing filename argument")
		}
		
		// Convert to string if numeric
		var filenameStr string
		if filename.IsNumeric {
			filenameStr = vm.toString(filename)
		} else {
			filenameStr = filename.StrValue
		}
		
		// Execute MUSIC command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			err := vm.tinybasic.cmdMusic(filenameStr)
			if err != nil {
				return fmt.Errorf("MUSIC execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_SPEAK:
		// Get text from stack
		text, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("SPEAK: missing text argument")
		}
		
		// Convert to string if numeric
		var textStr string
		if text.IsNumeric {
			textStr = vm.toString(text)
		} else {
			textStr = text.StrValue
		}
		
		// Execute SPEAK command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			_, err := vm.tinybasic.cmdSpeak(textStr)
			if err != nil {
				return fmt.Errorf("SPEAK execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_PLOT:
		// Get y coordinate from stack
		y, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("PLOT: missing y coordinate")
		}
		if !y.IsNumeric {
			return fmt.Errorf("PLOT: y coordinate must be numeric")
		}
		
		// Get x coordinate from stack
		x, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("PLOT: missing x coordinate")
		}
		if !x.IsNumeric {
			return fmt.Errorf("PLOT: x coordinate must be numeric")
		}
		
		// Execute PLOT command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			args := fmt.Sprintf("%g, %g", x.NumValue, y.NumValue)
			err := vm.tinybasic.cmdPlot(args)
			if err != nil {
				return fmt.Errorf("PLOT execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_LINE:
		// Get y2 coordinate from stack
		y2, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("LINE: missing y2 coordinate")
		}
		if !y2.IsNumeric {
			return fmt.Errorf("LINE: y2 coordinate must be numeric")
		}
		
		// Get x2 coordinate from stack
		x2, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("LINE: missing x2 coordinate")
		}
		if !x2.IsNumeric {
			return fmt.Errorf("LINE: x2 coordinate must be numeric")
		}
		
		// Get y1 coordinate from stack
		y1, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("LINE: missing y1 coordinate")
		}
		if !y1.IsNumeric {
			return fmt.Errorf("LINE: y1 coordinate must be numeric")
		}
		
		// Get x1 coordinate from stack
		x1, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("LINE: missing x1 coordinate")
		}
		if !x1.IsNumeric {
			return fmt.Errorf("LINE: x1 coordinate must be numeric")
		}
		
		// Execute LINE command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			args := fmt.Sprintf("%g, %g, %g, %g", x1.NumValue, y1.NumValue, x2.NumValue, y2.NumValue)
			err := vm.tinybasic.cmdLine(args)
			if err != nil {
				return fmt.Errorf("LINE execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_RECT:
		// Get height from stack
		height, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("RECT: missing height argument")
		}
		if !height.IsNumeric {
			return fmt.Errorf("RECT: height must be numeric")
		}
		
		// Get width from stack
		width, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("RECT: missing width argument")
		}
		if !width.IsNumeric {
			return fmt.Errorf("RECT: width must be numeric")
		}
		
		// Get y coordinate from stack
		y, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("RECT: missing y coordinate")
		}
		if !y.IsNumeric {
			return fmt.Errorf("RECT: y coordinate must be numeric")
		}
		
		// Get x coordinate from stack
		x, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("RECT: missing x coordinate")
		}
		if !x.IsNumeric {
			return fmt.Errorf("RECT: x coordinate must be numeric")
		}
		
		// Execute RECT command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			args := fmt.Sprintf("%g, %g, %g, %g", x.NumValue, y.NumValue, width.NumValue, height.NumValue)
			err := vm.tinybasic.cmdRect(args)
			if err != nil {
				return fmt.Errorf("RECT execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_CIRCLE:
		// Get radius from stack
		radius, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("CIRCLE: missing radius argument")
		}
		if !radius.IsNumeric {
			return fmt.Errorf("CIRCLE: radius must be numeric")
		}
		
		// Get y coordinate from stack
		y, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("CIRCLE: missing y coordinate")
		}
		if !y.IsNumeric {
			return fmt.Errorf("CIRCLE: y coordinate must be numeric")
		}
		
		// Get x coordinate from stack
		x, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("CIRCLE: missing x coordinate")
		}
		if !x.IsNumeric {
			return fmt.Errorf("CIRCLE: x coordinate must be numeric")
		}
		
		// Execute CIRCLE command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			args := fmt.Sprintf("%g, %g, %g", x.NumValue, y.NumValue, radius.NumValue)
			err := vm.tinybasic.cmdCircle(args)
			if err != nil {
				return fmt.Errorf("CIRCLE execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_SPRITE:
		// Get arguments from stack
		args, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("SPRITE: missing arguments")
		}
		
		// Convert to string
		var argsStr string
		if args.IsNumeric {
			argsStr = vm.toString(args)
		} else {
			argsStr = args.StrValue
		}
		
		// Execute SPRITE command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			err := vm.tinybasic.cmdSprite(argsStr)
			if err != nil {
				return fmt.Errorf("SPRITE execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_VECTOR:
		// Get arguments from stack
		args, err := vm.stack.Pop()
		if err != nil {
			return fmt.Errorf("VECTOR: missing arguments")
		}
		
		// Convert to string
		var argsStr string
		if args.IsNumeric {
			argsStr = vm.toString(args)
		} else {
			argsStr = args.StrValue
		}
		
		// Execute VECTOR command through TinyBASIC interpreter
		if vm.tinybasic != nil {
			err := vm.tinybasic.cmdVector(argsStr)
			if err != nil {
				return fmt.Errorf("VECTOR execution error: %v", err)
			}
		}
		
		vm.pc++
		
	case OP_DEBUG:
		// Debug breakpoint - could be used for debugging
		vm.pc++
		
	case OP_CALL_FUNC:
		// Call built-in function
		funcName := inst.Operand1.(string)
		
		// Check for fallback to interpreted execution
		if funcName == "FALLBACK_TO_INTERPRETED" {
			if inst.Operand2 != nil {
				commandLine := inst.Operand2.(string)
				// Stop VM execution and delegate to TinyBASIC interpreter
				vm.running = false
				
				// Execute the command using the TinyBASIC interpreter
				go func() {
					// This is a simplified fallback - in practice you'd need to
					// save VM state, execute the command, and resume if needed
					vm.tinybasic.sendMessageWrapped(shared.MessageTypeText, 
						fmt.Sprintf("Fallback to interpreted execution: %s", commandLine))
				}()
				return nil
			}
		}
		
		argCount := 0
		if inst.Operand2 != nil {
			if count, ok := inst.Operand2.(int); ok {
				argCount = count
			}
		}
		
		// For native functions, delegate to TinyBASIC for function execution
		err := vm.callBuiltinFunction(funcName, argCount)
		if err != nil {
			return err
		}
		vm.pc++
		
	// String operations
	case OP_STR_CONCAT:
		b, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		a, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		result := vm.toString(a) + vm.toString(b)
		vm.stack.Push(newStringBASICValue(result))
		vm.pc++
		
	case OP_STR_LEN:
		str, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		length := float64(len(vm.toString(str)))
		vm.stack.Push(newNumericBASICValue(length))
		vm.pc++
		
	default:
		return fmt.Errorf("unknown opcode: %v", inst.OpCode)
	}
	
	return nil
}

// Helper functions for VM operations

// execBinaryOp executes a binary operation
func (vm *BytecodeVM) execBinaryOp(op func(a, b BASICValue) (BASICValue, error)) error {
	b, err := vm.stack.Pop()
	if err != nil {
		return err
	}
	a, err := vm.stack.Pop()
	if err != nil {
		return err
	}
	
	result, err := op(a, b)
	if err != nil {
		return err
	}
	
	vm.stack.Push(result)
	vm.pc++
	return nil
}

// execUnaryOp executes a unary operation
func (vm *BytecodeVM) execUnaryOp(op func(a BASICValue) (BASICValue, error)) error {
	a, err := vm.stack.Pop()
	if err != nil {
		return err
	}
	
	result, err := op(a)
	if err != nil {
		return err
	}
	
	vm.stack.Push(result)
	vm.pc++
	return nil
}

// execComparison executes a comparison operation
func (vm *BytecodeVM) execComparison(cmp func(a, b BASICValue) bool) error {
	b, err := vm.stack.Pop()
	if err != nil {
		return err
	}
	a, err := vm.stack.Pop()
	if err != nil {
		return err
	}
	
	result := cmp(a, b)
	vm.stack.Push(newNumericBASICValue(vm.boolToNum(result)))
	vm.pc++
	return nil
}

// toString converts a BASICValue to string
func (vm *BytecodeVM) toString(value BASICValue) string {
	if value.IsNumeric {
		// Format number appropriately
		if value.NumValue == float64(int64(value.NumValue)) {
			return strconv.FormatInt(int64(value.NumValue), 10)
		}
		return strconv.FormatFloat(value.NumValue, 'f', -1, 64)
	}
	return value.StrValue
}

// isTrue determines if a BASICValue is true
func (vm *BytecodeVM) isTrue(value BASICValue) bool {
	if value.IsNumeric {
		return value.NumValue != 0
	}
	return value.StrValue != ""
}

// boolToNum converts a boolean to numeric value (0 or -1, like BASIC)
func (vm *BytecodeVM) boolToNum(b bool) float64 {
	if b {
		return -1 // BASIC uses -1 for true
	}
	return 0
}

// callBuiltinFunction calls a built-in TinyBASIC function
func (vm *BytecodeVM) callBuiltinFunction(funcName string, argCount int) error {
	// For simple math functions, implement them natively
	switch strings.ToUpper(funcName) {
	case "ABS":
		if argCount != 1 {
			return fmt.Errorf("ABS requires 1 argument, got %d", argCount)
		}
		arg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		if !arg.IsNumeric {
			return fmt.Errorf("ABS requires numeric argument")
		}
		result := arg.NumValue
		if result < 0 {
			result = -result
		}
		vm.stack.Push(newNumericBASICValue(result))
		return nil
		
	case "INT":
		if argCount != 1 {
			return fmt.Errorf("INT requires 1 argument, got %d", argCount)
		}
		arg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		if !arg.IsNumeric {
			return fmt.Errorf("INT requires numeric argument")
		}
		result := float64(int64(arg.NumValue))
		vm.stack.Push(newNumericBASICValue(result))
		return nil
		
	case "RND":
		// RND with 0 or 1 arguments
		if argCount > 1 {
			return fmt.Errorf("RND requires 0 or 1 arguments, got %d", argCount)
		}
		if argCount == 1 {
			// Pop the argument but ignore it for now
			_, err := vm.stack.Pop()
			if err != nil {
				return err
			}
		}
		// Generate random number between 0 and 1
		result := rand.Float64()
		vm.stack.Push(newNumericBASICValue(result))
		return nil
		
	case "LEN":
		if argCount != 1 {
			return fmt.Errorf("LEN requires 1 argument, got %d", argCount)
		}
		arg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		var length float64
		if arg.IsNumeric {
			// Convert number to string first
			length = float64(len(vm.toString(arg)))
		} else {
			length = float64(len(arg.StrValue))
		}
		vm.stack.Push(newNumericBASICValue(length))
		return nil
		
	case "MID$":
		if argCount != 2 && argCount != 3 {
			return fmt.Errorf("MID$ requires 2 or 3 arguments, got %d", argCount)
		}
		
		var lengthArg BASICValue
		if argCount == 3 {
			var err error
			lengthArg, err = vm.stack.Pop()
			if err != nil {
				return err
			}
			if !lengthArg.IsNumeric {
				return fmt.Errorf("MID$ length must be numeric")
			}
		}
		
		startArg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		if !startArg.IsNumeric {
			return fmt.Errorf("MID$ start position must be numeric")
		}
		
		strArg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		var str string
		if strArg.IsNumeric {
			str = vm.toString(strArg)
		} else {
			str = strArg.StrValue
		}
		
		start := int(startArg.NumValue) - 1 // BASIC uses 1-based indexing
		if start < 0 {
			start = 0
		}
		
		var result string
		if start >= len(str) {
			result = ""
		} else if argCount == 2 {
			result = str[start:]
		} else {
			length := int(lengthArg.NumValue)
			if length <= 0 {
				result = ""
			} else {
				end := start + length
				if end > len(str) {
					end = len(str)
				}
				result = str[start:end]
			}
		}
		
		vm.stack.Push(newStringBASICValue(result))
		return nil
		
	case "LEFT$":
		if argCount != 2 {
			return fmt.Errorf("LEFT$ requires 2 arguments, got %d", argCount)
		}
		
		lengthArg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		if !lengthArg.IsNumeric {
			return fmt.Errorf("LEFT$ length must be numeric")
		}
		
		strArg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		var str string
		if strArg.IsNumeric {
			str = vm.toString(strArg)
		} else {
			str = strArg.StrValue
		}
		
		length := int(lengthArg.NumValue)
		if length <= 0 {
			vm.stack.Push(newStringBASICValue(""))
		} else if length >= len(str) {
			vm.stack.Push(newStringBASICValue(str))
		} else {
			vm.stack.Push(newStringBASICValue(str[:length]))
		}
		return nil
		
	case "RIGHT$":
		if argCount != 2 {
			return fmt.Errorf("RIGHT$ requires 2 arguments, got %d", argCount)
		}
		
		lengthArg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		if !lengthArg.IsNumeric {
			return fmt.Errorf("RIGHT$ length must be numeric")
		}
		
		strArg, err := vm.stack.Pop()
		if err != nil {
			return err
		}
		
		var str string
		if strArg.IsNumeric {
			str = vm.toString(strArg)
		} else {
			str = strArg.StrValue
		}
		
		length := int(lengthArg.NumValue)
		if length <= 0 {
			vm.stack.Push(newStringBASICValue(""))
		} else if length >= len(str) {
			vm.stack.Push(newStringBASICValue(str))
		} else {
			start := len(str) - length
			vm.stack.Push(newStringBASICValue(str[start:]))
		}
		return nil
		
	default:
		// For complex functions, fall back to TinyBASIC
		// This is not ideal but ensures compatibility
		return fmt.Errorf("unsupported function in bytecode: %s", funcName)
	}
}

// GetVariables returns current variable state (for debugging)
func (vm *BytecodeVM) GetVariables() map[string]BASICValue {
	result := make(map[string]BASICValue)
	for k, v := range vm.variables {
		result[k] = v
	}
	return result
}

// GetPC returns current program counter (for debugging)
func (vm *BytecodeVM) GetPC() int {
	return vm.pc
}

// IsRunning returns whether VM is currently running
func (vm *BytecodeVM) IsRunning() bool {
	return vm.running
}

// Resume resumes VM execution from a specific program counter
func (vm *BytecodeVM) Resume(pc int, inputValue BASICValue, varName string) error {
	if vm.program == nil {
		return fmt.Errorf("no program loaded")
	}
	
	// Store the input value to the specified variable
	if varName != "" {
		vm.variables[strings.ToUpper(varName)] = inputValue
	}
	
	// Resume execution from the specified PC
	vm.pc = pc
	vm.running = true
	
	// Continue execution
	for vm.running && vm.pc < len(vm.program.Instructions) {
		// Check for cancellation
		select {
		case <-vm.ctx.Done():
			vm.running = false
			return vm.ctx.Err()
		default:
		}
		
		// Execute current instruction
		err := vm.executeInstruction()
		if err != nil {
			vm.running = false
			return err
		}
		
		// Small yield to prevent blocking
		if vm.pc%100 == 0 {
			time.Sleep(0)
		}
	}
	
	return nil
}