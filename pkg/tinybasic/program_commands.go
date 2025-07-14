package tinybasic

import (
	"fmt"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/editor"
	"github.com/antibyte/retroterm/pkg/shared"
	"github.com/antibyte/retroterm/pkg/virtualfs"
)

// cmdNew clears the current program and state. Assumes lock is held.
func (b *TinyBASIC) cmdNew(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("NEW")
	}
	b.program = make(map[int]string)
	b.programLines = make([]int, 0)
	b.variables = make(map[string]BASICValue)
	b.initializeKeyConstants() // Tastaturkonstanten nach Reset wiederherstellen
	b.gosubStack = b.gosubStack[:0]
	b.forLoops = b.forLoops[:0]
	b.data = make([]string, 0)
	b.dataPointer = 0
	b.currentLine = 0
	b.running = false
	b.inputVar = ""
	b.closeAllFiles()

	return nil
}

// cmdRun starts asynchronous program execution.
// Optionally accepts a filename: RUN "filename.bas"
func (b *TinyBASIC) cmdRun(args string) (string, error) {
	// If filename provided, load it first
	if args != "" {
		// Parse filename argument
		filenameExpr := strings.TrimSpace(args)
		filenameVal, err := b.evalExpression(filenameExpr)
		if err != nil || filenameVal.IsNumeric {
			return "", NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("RUN").WithUsageHint("Usage: RUN or RUN \"filename.bas\"")
		}

		filename := filenameVal.StrValue
		if filename == "" {
			return "", NewBASICError(ErrCategorySyntax, "MISSING_FILENAME", b.currentLine == 0, b.currentLine).WithCommand("RUN")
		}

		// Load the file first
		err = b.cmdLoad(filenameExpr)
		if err != nil {
			return "", err // Return load error directly
		}
	}

	// Nur Ausführungszustand zurücksetzen, nicht das geladene Programm
	b.ResetExecutionState()
	if len(b.program) == 0 {
		return "", NewBASICError(ErrCategoryExecution, "NO_PROGRAM_LOADED", true, 0).WithCommand("RUN")
	}
	b.rebuildProgramLines()
	if len(b.programLines) == 0 {
		return "", NewBASICError(ErrCategoryExecution, "NO_PROGRAM_LINES", true, 0).WithCommand("RUN")
	} // Rebuild DATA statements before running
	b.rebuildData()

	b.currentLine = b.programLines[0]
	b.running = true // Reset cursor state at start of program
	b.printCursorOnSameLine = false

	// Resetze INPUT_CONTROL enable Flag für neuen RUN
	b.inputControlEnableSent = false

	// Aktiviere RUN-Modus (INKEY$ und Strg+C aktiv, normale Eingabe deaktiviert)
	b.sendInputControl("run_mode")

	// Send empty line to separate RUN command from program output
	b.sendEmptyLine() // Send empty line for separation

	// Try bytecode execution first, fall back to interpreted if needed
	if b.useBytecode {
		err := b.compileProgramIfNeeded()
		if err == nil {
			// Run bytecode version
			tinyBasicDebugLog("Running program with bytecode VM")
			go b.runBytecodeProgram()
		} else {
			// Fall back to interpreted execution
			tinyBasicDebugLog("Bytecode compilation failed, falling back to interpreted execution: %v", err)
			go b.runProgramInternal(b.ctx)
		}
	} else {
		// Run interpreted version
		go b.runProgramInternal(b.ctx)
	}
	
	return "", nil
}

// cmdList displays program lines. Sends output via channel. Assumes lock is held.
func (b *TinyBASIC) cmdList(args string) error {
	if len(b.programLines) == 0 {
		b.sendMessageWrapped(shared.MessageTypeText, "Program empty.")
		return nil
	}
	startLine, endLine, _ := parseListRange(args)
	// parseListRange returns always nil error in current implementation

	// Sammle alle Zeilen in einem Buffer und sende sie in Blöcken
	var outputBuffer strings.Builder
	linesListed := 0
	linesSinceLastSend := 0
	const linesPerBatch = 50 // Sende in 50-Zeilen-Blöcken

	for _, lineNum := range b.programLines {
		if lineNum >= startLine && lineNum <= endLine {
			lineStr := fmt.Sprintf("%d %s\n", lineNum, b.program[lineNum])
			outputBuffer.WriteString(lineStr)
			linesListed++
			linesSinceLastSend++

			// Sende in Blöcken von 50 Zeilen
			if linesSinceLastSend >= linesPerBatch {
				b.sendMessage(shared.MessageTypeText, strings.TrimSuffix(outputBuffer.String(), "\n"))
				outputBuffer.Reset()
				linesSinceLastSend = 0

				// Kurze Pause zwischen Blöcken, um WebSocket nicht zu überlasten
				time.Sleep(10 * time.Millisecond)
			}
		}
		if lineNum > endLine {
			break
		}
	}

	// Sende den letzten Block, falls noch Zeilen im Buffer sind
	if outputBuffer.Len() > 0 {
		b.sendMessage(shared.MessageTypeText, strings.TrimSuffix(outputBuffer.String(), "\n"))
	}

	if linesListed == 0 && args != "" {
		b.sendMessageWrapped(shared.MessageTypeText, "No lines found in specified range.")
	}
	return nil
}

// cmdEnd terminates program execution. Assumes lock is held.
func (b *TinyBASIC) cmdEnd(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("END")
	}

	// Stop any playing SID music when program ends
	musicStopMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"action": "music_stop",
		},
	}
	b.sendMessageObject(musicStopMsg)

	b.running = false // Signal run loop to stop.
	b.currentLine = 0 // Set line to 0 for clean stop state.
	// Stacks and inputVar cleared by run loop's defer. Files closed too.
	return nil // runProgramInternal will detect running=false and exit.
}

// cmdExit signals intent to leave BASIC mode. Assumes lock is held by caller.
func (b *TinyBASIC) cmdExit(args string) error {
	if args != "" {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("EXIT")
	}

	b.closeAllFiles() // Close files before signaling exit

	// Stop any playing SID music before exiting
	musicStopMsg := shared.Message{
		Type: shared.MessageTypeSound,
		Params: map[string]interface{}{
			"action": "music_stop",
		},
	}
	b.sendMessageObject(musicStopMsg) // Don't check return value, as we're exiting anyway

	// Signal to the caller (Execute method) that EXIT was called.
	// The Execute method will then be responsible for sending appropriate messages
	// to the terminal handler, which in turn informs the client.
	return ErrExit
}

// cmdRem is a remark/comment. Does nothing. No lock needed.
func (b *TinyBASIC) cmdRem(args string) error {
	return nil
}

// cmdVars displays all defined variables and their values. Sends output via channel. Assumes lock is held.
func (b *TinyBASIC) cmdVars(args string) error {
	if len(b.variables) == 0 {
		b.sendMessageWrapped(shared.MessageTypeText, "No variables defined.")
		return nil
	}

	b.sendMessageWrapped(shared.MessageTypeText, "Defined variables:")
	for name, value := range b.variables {
		var valueStr string
		if value.IsNumeric {
			valueStr = fmt.Sprintf("%.6g", value.NumValue)
		} else {
			// Show string values with quotes and escape sequences visible
			valueStr = fmt.Sprintf("\"%s\" (hex: %x)", value.StrValue, []byte(value.StrValue))
		}
		varStr := fmt.Sprintf("  %s = %s", name, valueStr)
		b.sendMessageWrapped(shared.MessageTypeText, varStr)
	}
	return nil
}

// cmdEditor opens the full-screen text editor
func (b *TinyBASIC) cmdEditor(args string) error {
	filename := strings.TrimSpace(args)

	// Get the editor manager
	editorManager := editor.GetEditorManager()

	// Create output channel for editor messages
	outputChan := make(chan shared.Message, 100)
	// Get terminal dimensions (default values, could be made configurable)
	rows := 24
	cols := 80
	config := editor.EditorConfig{
		Filename:   filename,
		Rows:       rows,
		Cols:       cols,
		SessionID:  b.sessionID,
		OutputChan: outputChan, VFS: b.fs.(*virtualfs.VFS), // Type assertion: TinyBASIC filesystem (basic directory)
	}

	editorInstance := editorManager.StartEditor(config)

	// Handle special case: if no filename specified, edit the current program
	if filename == "" {
		// Load current program into editor AFTER initialization
		programText := b.getProgramAsText()
		editorInstance.SetContent(programText)
		editorInstance.SetFilename("<current program>")
	}

	// Send editor start message to frontend
	msg := shared.Message{
		Type:          shared.MessageTypeEditor,
		EditorCommand: "start",
		Params: map[string]interface{}{
			"filename": filename,
			"rows":     rows,
			"cols":     cols,
		},
		SessionID: b.sessionID,
	}
	b.sendMessageObject(msg)

	// Always send initial render message to ensure editor is displayed immediately
	renderParams := editorInstance.GetRenderParams()
	if renderParams != nil {
		renderMsg := shared.Message{
			Type:          shared.MessageTypeEditor,
			EditorCommand: "render",
			Params:        renderParams,
			SessionID:     b.sessionID,
		}
		b.sendMessageObject(renderMsg)
	} // Start a goroutine to process editor output messages
	go func() {
		outputChan := editorInstance.GetOutputChannel()
		// Use structured logging instead of fmt.Printf
		// fmt.Printf("[BASIC-EDITOR] Starting output message processor for session %s\n", b.sessionID)

		// Use a ticker to periodically check if editor is still active
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case msg := <-outputChan:
				// Forward editor output message to frontend
				// Use structured logging instead of fmt.Printf
				// fmt.Printf("[BASIC-EDITOR] Received editor output message: Type=%d, Command=%s\n",
				//	int(msg.Type), msg.EditorCommand)
				b.sendMessageObject(msg)
			case <-ticker.C:
				// Check if editor is still active
				editorManager := editor.GetEditorManager()
				if editorManager.GetEditor(b.sessionID) == nil {
					// Editor was closed, stop the goroutine
					// Use structured logging instead of fmt.Printf
					// fmt.Printf("[BASIC-EDITOR] Editor closed, stopping output processor for session %s\n", b.sessionID)
					return
				}
			}
		}
	}()

	return nil
}

// getProgramAsText returns the current BASIC program as text
func (b *TinyBASIC) getProgramAsText() string {
	var lines []string

	// b.programLines is already sorted, so we can iterate over it directly.
	// This avoids an inefficient O(n^2) bubble sort.
	// Build program text from the pre-sorted line numbers.
	for _, lineNum := range b.programLines {
		if line, exists := b.program[lineNum]; exists {
			lines = append(lines, fmt.Sprintf("%d %s", lineNum, line))
		}
	}

	return strings.Join(lines, "\n")
}
