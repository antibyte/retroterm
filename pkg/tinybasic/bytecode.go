package tinybasic

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Bytecode instruction opcodes for TinyBASIC
type OpCode byte

const (
	// Arithmetic operations
	OP_ADD OpCode = iota
	OP_SUB
	OP_MUL
	OP_DIV
	OP_MOD
	OP_POW
	OP_NEG

	// Comparison operations
	OP_EQ
	OP_NE
	OP_LT
	OP_LE
	OP_GT
	OP_GE

	// Logical operations
	OP_AND
	OP_OR
	OP_NOT

	// Stack operations
	OP_PUSH_NUM  // Push numeric literal
	OP_PUSH_STR  // Push string literal
	OP_LOAD_VAR  // Load variable value
	OP_STORE_VAR // Store value to variable
	OP_POP       // Pop top of stack

	// Control flow
	OP_JUMP        // Unconditional jump
	OP_JUMP_IF     // Jump if top of stack is true
	OP_JUMP_UNLESS // Jump if top of stack is false
	OP_CALL        // Call subroutine (GOSUB)
	OP_RETURN      // Return from subroutine

	// Loop operations
	OP_FOR_INIT  // Initialize FOR loop
	OP_FOR_CHECK // Check FOR loop condition
	OP_FOR_NEXT  // Increment FOR loop variable

	// I/O operations
	OP_PRINT    // Print value
	OP_PRINT_NL // Print newline
	OP_INPUT    // Input to variable

	// Special operations
	OP_HALT          // End program execution
	OP_NOP           // No operation
	OP_SOUND         // Play sound
	OP_WAIT          // Wait/delay
	OP_NOISE         // Play noise
	OP_BEEP          // Play beep
	OP_CLS           // Clear screen
	OP_MUSIC         // Music command
	OP_SPEAK         // Speak command
	OP_PLOT          // Plot pixel
	OP_LINE          // Draw line
	OP_RECT          // Draw rectangle
	OP_CIRCLE        // Draw circle
	OP_SPRITE        // Sprite command
	OP_VECTOR        // Vector command
	OP_PYRAMID       // Pyramid command
	OP_CYLINDER      // Cylinder command
	OP_SAY           // Say command
	OP_LOCATE        // Locate cursor
	OP_COLOR         // Set color
	OP_KEY           // Key input
	OP_DATA          // Data statement
	OP_READ          // Read data
	OP_DIM           // Dimension arrays
	OP_TEXTGFX       // Text graphics
	OP_CLEARGRAPHICS // Clear graphics
	OP_INVERSE       // Inverse text
	OP_RANDOMIZE     // Randomize seed
	OP_DEBUG         // Debug breakpoint

	// Function calls
	OP_CALL_FUNC // Call built-in function

	// String operations
	OP_STR_CONCAT // String concatenation
	OP_STR_LEN    // String length
	OP_STR_MID    // String substring
)

// Bytecode instruction with opcode and operands
type Instruction struct {
	OpCode   OpCode
	Operand1 interface{} // Can be int, float64, string, or nil
	Operand2 interface{} // Second operand for some instructions
	LineNum  int         // Original BASIC line number for debugging
}

// Compiled bytecode program
type BytecodeProgram struct {
	Instructions []Instruction  // Compiled instructions
	Constants    []interface{}  // Constant pool for literals
	Labels       map[int]int    // Line number to instruction index mapping
	OriginalCode map[int]string // Original BASIC code for debugging
	mutex        sync.RWMutex   // Protects concurrent access
}

// Bytecode virtual machine stack
type VMStack struct {
	data []BASICValue
	top  int
	size int
}

// StringInterningSystem provides thread-safe string interning with memory management
type StringInterningSystem struct {
	table     map[string]string
	mutex     sync.RWMutex
	maxSize   int
	hits      int64
	misses    int64
	evictions int64
}

// Global string interning system
var globalStringInterning = &StringInterningSystem{
	table:   make(map[string]string),
	maxSize: 10000, // Limit to 10k strings
}

// InternString interns a string to reduce allocations with memory management
func InternString(s string) string {
	return globalStringInterning.InternString(s)
}

// InternString interns a string with the system
func (sis *StringInterningSystem) InternString(s string) string {
	// Skip interning very long strings to prevent memory bloat
	if len(s) > 1000 {
		return s
	}
	
	sis.mutex.RLock()
	if interned, exists := sis.table[s]; exists {
		sis.mutex.RUnlock()
		atomic.AddInt64(&sis.hits, 1)
		return interned
	}
	sis.mutex.RUnlock()

	sis.mutex.Lock()
	defer sis.mutex.Unlock()
	
	// Double-check after acquiring write lock
	if interned, exists := sis.table[s]; exists {
		atomic.AddInt64(&sis.hits, 1)
		return interned
	}
	
	// Check if we need to evict entries
	if len(sis.table) >= sis.maxSize {
		sis.evictOldEntries()
	}
	
	sis.table[s] = s
	atomic.AddInt64(&sis.misses, 1)
	return s
}

// evictOldEntries removes some entries to make room (improved batch eviction)
func (sis *StringInterningSystem) evictOldEntries() {
	// Batch eviction: remove 20% of entries to reduce eviction frequency
	removeCount := len(sis.table) / 5
	if removeCount < 10 {
		removeCount = 10 // Minimum batch size
	}
	
	// Collect entries to remove (prefer longer strings for eviction)
	type entryInfo struct {
		key    string
		length int
	}
	
	entries := make([]entryInfo, 0, len(sis.table))
	for key := range sis.table {
		entries = append(entries, entryInfo{key: key, length: len(key)})
	}
	
	// Sort by length (descending) to evict longer strings first
	// This keeps frequently used short strings in cache longer
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].length < entries[j].length {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	// Remove the longest strings first
	count := 0
	for _, entry := range entries {
		if count >= removeCount {
			break
		}
		delete(sis.table, entry.key)
		count++
	}
	
	atomic.AddInt64(&sis.evictions, int64(count))
}

// GetStats returns interning statistics
func (sis *StringInterningSystem) GetStats() map[string]interface{} {
	sis.mutex.RLock()
	defer sis.mutex.RUnlock()
	
	hits := atomic.LoadInt64(&sis.hits)
	misses := atomic.LoadInt64(&sis.misses)
	total := hits + misses
	
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	
	return map[string]interface{}{
		"hits":       hits,
		"misses":     misses,
		"evictions":  atomic.LoadInt64(&sis.evictions),
		"entries":    len(sis.table),
		"hit_rate":   hitRate,
		"max_size":   sis.maxSize,
	}
}

// GetPooledBASICValue gets a BASICValue from the pool (use existing pool from tinybasic.go)
func GetPooledBASICValue() *BASICValue {
	return getBASICValue()
}

// PutPooledBASICValue returns a BASICValue to the pool (use existing pool from tinybasic.go)
func PutPooledBASICValue(v *BASICValue) {
	returnBASICValue(v)
}

// VMStackPool provides pooled VM stacks for reuse
type VMStackPool struct {
	pool    sync.Pool
	maxSize int
}

// Global VM stack pool
var globalVMStackPool = &VMStackPool{
	maxSize: 1000,
}

func init() {
	globalVMStackPool.pool = sync.Pool{
		New: func() interface{} {
			return &VMStack{
				data: make([]BASICValue, globalVMStackPool.maxSize),
				top:  -1,
				size: globalVMStackPool.maxSize,
			}
		},
	}
}

// NewVMStack creates a new VM stack with pooling
func NewVMStack(size int) *VMStack {
	if size <= globalVMStackPool.maxSize {
		// Use pooled stack
		stack := globalVMStackPool.pool.Get().(*VMStack)
		stack.Clear()
		return stack
	}
	
	// Create new stack for large sizes
	return &VMStack{
		data: make([]BASICValue, size),
		top:  -1,
		size: size,
	}
}

// ReturnVMStack returns a stack to the pool for reuse
func ReturnVMStack(stack *VMStack) {
	if stack.size <= globalVMStackPool.maxSize {
		stack.Clear()
		globalVMStackPool.pool.Put(stack)
	}
}

// Push value onto stack (optimized)
func (s *VMStack) Push(value BASICValue) error {
	if s.top >= s.size-1 {
		return fmt.Errorf("stack overflow")
	}
	s.top++
	s.data[s.top] = value
	return nil
}

// FastPush value onto stack with minimal bounds checking (optimized but safe)
func (s *VMStack) FastPush(value BASICValue) {
	s.top++
	if s.top >= s.size {
		panic(fmt.Sprintf("stack overflow: attempted to push at index %d, stack size %d", s.top, s.size))
	}
	s.data[s.top] = value
}

// Pop value from stack (optimized)
func (s *VMStack) Pop() (BASICValue, error) {
	if s.top < 0 {
		return BASICValue{}, fmt.Errorf("stack underflow")
	}
	value := s.data[s.top]
	s.top--
	return value, nil
}

// FastPop value from stack with minimal bounds checking (optimized but safe)
func (s *VMStack) FastPop() BASICValue {
	if s.top < 0 {
		panic(fmt.Sprintf("stack underflow: attempted to pop at index %d", s.top))
	}
	value := s.data[s.top]
	s.top--
	return value
}

// Peek at top of stack without popping
func (s *VMStack) Peek() (BASICValue, error) {
	if s.top < 0 {
		return BASICValue{}, fmt.Errorf("empty stack")
	}
	return s.data[s.top], nil
}

// IsEmpty checks if stack is empty
func (s *VMStack) IsEmpty() bool {
	return s.top < 0
}

// HasSpace returns true if stack has space for n more items
func (s *VMStack) HasSpace(n int) bool {
	return s.top+n < s.size
}

// HasItems returns true if stack has at least n items
func (s *VMStack) HasItems(n int) bool {
	return s.top >= n-1
}

// SafeFastPush pushes value if there's space, panics otherwise
func (s *VMStack) SafeFastPush(value BASICValue) {
	if !s.HasSpace(1) {
		panic(fmt.Sprintf("stack overflow: no space for push, current size %d/%d", s.top+1, s.size))
	}
	s.FastPush(value)
}

// SafeFastPop pops value if items exist, panics otherwise
func (s *VMStack) SafeFastPop() BASICValue {
	if !s.HasItems(1) {
		panic(fmt.Sprintf("stack underflow: no items to pop, current size %d", s.top+1))
	}
	return s.FastPop()
}

// Size returns the current stack size
func (s *VMStack) Size() int {
	return s.top + 1
}

// Clear empties the stack
func (s *VMStack) Clear() {
	s.top = -1
}

// Bytecode compiler state
type BytecodeCompiler struct {
	program      *BytecodeProgram
	currentLine  int
	instructions []Instruction
	constants    []interface{}
	labels       map[int]int
	originalCode map[int]string
}

// NewBytecodeCompiler creates a new bytecode compiler
func NewBytecodeCompiler() *BytecodeCompiler {
	return &BytecodeCompiler{
		program: &BytecodeProgram{
			Instructions: make([]Instruction, 0),
			Constants:    make([]interface{}, 0),
			Labels:       make(map[int]int),
			OriginalCode: make(map[int]string),
		},
		instructions: make([]Instruction, 0),
		constants:    make([]interface{}, 0),
		labels:       make(map[int]int),
		originalCode: make(map[int]string),
	}
}

// Emit adds an instruction to the current compilation
func (c *BytecodeCompiler) Emit(opcode OpCode, operands ...interface{}) {
	inst := Instruction{
		OpCode:  opcode,
		LineNum: c.currentLine,
	}

	if len(operands) > 0 {
		inst.Operand1 = operands[0]
	}
	if len(operands) > 1 {
		inst.Operand2 = operands[1]
	}

	c.instructions = append(c.instructions, inst)
}

// EmitConstant adds a constant to the constant pool and emits appropriate load instruction
func (c *BytecodeCompiler) EmitConstant(value interface{}) {
	constIndex := len(c.constants)
	c.constants = append(c.constants, value)

	switch value.(type) {
	case float64, int:
		c.Emit(OP_PUSH_NUM, constIndex)
	case string:
		c.Emit(OP_PUSH_STR, constIndex)
	}
}

// SetLabel marks the current instruction position as a label for the given line number
func (c *BytecodeCompiler) SetLabel(lineNum int) {
	c.labels[lineNum] = len(c.instructions)
}

// CompileProgram compiles a BASIC program to bytecode
func (c *BytecodeCompiler) CompileProgram(program map[int]string, programLines []int) (*BytecodeProgram, error) {
	// Reset compiler state
	c.instructions = make([]Instruction, 0)
	c.constants = make([]interface{}, 0)
	c.labels = make(map[int]int)
	c.originalCode = make(map[int]string)

	// Store original code
	for _, lineNum := range programLines {
		c.originalCode[lineNum] = program[lineNum]
	}

	// Compile each line and set labels correctly
	for _, lineNum := range programLines {
		// Set label to current instruction position
		c.labels[lineNum] = len(c.instructions)

		c.currentLine = lineNum
		code := program[lineNum]

		err := c.compileLine(code)
		if err != nil {
			return nil, fmt.Errorf("compilation error at line %d: %v", lineNum, err)
		}
	}

	// Emit HALT instruction at the end
	c.Emit(OP_HALT)

	// Build final program
	c.program.Instructions = c.instructions
	c.program.Constants = c.constants
	c.program.Labels = c.labels
	c.program.OriginalCode = c.originalCode

	return c.program, nil
}

// compileLine compiles a single BASIC line to bytecode
func (c *BytecodeCompiler) compileLine(line string) error {
	// The line already has the line number removed by the compiler
	// since it comes from the program map which stores just the code part

	// Split line into statements by colon
	statements := c.splitStatements(line)

	for _, stmt := range statements {
		err := c.compileStatement(stmt)
		if err != nil {
			return err
		}
	}

	return nil
}

// compileStatement compiles a single BASIC statement to bytecode
func (c *BytecodeCompiler) compileStatement(stmt string) error {
	stmt = strings.TrimSpace(stmt)
	if stmt == "" {
		return nil
	}

	parts := strings.Fields(stmt)
	if len(parts) == 0 {
		return nil
	}

	command := strings.ToUpper(parts[0])
	args := ""
	if len(parts) > 1 {
		// Preserve original spacing after command
		commandLen := len(parts[0])
		if spaceIdx := strings.Index(stmt[commandLen:], " "); spaceIdx != -1 {
			args = stmt[commandLen+spaceIdx+1:]
		}
	}

	switch command {
	case "REM":
		// Comments are no-ops
		c.Emit(OP_NOP)

	case "LET":
		return c.compileLet(stmt)

	case "PRINT", "PR.", "?":
		return c.compilePrint(args)

	case "INPUT":
		return c.compileInput(args)

	case "IF":
		return c.compileIf(args)

	case "GOTO":
		return c.compileGoto(args)

	case "GOSUB":
		return c.compileGosub(args)

	case "RETURN":
		c.Emit(OP_RETURN)

	case "FOR":
		return c.compileFor(args)

	case "NEXT":
		return c.compileNext(args)

	case "END", "STOP":
		c.Emit(OP_HALT)

	case "SOUND":
		return c.compileSound(args)

	case "WAIT":
		return c.compileWait(args)

	case "NOISE":
		return c.compileNoise(args)

	case "BEEP":
		return c.compileBeep(args)

	case "CLS":
		return c.compileCls(args)

	case "MUSIC":
		return c.compileMusic(args)

	case "SPEAK":
		return c.compileSpeak(args)

	case "PLOT":
		return c.compilePlot(args)

	case "LINE":
		return c.compileLineCommand(args)

	case "RECT":
		return c.compileRect(args)

	case "CIRCLE":
		return c.compileCircle(args)

	case "SPRITE":
		return c.compileSprite(args)

	case "VECTOR":
		return c.compileVector(args)

	case "PYRAMID":
		return c.compilePyramid(args)

	case "CYLINDER":
		return c.compileCylinder(args)

	case "VECFLOOR":
		return c.compileVecFloor(args)

	case "VECNODE":
		return c.compileVecNode(args)

	case "SAY":
		return c.compileSay(args)

	case "LOCATE":
		return c.compileLocate(args)

	case "COLOR":
		return c.compileColor(args)

	case "KEY":
		return c.compileKey(args)

	case "DATA":
		return c.compileData(args)

	case "READ":
		return c.compileRead(args)

	case "DIM":
		return c.compileDim(args)

	case "TEXTGFX":
		return c.compileTextGfx(args)

	case "CLEARGRAPHICS":
		return c.compileClearGraphics(args)

	case "INVERSE":
		return c.compileInverse(args)

	case "RANDOMIZE":
		return c.compileRandomize(args)

	case "PHYSICS":
		return c.compilePhysics(args)

	default:
		// Unknown command - emit as function call
		return c.compileFunction(command, args)
	}

	return nil
}

// compileLet compiles LET statements (variable assignments)
func (c *BytecodeCompiler) compileLet(stmt string) error {
	// Handle both "LET var = expr" and "var = expr" forms
	letStmt := strings.TrimSpace(stmt)

	// Remove LET keyword if present
	if strings.HasPrefix(strings.ToUpper(letStmt), "LET ") {
		letStmt = strings.TrimSpace(letStmt[4:])
	}

	// Parse assignment: var = expression
	parts := strings.SplitN(letStmt, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid assignment syntax: %s", stmt)
	}

	varName := strings.TrimSpace(strings.ToUpper(parts[0]))
	expression := strings.TrimSpace(parts[1])

	// Validate variable name
	if !c.isValidVariableName(varName) {
		return fmt.Errorf("invalid variable name: %s", varName)
	}

	// Compile expression to stack
	err := c.compileExpression(expression)
	if err != nil {
		return fmt.Errorf("error compiling expression '%s': %v", expression, err)
	}

	// Store result to variable
	c.Emit(OP_STORE_VAR, varName)

	return nil
}

// compilePrint compiles PRINT statements
func (c *BytecodeCompiler) compilePrint(args string) error {
	if args == "" {
		// Empty PRINT - just print newline
		c.Emit(OP_PRINT_NL)
		return nil
	}

	// Check if statement ends with separator
	args = strings.TrimSpace(args)
	endsWithSeparator := false
	lastChar := ""

	if strings.HasSuffix(args, ";") {
		endsWithSeparator = true
		lastChar = ";"
		args = strings.TrimSpace(args[:len(args)-1])
	} else if strings.HasSuffix(args, ",") {
		endsWithSeparator = true
		lastChar = ","
		args = strings.TrimSpace(args[:len(args)-1])
	}

	// Parse print items separated by semicolons or commas
	items := c.parsePrintItems(args)

	for i, item := range items {
		// Compile expression
		err := c.compileExpression(item.expression)
		if err != nil {
			return fmt.Errorf("error compiling PRINT expression '%s': %v", item.expression, err)
		}

		// Emit print instruction
		c.Emit(OP_PRINT)

		// Handle separator between items
		if i < len(items)-1 {
			if item.separator == "," {
				// Tab to next print zone (approximate with spaces)
				c.EmitConstant("    ") // 4 spaces for tab
				c.Emit(OP_PRINT)
			}
			// Semicolon means no space/newline between items
		}
	}

	// Handle final separator
	if endsWithSeparator {
		if lastChar == "," {
			// Trailing comma - add tab spacing
			c.EmitConstant("    ")
			c.Emit(OP_PRINT)
		}
		// Trailing semicolon means no newline
	} else {
		// No trailing separator means newline
		c.Emit(OP_PRINT_NL)
	}

	return nil
}

// BytecodePrintItem represents a single item in a PRINT statement for bytecode compilation
type BytecodePrintItem struct {
	expression string
	separator  string // ";" or "," or ""
}

// parsePrintItems parses PRINT statement arguments
func (c *BytecodeCompiler) parsePrintItems(args string) []BytecodePrintItem {
	items := make([]BytecodePrintItem, 0)
	current := ""
	inString := false

	for _, char := range args {
		if char == '"' {
			inString = !inString
			current += string(char)
		} else if !inString && (char == ';' || char == ',') {
			if strings.TrimSpace(current) != "" {
				items = append(items, BytecodePrintItem{
					expression: strings.TrimSpace(current),
					separator:  string(char),
				})
			}
			current = ""
		} else {
			current += string(char)
		}
	}

	// Add final item
	if strings.TrimSpace(current) != "" {
		items = append(items, BytecodePrintItem{
			expression: strings.TrimSpace(current),
			separator:  "", // No trailing separator
		})
	}

	return items
}

// compileInput compiles INPUT statements
func (c *BytecodeCompiler) compileInput(args string) error {
	// Simple INPUT variable
	varName := strings.TrimSpace(args)
	if varName == "" {
		return fmt.Errorf("INPUT requires variable name")
	}

	c.Emit(OP_INPUT, varName)
	return nil
}

// compileIf compiles IF statements
func (c *BytecodeCompiler) compileIf(args string) error {
	// Parse IF condition THEN statement [ELSE statement]
	args = strings.TrimSpace(args)

	// Find THEN keyword
	thenIdx := -1
	upperArgs := strings.ToUpper(args)
	for i := 0; i < len(upperArgs)-4; i++ {
		if upperArgs[i:i+5] == " THEN" && (i+5 >= len(upperArgs) || upperArgs[i+5] == ' ') {
			thenIdx = i + 1
			break
		}
	}

	if thenIdx == -1 {
		return fmt.Errorf("IF statement missing THEN keyword")
	}

	condition := strings.TrimSpace(args[:thenIdx-1])
	remaining := strings.TrimSpace(args[thenIdx+4:]) // Skip "THEN"

	// Check for ELSE
	elseIdx := -1
	upperRemaining := strings.ToUpper(remaining)
	for i := 0; i < len(upperRemaining)-4; i++ {
		if upperRemaining[i:i+5] == " ELSE" && (i+5 >= len(upperRemaining) || upperRemaining[i+5] == ' ') {
			elseIdx = i + 1
			break
		}
	}

	var thenPart, elsePart string
	if elseIdx != -1 {
		thenPart = strings.TrimSpace(remaining[:elseIdx-1])
		elsePart = strings.TrimSpace(remaining[elseIdx+4:]) // Skip "ELSE"
	} else {
		thenPart = remaining
	}

	// Compile condition
	err := c.compileExpression(condition)
	if err != nil {
		return fmt.Errorf("error compiling IF condition '%s': %v", condition, err)
	}

	// Check if THEN part is a line number (GOTO) or statement
	if lineNum, err := strconv.Atoi(thenPart); err == nil {
		// THEN line number - conditional jump
		if elsePart != "" {
			// IF condition THEN linenum ELSE statement
			c.Emit(OP_JUMP_IF, lineNum)
			return c.compileStatement(elsePart)
		} else {
			// IF condition THEN linenum
			c.Emit(OP_JUMP_IF, lineNum)
		}
	} else {
		// THEN statement - conditional execution
		jumpAddr := len(c.instructions)
		c.Emit(OP_JUMP_UNLESS, 0) // Placeholder address

		// Compile the THEN statement
		err = c.compileStatement(thenPart)
		if err != nil {
			return fmt.Errorf("error compiling THEN statement '%s': %v", thenPart, err)
		}

		if elsePart != "" {
			// Jump over ELSE part
			elseJumpAddr := len(c.instructions)
			c.Emit(OP_JUMP, 0) // Placeholder address

			// Update JUMP_UNLESS to point here (start of ELSE)
			c.instructions[jumpAddr].Operand1 = len(c.instructions)

			// Compile ELSE statement
			err = c.compileStatement(elsePart)
			if err != nil {
				return fmt.Errorf("error compiling ELSE statement '%s': %v", elsePart, err)
			}

			// Update ELSE jump to point after ELSE
			c.instructions[elseJumpAddr].Operand1 = len(c.instructions)
		} else {
			// Update JUMP_UNLESS to point after THEN statement
			c.instructions[jumpAddr].Operand1 = len(c.instructions)
		}
	}

	return nil
}

// compileGoto compiles GOTO statements
func (c *BytecodeCompiler) compileGoto(args string) error {
	lineNum, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil {
		return fmt.Errorf("GOTO requires line number")
	}

	c.Emit(OP_JUMP, lineNum)
	return nil
}

// compileGosub compiles GOSUB statements
func (c *BytecodeCompiler) compileGosub(args string) error {
	lineNum, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil {
		return fmt.Errorf("GOSUB requires line number")
	}

	c.Emit(OP_CALL, lineNum)
	return nil
}

// compileFor compiles FOR statements
func (c *BytecodeCompiler) compileFor(args string) error {
	// Parse FOR var = start TO end [STEP step]
	args = strings.TrimSpace(args)

	// Find variable name and equals sign
	eqIdx := strings.Index(args, "=")
	if eqIdx == -1 {
		return fmt.Errorf("FOR statement missing '='")
	}

	varName := strings.TrimSpace(strings.ToUpper(args[:eqIdx]))
	if !c.isValidVariableName(varName) {
		return fmt.Errorf("invalid FOR variable name: %s", varName)
	}

	remainder := strings.TrimSpace(args[eqIdx+1:])

	// Find TO keyword
	toIdx := -1
	upperRemainder := strings.ToUpper(remainder)
	for i := 0; i < len(upperRemainder)-2; i++ {
		if upperRemainder[i:i+3] == " TO" && (i+3 >= len(upperRemainder) || upperRemainder[i+3] == ' ') {
			toIdx = i + 1
			break
		}
	}

	if toIdx == -1 {
		return fmt.Errorf("FOR statement missing TO keyword")
	}

	startExpr := strings.TrimSpace(remainder[:toIdx-1])
	remaining2 := strings.TrimSpace(remainder[toIdx+2:]) // Skip "TO"

	// Check for STEP keyword
	stepIdx := -1
	upperRemaining2 := strings.ToUpper(remaining2)
	for i := 0; i < len(upperRemaining2)-4; i++ {
		if upperRemaining2[i:i+5] == " STEP" && (i+5 >= len(upperRemaining2) || upperRemaining2[i+5] == ' ') {
			stepIdx = i + 1
			break
		}
	}

	var endExpr, stepExpr string
	if stepIdx != -1 {
		endExpr = strings.TrimSpace(remaining2[:stepIdx-1])
		stepExpr = strings.TrimSpace(remaining2[stepIdx+4:]) // Skip "STEP"
	} else {
		endExpr = remaining2
		stepExpr = "1" // Default step
	}

	// Compile start expression and store to variable
	err := c.compileExpression(startExpr)
	if err != nil {
		return fmt.Errorf("error compiling FOR start expression '%s': %v", startExpr, err)
	}
	c.Emit(OP_STORE_VAR, varName)

	// Compile end expression
	err = c.compileExpression(endExpr)
	if err != nil {
		return fmt.Errorf("error compiling FOR end expression '%s': %v", endExpr, err)
	}

	// Compile step expression
	err = c.compileExpression(stepExpr)
	if err != nil {
		return fmt.Errorf("error compiling FOR step expression '%s': %v", stepExpr, err)
	}

	// Initialize FOR loop - stack now has [end, step]
	c.Emit(OP_FOR_INIT, varName)

	return nil
}

// compileNext compiles NEXT statements
func (c *BytecodeCompiler) compileNext(args string) error {
	varName := strings.TrimSpace(args)
	if varName == "" {
		// NEXT without variable - use most recent FOR loop
		c.Emit(OP_FOR_NEXT, nil)
	} else {
		c.Emit(OP_FOR_NEXT, varName)
	}
	return nil
}

// compileSound compiles SOUND statements
func (c *BytecodeCompiler) compileSound(args string) error {
	if args == "" {
		return fmt.Errorf("SOUND requires frequency and duration arguments")
	}

	// Parse SOUND frequency, duration
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return fmt.Errorf("SOUND requires exactly 2 arguments: frequency, duration")
	}

	// Compile frequency expression
	freq := strings.TrimSpace(parts[0])
	err := c.compileExpression(freq)
	if err != nil {
		return fmt.Errorf("error compiling frequency expression: %v", err)
	}

	// Compile duration expression
	duration := strings.TrimSpace(parts[1])
	err = c.compileExpression(duration)
	if err != nil {
		return fmt.Errorf("error compiling duration expression: %v", err)
	}

	// Emit SOUND instruction
	c.Emit(OP_SOUND)
	return nil
}

// compileWait compiles WAIT statements
func (c *BytecodeCompiler) compileWait(args string) error {
	if args == "" {
		return fmt.Errorf("WAIT requires duration argument")
	}

	// Compile duration expression
	err := c.compileExpression(args)
	if err != nil {
		return fmt.Errorf("error compiling WAIT duration: %v", err)
	}

	// Emit WAIT instruction
	c.Emit(OP_WAIT)
	return nil
}

// compileNoise compiles NOISE statements
func (c *BytecodeCompiler) compileNoise(args string) error {
	if args == "" {
		return fmt.Errorf("NOISE requires pitch, attack, and decay arguments")
	}

	// Parse NOISE pitch, attack, decay
	parts := strings.Split(args, ",")
	if len(parts) != 3 {
		return fmt.Errorf("NOISE requires exactly 3 arguments: pitch, attack, decay")
	}

	// Compile pitch expression
	pitch := strings.TrimSpace(parts[0])
	err := c.compileExpression(pitch)
	if err != nil {
		return fmt.Errorf("error compiling pitch expression: %v", err)
	}

	// Compile attack expression
	attack := strings.TrimSpace(parts[1])
	err = c.compileExpression(attack)
	if err != nil {
		return fmt.Errorf("error compiling attack expression: %v", err)
	}

	// Compile decay expression
	decay := strings.TrimSpace(parts[2])
	err = c.compileExpression(decay)
	if err != nil {
		return fmt.Errorf("error compiling decay expression: %v", err)
	}

	// Emit NOISE instruction
	c.Emit(OP_NOISE)
	return nil
}

// compileBeep compiles BEEP statements
func (c *BytecodeCompiler) compileBeep(args string) error {
	if args != "" {
		return fmt.Errorf("BEEP does not take arguments")
	}

	// Emit BEEP instruction
	c.Emit(OP_BEEP)
	return nil
}

// compileCls compiles CLS statements
func (c *BytecodeCompiler) compileCls(args string) error {
	if args != "" {
		return fmt.Errorf("CLS does not take arguments")
	}

	// Emit CLS instruction
	c.Emit(OP_CLS)
	return nil
}

// compileMusic compiles MUSIC statements
func (c *BytecodeCompiler) compileMusic(args string) error {
	// MUSIC command takes a string argument
	if args == "" {
		return fmt.Errorf("MUSIC requires a filename argument")
	}

	// Compile the filename expression
	err := c.compileExpression(args)
	if err != nil {
		return fmt.Errorf("error compiling MUSIC filename: %v", err)
	}

	// Emit MUSIC instruction
	c.Emit(OP_MUSIC)
	return nil
}

// compileSpeak compiles SPEAK statements
func (c *BytecodeCompiler) compileSpeak(args string) error {
	// SPEAK command takes a string argument
	if args == "" {
		return fmt.Errorf("SPEAK requires a text argument")
	}

	// Compile the text expression
	err := c.compileExpression(args)
	if err != nil {
		return fmt.Errorf("error compiling SPEAK text: %v", err)
	}

	// Emit SPEAK instruction
	c.Emit(OP_SPEAK)
	return nil
}

// compilePlot compiles PLOT statements
func (c *BytecodeCompiler) compilePlot(args string) error {
	if args == "" {
		return fmt.Errorf("PLOT requires x and y coordinates")
	}

	// Parse PLOT x, y
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return fmt.Errorf("PLOT requires exactly 2 arguments: x, y")
	}

	// Compile x coordinate
	x := strings.TrimSpace(parts[0])
	err := c.compileExpression(x)
	if err != nil {
		return fmt.Errorf("error compiling x coordinate: %v", err)
	}

	// Compile y coordinate
	y := strings.TrimSpace(parts[1])
	err = c.compileExpression(y)
	if err != nil {
		return fmt.Errorf("error compiling y coordinate: %v", err)
	}

	// Emit PLOT instruction
	c.Emit(OP_PLOT)
	return nil
}

// compileLineCommand compiles LINE statements
func (c *BytecodeCompiler) compileLineCommand(args string) error {
	if args == "" {
		return fmt.Errorf("LINE requires coordinates")
	}

	// Parse LINE x1, y1, x2, y2
	parts := strings.Split(args, ",")
	if len(parts) != 4 {
		return fmt.Errorf("LINE requires exactly 4 arguments: x1, y1, x2, y2")
	}

	// Compile all coordinates
	for i, part := range parts {
		coord := strings.TrimSpace(part)
		err := c.compileExpression(coord)
		if err != nil {
			return fmt.Errorf("error compiling coordinate %d: %v", i+1, err)
		}
	}

	// Emit LINE instruction
	c.Emit(OP_LINE)
	return nil
}

// compileRect compiles RECT statements
func (c *BytecodeCompiler) compileRect(args string) error {
	if args == "" {
		return fmt.Errorf("RECT requires coordinates and dimensions")
	}

	// Parse RECT x, y, width, height
	parts := strings.Split(args, ",")
	if len(parts) != 4 {
		return fmt.Errorf("RECT requires exactly 4 arguments: x, y, width, height")
	}

	// Compile all parameters
	for i, part := range parts {
		param := strings.TrimSpace(part)
		err := c.compileExpression(param)
		if err != nil {
			return fmt.Errorf("error compiling parameter %d: %v", i+1, err)
		}
	}

	// Emit RECT instruction
	c.Emit(OP_RECT)
	return nil
}

// compileCircle compiles CIRCLE statements
func (c *BytecodeCompiler) compileCircle(args string) error {
	if args == "" {
		return fmt.Errorf("CIRCLE requires coordinates and radius")
	}

	// Parse CIRCLE x, y, radius
	parts := strings.Split(args, ",")
	if len(parts) != 3 {
		return fmt.Errorf("CIRCLE requires exactly 3 arguments: x, y, radius")
	}

	// Compile all parameters
	for i, part := range parts {
		param := strings.TrimSpace(part)
		err := c.compileExpression(param)
		if err != nil {
			return fmt.Errorf("error compiling parameter %d: %v", i+1, err)
		}
	}

	// Emit CIRCLE instruction
	c.Emit(OP_CIRCLE)
	return nil
}

// compileSprite compiles SPRITE statements
func (c *BytecodeCompiler) compileSprite(args string) error {
	if args == "" {
		return fmt.Errorf("SPRITE requires arguments")
	}

	// For SPRITE, we need to handle variable argument count
	// Just compile the entire argument string as a single expression for now
	// The VM will handle the complex parsing
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling SPRITE arguments: %v", err)
	}

	// Emit SPRITE instruction
	c.Emit(OP_SPRITE)
	return nil
}

// compileVector compiles VECTOR statements
func (c *BytecodeCompiler) compileVector(args string) error {
	if args == "" {
		return fmt.Errorf("VECTOR requires arguments")
	}

	// For VECTOR, we push the raw argument string as-is
	// The VM will handle the parsing
	c.EmitConstant(BASICValue{IsNumeric: false, StrValue: args})

	// Emit VECTOR instruction
	c.Emit(OP_VECTOR)
	return nil
}

// compilePyramid compiles PYRAMID statements
func (c *BytecodeCompiler) compilePyramid(args string) error {
	if args == "" {
		return fmt.Errorf("PYRAMID requires arguments")
	}

	// For PYRAMID, we need to handle variable argument count
	// Just compile the entire argument string as a single expression for now
	// The VM will handle the complex parsing
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling PYRAMID arguments: %v", err)
	}

	// Emit PYRAMID instruction
	c.Emit(OP_PYRAMID)
	return nil
}

// compileCylinder compiles CYLINDER statements
func (c *BytecodeCompiler) compileCylinder(args string) error {
	if args == "" {
		return fmt.Errorf("CYLINDER requires arguments")
	}

	// For CYLINDER, we need to handle variable argument count
	// Just compile the entire argument string as a single expression for now
	// The VM will handle the complex parsing
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling CYLINDER arguments: %v", err)
	}

	// Emit CYLINDER instruction
	c.Emit(OP_CYLINDER)
	return nil
}

// compileVecFloor compiles VECFLOOR statements
func (c *BytecodeCompiler) compileVecFloor(args string) error {
	if args == "" {
		return fmt.Errorf("VECFLOOR requires arguments")
	}

	// For VECFLOOR, we need to handle variable argument count
	// Just compile the entire argument string as a single expression for now
	// The VM will handle the complex parsing
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling VECFLOOR arguments: %v", err)
	}

	// Emit VECFLOOR instruction
	c.Emit(OP_CALL_FUNC, "VECFLOOR", args)
	return nil
}

// compileVecNode compiles VECNODE statements
func (c *BytecodeCompiler) compileVecNode(args string) error {
	if args == "" {
		return fmt.Errorf("VECNODE requires arguments")
	}

	// For VECNODE, we need to handle variable argument count
	// Just compile the entire argument string as a single expression for now
	// The VM will handle the complex parsing
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling VECNODE arguments: %v", err)
	}

	// Emit VECNODE instruction
	c.Emit(OP_CALL_FUNC, "VECNODE", args)
	return nil
}

// compileSay compiles SAY statements
func (c *BytecodeCompiler) compileSay(args string) error {
	// SAY is an alias for SPEAK
	return c.compileSpeak(args)
}

// compileLocate compiles LOCATE statements
func (c *BytecodeCompiler) compileLocate(args string) error {
	if args == "" {
		return fmt.Errorf("LOCATE requires x and y coordinates")
	}

	// Parse LOCATE x, y
	parts := strings.Split(args, ",")
	if len(parts) != 2 {
		return fmt.Errorf("LOCATE requires exactly 2 arguments: x, y")
	}

	// Compile x coordinate
	x := strings.TrimSpace(parts[0])
	err := c.compileExpression(x)
	if err != nil {
		return fmt.Errorf("error compiling x coordinate: %v", err)
	}

	// Compile y coordinate
	y := strings.TrimSpace(parts[1])
	err = c.compileExpression(y)
	if err != nil {
		return fmt.Errorf("error compiling y coordinate: %v", err)
	}

	// Emit LOCATE instruction
	c.Emit(OP_LOCATE)
	return nil
}

// compileColor compiles COLOR statements
func (c *BytecodeCompiler) compileColor(args string) error {
	if args == "" {
		return fmt.Errorf("COLOR requires color argument")
	}

	// Compile color expression
	err := c.compileExpression(args)
	if err != nil {
		return fmt.Errorf("error compiling COLOR argument: %v", err)
	}

	// Emit COLOR instruction
	c.Emit(OP_COLOR)
	return nil
}

// compileKey compiles KEY statements
func (c *BytecodeCompiler) compileKey(args string) error {
	if args == "" {
		return fmt.Errorf("KEY requires arguments")
	}

	// For KEY, compile the entire argument string
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling KEY arguments: %v", err)
	}

	// Emit KEY instruction
	c.Emit(OP_KEY)
	return nil
}

// compileData compiles DATA statements
func (c *BytecodeCompiler) compileData(args string) error {
	if args == "" {
		return fmt.Errorf("DATA requires data values")
	}

	// For DATA, compile the entire argument string as a string literal
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling DATA arguments: %v", err)
	}

	// Emit DATA instruction
	c.Emit(OP_DATA)
	return nil
}

// compileRead compiles READ statements
func (c *BytecodeCompiler) compileRead(args string) error {
	if args == "" {
		return fmt.Errorf("READ requires variable arguments")
	}

	// For READ, compile the entire argument string as a string literal
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling READ arguments: %v", err)
	}

	// Emit READ instruction
	c.Emit(OP_READ)
	return nil
}

// compileDim compiles DIM statements
func (c *BytecodeCompiler) compileDim(args string) error {
	if args == "" {
		return fmt.Errorf("DIM requires array declaration")
	}

	// For DIM, compile the entire argument string as a string literal
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling DIM arguments: %v", err)
	}

	// Emit DIM instruction
	c.Emit(OP_DIM)
	return nil
}

// compileTextGfx compiles TEXTGFX statements
func (c *BytecodeCompiler) compileTextGfx(args string) error {
	if args == "" {
		return fmt.Errorf("TEXTGFX requires arguments")
	}

	// For TEXTGFX, compile the entire argument string as a string literal
	err := c.compileExpression(fmt.Sprintf("\"%s\"", args))
	if err != nil {
		return fmt.Errorf("error compiling TEXTGFX arguments: %v", err)
	}

	// Emit TEXTGFX instruction
	c.Emit(OP_TEXTGFX)
	return nil
}

// compileClearGraphics compiles CLEARGRAPHICS statements
func (c *BytecodeCompiler) compileClearGraphics(args string) error {
	if args != "" {
		return fmt.Errorf("CLEARGRAPHICS does not take arguments")
	}

	// Emit CLEARGRAPHICS instruction
	c.Emit(OP_CLEARGRAPHICS)
	return nil
}

// compileInverse compiles INVERSE statements
func (c *BytecodeCompiler) compileInverse(args string) error {
	if args == "" {
		return fmt.Errorf("INVERSE requires on/off argument")
	}

	// Compile inverse argument
	err := c.compileExpression(args)
	if err != nil {
		return fmt.Errorf("error compiling INVERSE argument: %v", err)
	}

	// Emit INVERSE instruction
	c.Emit(OP_INVERSE)
	return nil
}

// compileRandomize compiles RANDOMIZE statements
func (c *BytecodeCompiler) compileRandomize(args string) error {
	if args == "" {
		// RANDOMIZE without argument - use current time
		c.Emit(OP_RANDOMIZE)
		return nil
	}

	// Compile seed argument
	err := c.compileExpression(args)
	if err != nil {
		return fmt.Errorf("error compiling RANDOMIZE seed: %v", err)
	}

	// Emit RANDOMIZE instruction
	c.Emit(OP_RANDOMIZE)
	return nil
}

// compileFunction compiles function calls and other commands
func (c *BytecodeCompiler) compileFunction(command, args string) error {
	// Check if this is a supported TinyBASIC extension command
	extendedCommands := map[string]bool{
		"CALL":   true,
		"LOAD":   true,
		"SAVE":   true,
		"OPEN":   true,
		"CLOSE":  true,
		"MCP":    true,
		"NEW":    true,
		"LIST":   true,
		"EDITOR": true,
		"VARS":   true,
		"DIR":    true,
	}

	// If it's an extended command, emit a fallback instruction
	if extendedCommands[strings.ToUpper(command)] {
		// For extended commands, we'll emit a special fallback instruction
		// that will cause the VM to fall back to interpreted execution
		c.Emit(OP_CALL_FUNC, "FALLBACK_TO_INTERPRETED", command+" "+args)
		return nil
	}

	// For basic math functions and other simple functions, compile normally
	if args != "" {
		err := c.compileExpression(args)
		if err != nil {
			return err
		}
	}

	// Call function
	c.Emit(OP_CALL_FUNC, command, args)
	return nil
}

// compileExpression compiles BASIC expressions to bytecode using the full parser
func (c *BytecodeCompiler) compileExpression(expr string) error {
	return c.CompileExpression(expr)
}

// compileComplexExpression handles complex expressions with operators
func (c *BytecodeCompiler) compileComplexExpression(expr string) error {
	// This function is deprecated - use the full expression parser instead
	// Delegate to the complete expression parser
	return c.CompileExpression(expr)
}

// isValidVariableName checks if a string is a valid variable name
func (c *BytecodeCompiler) isValidVariableName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// Check first character is letter
	first := name[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z')) {
		return false
	}

	// Check remaining characters are alphanumeric
	for i := 1; i < len(name); i++ {
		char := name[i]
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') || char == '$') {
			return false
		}
	}

	return true
}

// splitStatements splits a line into individual statements, handling string literals properly
func (c *BytecodeCompiler) splitStatements(line string) []string {
	statements := make([]string, 0)
	current := ""
	inString := false
	escaped := false

	for _, char := range line {
		switch char {
		case '\\':
			if inString && !escaped {
				escaped = true
				current += string(char)
				continue
			}
		case '"':
			if !escaped {
				inString = !inString
			}
			current += string(char)
		case ':':
			if !inString && !escaped {
				trimmed := strings.TrimSpace(current)
				if trimmed != "" {
					statements = append(statements, trimmed)
				}
				current = ""
				continue
			}
			current += string(char)
		default:
			current += string(char)
		}
		escaped = false
	}

	// Add final statement
	if trimmed := strings.TrimSpace(current); trimmed != "" {
		statements = append(statements, trimmed)
	}

	return statements
}

// String returns string representation of an instruction
func (inst Instruction) String() string {
	switch inst.OpCode {
	case OP_PUSH_NUM, OP_PUSH_STR:
		return fmt.Sprintf("%s %v", inst.OpCode, inst.Operand1)
	case OP_LOAD_VAR, OP_STORE_VAR:
		return fmt.Sprintf("%s %s", inst.OpCode, inst.Operand1)
	case OP_JUMP, OP_JUMP_IF, OP_JUMP_UNLESS, OP_CALL:
		return fmt.Sprintf("%s %v", inst.OpCode, inst.Operand1)
	default:
		return string(inst.OpCode)
	}
}

// compilePhysics compiles PHYSICS statements
func (c *BytecodeCompiler) compilePhysics(args string) error {
	// PHYSICS commands need to be delegated to the TinyBASIC interpreter
	// because they involve complex state management and frontend communication
	// We emit a fallback instruction that will be handled by the VM
	
	// Add the arguments string to constants and push it onto the stack
	c.EmitConstant(args)
	// Call the PHYSICS function with 1 argument (function name, arg count)
	c.Emit(OP_CALL_FUNC, "PHYSICS", 1)
	return nil
}

// String returns string representation of an opcode
func (op OpCode) String() string {
	names := []string{
		"ADD", "SUB", "MUL", "DIV", "MOD", "POW", "NEG",
		"EQ", "NE", "LT", "LE", "GT", "GE",
		"AND", "OR", "NOT",
		"PUSH_NUM", "PUSH_STR", "LOAD_VAR", "STORE_VAR", "POP",
		"JUMP", "JUMP_IF", "JUMP_UNLESS", "CALL", "RETURN",
		"FOR_INIT", "FOR_CHECK", "FOR_NEXT",
		"PRINT", "PRINT_NL", "INPUT",
		"HALT", "NOP", "SOUND", "WAIT", "NOISE", "BEEP", "CLS", "MUSIC", "SPEAK", "PLOT", "LINE", "RECT", "CIRCLE", "SPRITE", "VECTOR", "SAY", "LOCATE", "COLOR", "KEY", "DATA", "READ", "DIM", "TEXTGFX", "CLEARGRAPHICS", "INVERSE", "RANDOMIZE", "DEBUG",
		"CALL_FUNC", "STR_CONCAT", "STR_LEN", "STR_MID",
	}

	if int(op) < len(names) {
		return names[op]
	}
	return fmt.Sprintf("UNKNOWN(%d)", op)
}
