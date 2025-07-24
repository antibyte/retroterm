package tinybasic

import (
	"context" // Used only by toJSON helper (currently unused but kept)
	"fmt"
	"log"
	"math/rand"
	"os" // Added for debugFP
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/antibyte/retroterm/pkg/logger"

	"errors" // Import für errors.Is und errors.As

	"github.com/antibyte/retroterm/pkg/shared"
	"github.com/antibyte/retroterm/pkg/tinyos"
)

// Helper function for TinyBASIC debug logging that respects configuration
func tinyBasicDebugLog(format string, args ...interface{}) {
	logger.Debug(logger.AreaTinyBasic, format, args...)
}

// Performance Optimizations: String Interning, Compiled Patterns, Expression Caching, and String Builder Pooling
var (
	// String interning cache for frequently used strings
	internedStrings = make(map[string]string)
	internMutex     sync.RWMutex
	
	// Compiled regex patterns for statement splitting (compiled once, reused many times)
	colonSplitPattern = regexp.MustCompile(`([^"]*"[^"]*")*[^"]*:`)
	remPattern        = regexp.MustCompile(`^\s*REM\b`)
	ifThenPattern     = regexp.MustCompile(`^\s*IF\s+.*\s+THEN\s+`)
	
	// Expression caching for repeated evaluations (especially in loops)
	expressionCache = make(map[string]BASICValue)
	exprCacheMutex  sync.RWMutex
	exprCacheHits   int64
	exprCacheMisses int64
	
	// String builder pool for reducing allocations
	stringBuilderPool = sync.Pool{
		New: func() interface{} {
			sb := &strings.Builder{}
			sb.Grow(256) // Pre-allocate reasonable capacity
			return sb
		},
	}
	
	// Variable lookup cache for frequently accessed variables
	varCache      = make(map[string]string) // Maps raw names to normalized names
	varCacheMutex sync.RWMutex
)

// internString returns an interned version of the string to reduce memory usage
func internString(s string) string {
	if len(s) > 50 { // Don't intern very long strings
		return s
	}
	
	internMutex.RLock()
	if interned, exists := internedStrings[s]; exists {
		internMutex.RUnlock()
		return interned
	}
	internMutex.RUnlock()
	
	internMutex.Lock()
	defer internMutex.Unlock()
	
	// Double-check after acquiring write lock
	if interned, exists := internedStrings[s]; exists {
		return interned
	}
	
	// Limit cache size to prevent memory leaks
	if len(internedStrings) > 1000 {
		// Clear cache if it gets too large
		internedStrings = make(map[string]string)
	}
	
	internedStrings[s] = s
	return s
}

// BASICValue object pool for reducing memory allocations
var basicValuePool = sync.Pool{
	New: func() interface{} {
		return &BASICValue{}
	},
}

// getBASICValue gets a BASICValue from the pool
func getBASICValue() *BASICValue {
	return basicValuePool.Get().(*BASICValue)
}

// returnBASICValue returns a BASICValue to the pool after clearing it
func returnBASICValue(v *BASICValue) {
	if v != nil {
		// Clear the value before returning to pool
		v.NumValue = 0
		v.StrValue = ""
		v.IsNumeric = false
		basicValuePool.Put(v)
	}
}

// Expression caching functions
func getCachedExpression(expr string) (BASICValue, bool) {
	exprCacheMutex.RLock()
	value, exists := expressionCache[expr]
	exprCacheMutex.RUnlock()
	
	if exists {
		exprCacheHits++
		return value, true
	}
	exprCacheMisses++
	return BASICValue{}, false
}

func setCachedExpression(expr string, value BASICValue) {
	if len(expr) > 100 { // Don't cache very long expressions
		return
	}
	
	exprCacheMutex.Lock()
	defer exprCacheMutex.Unlock()
	
	// Limit cache size to prevent memory leaks
	if len(expressionCache) > 500 {
		// Clear cache if it gets too large
		expressionCache = make(map[string]BASICValue)
	}
	
	expressionCache[expr] = value
}

func clearExpressionCache() {
	exprCacheMutex.Lock()
	defer exprCacheMutex.Unlock()
	expressionCache = make(map[string]BASICValue)
}

// String builder pool functions
func getStringBuilder() *strings.Builder {
	return stringBuilderPool.Get().(*strings.Builder)
}

func returnStringBuilder(sb *strings.Builder) {
	if sb != nil {
		sb.Reset()
		stringBuilderPool.Put(sb)
	}
}

// Variable name caching functions
func getCachedVarName(rawName string) string {
	varCacheMutex.RLock()
	normalized, exists := varCache[rawName]
	varCacheMutex.RUnlock()
	
	if exists {
		return normalized
	}
	
	// Cache miss - normalize and cache
	normalized = strings.ToUpper(rawName)
	
	varCacheMutex.Lock()
	defer varCacheMutex.Unlock()
	
	// Limit cache size
	if len(varCache) > 1000 {
		varCache = make(map[string]string)
	}
	
	varCache[rawName] = normalized
	return normalized
}

// Helper functions for creating BASICValues from the pool
func newNumericBASICValue(num float64) BASICValue {
	return BASICValue{NumValue: num, IsNumeric: true}
}

func newStringBASICValue(str string) BASICValue {
	return BASICValue{StrValue: str, IsNumeric: false}
}

// ErrExit wird zurückgegeben, wenn der EXIT-Befehl ausgeführt wird.
var ErrExit = errors.New("EXIT command executed")

// cleanCodeForLoading removes non-printable characters except newlines from code
// This prevents parsing issues caused by invisible characters in MCP-generated or editor-created code
func cleanCodeForLoading(content string) string {
	var cleaned strings.Builder

	for _, r := range content {
		// Keep printable characters (including Unicode like üöä), newlines, and essential whitespace
		if r == '\n' || r == '\r' || r == '\t' || unicode.IsPrint(r) {
			cleaned.WriteRune(r)
		}
		// Skip all other non-printable characters (including zero-width spaces, etc.)
	}

	result := cleaned.String()

	// Normalize line endings
	result = strings.ReplaceAll(result, "\r\n", "\n")
	result = strings.ReplaceAll(result, "\r", "\n")

	// Remove any trailing/leading whitespace from each line but preserve structure
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t") // Remove trailing whitespace but keep leading
		// Normalize BASIC code while preserving strings
		lines[i] = normalizeBASICLine(lines[i])
	}

	return strings.Join(lines, "\n")
}

// normalizeBASICLine converts BASIC commands and variables to uppercase while preserving string literals
func normalizeBASICLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}

	var result strings.Builder
	inString := false
	var quote rune

	for _, r := range line {
		if !inString {
			// Outside of string literals - check for quote start
			if r == '"' || r == '\'' {
				inString = true
				quote = r
				result.WriteRune(r)
			} else {
				// Convert to uppercase outside of strings
				result.WriteRune(unicode.ToUpper(r))
			}
		} else {
			// Inside string literal - preserve case and check for end
			result.WriteRune(r)
			if r == quote {
				// Check if it's an escaped quote (not implemented in TinyBASIC, so simple end)
				inString = false
			}
		}
	}

	return result.String()
}

// ResetExecutionState clears only the execution-related state, not the loaded program.
func (b *TinyBASIC) ResetExecutionState() {
	b.mu.Lock()
	b.variables = make(map[string]BASICValue)
	b.initializeKeyConstants() // Tastaturkonstanten nach Reset wiederherstellen
	b.currentLine = 0
	b.running = false
	b.inputVar = ""
	b.waitingForMCPInput = false // Clear MCP input flag
	b.pendingMCPCode = ""        // Clear pending MCP code
	b.pendingMCPFilename = ""    // Clear pending MCP filename
	b.gosubStack = b.gosubStack[:0]
	b.forLoops = b.forLoops[:0]
	b.forLoopIndexMap = make(map[string]int) // Clear loop index map
	// Clear any cached expressions when resetting execution state
	clearExpressionCache()
	// Reset performance counters
	b.loopIterationCount = 0
	b.dataPointer = 0
	b.printCursorOnSameLine = false
	b.inputControlEnableSent = false
	// GOTO Cleanup Counter zurücksetzen
	b.gotoCleanupCount = make(map[string]int)
	// Reset bytecode VM state
	if b.bytecodeVM != nil {
		b.bytecodeVM.Reset()
	}
	// Kontext für RUN erneuern
	b.ctx, b.cancel = context.WithCancel(context.Background())
	b.mu.Unlock()
}

// TinyBASIC is the main interpreter struct.
// Encapsulates all state and provides methods for interaction.
type TinyBASIC struct {
	lastSayText string // Zuletzt gesprochener Text für Timeout-Schätzung
	// Dependencies and Configuration (External)
	os *tinyos.TinyOS // Reference to the underlying OS (optional).
	fs FileSystem     // Filesystem interface implementation.

	// Communication (External)
	OutputChan chan shared.Message // Channel for sending messages to the frontend.
	// Interpreter State (Internal, protected by mu)
	program                  map[int]string        // Stores program lines (line number -> code).
	variables                map[string]BASICValue // Stores variable values (name -> value).
	programLines             []int                 // Sorted list of program line numbers for efficient lookup and LIST.
	
	// Bytecode compilation and execution
	compiledProgram          *BytecodeProgram      // Compiled bytecode version of the program
	bytecodeVM               *BytecodeVM           // Virtual machine for bytecode execution
	useBytecode              bool                  // Flag to enable/disable bytecode execution
	compiledHash             string                // Hash of compiled program to detect changes
	currentLine              int                   // The line number currently being executed (0 if not running).
	inputVar                 string                // Name of the variable waiting for INPUT, empty otherwise.
	inputPC                  int                   // Program counter for bytecode VM to resume after INPUT
	forLoops                 []ForLoopInfo         // Stack for tracking active FOR loops.
	forLoopIndexMap          map[string]int        // Maps variable names to forLoops indices for O(1) lookup
	gosubStack               []int                 // Stack for tracking GOSUB return points (renamed from runningStack).
	data                     []string              // Stores DATA statement values, populated by rebuildData.
	dataPointer              int                   // Current position within the data items for READ.
	openFiles                map[int]*OpenFile     // Map of active file handles (handle number -> OpenFile).
	nextHandle               int                   // Next available file handle number (auto-incrementing).
	running                  bool                  // Flag indicating if a program is currently executing via RUN.
	sessionID                string                // Identifier for filesystem operations (e.g., user session).
	termCols                 int                   // Terminal width (for PRINT formatting).
	termRows                 int                   // Terminal height.
	printCursorOnSameLine    bool                  // Flag indicating if cursor should stay on same line (for semicolon behavior)
	currentSubStatementIndex int                   // Current index in colon-separated statements for FOR-NEXT loops
	debugFP                  *os.File              // File pointer for debug logging

	// Concurrency Control (Internal)
	mu     sync.Mutex         // Protects access to all interpreter state above.
	ctx    context.Context    // Context for managing execution lifecycle and cancellation.
	cancel context.CancelFunc // Function to cancel the current execution context (for RUN).

	// Sprachausgabe-Synchronisation
	waitForSpeechDone int       // ID der Sprachausgabe, auf deren Abschluss gewartet wird (0 = keine)
	speechTimeout     time.Time // Zeitpunkt, zu dem das Timeout für die Sprachausgabe abläuft
	lastSpeechID      int       // Fortlaufende ID für Sprachausgaben (atomarer Zähler)
	lastSayDoneID     int64     // Zuletzt empfangene SAY_DONE ID vom Frontend
	// SAY command synchronization
	waitingForSayDone bool          // Flag: wartet auf SAY_DONE
	sayWaitChan       chan struct{} // Channel für Synchronisation
	// Index für den Neustart der Sub-Statement-Verarbeitung innerhalb einer Zeile
	resumeSubStatementIndex int
	
	// Expression Token Caching for Performance Optimization
	exprTokenCache *ExpressionTokenCache // Cache for tokenized expressions
	
	// JIT Compiler for Hot Loop Optimization
	jitCompiler   *JITCompiler   // Just-In-Time compiler for performance-critical loops (DISABLED)
	simpleJIT     *SimpleJIT     // Simple in-memory JIT compiler for better performance (DISABLED)
	expressionJIT *ExpressionJIT // Safe, non-invasive expression-level JIT compiler
	// SAY/SAY_DONE Synchronisation mit IDs
	sayID        int64            // Fortlaufende Nummer für SAY/SAY_DONE (alt)
	waitingSayID int64            // ID, auf die aktuell gewartet wird (alt)	// INKEY$ Support - Channel-basierte thread-safe Implementierung
	currentKey   string           // Aktuell gedrückte Taste (nur für interne Verwendung)
	keyChannel   chan string      // Channel für Key-Updates
	keyRequests  chan chan string // Channel für INKEY$-Anfragen

	// Erweiterte Tastaturstatus-Tracking für Spielsteuerung
	keyStates    map[string]bool // Status aller Tasten (gedrückt/nicht gedrückt)
	lastKeyEvent time.Time       // Zeitstempel des letzten Tastenereignisses

	// Rate Limiting für SAY-Befehle
	sayCommandTimestamps []time.Time
	maxSayRatePerSecond  int

	// GOTO Cleanup Protection
	gotoCleanupCount map[string]int // Counter für GOTO-Cleanup-Aufrufe zur Endlosschleifen-Erkennung

	// Rate Limiting für NOISE-Befehle
	noiseCommandTimestamps []time.Time
	maxNoiseRatePerSecond  int
	
	// Performance optimization counters
	loopIterationCount       int                   // Count iterations since last context check
	contextCheckInterval     int                   // How often to check context (default: every 1000 iterations)

	// Text-Cursor und Text-Attribute
	cursorX         int  // Aktuelle Cursor X-Position (0-basiert)
	cursorY         int  // Aktuelle Cursor Y-Position (0-basiert)
	inverseTextMode bool // Flag für inversen Text-Modus

	// Flag um mehrfache INPUT_CONTROL enable Nachrichten zu verhindern
	inputControlEnableSent bool // Wird auf true gesetzt wenn INPUT_CONTROL enable gesendet wurde
	// MCP Rate-Limiting (Internal)
	mcpUserUsage       map[string][]time.Time // Pro User: Liste der MCP-Nutzungszeiten in den letzten 24h
	mcpSystemUsage     []time.Time            // Systemweit: Liste aller MCP-Nutzungszeiten am aktuellen Tag
	mcpMutex           sync.Mutex             // Schützt den Zugriff auf MCP-Nutzungsdaten	// MCP Generated Code Storage (Internal)
	pendingMCPCode     string                 // Temporarily stores generated MCP code until filename is provided
	pendingMCPFilename string                 // Stores the original filename for MCP edit operations
	waitingForMCPInput bool                   // Flag indicating if we're waiting for MCP filename input
	waitingInput       bool                   // Flag indicating if we're waiting for any user input

	// Sprite Batching System for Performance
	spriteBatch      []shared.Message // Batch of sprite updates to send together
	spriteBatchTimer *time.Timer      // Timer for automatic batch sending
	spriteBatchMutex sync.Mutex       // Protects sprite batch operations
	batchingEnabled  bool             // Flag to enable/disable batching

	// Autorun callback for returning to TinyOS after program completion
	onProgramEnd func() // Optional callback executed when program ends
}

// MCP Rate-Limiting Konstanten
const (
	MCPUserDailyLimit   = 10  // Max 10 MCP-Nutzungen pro User in 24h
	MCPSystemDailyLimit = 250 // Max 250 MCP-Nutzungen systemweit pro Tag
)

// BASICValue represents a value within the BASIC interpreter (number or string).
type BASICValue struct {
	NumValue  float64 // Numeric value (if IsNumeric is true).
	StrValue  string  // String value (if IsNumeric is false).
	IsNumeric bool    // Flag indicating whether the value is numeric or string.
}

// NewTinyBASIC creates and initializes a new TinyBASIC interpreter instance.
func NewTinyBASIC(osys *tinyos.TinyOS) *TinyBASIC {
	ctx, cancel := context.WithCancel(context.Background())

	var fs FileSystem
	if osys != nil && osys.Vfs != nil {
		fs = osys.Vfs // Use VFS from TinyOS as the filesystem provider.
	} // else: still nil, but no log output

	// Attempt to open or create the debug log file
	debugFile, err := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		debugFile = nil // Ensure it's nil if opening failed
	}

	b := &TinyBASIC{
		os:           osys,
		fs:           fs,
		program:      make(map[int]string),
		variables:    make(map[string]BASICValue),
		programLines: make([]int, 0),
		openFiles:    make(map[int]*OpenFile),
		forLoops:        make([]ForLoopInfo, 0, MaxForLoopDepth/2), // Pre-allocate reasonably
		forLoopIndexMap: make(map[string]int),                       // Initialize loop index map
		gosubStack:   make([]int, 0, MaxGosubDepth/2),           // Pre-allocate reasonably
		data:         make([]string, 0),
		OutputChan:   make(chan shared.Message, OutputChannelBufferSize), termCols: DefaultTermCols,
		termRows:                DefaultTermRows,
		printCursorOnSameLine:   false, // Initially cursor is at start of line
		nextHandle:              1,     // File handles start from 1
		ctx:                     ctx,
		cancel:                  cancel,
		sayWaitChan:             make(chan struct{}, 1),
		resumeSubStatementIndex: 0, sayCommandTimestamps: make([]time.Time, 0), // Initialisieren
		maxSayRatePerSecond:    2,                            // Maximal 2 SAY-Befehle pro Sekunde
		noiseCommandTimestamps: make([]time.Time, 0),         // Initialisieren für NOISE
		maxNoiseRatePerSecond:  10,                           // Maximal 10 NOISE-Befehle pro Sekunde
		cursorX:                0,                            // Cursor X-Position (0-basiert)
		cursorY:                0,                            // Cursor Y-Position (0-basiert)
		inverseTextMode:        false,                        // Normaler Text-Modus
		inputControlEnableSent: false,                        // Initialwert für das Flag
		keyStates:              make(map[string]bool),        // Initialisiere die Tastaturstatus-Map
		debugFP:                debugFile,                    // Assign the file pointer		gotoCleanupCount:       make(map[string]int),         // Initialize GOTO Cleanup Protection
		mcpUserUsage:           make(map[string][]time.Time), // Initialize MCP User Usage
		mcpSystemUsage:         make([]time.Time, 0),         // Initialize MCP System Usage		pendingMCPCode:         "",                           // Initialize MCP pending code
		pendingMCPFilename:     "",                           // Initialize MCP pending filename
		waitingForMCPInput:     false,                        // Initialize MCP input flag		// Sprite Batching System for Performance
		spriteBatch:            make([]shared.Message, 0),
		spriteBatchTimer:       nil,          // Will be initialized when needed
		spriteBatchMutex:       sync.Mutex{}, // Initialize mutex
		batchingEnabled:        true,         // Enable batching by default
		contextCheckInterval:   1000,         // Check context every 1000 loop iterations for performance
		
		// Bytecode compilation and execution
		useBytecode:            true,         // Enable bytecode by default for performance
		compiledHash:           "",           // No program compiled yet
		
		// Expression Token Caching
		exprTokenCache:         NewExpressionTokenCache(1000, 5*time.Minute), // Cache 1000 expressions for 5 minutes
		
		// JIT Compiler (disabled by default, can be enabled for performance testing)
		jitCompiler:            NewJITCompiler(), // JIT compiler for hot loop optimization (DISABLED)
		simpleJIT:              NewSimpleJIT(),  // Simple JIT compiler for better performance (DISABLED)
		expressionJIT:          NewExpressionJIT(), // Safe expression-level JIT compiler
	}

	// Seed the random number generator once
	//nolint:staticcheck // Using Seed is acceptable for simple non-crypto random numbers needed here.
	rand.Seed(time.Now().UnixNano())

	b.noiseCommandTimestamps = make([]time.Time, 0) // Initialisieren für NOISE
	b.maxNoiseRatePerSecond = 10                    // Maximal 10 NOISE-Befehle pro Sekunde
	// Initialisiere Tastenkonstanten für INKEY$
	b.initializeKeyConstants()

	// DEBUG: Log der initialisierten Konstanten
	// tinyBasicDebugLog("KEYESC initialized = '%s' (len=%d)", b.variables["KEYESC"].StrValue, len(b.variables["KEYESC"].StrValue))
	// tinyBasicDebugLog("KEYLEFT initialized = '%s' (len=%d)", b.variables["KEYLEFT"].StrValue, len(b.variables["KEYLEFT"].StrValue))
	// tinyBasicDebugLog("KEYCURLEFT initialized = '%s' (len=%d)", b.variables["KEYCURLEFT"].StrValue, len(b.variables["KEYCURLEFT"].StrValue))
	// tinyBasicDebugLog("KEYSPACE initialized = '%s' (len=%d)", b.variables["KEYSPACE"].StrValue, len(b.variables["KEYSPACE"].StrValue))

	// INKEY$ Variable initialisieren (leer)
	b.variables["INKEY$"] = BASICValue{StrValue: "", IsNumeric: false}

	// Initialize bytecode VM
	b.bytecodeVM = NewBytecodeVM(b)

	return b
}

// SetSessionID sets the session identifier used for filesystem operations.
func (b *TinyBASIC) SetSessionID(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sessionID = sessionID
}

// Setzt die zuletzt empfangene SAY_DONE-ID (wird vom Terminal-Handler gesetzt)
func (b *TinyBASIC) SetLastSayDoneID(id int64) {
	b.mu.Lock()
	b.lastSayDoneID = id
	b.mu.Unlock()
}

// SetTerminalDimensions updates the interpreter's knowledge of the terminal size.
func (b *TinyBASIC) SetTerminalDimensions(cols, rows int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Basic validation for terminal dimensions
	if cols > 0 {
		b.termCols = cols
	} else {
		b.termCols = DefaultTermCols // Fallback to default if invalid
	}
	if rows > 0 {
		b.termRows = rows
	} else {
		b.termRows = DefaultTermRows // Fallback to default if invalid
	}
}

// HandleSayDone wird vom Terminal-Handler aufgerufen, wenn eine SAY_DONE-Nachricht empfangen wird.
func (b *TinyBASIC) HandleSayDone(sayID int) {
	b.mu.Lock()

	if b.waitingForSayDone && b.waitForSpeechDone == sayID {
		b.waitingForSayDone = false // Zurücksetzen des Flags
		b.waitForSpeechDone = 0     // Zurücksetzen der erwarteten ID

		// Non-blocking send to sayWaitChan to avoid deadlock if runProgramInternal is not waiting
		// This can happen if StopExecution was called or the program ended for other reasons.
		select {
		case b.sayWaitChan <- struct{}{}:
			// Erfolgreich gesendet
		}
	}
	b.mu.Unlock()
}

// GetOutputChannel returns the read-only output channel for external components.
func (b *TinyBASIC) GetOutputChannel() <-chan shared.Message {
	return b.OutputChan // No lock needed, channel receive is safe
}

// IsRunning safely checks if a BASIC program is currently executing via RUN.
func (b *TinyBASIC) IsRunning() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.running
}

// IsWaitingForInput safely checks if the interpreter is paused waiting for user input.
func (b *TinyBASIC) IsWaitingForInput() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	waiting := b.inputVar != "" || b.waitingForMCPInput
	if waiting {
		tinyBasicDebugLog("[WAITING] IsWaitingForInput=true, inputVar='%s', waitingForMCPInput=%v", b.inputVar, b.waitingForMCPInput)
	}
	return waiting
}

// StopExecution forcefully stops the currently running program execution (async RUN).
// Returns the messages that were sent (BREAK, INPUT_CONTROL, etc.)
func (b *TinyBASIC) StopExecution() []shared.Message {
	b.mu.Lock() // Lock for state modification

	// Cancel the context to signal the running goroutine
	if b.cancel != nil {
		b.cancel()
	}
	// Reset execution state immediately
	wasRunning := b.running
	b.running = false
	b.currentLine = 0
	b.inputVar = ""                 // Clear pending input
	b.waitingForMCPInput = false    // Clear MCP input flag
	b.pendingMCPCode = ""           // Clear pending MCP code
	b.pendingMCPFilename = ""       // Clear pending MCP filename
	b.gosubStack = b.gosubStack[:0] // Clear stacks
	b.forLoops = b.forLoops[:0]

	// Create a new context for potential future RUN commands
	b.ctx, b.cancel = context.WithCancel(context.Background())
	// Ensure files opened during the run are closed
	b.closeAllFiles() // Assumes lock is held

	var messages []shared.Message

	// Stop any playing SID music when execution is stopped
	if wasRunning {
		musicStopMsg := shared.Message{
			Type: shared.MessageTypeSound,
			Params: map[string]interface{}{
				"action": "music_stop",
			},
		}
		b.sendMessageObject(musicStopMsg)
		messages = append(messages, musicStopMsg)
	}

	// Ensure terminal input is re-enabled if it was stopped by INPUT or RUN
	log.Printf("[DEBUG] StopExecution: Sende INPUT_CONTROL enable")
	enableMsg := shared.Message{Type: shared.MessageTypeInputControl, Content: "enable", SessionID: b.sessionID}
	b.sendMessageObject(enableMsg)
	messages = append(messages, enableMsg)
	b.inputControlEnableSent = true // Markiere, dass enable bereits gesendet wurde

	b.mu.Unlock()

	// Send BREAK message only if a program was actually running
	if wasRunning {
		breakMsg := shared.Message{Type: shared.MessageTypeText, Content: "BREAK", SessionID: b.sessionID}
		b.sendMessageObject(breakMsg)
		messages = append(messages, breakMsg)
	}

	return messages
}

// Reset clears the entire interpreter state and prepares it for fresh use.
func (b *TinyBASIC) Reset() {
	b.mu.Lock() // Lock for state modification

	// Stop any ongoing execution first
	if b.cancel != nil {
		b.cancel()
	}

	// Clear program, variables, and execution state
	b.program = make(map[int]string)
	b.programLines = make([]int, 0)
	b.variables = make(map[string]BASICValue)
	b.currentLine = 0
	b.running = false
	b.inputVar = ""
	b.gosubStack = b.gosubStack[:0]
	b.forLoops = b.forLoops[:0]
	b.data = make([]string, 0)
	b.dataPointer = 0

	// Close files and reset file handling state
	b.closeAllFiles() // Assumes lock is held
	b.openFiles = make(map[int]*OpenFile)
	b.nextHandle = 1

	// Reset terminal dimensions to default (optional, maybe keep user setting?)
	// b.termCols = DefaultTermCols
	// b.termRows = DefaultTermRows

	// Recreate context
	b.ctx, b.cancel = context.WithCancel(context.Background())

	// Keep OutputChan, sessionID, fs, os

	b.mu.Unlock()

	b.sendMessageWrapped(shared.MessageTypeText, "Ready.") // Simple ready message
}

// SetOnProgramEnd sets a callback function that is called when a program ends
func (b *TinyBASIC) SetOnProgramEnd(callback func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onProgramEnd = callback
}

// ExecuteCommand is a placeholder from the original code, not fully implemented.
// It seems intended for external command execution, maybe from the OS?
// For now, it just echoes the input.
func (b *TinyBASIC) ExecuteCommand(input string, sessionID string) []shared.Message {
	b.SetSessionID(sessionID) // Use the setter to be safe
	// Return an echo or a "not implemented" message
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("Command '%s' not implemented.", input)},
	}
}

// Execute processes a single line of input (direct command or program line).
// Returns messages for immediate display (e.g., direct mode errors or OK).
// Program output during RUN is sent via OutputChan.
func (b *TinyBASIC) Execute(input string) []shared.Message {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil // Ignore empty input
	} // Handle __BREAK__ immediately, it needs to interrupt even if locked elsewhere momentarily.
	if input == "__BREAK__" {
		stopMessages := b.StopExecution()
		return stopMessages // Return the stop messages directly
	}

	b.mu.Lock() // Lock for state checks and modifications

	// If waiting for input, delegate to ExecuteInputResponse.
	if b.inputVar != "" {
		// ExecuteInputResponse handles locking/unlocking and message sending.
		b.mu.Unlock()
		return b.ExecuteInputResponse(input) // Returns nil, sends messages via channel.
	}

	// If a program is running via RUN, reject other commands.
	if b.running {
		b.mu.Unlock()
		return FormatErrorAsMessages(WrapError(ErrProgramAlreadyRunning, "", true, 0))
	}

	// --- Direct Mode or Program Line Input ---

	// Check if it's a program line (number + code).
	if lineNum, code, isLine := parseProgramLine(input); isLine {
		if code == "" {
			// Delete line.
			delete(b.program, lineNum)
		} else {
			code = upperOutsideQuotes(code)
			b.program[lineNum] = code
		}
		// Rebuild internal structures after modification.
		b.rebuildProgramLines() // Assumes lock held
		b.rebuildData()         // Assumes lock held
		b.mu.Unlock()
		return []shared.Message{{Type: shared.MessageTypeText, Content: "OK"}}
	} // --- Direct Mode Command Execution ---
	// The interpreter is not currently RUNning a program here.
	b.currentLine = 0   // Direct mode doesn't have a persistent line number context.
	currentCtx := b.ctx // Use the main context (though direct commands are synchronous).

	// Create a temporary channel to capture messages during direct execution
	originalOutputChan := b.OutputChan
	tempOutputChan := make(chan shared.Message, 100)
	b.OutputChan = tempOutputChan

	b.mu.Unlock() // Temporarily release lock before calling executeStatement

	_, err := b.executeStatement(input, currentCtx)

	// Restore the original output channel
	b.mu.Lock()
	b.OutputChan = originalOutputChan
	b.mu.Unlock()

	// Collect all messages from the temporary channel
	var collectedMessages []shared.Message
	close(tempOutputChan) // Close the temp channel to stop receiving
	for msg := range tempOutputChan {
		msg.SessionID = b.sessionID // Set session ID for collected messages
		collectedMessages = append(collectedMessages, msg)
	}

	// Process result/error from direct mode execution.
	if err != nil {
		// Special handling for EXIT command.
		if errors.Is(err, ErrExit) {
			// Return a specific message or signal that can be interpreted by the terminal handler
			// to switch modes. The actual mode switch message (MessageTypeMode) should be sent
			// by the terminal handler, not directly from here.
			return []shared.Message{{Type: shared.MessageTypeText, Content: ErrExit.Error()}}
		}
		// Special handling for HELP output, which returns messages in the error struct.
		if helpErr, ok := err.(*helpLinesAsError); ok {
			return helpErr.lines
		}
		return FormatErrorAsMessages(err)
	}
	// Check if this was a RUN command - if so, don't send OK immediately
	// since RUN executes asynchronously and will send OK when finished
	// Also exclude LOAD commands since they have their own OK handling
	inputUpper := strings.ToUpper(strings.TrimSpace(input))
	if inputUpper == "RUN" || strings.HasPrefix(inputUpper, "LOAD ") {
		return collectedMessages // No immediate OK for RUN or LOAD command
	}

	// Combine collected messages with success message
	result := collectedMessages
	result = append(result, shared.Message{Type: shared.MessageTypeText, Content: "OK"})
	return result
}

// ExecuteInputResponse handles user input provided after an INPUT statement paused execution.
// Returns nil because output/prompts are sent via OutputChan.
func (b *TinyBASIC) ExecuteInputResponse(input string) []shared.Message {
	b.mu.Lock() // Lock to modify interpreter state

	// Check if we're waiting for MCP filename input
	if b.waitingForMCPInput {
		return b.processMCPFilenameInput(input) // This will unlock mutex
	}

	if b.inputVar == "" {
		b.mu.Unlock()
		b.sendMessageWrapped(shared.MessageTypeText, "ERROR: "+ErrInputNotExpected.Error())
		return nil // Message sent via channel
	}

	varName := b.inputVar
	// Clear the input request flag *before* potentially resuming or erroring.
	b.inputVar = ""

	// Assign the input value to the variable.
	var assignErr error
	if strings.HasSuffix(varName, "$") { // String variable
		b.variables[strings.ToUpper(varName)] = BASICValue{StrValue: input, IsNumeric: false}
	} else { // Numeric variable
		// Attempt to parse the input as a float.
		val, err := strconv.ParseFloat(input, 64)
		if err != nil {
			// Invalid numeric input: Restore inputVar and prompt again.
			b.inputVar = varName // Restore the input request flag
			assignErr = WrapError(ErrInvalidExpression, "INPUT", false, b.currentLine)
		} else {
			b.variables[strings.ToUpper(varName)] = BASICValue{NumValue: val, IsNumeric: true}
		}
	}

	if assignErr != nil {
		// Failed assignment (invalid number) - prompt user again.
		b.mu.Unlock()                                                    // Unlock before sending message.
		b.sendMessageWrapped(shared.MessageTypeText, "?REDO FROM START") // Classic BASIC message
		return nil
	}

	// Input processed successfully, re-enable terminal input.
	b.sendInputControl("enable") // Assumes lock held

	// Check if a program was running and waiting for this input.
	if b.running {
		// Check if we're using bytecode execution and need to resume VM
		if b.useBytecode && b.bytecodeVM != nil && b.inputPC > 0 {
			// Resume bytecode VM execution from stored PC
			resumePC := b.inputPC
			b.inputPC = 0 // Clear stored PC
			b.mu.Unlock() // Unlock before resuming VM
			
			// Convert input to BASICValue
			var inputValue BASICValue
			if strings.HasSuffix(varName, "$") {
				inputValue = newStringBASICValue(input)
			} else {
				if val, err := strconv.ParseFloat(input, 64); err == nil {
					inputValue = newNumericBASICValue(val)
				} else {
					inputValue = newNumericBASICValue(0) // Default to 0 for invalid input
				}
			}
			
			// Resume VM execution
			go func() {
				err := b.bytecodeVM.Resume(resumePC, inputValue, varName)
				if err != nil {
					b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("Runtime error: %s", err.Error()))
				}
			}()
			return nil
		}
		
		// Traditional interpreted execution
		prevLine := b.currentLine
		nextLine, found := b.findNextLine(prevLine) // Assumes lock held
		if !found {
			b.running = false // Mark as not running
			b.currentLine = 0 // Set line to 0
			b.mu.Unlock()
			b.sendMessageWrapped(shared.MessageTypeText, "Program finished.")
			return nil
		}
		b.currentLine = nextLine
		b.mu.Unlock() // Unlock to allow the run loop goroutine to proceed.
	} else {
		b.mu.Unlock() // Unlock the state.
		b.sendMessageWrapped(shared.MessageTypeText, "OK")
	}

	return nil // All output/status sent via channel.
}

// parseProgramLine attempts to parse a string as a BASIC program line (number + code).
// Returns line number, code, and a boolean indicating success. Pure function.
func parseProgramLine(line string) (int, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return 0, "", false
	}

	spaceIdx := strings.IndexAny(line, " \t") // Find first space or tab

	if spaceIdx == -1 {
		// No space/tab, could be just a number (delete line).
		num, err := strconv.Atoi(line)
		if err == nil && num > 0 {
			return num, "", true // Valid line number, empty code means delete.
		}
		return 0, "", false // Not a number or not positive.
	}

	// Potential line number before the first space/tab.
	numStr := line[:spaceIdx]
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 {
		return 0, "", false // First part is not a valid positive line number.
	}
	// The rest is code.
	code := strings.TrimSpace(line[spaceIdx:])

	// Debug-Log für PRINT-Zeilen
	if strings.Contains(code, "PRINT") && strings.Contains(code, "R E T R O") {
		tinyBasicDebugLog("[PARSE] Original line: '%s'", line)
		tinyBasicDebugLog("[PARSE] Parsed code: '%s'", code)
	}

	return num, code, true
}

// rebuildProgramLines updates the sorted slice of line numbers from the program map.
// Must be called whenever b.program is modified. Assumes lock is held.
func (b *TinyBASIC) rebuildProgramLines() {
	// Efficiently reuse or create slice capacity.
	if cap(b.programLines) < len(b.program) {
		b.programLines = make([]int, 0, len(b.program))
	} else {
		b.programLines = b.programLines[:0] // Reuse existing slice.
	}
	for lineNum := range b.program {
		b.programLines = append(b.programLines, lineNum)
	}
	sort.Ints(b.programLines) // Keep sorted for efficient lookups.
}

// findNextLine finds the line number immediately following the given currentLine.
// Returns the next line number and true if found, or 0 and false if it's the last line.
// Uses binary search on the sorted programLines slice. Assumes lock is held.
func (b *TinyBASIC) findNextLine(currentLine int) (int, bool) {
	lines := b.programLines
	numLines := len(lines)
	if numLines == 0 {
		return 0, false
	}

	// Binary search for the first index `i` where lines[i] > currentLine.
	index := sort.Search(numLines, func(i int) bool { return lines[i] > currentLine })

	if index < numLines {
		return lines[index], true // Found a line number greater than currentLine.
	}

	return 0, false // No line found after currentLine.
}

// runProgramInternal is the core asynchronous execution loop for RUN.
// It executes statements line by line, handling control flow and context cancellation.
// Assumes initial state (running=true, currentLine set) is prepared by cmdRun.
func (b *TinyBASIC) runProgramInternal(ctx context.Context) {
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
	// Counter für periodische Checks (alle 1000 Iterationen)
	checkCounter := 0

	for { // CRITICAL: Context cancellation check to prevent deadlocks
		select {
		case <-ctx.Done():
			b.mu.Lock()
			b.running = false
			wasEnableSent := b.inputControlEnableSent
			callback := b.onProgramEnd // Get callback reference
			b.mu.Unlock()

			if !wasEnableSent {
				b.sendInputControl("enable")
			}
			b.sendMessageWrapped(shared.MessageTypeText, "EXECUTION CANCELLED")
			// Call callback if set
			if callback != nil {
				callback()
				// Clear callback after execution
				b.mu.Lock()
				b.onProgramEnd = nil
				b.mu.Unlock()
			}
			return
		default:
		} // Periodische Checks alle 50 Iterationen für bessere Responsiveness
		checkCounter++
		if checkCounter >= 50 {
			checkCounter = 0

			// Prüfe ob das System noch läuft und Benutzer verbunden ist
			// Hier könnten zusätzliche Checks für Benutzerverbindung hinzugefügt werden
			// Beispiel: if !b.isUserConnected() { return }

			// Kleine Pause für andere Goroutines und Garbage Collector
			time.Sleep(time.Millisecond)
		}

		// Execution Loop Batching: Reduce lock granularity by batching state operations
		b.mu.Lock()
		if !b.running {
			b.mu.Unlock()
			break
		}
		currentLine := b.currentLine
		code, ok := b.program[currentLine]
		b.mu.Unlock()

		// Early exit checks without additional locking
		if currentLine == 0 {
			b.mu.Lock()
			b.running = false
			b.mu.Unlock()
			break
		}
		if !ok {
			b.mu.Lock()
			b.running = false
			b.mu.Unlock()
			break
		} // Store the original line before execution for comparison
		originalLineBeforeExecution := currentLine
		nextLine, err := b.executeStatement(code, nil)
		if err != nil {
			b.mu.Lock()
			b.running = false
			terminatedLine := b.currentLine
			// CRITICAL: Reset input state on program abort to prevent "?REDO FROM START"
			b.inputVar = ""
			b.waitingForMCPInput = false
			b.pendingMCPCode = ""
			b.pendingMCPFilename = ""
			b.mu.Unlock()

			// Display program termination message with error
			if !errors.Is(err, ErrExit) {
				terminationMsg := fmt.Sprintf("PROGRAM TERMINATED IN LINE %d", terminatedLine)
				b.sendMessageWrapped(shared.MessageTypeText, terminationMsg)

				// Display the actual error message
				if basicErr, ok := err.(*BASICError); ok {
					b.sendMessageWrapped(shared.MessageTypeText, basicErr.Error())
				} else {
					b.sendMessageWrapped(shared.MessageTypeText, err.Error())
				}
			}
			break
		}

		// Check if END or STOP was executed (returns nextLine = 0)
		if nextLine == 0 {
			b.mu.Lock()
			b.running = false
			b.mu.Unlock()
			break
		}
		b.mu.Lock()
		waiting := b.waitingForSayDone
		stillRunning := b.running // Check running status while holding the lock
		b.mu.Unlock()

		// If program was stopped by END or error, exit immediately
		if !stillRunning {
			break
		}

		if waiting {
			timeout := 30 * time.Second
			select {
			case <-b.sayWaitChan:
				// ok
			case <-time.After(timeout):
			case <-b.ctx.Done():
				return
			}
			b.mu.Lock()
			b.waitingForSayDone = false
			b.mu.Unlock()
		}
		// Calculate next line while holding the lock
		b.mu.Lock()
		nextLineToUse := 0

		// Check if currentLine was changed by the command (e.g., GOTO, IF with jump)
		if b.currentLine != originalLineBeforeExecution {
			// Command changed the current line, use it
			nextLineToUse = b.currentLine
		} else if nextLine != originalLineBeforeExecution {
			// executeStatement returned a different line
			nextLineToUse = nextLine
		} else {
			// Normal execution: advance to next line
			nextLineFound, found := b.findNextLine(originalLineBeforeExecution)
			if found {
				nextLineToUse = nextLineFound
			} else {
				nextLineToUse = 0 // End of program
			}
		}

		b.currentLine = nextLineToUse
		b.mu.Unlock()
		time.Sleep(0)
	}
}

// executeStatement parses and executes a single BASIC statement.
// Acquires lock only for the duration of the specific command logic.
// Takes context for potential cancellation.
// isMultiCommand indicates if this is part of a multi-statement line
// subStatementIndex is the index of this statement in a multi-statement line (0-based)
func (b *TinyBASIC) executeStatement(line string, ctx context.Context) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Store the original currentLine to preserve it during sub-statement execution
	originalCurrentLine := b.currentLine

	startIndex := 0
	if b.resumeSubStatementIndex > 0 {
		startIndex = b.resumeSubStatementIndex
		b.resumeSubStatementIndex = 0
	}
	// Korrektur: splitStatementsByColon zu b.splitStatementsByColon
	subStatements := b.splitStatementsByColon(line)
	finalNextLine := b.currentLine // Deklaration hinzugefügt
	var finalError error           // Deklaration hinzugefügt

	for i := startIndex; i < len(subStatements); i++ {
		subStatement := subStatements[i]

		if ctx != nil {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
				// Fortfahren
			}
		}

		// Set current sub-statement index for FOR-NEXT loops
		b.currentSubStatementIndex = i

		// Restore the original line number before executing each sub-statement
		// This ensures that all sub-statement on the same line report the same line number		b.currentLine = originalCurrentLine

		// Die Sperre wird für die Dauer des Aufrufs gehalten
		nextLine, err := b.executeSingleStatementInternal(subStatement, ctx)

		if err != nil {
			// Spezielle Behandlung für ErrExit: Direkte Weitergabe ohne Formatierung.
			if errors.Is(err, ErrExit) {
				finalError = err
			} else {
				// NEUE LOGIK: Wenn es bereits ein *BASICError ist, direkt verwenden.
				if basicErr, ok := err.(*BASICError); ok {
					finalError = basicErr
				} else {
					// Andere Fehler wie gewohnt formatieren.
					finalError = b.FormatErrorForDisplay(err, subStatement)
				}
			}
			finalNextLine = 0
			break
		}
		finalNextLine = nextLine
		// Check if NEXT command set resumeSubStatementIndex (FOR loop continuation)
		if b.resumeSubStatementIndex > 0 {
			// Restart the loop from the specified index
			startIndex = b.resumeSubStatementIndex
			b.resumeSubStatementIndex = 0
			i = startIndex - 1 // Will be incremented by the loop
			continue
		}
		// --- Verbesserte Abbruchbedingung für Kontrollflussbefehle ---
		// Wenn sich b.currentLine (gesetzt durch GOTO, GOSUB, IF-THEN-GOTO etc. in executeSingleStatementInternal)
		// von der originalCurrentLine (der Zeilennummer der gesamten BASIC-Zeile) unterscheidet,
		// bedeutet das, dass ein Sprung stattgefunden hat und wir die Verarbeitung der aktuellen Multi-Statement-Zeile abbrechen müssen.
		if b.currentLine != originalCurrentLine {
			finalNextLine = b.currentLine // Der GOSUB/GOTO hat die nächste Zeile bereits gesetzt.
			break
		}

		// Originale, komplexere Abbruchlogik (kann ggf. später wieder integriert oder verfeinert werden)
		/*			// Only break if there's an actual control flow change, input needed, or SAY waiting
					// Don't break just because nextLine is different from currentLine (that's normal for sequential execution)
					if nextLine == 0 || b.inputVar != "" || b.waitingForSayDone {
						if b.waitingForSayDone && i+1 < len(subStatements) {
							b.resumeSubStatementIndex = i + 1
						}
						break
					}
					// Check for control flow changes that require immediate interruption
					// Only break if the next line is different AND it's not just advancing to the next sequential line
					nextPhysicalLine, _ := b.findNextLine(b.currentLine)
					if b.currentLine != finalNextLine && finalNextLine != nextPhysicalLine && b.currentLine != 0 {
						// This indicates a genuine control flow change (GOTO, GOSUB, etc.)
						// But NOT normal sequential progression or FOR loop continuation
						break
					}
		*/
	}

	return finalNextLine, finalError
}

// executeSingleStatementInternal führt eine einzelne Anweisung aus.
// Gibt die nächste Zeilennummer und einen Fehler zurück.
// Wichtig: Diese Funktion kann die b.mu-Sperre freigeben und wiedererlangen, insbesondere für Befehle wie SAY.
func (b *TinyBASIC) executeSingleStatementInternal(statement string, ctx context.Context) (int, error) {
	var err error // Variable für Fehlerbehandlung
	trimmedStatement := strings.TrimSpace(statement)
	if trimmedStatement == "" {
		// Leere Anweisung, zur nächsten Zeile gehen
		nl, _ := b.findNextLine(b.currentLine)
		return nl, nil
	}
	parts := strings.Fields(trimmedStatement)
	if len(parts) == 0 {
		// Sollte nicht passieren, wenn trimmedStatement nicht leer ist
		nl, _ := b.findNextLine(b.currentLine)
		return nl, nil
	}

	command := strings.ToUpper(parts[0])

	args := "" // args wird hier definiert
	if len(parts) > 1 {
		// WICHTIG: Preserve original spacing by finding the command and taking the rest
		commandLen := len(parts[0])
		// Find the first non-space character after the command
		spaceAfterCommand := strings.Index(trimmedStatement[commandLen:], " ")
		if spaceAfterCommand != -1 {
			args = trimmedStatement[commandLen+spaceAfterCommand+1:]
		}
	}
	// trimmedStatement ist ebenfalls hier im Scope verfügbar

	// Default next line is the physical next line, unless a command changes it.
	physicalNextLine, _ := b.findNextLine(b.currentLine)

	switch command {
	case "REM":
		return physicalNextLine, nil
	case "LET":
		err := b.cmdLet(trimmedStatement)
		return physicalNextLine, err
	case "PRINT", "PR.", "?":
		err := b.cmdPrint(args)
		return physicalNextLine, err
	case "INPUT":
		err := b.cmdInput(args)
		if err != nil {
			return 0, err
		}
		return b.currentLine, nil
	case "IF":
		err := b.cmdIf(args)
		if err != nil {
			return 0, err
		}
		return b.currentLine, nil
	case "GOTO":
		err := b.cmdGoto(args)
		if err != nil {
			return 0, err
		}
		return b.currentLine, nil
	case "GOSUB":
		err := b.cmdGosub(args)
		if err != nil {
			return 0, err
		}
		return b.currentLine, nil // GOSUB setzt b.currentLine
	case "RETURN":
		err := b.cmdReturn(args)
		if err != nil {
			return 0, err
		}
		return b.currentLine, nil // RETURN setzt b.currentLine
	case "FOR":
		nextSubStatementIndex, _ := b.getCurrentSubStatementInfo(trimmedStatement)
		err = b.cmdFor(args, nextSubStatementIndex)
		if err != nil {
			return 0, err
		}
		return b.currentLine, nil
	case "NEXT":
		err = b.cmdNext(args)
		if err != nil {
			return 0, err
		}
		return b.currentLine, nil
	case "END", "STOP":
		return 0, nil
	case "LIST":
		err := b.cmdList(args)
		return physicalNextLine, err
	case "EDITOR":
		err := b.cmdEditor(args)
		return physicalNextLine, err
	case "VARS":
		err := b.cmdVars(args)
		return physicalNextLine, err
	case "RUN":
		b.mu.Unlock()
		_, err := b.cmdRun(args)
		b.mu.Lock()
		if err != nil {
			return 0, err
		}
		return b.currentLine, nil
	case "NEW":
		b.cmdNew("")
		return physicalNextLine, nil
	case "CLEAR":
		b.cmdNew("")
		return physicalNextLine, nil
	case "LOAD":
		err := b.cmdLoad(args)
		return physicalNextLine, err
	case "SAVE":
		err := b.cmdSave(args)
		return physicalNextLine, err
	case "DIR":
		listing, err := b.cmdDir(args)
		if err != nil {
			return 0, err
		}
		b.sendMessageWrapped(shared.MessageTypeText, listing)
		return physicalNextLine, nil
	case "MCP":
		err := b.cmdMCP(args)
		if err != nil {
			return 0, err
		}
		return physicalNextLine, nil
	case "HELP":
		return 0, HandleHelpCommand(args)
	
	// JIT Compiler Commands
	case "JITON":
		err := b.cmdJITOn(args)
		return physicalNextLine, err
	case "JITOFF":
		err := b.cmdJITOff(args)
		return physicalNextLine, err
	case "JITSTATS":
		err := b.cmdJITStats(args)
		return physicalNextLine, err
	case "JITCLEAR":
		err := b.cmdJITClear(args)
		return physicalNextLine, err
	case "JITCONFIG":
		err := b.cmdJITConfig(args)
		return physicalNextLine, err
	case "JITBENCH":
		err := b.cmdJITBench(args)
		return physicalNextLine, err
		
	case "EXIT", "QUIT":
		// cmdExit now returns ErrExit. This error will be propagated up.
		// The Execute function will handle it and inform the terminal handler.
		return 0, b.cmdExit(args) // cmdExit returns ErrExit on success
	case "SAY", "SPEAK":
		_, err := b.cmdSpeak(args)
		if err != nil {
			return 0, err
		}
		if !b.waitingForSayDone {
			return physicalNextLine, nil
		}
		return b.currentLine, nil
	case "SOUND":
		err := b.cmdSound(args)
		if err != nil {
			return 0, err
		}
		return physicalNextLine, nil
	case "NOISE":
		err := b.cmdNoise(args)
		if err != nil {
			return 0, err
		}
		return physicalNextLine, nil
	case "BEEP":
		err := b.cmdBeep(args)
		return physicalNextLine, err
	case "MUSIC":
		err := b.cmdMusic(args)
		if err != nil {
			return 0, err
		}
		return physicalNextLine, nil
	case "CLS":
		err := b.cmdCls(args)
		return physicalNextLine, err
	case "LOCATE":
		err := b.cmdLocate(args)
		return physicalNextLine, err
	case "INVERSE":
		err := b.cmdInverse(args)
		return physicalNextLine, err
	case "PLOT":
		err := b.cmdPlot(args)
		return physicalNextLine, err
	case "LINE":
		err := b.cmdLine(args)
		return physicalNextLine, err
	case "RECT":
		err := b.cmdRect(args)
		return physicalNextLine, err
	case "CIRCLE":
		err := b.cmdCircle(args)
		return physicalNextLine, err
	// FILL, INK, PAPER, MODE sind noch nicht in gfx_commands.go implementiert
	// case "FILL":
	// 	err := b.cmdFill(args)
	// 	return physicalNextLine, err
	// case "INK":
	// 	err := b.cmdInk(args)
	// 	return physicalNextLine, err
	// case "PAPER":
	// 	err := b.cmdPaper(args)
	// 	return physicalNextLine, err
	// case "MODE":
	// 	err := b.cmdMode(args)
	// 	return physicalNextLine, err
	case "CLEAR GRAPHICS": // Korrigierter Name
		err := b.cmdClearGraphics(args) // Korrigierter Funktionsname
		return physicalNextLine, err
	case "OPEN":
		err := b.cmdOpen(args)
		return physicalNextLine, err
	case "CLOSE":
		err := b.cmdClose(args)
		return physicalNextLine, err
	case "PRINT#":
		err := b.cmdPrintFile(args)
		return physicalNextLine, err
	case "INPUT#":
		err := b.cmdInputFile(args)
		return physicalNextLine, err
	case "LINE INPUT#":
		err := b.cmdLineInputFile(args)
		return physicalNextLine, err
	case "DATA":
		err := b.cmdData(args)
		return physicalNextLine, err
	case "READ":
		err := b.cmdRead(args)
		return physicalNextLine, err
	case "RESTORE":
		err := b.cmdRestore(args)
		return physicalNextLine, err
	case "SYSTEM", "SYS":
		return 0, NewBASICError(ErrCategorySyntax, "UNKNOWN_COMMAND", b.currentLine == 0, b.currentLine).
			WithCommand("SYSTEM").
			WithUsageHint("SYSTEM command is not implemented")
	case "WAIT":
		err := b.cmdWait(args)
		return physicalNextLine, err
	case "DIM":
		err := b.cmdDim(args)
		return physicalNextLine, err
	case "SPRITE":
		err := b.cmdSprite(args)
		return physicalNextLine, err
	case "VECTOR":
		err := b.cmdVector(args)
		return physicalNextLine, err
	case "VECTOR.SCALE":
		err := b.cmdVector3DScale(args)
		return physicalNextLine, err
	case "VECTOR.HIDE":
		err := b.cmdVectorHide(args)
		return physicalNextLine, err
	case "VECTOR.SHOW":
		err := b.cmdVectorShow(args)
		return physicalNextLine, err
	case "VECFLOOR":
		err := b.cmdVecFloor(args)
		return physicalNextLine, err
	case "VECNODE":
		err := b.cmdVecNode(args)
		return physicalNextLine, err
	default:
		// Check if it's an implicit LET statement (e.g., A=10)
		// The original BASIC often allowed LET to be omitted.
		if strings.Contains(trimmedStatement, "=") && !isKnownCommand(command) {
			err := b.cmdLet(trimmedStatement)
			return physicalNextLine, err
		}
		return 0, NewBASICError(ErrCategorySyntax, "UNKNOWN_COMMAND", b.currentLine == 0, b.currentLine).
			WithCommand(command)

	}
}

// Hilfsfunktion, um zu prüfen, ob ein Kommando bekannt ist (um Zuweisungen wie A=B von Kommandos zu unterscheiden)
func isKnownCommand(cmd string) bool {
	// Diese Liste sollte mit den Kommandos in executeSingleStatementInternal synchronisiert werden
	knownCmds := []string{
		"REM", "LET", "PRINT", "PR.", "?", "INPUT", "IF", "GOTO", "GOSUB", "RETURN", "FOR", "NEXT",
		"END", "CLS", "LIST", "EDITOR", "RUN", "NEW", "LOAD", "SAVE", "DIR", "DEL", "HELP", "QUIT", "EXIT", "MCP",
		"PLOT", "LINE", "RECT", "CIRCLE", "FILL", "INK", "PAPER", "MODE", "BEEP", "SOUND", "SAY", "SPEAK", "NOISE",
		"OPEN", "CLOSE", "PRINT#", "INPUT#", "LINE INPUT#", "EOF", "DATA", "READ", "RESTORE",
		"DIM", "SPRITE", "SPRITE ON", "SPRITE OFF", "SPRITE AT", "SPRITE COLOR", "SPRITE DEL", "SPRITE LOAD", "SPRITE SAVE",
		"VECTOR", "VECTOR.SCALE", "VECTOR.HIDE", "VECTOR.SHOW", "VECTOR ON", "VECTOR OFF", "VECTOR AT", "VECTOR COLOR", "VECTOR DEL", "VECTOR LOAD", "VECTOR SAVE",
		"SYSTEM", "SYS", "WAIT",
	}
	for _, known := range knownCmds {
		if cmd == known {
			return true
		}
	}
	return false
}

// FormatErrorForDisplay formats an error for display and possibly enriches it with context
// from the BASIC interpreter state.
func (b *TinyBASIC) FormatErrorForDisplay(originalErr error, statementContext ...string) error {
	if b == nil {
		return originalErr
	}

	// Special case: Help errors should never be processed as BASIC errors
	if _, ok := originalErr.(*helpLinesAsError); ok {
		return originalErr
	}

	currentLineNum := b.currentLine

	// Wenn der Fehler bereits ein BASICError ist, aktualisiere nur den Kontext
	var be *BASICError
	if errors.As(originalErr, &be) {
		if be.LineNumber == 0 && currentLineNum != 0 {
			be.LineNumber = currentLineNum
			be.DirectMode = false
		}
		if len(statementContext) > 0 && be.Command == "" {
			parts := strings.Fields(statementContext[0])
			if len(parts) > 0 {
				be.Command = strings.ToUpper(parts[0])
				if be.Category == ErrCategorySyntax {
					be.UsageHint = GetCommandUsageHint(be.Command)
				}
			}
		}
		return be
	}

	// Neuen BASICError aus originalErr erstellen
	errMsg := originalErr.Error()
	errLower := strings.ToLower(errMsg)

	// Kategorisierung basierend auf Fehlertext
	var categoryString string
	switch {
	case strings.Contains(errLower, "syntax"):
		categoryString = ErrCategorySyntax
	case strings.Contains(errLower, "file") || strings.Contains(errLower, "not found"):
		categoryString = ErrCategoryFileSystem
	case strings.Contains(errLower, "type"):
		categoryString = ErrCategoryEvaluation
	case strings.Contains(errLower, "execution"):
		categoryString = ErrCategoryExecution
	case strings.Contains(errLower, "i/o") || strings.Contains(errLower, "input") || strings.Contains(errLower, "output"):
		categoryString = ErrCategoryIO
	case strings.Contains(errLower, "overflow") || strings.Contains(errLower, "memory"):
		categoryString = ErrCategoryResource
	default:
		categoryString = ErrCategoryExecution
	}

	newErr := NewBASICError(categoryString, errMsg, currentLineNum == 0, currentLineNum)
	if len(statementContext) > 0 {
		parts := strings.Fields(statementContext[0])
		if len(parts) > 0 {
			command := strings.ToUpper(parts[0])
			newErr = newErr.WithCommand(command)
			if categoryString == ErrCategorySyntax {
				newErr.UsageHint = GetCommandUsageHint(command)
			}
		}
	}
	return newErr
}

// splitStatementsByColon teilt eine BASIC-Zeile anhand von Doppelpunkten in einzelne Anweisungen,
// wobei Strings und andere Syntaxelemente berücksichtigt werden.
// REM-Kommentare werden nicht aufgeteilt, da alles nach REM als Kommentar gilt.
func (b *TinyBASIC) splitStatementsByColon(line string) []string {
	// Performance optimization: Quick checks using pre-compiled patterns
	trimmed := strings.TrimSpace(line)
	
	// Fast REM check using compiled pattern
	if remPattern.MatchString(trimmed) {
		return []string{line}
	}
	
	// Fast IF-THEN check using compiled pattern  
	if ifThenPattern.MatchString(trimmed) {
		return []string{line}
	}
	
	// Performance optimization: if no colons, return early
	if !strings.Contains(trimmed, ":") {
		return []string{line}
	}

	// Use efficient string splitting with pre-allocated slice
	statements := make([]string, 0, 4) // Pre-allocate for common case
	var currentStatement strings.Builder
	currentStatement.Grow(len(trimmed) / 2) // Pre-allocate builder capacity
	inString := false

	for i := 0; i < len(trimmed); i++ { // Iteriere über die getrimmte Zeile
		char := trimmed[i]

		if inString {
			if char == '"' {
				// Behandle maskierte doppelte Anführungszeichen ""
				if i+1 < len(trimmed) && trimmed[i+1] == '"' {
					currentStatement.WriteByte('"')
					i++ // Überspringe das zweite Anführungszeichen des Paares
				} else {
					inString = false // Ende des Strings
					currentStatement.WriteByte('"')
				}
			} else {
				currentStatement.WriteByte(char) // Zeichen innerhalb des Strings
			}
		} else { // Nicht in einem String
			if char == '"' {
				inString = true
				currentStatement.WriteByte('"')
			} else if char == ':' {
				// End of statement - intern the result for memory efficiency
				stmt := strings.TrimSpace(currentStatement.String())
				if len(stmt) > 0 {
					statements = append(statements, internString(stmt))
				}
				currentStatement.Reset()

				// Check if next statement starts with REM using compiled pattern
				remainingLine := strings.TrimSpace(trimmed[i+1:])
				if remPattern.MatchString(remainingLine) {
					// Rest of line is a REM comment
					statements = append(statements, internString(remainingLine))
					break // Stop processing; rest is comment
				}
			} else {
				// Teil einer Anweisung (könnte der Anfang von REM sein)
				// Prüfe, ob dies der Anfang eines REM-Schlüsselworts ist
				// Wir müssen von trimmed[i:] prüfen
				// Use compiled pattern for REM detection
				if remPattern.MatchString(trimmed[i:]) {
					// First add any collected statement
					if currentStatement.Len() > 0 {
						stmt := strings.TrimSpace(currentStatement.String())
						if len(stmt) > 0 {
							statements = append(statements, internString(stmt))
						}
						currentStatement.Reset()
					}
					// Then add entire rest of line as REM statement
					statements = append(statements, internString(strings.TrimSpace(trimmed[i:])))
					break // Stop processing; rest is comment
				} else {
					currentStatement.WriteByte(char) // Normal character processing
				}
			}
		}
	}

	// Add final statement if non-empty
	if currentStatement.Len() > 0 {
		stmt := strings.TrimSpace(currentStatement.String())
		if len(stmt) > 0 {
			statements = append(statements, internString(stmt))
		}
	}

	// Note: Empty statements are already filtered out during processing
	// No need for additional filtering loop - this improves performance
	return statements
}

// isValidVarNameInternal prüft, ob ein Name ein gültiger Variablenname für Zuweisungen ist.
// Erlaubt Suffix '$' für Strings. Verwendet isIdStart und isIdChar.
func isValidVarNameInternal(name string) bool {
	if name == "" {
		return false
	}
	hasDollar := strings.HasSuffix(name, "$")
	if hasDollar {
		name = name[:len(name)-1]
		if name == "" { // Nur "$" ist ungültig
			return false
		}
	}

	if len(name) == 0 { // z.B. wenn ursprünglicher Name nur "$" war
		return false
	}

	if !isIdStart(name[0]) { // isIdStart ist in expression.go definiert
		return false
	}
	for i := 1; i < len(name); i++ {
		if !isIdChar(name[i]) { // isIdChar ist in expression.go definiert
			return false
		}
	}
	return true
}

// isValidLValueInternal prüft, ob ein Teil eines Statements ein gültiger L-Wert ist (Variable oder Array-Referenz).
func isValidLValueInternal(varPart string) bool {
	varPart = strings.TrimSpace(varPart)
	if varPart == "" {
		return false
	}

	openParen := strings.IndexByte(varPart, '(')
	closeParen := strings.LastIndexByte(varPart, ')')

	// Prüft auf Array-Syntax wie VAR(...) oder VAR$(...)
	if openParen > 0 && closeParen == len(varPart)-1 && openParen < closeParen {
		arrayName := strings.TrimSpace(varPart[:openParen])
		// Die Indizes selbst werden hier nicht validiert, nur die Struktur und der Array-Name
		// indexPart := varPart[openParen+1 : closeParen]
		// if indexPart == "" { // A() ist normalerweise ungültig ohne DIM oder spezifische Sprachfeatures
		// return false
		// }
		return isValidVarNameInternal(arrayName)
	} else if openParen == -1 && closeParen == -1 { // Keine Klammern, sollte einfacher Variablenname sein
		return isValidVarNameInternal(varPart)
	}
	// Ungültige Klammerverwendung, z.B. (VAR) oder VAR) oder (VAR
	return false
}

// isPotentialAssignment prüft, ob ein Statement eine Zuweisung sein könnte.
// Berücksichtigt einfache Variablen, Array-Zuweisungen und LET.
func (b *TinyBASIC) isPotentialAssignment(statement string) bool {
	trimmedStatement := strings.TrimSpace(statement)
	if trimmedStatement == "" {
		return false
	}

	isLetStatement := false
	statementToCheckForAssignment := trimmedStatement

	if strings.HasPrefix(strings.ToUpper(trimmedStatement), "LET ") {
		isLetStatement = true
		// Entferne "LET " vom Anfang des zu prüfenden Teils
		statementToCheckForAssignment = strings.TrimSpace(trimmedStatement[4:])
		if statementToCheckForAssignment == "" { // Nur "LET "
			return false
		}
	}

	equalSignIndex := strings.Index(statementToCheckForAssignment, "=")
	// Das Gleichheitszeichen darf nicht am Anfang oder Ende stehen und muss existieren.
	if equalSignIndex <= 0 || equalSignIndex == len(statementToCheckForAssignment)-1 {
		return false
	}

	potentialVarPart := strings.TrimSpace(statementToCheckForAssignment[:equalSignIndex])
	// Der Teil nach dem '=' muss auch existieren (wird durch equalSignIndex == len-1 abgedeckt)

	if !isValidLValueInternal(potentialVarPart) {
		return false
	}

	// Wenn es keine LET-Anweisung ist, prüfen, ob die ursprüngliche Anweisung mit einem anderen Befehl beginnt.
	// Dies verhindert, dass z.B. FOR I = 1 TO 10 als Zuweisung erkannt wird.
	if !isLetStatement {
		words := strings.Fields(trimmedStatement) // Verwende die ursprüngliche Anweisung für die Befehlsprüfung
		if len(words) > 0 {
			firstWordUpper := strings.ToUpper(words[0])
			// isKnownCommand ist eine globale Funktion in tinybasic.go
			if isKnownCommand(firstWordUpper) {
				// Es ist ein bekannter Befehl (und nicht LET, da dieser Fall oben behandelt wurde),
				// also ist es keine reine Zuweisungsanweisung.
				return false
			}
		}
	}
	return true
}

// getCurrentSubStatementInfo returns the current sub-statement index and total count
// for use in FOR loops. This is needed for single-line FOR loops with colon separation.
func (b *TinyBASIC) getCurrentSubStatementInfo(statement string) (int, int) {
	// Split the current line to find which sub-statement we're in
	if b.currentLine == 0 {
		return -1, 0
	}

	currentLineCode, exists := b.program[b.currentLine]
	if !exists {
		return -1, 0
	}

	subStatements := b.splitStatementsByColon(currentLineCode)

	// Find which sub-statement matches our current statement
	trimmedStatement := strings.TrimSpace(statement)
	for i, subStmt := range subStatements {
		if strings.TrimSpace(subStmt) == trimmedStatement {
			// Return the index of the NEXT statement after FOR
			// FOR is at index i, so the statement after FOR is at i+1
			if i+1 < len(subStatements) {
				return i + 1, len(subStatements)
			}
			return -1, len(subStatements) // No statement after FOR on this line
		}
	}

	return -1, len(subStatements)
}

// Anfang und Ende verschiedener Hilfsfunktionen. Dies ist nur ein Platzhalter für zusätzliche Funktionalität.

// SetKeyPressed setzt die aktuell gedrückte Taste für INKEY$ Abfrage (lock-free)
func (b *TinyBASIC) SetKeyPressed(key string) {
	// Einfache String-Zuweisung - sollte atomisch sein bei Strings in Go
	b.currentKey = key

	// Erweiterte Zustandsverfolgung (thread-safe)
	b.mu.Lock()
	b.keyStates[key] = true
	b.lastKeyEvent = time.Now()
	b.mu.Unlock()
}

// SetKeyReleased löscht eine bestimmte Taste oder alle Tasten (setzt INKEY$ auf leer) (lock-free)
func (b *TinyBASIC) SetKeyReleased(key ...string) {
	// Erweiterte Zustandsverfolgung (thread-safe)
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(key) == 0 {
		// Alle Tasten löschen (Legacy-Verhalten)
		for k := range b.keyStates {
			b.keyStates[k] = false
		}
		b.currentKey = ""
	} else {
		// Nur spezifische Taste löschen
		releaseKey := key[0]
		b.keyStates[releaseKey] = false

		// Wenn die losgelassene Taste die aktuelle INKEY$ Taste war, lösche currentKey
		if b.currentKey == releaseKey {
			b.currentKey = ""
		}
	}

	b.lastKeyEvent = time.Now()
}

// GetCurrentKey gibt die aktuell gedrückte Taste zurück und aktualisiert INKEY$
func (b *TinyBASIC) GetCurrentKey() string {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Aktualisiere INKEY$ mit dem aktuellen Tastenzustand
	b.variables["INKEY$"] = BASICValue{StrValue: b.currentKey, IsNumeric: false}
	return b.currentKey
}

// IsKeyPressed prüft, ob eine bestimmte Taste aktuell gedrückt ist (neue Funktion)
func (b *TinyBASIC) IsKeyPressed(key string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := b.keyStates[key]
	return result
}

// GetKeyState gibt den Zustand einer bestimmten Taste zurück (0=losgelassen, 1=gedrückt)
// Lock-freie Version um Deadlocks zu vermeiden
func (b *TinyBASIC) GetKeyState(key string) float64 {
	// Direkte, lock-freie Abfrage der keyStates Map
	// Go Maps sind thread-safe für gleichzeitige Lesezugriffe
	if pressed, exists := b.keyStates[key]; exists && pressed {
		return 1.0
	}
	return 0.0
}

// initializeKeyConstants initialisiert die Tastaturkonstanten
func (b *TinyBASIC) initializeKeyConstants() {
	// Originale Tastenkonstanten für INKEY$
	b.variables["KEYESC"] = BASICValue{StrValue: KeyEsc, IsNumeric: false}
	b.variables["KEYCURUP"] = BASICValue{StrValue: KeyCurUp, IsNumeric: false}
	b.variables["KEYCURDOWN"] = BASICValue{StrValue: KeyCurDown, IsNumeric: false}
	b.variables["KEYCURLEFT"] = BASICValue{StrValue: KeyCurLeft, IsNumeric: false}
	b.variables["KEYCURRIGHT"] = BASICValue{StrValue: KeyCurRight, IsNumeric: false}

	// Erweiterte Tastaturkonstanten für bessere Spielsteuerung
	b.variables["KEYSPACE"] = BASICValue{StrValue: " ", IsNumeric: false}         // Leertaste
	b.variables["KEYLEFT"] = BASICValue{StrValue: KeyCurLeft, IsNumeric: false}   // Alias für Links
	b.variables["KEYRIGHT"] = BASICValue{StrValue: KeyCurRight, IsNumeric: false} // Alias für Rechts
	b.variables["KEYUP"] = BASICValue{StrValue: KeyCurUp, IsNumeric: false}       // Alias für Hoch
	b.variables["KEYDOWN"] = BASICValue{StrValue: KeyCurDown, IsNumeric: false}   // Alias für Runter

	// INKEY$ Variable initialisieren (leer)
	b.variables["INKEY$"] = BASICValue{StrValue: "", IsNumeric: false}

	// Debug-Ausgabe für erweiterte Konstanten
	tinyBasicDebugLog("KEYLEFT re-initialized = '%s' (len=%d)", b.variables["KEYLEFT"].StrValue, len(b.variables["KEYLEFT"].StrValue))
	tinyBasicDebugLog("KEYSPACE re-initialized = '%s' (len=%d)", b.variables["KEYSPACE"].StrValue, len(b.variables["KEYSPACE"].StrValue))
}

// checkMCPRateLimit prüft sowohl User- als auch System-Rate-Limits für MCP
func (b *TinyBASIC) checkMCPRateLimit(username string) (bool, string, int) {
	b.mcpMutex.Lock()
	defer b.mcpMutex.Unlock()

	now := time.Now()
	today := now.Truncate(24 * time.Hour)

	// 1. Systemweites Limit prüfen (pro Kalendertag)
	var systemUsageToday []time.Time
	for _, t := range b.mcpSystemUsage {
		if t.Truncate(24 * time.Hour).Equal(today) {
			systemUsageToday = append(systemUsageToday, t)
		}
	}
	b.mcpSystemUsage = systemUsageToday // Bereinige alte Einträge

	if len(systemUsageToday) >= MCPSystemDailyLimit {
		return false, "Daily usage quota for MCP exceeded. Try again tomorrow.", 0
	}

	// 2. User-spezifisches Limit prüfen (24h-Fenster)
	last24h := now.Add(-24 * time.Hour)
	userTimes, exists := b.mcpUserUsage[username]
	if !exists {
		userTimes = make([]time.Time, 0)
	}

	// Bereinige User-Timestamps älter als 24h
	var recentUserTimes []time.Time
	for _, t := range userTimes {
		if t.After(last24h) {
			recentUserTimes = append(recentUserTimes, t)
		}
	}
	b.mcpUserUsage[username] = recentUserTimes

	if len(recentUserTimes) >= MCPUserDailyLimit {
		return false, fmt.Sprintf("User daily limit exceeded. You can use MCP %d times in 24h.", MCPUserDailyLimit), len(recentUserTimes)
	}

	return true, "", len(recentUserTimes)
}

// recordMCPUsage verzeichnet eine MCP-Nutzung für User und System
func (b *TinyBASIC) recordMCPUsage(username string) int {
	b.mcpMutex.Lock()
	defer b.mcpMutex.Unlock()

	now := time.Now()

	// Systemweite Nutzung verzeichnen
	b.mcpSystemUsage = append(b.mcpSystemUsage, now)

	// User-spezifische Nutzung verzeichnen
	if b.mcpUserUsage[username] == nil {
		b.mcpUserUsage[username] = make([]time.Time, 0)
	}
	b.mcpUserUsage[username] = append(b.mcpUserUsage[username], now)

	return len(b.mcpUserUsage[username])
}

// processMCPFilenameInput handles filename input after MCP code generation
// Assumes mutex is already locked and will unlock it before returning
func (b *TinyBASIC) processMCPFilenameInput(filename string) []shared.Message { // Clear MCP input flag
	b.waitingForMCPInput = false

	// Get the pending code and filename
	code := b.pendingMCPCode
	originalFilename := b.pendingMCPFilename
	b.pendingMCPCode = ""
	b.pendingMCPFilename = ""

	b.mu.Unlock() // Unlock before processing

	// Validate filename
	filename = strings.TrimSpace(filename)
	if filename == "" {
		// If no filename provided, use the original filename from MCP EDIT
		if originalFilename != "" {
			filename = originalFilename
		} else {
			b.sendMessageWrapped(shared.MessageTypeText, "Error: Filename cannot be empty")
			return nil
		}
	}

	// Ensure filename ends with .bas
	if !strings.HasSuffix(strings.ToLower(filename), ".bas") {
		filename += ".bas"
	}

	// Save the generated code to the file
	if b.fs == nil {
		b.sendMessageWrapped(shared.MessageTypeText, "Error: File system not available")
		return nil

	}
	err := b.fs.WriteFile(filename, code, b.sessionID)
	if err != nil {
		b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("Error saving file: %s", err.Error()))
		return nil
	}
	b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("Program saved to %s", filename))

	// Automatically load the program into BASIC
	b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("Loading %s into BASIC...", filename))

	// Re-acquire the mutex for cmdLoad (it expects lock to be held)
	b.mu.Lock()

	// Use the existing cmdLoad function to load the program
	// cmdLoad expects a quoted string, so we need to wrap the filename in quotes
	quotedFilename := fmt.Sprintf("\"%s\"", filename)
	err = b.cmdLoad(quotedFilename)
	if err != nil {
		b.mu.Unlock()
		b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("Error loading program: %s", err.Error()))
		return nil
	}

	// Ensure no INPUT variables are pending after load

	b.inputVar = ""
	b.waitingForMCPInput = false
	b.running = false

	b.mu.Unlock()

	b.sendMessageWrapped(shared.MessageTypeText, "Program loaded successfully. Type RUN to execute.")
	return nil
}
