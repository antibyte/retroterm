// Package tinybasic implements a simple BASIC interpreter.
package tinybasic

import (
	"fmt"
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// --- Help Text System ---
type helpLinesAsError struct{ lines []shared.Message }

func (e *helpLinesAsError) Error() string {
	return "help output"
}

// HandleHelpCommand processes the HELP command and returns either general help
// or help for a specific command
func HandleHelpCommand(args string) error {
	args = strings.TrimSpace(args)

	var msgs []shared.Message
	if args == "" {
		// Show general help overview
		msgs = GetOverviewHelpText()
	} else {
		// Show help for a specific command
		msgs = GetCommandHelpText(args)
	}

	// Return as the expected error type that TinyBASIC knows how to handle
	return &helpLinesAsError{lines: msgs}
}

// GetCommandSyntax gibt die Syntax für einen bestimmten Befehl zurück
func GetCommandSyntax(command string) string {
	command = strings.ToUpper(strings.TrimSpace(command))
	syntax, exists := commandUsageHints[command]
	if !exists {
		return ""
	}
	return syntax
}

// GetOverviewHelpText gibt eine allgemeine Hilfeübersicht zurück
func GetOverviewHelpText() []shared.Message {
	messages := []shared.Message{
		{Type: shared.MessageTypeText, Content: "TinyBASIC HELP OVERVIEW"},
		{Type: shared.MessageTypeText, Content: "-------------------"},
		{Type: shared.MessageTypeText, Content: "Available commands:"},
		{Type: shared.MessageTypeText, Content: ""},
	}

	// All available commands in a compact list
	commands := []string{
		"REM", "END", "DATA", "READ", "RESTORE", "LET", "DIM", "PRINT", "INPUT",
		"CLS", "LOCATE", "INVERSE", "IF", "GOTO", "GOSUB", "RETURN", "FOR", "NEXT",
		"RUN", "LIST", "NEW", "LOAD", "SAVE", "DIR", "EDITOR", "VARS", "PLOT",
		"LINE", "RECT", "CIRCLE", "BEEP", "SOUND", "NOISE", "SAY", "MUSIC",
		"SPRITE", "VECTOR", "OPEN", "CLOSE", "PRINT#", "INPUT#", "LINE INPUT#",
		"EOF", "WAIT", "MCP", "EXIT", "HELP",
	}

	// Display commands in rows of 8 for compact display
	for i := 0; i < len(commands); i += 8 {
		end := i + 8
		if end > len(commands) {
			end = len(commands)
		}
		row := strings.Join(commands[i:end], "  ")
		messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: "  " + row})
	}

	messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: ""})
	messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: "Type HELP followed by command name for specific help."})
	messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: "Example: HELP PRINT"})

	return messages
}

// GetCommandHelpText gibt detaillierte Hilfe für einen bestimmten Befehl zurück
func GetCommandHelpText(command string) []shared.Message {
	command = strings.ToUpper(strings.TrimSpace(command))

	// Suche nach dem Befehl in der Hilfetextsammlung
	helpText, exists := helpTexts[command]
	if !exists {
		return []shared.Message{
			{Type: shared.MessageTypeText, Content: fmt.Sprintf("No help available for '%s'", command)},
			{Type: shared.MessageTypeText, Content: "Type HELP for a list of commands."},
		}
	}

	// Finde die Syntax für den Befehl
	syntax := GetCommandSyntax(command)

	messages := []shared.Message{
		{Type: shared.MessageTypeText, Content: fmt.Sprintf("HELP: %s", command)},
		{Type: shared.MessageTypeText, Content: "-------------------"},
	}

	if syntax != "" {
		messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: fmt.Sprintf("Syntax: %s", syntax)})
		messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: ""})
	}

	// Teile den Hilfetext in Zeilen auf
	lines := strings.Split(helpText, "\n")
	for _, line := range lines {
		messages = append(messages, shared.Message{Type: shared.MessageTypeText, Content: line})
	}

	return messages
}

// BASICError repräsentiert einen Fehler im BASIC Interpreter
// Bereits in errors.go definiert
// type BASICError struct {
// 	Category ErrorCategory
// 	Code     string
// 	Message  string
// 	Line     int
// 	Command  string
// 	IsDirect bool // true, wenn der Fehler im Direktmodus auftrat
// }

// --- Hilfetexte für Befehle ---

// Hilfetext für alle Befehle
var helpTexts = map[string]string{
	"PRINT": `Outputs text or expressions to the screen.
- Use commas to tab to next column
- Use semicolons to join items without spaces
- End with semicolon to prevent line break
- Expressions can be strings or numbers
- String literals must be enclosed in quotes

Examples:
  PRINT "Hello, World!"
  PRINT A, B, C
  PRINT "The answer is"; A`,

	"LET": `Assigns a value to a variable.
- Variable name must start with a letter
- String variables end with $ and contain text
- Numeric variables store numbers
- The word LET is optional

Examples:
  LET A = 42
  B = A * 2
  LET NAME$ = "John"`,

	"INPUT": `Reads user input into a variable.
- Can display an optional prompt
- String variables receive text as entered
- Numeric variables require valid numbers

Examples:
  INPUT A
  INPUT "Enter your name"; NAME$
  INPUT "Value"; X`,

	"GOTO": `Jumps execution to specified line number.
- Program continues from that line

Example:
  GOTO 100`,

	"IF": `Conditionally executes a statement.
- Tests a condition, acts if true
- Can use =, <, >, <=, >=, <> comparisons
- THEN keyword required

Examples:
  IF A = 10 THEN PRINT "Equal"
  IF X > 0 THEN GOTO 200`,

	"FOR": `Starts a loop with a control variable.
- Loop executes until control variable exceeds end value
- STEP specifies increment (default is 1)
- Must end with matching NEXT statement

Examples:
  FOR I = 1 TO 10
  FOR J = 10 TO 1 STEP -1`,

	"NEXT": `Marks the end of a FOR loop.
- Variable name should match corresponding FOR
- Increments variable and continues loop if not done

Example:
  NEXT I`,

	"GOSUB": `Calls a subroutine at specified line number.
- Program returns to next line after RETURN statement

Example:
  GOSUB 5000`,

	"RETURN": `Returns from a subroutine to call point.
- Must correspond to a previous GOSUB

Example:
  RETURN`,

	"END": `Terminates program execution.
- Can appear anywhere in program

Example:
  END`,

	"REM": `Inserts a comment (remark) in the program.
- Has no effect on execution
- Used for documenting code

Example:
  REM This is a comment`,

	"LIST": `Displays program lines.
- Can specify a range of lines
- Useful to verify or review code

Examples:
  LIST
  LIST 100
  LIST 100-200`,

	"RUN": `Executes the program from the beginning.
- Starts with the lowest line number
- Clears previous results but not variables

Example:
  RUN`,

	"NEW": `Clears the current program and variables.
- Use with caution - data can't be recovered

Example:
  NEW`,

	"LOAD": `Loads a program from storage.
- Clears current program before loading
- Automatically adds .bas extension if none specified

Examples:
  LOAD "GAME"
  LOAD "PROGRAM.BAS"`,

	"SAVE": `Saves the current program to storage.
- Automatically adds .bas extension if none specified

Examples:
  SAVE "MYPROG"
  SAVE "BACKUP.BAS"`,

	"DIR": `Lists BASIC program files available.
- Shows files with .bas extension

Example:
  DIR`,

	"OPEN": `Opens a file for input or output operations.
- INPUT mode: read from file
- OUTPUT mode: write to file
- Each open file needs a unique handle number

Examples:
  OPEN "DATA.TXT" FOR INPUT AS #1
  OPEN "OUTPUT.TXT" FOR OUTPUT AS #2`,

	"CLOSE": `Closes an open file.
- Writes any buffered output
- Releases the file handle

Example:
  CLOSE #1`,

	"LINE INPUT": `Reads a full line from a file into a variable.
- Must specify file handle and string variable

Example:
  LINE INPUT #1, A$`,

	"DATA": `Defines data to be read by READ statements.
- Values separated by commas
- String literals can be quoted

Examples:
  DATA 10, 20, 30, 40
  DATA "Apple", "Orange", "Banana"`,

	"READ": `Reads values from DATA statements into variables.
- Advances through DATA items sequentially
- Type must match (numbers to numeric vars, strings to string vars)

Example:
  READ A, B$, C`,

	"RESTORE": `Resets the DATA pointer to first DATA item.
- Allows re-reading data

Example:
  RESTORE`,

	"CLS": `Clears the screen.
- Removes all text but keeps program running

Example:
  CLS`,

	"BEEP": `Produces a short beep sound.

Example:
  BEEP`,

	"SOUND": `Generates a tone with specified frequency and duration.
- Frequency in Hz
- Duration in milliseconds

Example:
  SOUND 440, 500 (A note for half a second)`,

	"SAY": `Outputs text as computer speech.
- Same as SPEAK command

Example:
  SAY "Hello, how are you today?"`,

	"SPEAK": `Outputs text as computer speech.
- Same as SAY command

Example:
  SPEAK "Welcome to TinyBASIC"`,

	"PLOT": `Plots a single point in graphics mode.
- Requires x,y coordinates
- Uses current color (set by INK)

Example:
  PLOT 160, 100`, "CIRCLE": `Draws a circle.
- Requires center (x,y) and radius

Example:
  CIRCLE 160, 100, 50`,

	"RECT": `Draws a rectangle.
- Requires top-left corner (x,y) and dimensions (width,height)

Example:
  RECT 50, 50, 100, 80`,

	"EXIT": `Exits BASIC and returns to system.
- Closes all open files

Example:
  EXIT`,

	"HELP": `Displays help information.
- Without arguments shows command list
- With command name shows specific help

Examples:
  HELP
  HELP PRINT`,
	// Aliases
	"?": `Short form of PRINT command.
See HELP PRINT for full details.`,

	"MCP": `Access the Master Control Program AI assistant.
- CREATE: Generate a new BASIC program
- EDIT: Modify an existing program file

Examples:
  MCP CREATE a program that draws a sine wave
  MCP EDIT sine.bas add a coordinate system`,

	"DIM": `Declares array variables with specified dimensions.
- Creates space for multiple values
- Arrays can be 1D or 2D

Examples:
  DIM A(10)        ' 1D array with 11 elements (0-10)
  DIM B(5,5)       ' 2D array 6x6 elements`,

	"LOCATE": `Positions the cursor at specified screen coordinates.
- Uses text coordinates (1-based)
- Screen is 80 columns by 24 rows

Example:
  LOCATE 10, 5     ' Column 10, Row 5`,

	"INVERSE": `Controls inverse text display mode.
- ON: Text appears in reverse video
- OFF: Normal text display

Examples:
  INVERSE ON
  INVERSE OFF`,

	"LINE": `Draws a line between two points.
- Requires start (x1,y1) and end (x2,y2) coordinates
- Optional brightness parameter (0-15)

Example:
  LINE 0, 0, 100, 100, 15`,

	"NOISE": `Generates noise sound with envelope control.
- Pitch: frequency of noise
- Attack: rise time
- Decay: fall time

Example:
  NOISE 1000, 100, 500`,

	"MUSIC": `Plays SID music files.
- Available files: ull.sid, sensory.sid, deep.sid

Example:
  MUSIC "ull.sid"`,

	"SPRITE": `Controls 32x32 pixel sprite graphics.
- Define: SPRITE id, pixelData$
- Position: SPRITE id AT x, y
- Color: SPRITE id COLOR brightness
- Visibility: SPRITE id ON/OFF

Examples:
  SPRITE 1, "1,4,6,1,0,15..." ' Define sprite
  SPRITE 1 AT 100, 50         ' Position
  SPRITE 1 ON                 ' Show`,

	"VECTOR": `Controls 3D vector graphics objects.
- Shapes: "cube", "pyramid", "sphere"
- Position with VECTOR id AT x, y
- Scale with VECTOR.SCALE

Examples:
  VECTOR 1, "cube", 0, 0, 0, 0, 0, 0, 1
  VECTOR 1 AT 100, 100`,

	"EDITOR": `Opens the full-screen text editor.
- Can specify filename to edit
- ESC to exit and return to BASIC

Examples:
  EDITOR
  EDITOR "myfile.bas"`,

	"VARS": `Lists all current variables and their values.
- Shows both numeric and string variables

Example:
  VARS`,

	"WAIT": `Pauses program execution for specified milliseconds.

Example:
  WAIT 1000        ' Wait 1 second`,
	"EOF": `Tests if file is at end-of-file.
- Returns true if no more data to read
- Used with file input operations

Example:
  IF EOF(1) THEN GOTO 100`,

	"PRINT#": `Writes data to an open file.
- Must specify file handle number
- Can output text or numeric values

Examples:
  PRINT #1, "Hello World"
  PRINT #2, A, B, C`,

	"INPUT#": `Reads data from an open file into variables.
- Must specify file handle number
- Variable types must match data

Examples:
  INPUT #1, A
  INPUT #1, NAME$, AGE`,

	"LINE INPUT#": `Reads a complete line from an open file.
- Must specify file handle and string variable
- Reads until end of line or file

Example:
  LINE INPUT #1, LINE$`,
}
