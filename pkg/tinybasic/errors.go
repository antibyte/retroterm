// Package tinybasic implements a simple BASIC interpreter.
package tinybasic

import (
	"errors"
	"fmt"
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// Error definitions specific to TinyBASIC operations.
var (
	ErrNilFileSystem          = errors.New("filesystem not available")
	ErrProgramNotRunning      = errors.New("program not running")
	ErrProgramAlreadyRunning  = errors.New("program already running")
	ErrNoProgramLoaded        = errors.New("no program loaded")
	ErrInputNotExpected       = errors.New("no input expected")
	ErrInvalidLineNumber      = errors.New("invalid line number")
	ErrLineNotFound           = errors.New("line not found")
	ErrReturnWithoutGosub     = errors.New("RETURN without GOSUB")
	ErrNextWithoutFor         = errors.New("NEXT without FOR")
	ErrForWithoutNextMismatch = errors.New("NEXT variable mismatch")
	ErrOutOfData              = errors.New("OUT OF DATA") // Classic BASIC message
	ErrReadMissingVariable    = errors.New("READ: missing variable name")
	ErrInvalidExpression      = errors.New("invalid expression")
	ErrTypeMismatch           = errors.New("type mismatch")
	ErrDivisionByZero         = errors.New("division by zero")
	ErrUnknownVariable        = errors.New("unknown variable")
	ErrInvalidVariableName    = errors.New("invalid variable name") // Neuer Fehler
	ErrUnknownCommand         = errors.New("unknown command")
	ErrSyntaxError            = errors.New("syntax error")
	ErrMissingParenthesis     = errors.New("missing closing parenthesis")
	ErrInvalidFileHandle      = errors.New("invalid file handle")
	ErrFileHandleInUse        = errors.New("file handle already in use")
	ErrFileNotOpen            = errors.New("file not open")
	ErrFileNotInInputMode     = errors.New("file not open for INPUT")
	ErrFileNotInOutputMode    = errors.New("file not open for OUTPUT")
	ErrEndOfFile              = errors.New("end of file")
	ErrFileReadError          = errors.New("file read error")
	ErrFileWriteError         = errors.New("file write error")
	ErrFileNotFound           = errors.New("file not found")
	ErrHelpNotFound           = errors.New("no help found for this command")
	ErrGosubDepthExceeded     = errors.New("GOSUB depth exceeded")
	ErrForLoopDepthExceeded   = errors.New("FOR loop depth exceeded")
	ErrInvalidColor           = errors.New("invalid color value (must be 1-16)")
	ErrInvalidCoordinates     = errors.New("invalid coordinates")
	ErrInvalidArguments       = errors.New("invalid arguments")         // Generic argument error
	ErrSayTextTooLong         = errors.New("say text too long")         // Neuer Fehler für SAY-Textlänge
	ErrRateLimitExceeded      = errors.New("rate limit exceeded")       // Neuer Fehler für SAY Rate Limiting
	ErrNoiseRateLimitExceeded = errors.New("noise rate limit exceeded") // Neuer Fehler für NOISE Rate Limiting
	ErrInvalidNoiseParameter  = errors.New("invalid noise parameter")   // Neuer Fehler für NOISE Parameter
	ErrMessageSendFailed      = errors.New("message send failed")       // Fehler beim Senden der Nachricht an den Client
)

// BASICError repräsentiert einen strukturierten Fehler im TinyBASIC-Interpreter
type BASICError struct {
	Category   string // Fehlerkategorie (z.B. "SYNTAX ERROR")
	Message    string // Spezifische Fehlermeldung
	Command    string // Der Befehl, bei dem der Fehler aufgetreten ist (optional)
	UsageHint  string // Hinweis zur korrekten Syntax (nur für Syntaxfehler)
	LineNumber int    // Zeilennummer im Programm (0 für Direktmodus)
	DirectMode bool   // Ob der Fehler im Direktmodus aufgetreten ist
	Detail     string // Detaillierte Fehlerbeschreibung (für spezifische Fehlercodes)
}

// Error implementiert das error-Interface
func (be *BASICError) Error() string {
	// Debug: Log error details
	fmt.Printf("DEBUG: Creating error - Category: '%s', Detail: '%s', Command: '%s', LineNumber: %d\n",
		be.Category, be.Detail, be.Command, be.LineNumber)

	friendly := GetFriendlyErrorText(be.Category, be.Detail)
	msg := ""
	if be.DirectMode {
		msg = be.Category + ": " + friendly
		if be.UsageHint != "" {
			msg += "\nUSAGE: " + be.UsageHint
		}
	} else if be.LineNumber > 0 {
		msg = be.Category + " IN LINE " + fmt.Sprint(be.LineNumber) + ": " + friendly
	} else {
		msg = be.Category + ": " + friendly
	}
	// Entfernt: Logausgabe für Fehlerobjekt (Fehler werden ohnehin als Rückgabe behandelt)
	return msg
}

// NewBASICError erstellt eine neue BASIC-Fehlerinstanz
func NewBASICError(category, message string, directMode bool, lineNumber int) *BASICError {
	return &BASICError{
		Category:   category,
		Message:    message,
		Detail:     message, // Fehlercode auch als Detail setzen!
		DirectMode: directMode,
		LineNumber: lineNumber,
	}
}

// WithCommand fügt dem Fehler einen Befehlsnamen hinzu
func (be *BASICError) WithCommand(cmd string) *BASICError {
	be.Command = cmd
	// Füge automatisch einen Syntaxhinweis hinzu, wenn verfügbar
	if be.Category == ErrCategorySyntax {
		be.UsageHint = GetCommandUsageHint(cmd)
	}
	return be
}

// WithUsageHint fügt dem Fehler einen expliziten Verwendungshinweis hinzu
func (be *BASICError) WithUsageHint(hint string) *BASICError {
	be.UsageHint = hint
	return be
}

// Fehlerkategorien
const (
	// ErrCategorySyntax kennzeichnet Syntaxfehler.
	ErrCategorySyntax = "SYNTAX ERROR"
	// ErrCategoryRuntime kennzeichnet Laufzeitfehler.
	ErrCategoryRuntime = "RUNTIME ERROR"
	// ErrCategoryFileSystem kennzeichnet Dateisystemfehler.
	ErrCategoryFileSystem = "FILE SYSTEM ERROR"
	// ErrCategoryEvaluation kennzeichnet Fehler bei der Ausdrucksauswertung.
	ErrCategoryEvaluation = "EVALUATION ERROR"
	// ErrCategoryCommand kennzeichnet Fehler bei der Befehlsausführung.
	ErrCategoryCommand = "COMMAND ERROR"
	// ErrCategoryResource kennzeichnet Fehler im Zusammenhang mit Ressourcen (z.B. Speicher).
	ErrCategoryResource = "RESOURCE ERROR"
	// ErrCategoryExecution kennzeichnet allgemeine Ausführungsfehler.
	ErrCategoryExecution = "EXECUTION ERROR"
	// ErrCategoryIO kennzeichnet Ein-/Ausgabefehler.
	ErrCategoryIO = "I/O ERROR"
	// ErrCategorySystem kennzeichnet systemnahe Fehler.
	ErrCategorySystem = "SYSTEM ERROR"
)

// FriendlyErrorTexts map error codes to user-friendly messages.
// Diese Variable wird nur einmal deklariert.
var FriendlyErrorTexts = map[string]map[string]string{
	ErrCategorySyntax: {
		"UNEXPECTED_TOKEN":          "UNEXPECTED TOKEN ENCOUNTERED",
		"MISSING_PARENTHESIS":       "MISSING CLOSING PARENTHESIS",
		"INVALID_LINE_NUMBER":       "INVALID LINE NUMBER SPECIFIED",
		"UNKNOWN_COMMAND":           "COMMAND NOT RECOGNIZED",
		"INVALID_VARIABLE_NAME":     "INVALID VARIABLE NAME",
		"INVALID_ARGUMENT":          "INVALID ARGUMENT PROVIDED",
		"MISSING_ARGUMENT":          "REQUIRED ARGUMENT IS MISSING",
		"TOO_MANY_ARGUMENTS":        "TOO MANY ARGUMENTS PROVIDED",
		"TEXT_TOO_LONG":             "INPUT TEXT EXCEEDS MAXIMUM ALLOWED LENGTH",
		"RATE_LIMIT_EXCEEDED":       "COMMAND RATE LIMIT EXCEEDED. PLEASE WAIT",
		"NOISE_RATE_LIMIT_EXCEEDED": "NOISE COMMAND RATE LIMIT EXCEEDED. PLEASE WAIT",
		"INVALID_NOISE_PARAMETER":   "INVALID PARAMETER FOR NOISE COMMAND",
		"EXPECTED_EQUALS":           "EQUALS SIGN (=) EXPECTED",
		"EXPECTED_VARIABLE":         "VARIABLE NAME EXPECTED",
		"EXPECTED_EXPRESSION":       "EXPRESSION EXPECTED",
		"EXPECTED_TO":               "TO KEYWORD EXPECTED IN FOR LOOP",
		"EXPECTED_THEN":             "THEN KEYWORD EXPECTED AFTER IF CONDITION", "INVALID_NUMBER": "INVALID NUMBER FORMAT",
		"MISSING_FILENAME":           "FILENAME EXPECTED",
		"INVALID_FILE_TYPE":          "ONLY .BAS FILES ARE SUPPORTED",
		"MISSING_TEXT":               "TEXT EXPECTED",
		"MISSING_HANDLE":             "FILE HANDLE NUMBER EXPECTED",
		"COMMA_EXPECTED":             "COMMA EXPECTED",
		"INVALID_FILE_MODE":          "INVALID FILE MODE (USE INPUT OR OUTPUT)",
		"MISSING_AS":                 "AS KEYWORD EXPECTED",
		"INVALID_PARAMETER_COUNT":    "INVALID NUMBER OF PARAMETERS",
		"INVALID_COLOR_FORMAT":       "INVALID COLOR FORMAT",
		"INVALID_PARAMETER_VALUE":    "INVALID PARAMETER VALUE",
		"SYNTAX_ERROR":               "SYNTAX ERROR",
		"INVALID_LOGICAL_EXPRESSION": "INVALID LOGICAL EXPRESSION",
		"VIRTUAL_SPRITE_COUNT_ERROR": "INVALID VIRTUAL SPRITE COUNT",
		"MISSING_QUOTES":             "STRING LITERAL REQUIRES QUOTES",
		"EXPECTED_STRING":            "STRING EXPECTED",
		"EXPECTED_NUMBER":            "NUMBER EXPECTED",
		"EXPECTED_OPEN_PAREN":        "OPEN PARENTHESIS (() EXPECTED",
		"UNEXPECTED_CHARACTER":       "UNEXPECTED CHARACTER IN EXPRESSION",
		"INVALID_ARRAY_INDEX":        "INVALID ARRAY INDEX",
		"INVALID_DIM_STATEMENT":      "INVALID DIM STATEMENT",
		"ARRAY_ALREADY_DIM":          "ARRAY ALREADY DIMENSIONED",
	},
	ErrCategoryRuntime: {
		"LINE_NOT_FOUND":       "PROGRAM LINE NOT FOUND",
		"RETURN_WITHOUT_GOSUB": "RETURN STATEMENT WITHOUT A CORRESPONDING GOSUB",
		"NEXT_WITHOUT_FOR":     "NEXT STATEMENT WITHOUT A CORRESPONDING FOR",
		"FOR_NEXT_MISMATCH":    "NEXT VARIABLE DOES NOT MATCH FOR VARIABLE", "OUT_OF_DATA": "READ STATEMENT WITH NO AVAILABLE DATA",
		"READ_MISSING_VARIABLE":   "READ STATEMENT IS MISSING A VARIABLE",
		"GOSUB_DEPTH_EXCEEDED":    "MAXIMUM GOSUB NESTING DEPTH EXCEEDED",
		"FOR_LOOP_DEPTH_EXCEEDED": "MAXIMUM FOR LOOP NESTING DEPTH EXCEEDED",
		"MISSING_NEXT":            "MISSING NEXT STATEMENT FOR FOR LOOP",
		"NEXT_VARIABLE_MISMATCH":  "NEXT VARIABLE DOES NOT MATCH FOR VARIABLE",
		"ARRAY_OUT_OF_BOUNDS":     "ARRAY INDEX OUT OF BOUNDS", "FOR_DEPTH": "FOR LOOP STACK OVERFLOW (TOO MANY NESTED LOOPS)",
		"GOSUB_DEPTH":        "GOSUB STACK OVERFLOW (TOO MANY NESTED CALLS)",
		"GOTO_INFINITE_LOOP": "INTERPRETER DEADLOCK DETECTED (EXCESSIVE GOTO LOOP ITERATIONS)",
		"FOR_NEXT_DEADLOCK":  "FOR/NEXT DEADLOCK DETECTED (GOTO SKIPS FOR LOOPS)",
	}, ErrCategoryEvaluation: {
		"INVALID_EXPRESSION":               "EXPRESSION CANNOT BE EVALUATED",
		"TYPE_MISMATCH":                    "TYPE MISMATCH IN EXPRESSION OR ASSIGNMENT",
		"ARRAY_INDEX_NOT_NUMERIC":          "ARRAY INDEX MUST BE NUMERIC",
		"DIVISION_BY_ZERO":                 "DIVISION BY ZERO",
		"UNKNOWN_VARIABLE":                 "VARIABLE NOT DEFINED",
		"OVERFLOW":                         "ARITHMETIC OVERFLOW",
		"OUT_OF_RANGE":                     "VALUE OUT OF RANGE",
		"NEGATIVE_SQRT":                    "NEGATIVE VALUE IN SQUARE ROOT",
		"INVALID_LOG":                      "NON-POSITIVE VALUE IN LOGARITHM",
		"INVALID_VALUE":                    "INVALID VALUE FOR OPERATION",
		"VECTOR_ID_PARAM_ERROR":            "VECTOR ID PARAMETER ERROR",
		"VECTOR_ID_TYPE_ERROR":             "VECTOR ID TYPE ERROR",
		"VECTOR_ID_RANGE_ERROR":            "VECTOR ID OUT OF RANGE",
		"VECTOR_BRIGHTNESS_PARAM_ERROR":    "VECTOR BRIGHTNESS PARAMETER ERROR",
		"VECTOR_BRIGHTNESS_TYPE_ERROR":     "VECTOR BRIGHTNESS TYPE ERROR",
		"NUMERIC_PARAM_ERROR":              "NUMERIC PARAMETER ERROR",
		"SPRITE_ID_PARAM_ERROR":            "SPRITE ID PARAMETER ERROR",
		"SPRITE_ID_TYPE_ERROR":             "SPRITE ID TYPE ERROR",
		"SPRITE_ID_RANGE_ERROR":            "SPRITE ID OUT OF RANGE",
		"SPRITE_PIXELDATA_PARAM_ERROR":     "SPRITE PIXEL DATA PARAMETER ERROR",
		"SPRITE_PIXELDATA_TYPE_ERROR":      "SPRITE PIXEL DATA TYPE ERROR",
		"SPRITE_DATA_SIZE_ERROR":           "SPRITE DATA SIZE ERROR",
		"SPRITE_PIXEL_VALUE_ERROR":         "SPRITE PIXEL VALUE ERROR",
		"SPRITE_PIXEL_RANGE_ERROR":         "SPRITE PIXEL RANGE ERROR",
		"SPRITE_PARAM_ERROR":               "SPRITE PARAMETER ERROR",
		"SPRITE_PARAM_TYPE_ERROR":          "SPRITE PARAMETER TYPE ERROR",
		"SPRITE_INSTANCE_ID_RANGE_ERROR":   "SPRITE INSTANCE ID OUT OF RANGE",
		"SPRITE_DEFINITION_ID_RANGE_ERROR": "SPRITE DEFINITION ID OUT OF RANGE",
		"VIRTUAL_SPRITE_ID_PARAM_ERROR":    "VIRTUAL SPRITE ID PARAMETER ERROR",
		"VIRTUAL_SPRITE_ID_TYPE_ERROR":     "VIRTUAL SPRITE ID TYPE ERROR",
		"VIRTUAL_SPRITE_ID_RANGE_ERROR":    "VIRTUAL SPRITE ID OUT OF RANGE",
		"LAYOUT_PARAM_ERROR":               "LAYOUT PARAMETER ERROR",
		"LAYOUT_TYPE_ERROR":                "LAYOUT TYPE ERROR",
		"LAYOUT_VALUE_ERROR":               "LAYOUT VALUE ERROR",
		"BASE_SPRITE_ID_PARAM_ERROR":       "BASE SPRITE ID PARAMETER ERROR",
		"BASE_SPRITE_ID_TYPE_ERROR":        "BASE SPRITE ID TYPE ERROR",
		"BASE_SPRITE_ID_RANGE_ERROR":       "BASE SPRITE ID OUT OF RANGE",
	},
	ErrCategoryFileSystem: {
		"NIL_FILESYSTEM":      "FILE SYSTEM IS NOT AVAILABLE",
		"INVALID_FILE_HANDLE": "INVALID FILE HANDLE NUMBER",
		"FILE_HANDLE_IN_USE":  "FILE HANDLE IS ALREADY IN USE",
		"FILE_NOT_OPEN":       "FILE IS NOT OPEN",
		"FILE_NOT_IN_INPUT":   "FILE IS NOT OPEN FOR INPUT",
		"FILE_NOT_IN_OUTPUT":  "FILE IS NOT OPEN FOR OUTPUT",
		"END_OF_FILE":         "ATTEMPTED TO READ PAST THE END OF FILE",
		"FILE_READ_ERROR":     "ERROR READING FROM FILE",
		"FILE_WRITE_ERROR":    "ERROR WRITING TO FILE",
		"FILE_NOT_FOUND":      "FILE NOT FOUND",
		"FILE_SYSTEM_ERROR":   "FILE SYSTEM OPERATION FAILED",
		"FILE_ALREADY_EXISTS": "FILE ALREADY EXISTS",
		"PERMISSION_DENIED":   "PERMISSION DENIED",
	},
	ErrCategoryCommand: {
		"HELP_NOT_FOUND": "NO HELP AVAILABLE FOR THE SPECIFIED COMMAND",
		"TEXT_TOO_LONG":  "TEXT TOO LONG FOR COMMAND",
	},
	ErrCategoryResource: {
		"MEMORY_FULL":       "INTERPRETER MEMORY IS FULL",
		"PROGRAM_TOO_LARGE": "PROGRAM TOO LARGE",
		"STACK_OVERFLOW":    "STACK OVERFLOW",
	}, ErrCategoryExecution: {
		"GENERAL_ERROR":         "AN UNEXPECTED ERROR OCCURRED DURING EXECUTION",
		"EXECUTION_CANCELLED":   "EXECUTION CANCELLED",
		"INTERNAL_ERROR":        "INTERNAL INTERPRETER ERROR",
		"NOT_IMPLEMENTED":       "FEATURE NOT IMPLEMENTED",
		"COMMAND_NOT_IN_DIRECT": "COMMAND NOT ALLOWED IN DIRECT MODE",
		"COMMAND_NOT_IN_PROG":   "COMMAND NOT ALLOWED IN PROGRAM MODE",
		"NO_PROGRAM_LINES":      "NO PROGRAM LINES TO EXECUTE",
		"NO_PROGRAM_LOADED":     "NO PROGRAM LOADED",
	},
	ErrCategoryIO: {
		"DEVICE_NOT_READY":    "DEVICE NOT READY FOR I/O OPERATION",
		"FILE_NOT_FOUND":      "FILE NOT FOUND",
		"FILE_ALREADY_OPEN":   "FILE ALREADY OPEN WITH THIS HANDLE",
		"FILE_NOT_OPEN":       "FILE NOT OPEN WITH THIS HANDLE",
		"WRONG_FILE_MODE":     "FILE NOT OPEN IN REQUIRED MODE (INPUT/OUTPUT)",
		"END_OF_FILE":         "END OF FILE REACHED",
		"IO_ERROR":            "INPUT/OUTPUT ERROR",
		"INVALID_FILE_HANDLE": "INVALID FILE HANDLE",
		"FILE_EXISTS":         "FILE ALREADY EXISTS",
		"PERMISSION_DENIED":   "PERMISSION DENIED",
		"FILE_IO_ERROR":       "FILE INPUT/OUTPUT ERROR",
		"INVALID_FILENAME":    "INVALID FILENAME",
		"FILE_SYSTEM_ERROR":   "FILE SYSTEM ERROR",
	},
	ErrCategorySystem: {
		"INTERNAL_ERROR":      "AN INTERNAL SYSTEM ERROR OCCURRED",
		"MESSAGE_SEND_FAILED": "FAILED TO SEND MESSAGE TO CLIENT",
	},
}

// GetCommandUsageHint gibt einen Syntaxhinweis für einen Befehl zurück
func GetCommandUsageHint(cmd string) string {
	cmd = strings.ToUpper(strings.TrimSpace(cmd))
	hint, exists := commandUsageHints[cmd]
	if !exists {
		return ""
	}
	return hint
}

// Spezifische Fehlermeldungen für häufige Fehlertypen
var specificErrorMessages = map[string]map[string]string{
	ErrCategorySyntax: {
		"MISSING_QUOTES":      "string literal requires quotes",
		"MISSING_PARENTHESIS": "missing closing parenthesis",
		"UNEXPECTED_TOKEN":    "unexpected token",
		"EXPECTED_EXPRESSION": "expression expected",
		"EXPECTED_VARIABLE":   "variable name expected",
		"EXPECTED_EQUALS":     "equals sign (=) expected",
		"EXPECTED_TO":         "TO keyword expected in FOR loop",
		"EXPECTED_THEN":       "THEN keyword expected after IF condition",
		"INVALID_NUMBER":      "invalid number format",
		"INVALID_LINE_NUMBER": "invalid line number (must be 1-9999)", "INVALID_VARIABLE_NAME": "invalid variable name format", // Sicherstellen, dass dieser Eintrag korrekt ist
		"MISSING_FILENAME":  "filename expected",
		"INVALID_FILE_TYPE": "only .bas files are supported",
		"MISSING_TEXT":      "text expected",
		"MISSING_HANDLE":    "file handle number expected",
		"COMMA_EXPECTED":    "comma expected",
		"INVALID_FILE_MODE": "invalid file mode (use INPUT or OUTPUT)",
		"MISSING_AS":        "AS keyword expected"}, ErrCategoryEvaluation: {
		"UNKNOWN_VARIABLE":        "variable not defined",
		"DIVISION_BY_ZERO":        "division by zero",
		"TYPE_MISMATCH":           "type mismatch (number/string)",
		"ARRAY_INDEX_NOT_NUMERIC": "array index must be numeric",
		"OVERFLOW":                "arithmetic overflow",
		"OUT_OF_RANGE":            "value out of range",
		"NEGATIVE_SQRT":           "negative value in square root",
		"INVALID_LOG":             "non-positive value in logarithm",
		"INVALID_ARRAY_INDEX":     "invalid array index",
	},
	ErrCategoryExecution: {
		"OUT_OF_DATA":          "out of DATA items to READ",
		"RETURN_WITHOUT_GOSUB": "RETURN without matching GOSUB",
		"NEXT_WITHOUT_FOR":     "NEXT without matching FOR",
		"FOR_NEXT_MISMATCH":    "FOR/NEXT variable mismatch",
		"GOSUB_DEPTH":          "GOSUB stack overflow (too many nested calls)", "FOR_DEPTH": "FOR loop stack overflow (too many nested loops)", "INVALID_VALUE": "invalid value for operation",
		"NO_PROGRAM_LINES":            "no program lines to execute",
		"NO_PROGRAM_LOADED":           "no program loaded",
		"unterminated string literal": "SYNTAX ERROR: MISSING CLOSING QUOTE",
	},
	ErrCategoryIO: {
		"FILE_NOT_FOUND":    "file not found",
		"FILE_ALREADY_OPEN": "file already open with given handle",
		"FILE_NOT_OPEN":     "file not open with this handle",
		"WRONG_FILE_MODE":   "file not open in required mode (INPUT/OUTPUT)",
		"END_OF_FILE":       "end of file reached",
		"IO_ERROR":          "input/output error",
		"FILE_SYSTEM_ERROR": "file system operation failed",
		"FILE_EXISTS":       "file already exists",
		"PERMISSION_DENIED": "permission denied",
	},
	ErrCategorySystem: {
		"MEMORY_FULL":       "memory full",
		"INTERNAL_ERROR":    "internal interpreter error",
		"PROGRAM_TOO_LARGE": "program too large",
		"STACK_OVERFLOW":    "stack overflow",
	},
}

// Tabelle mit Syntaxhinweisen für Befehle
var commandUsageHints = map[string]string{
	"PRINT":      "PRINT [expr][,|;]... or PRINT \"text\"",
	"LET":        "LET var = expr",
	"IF":         "IF condition THEN statement",
	"FOR":        "FOR var = start TO end [STEP value]",
	"NEXT":       "NEXT var",
	"INPUT":      "INPUT [\"prompt\";] var",
	"GOTO":       "GOTO lineNumber",
	"GOSUB":      "GOSUB lineNumber",
	"RETURN":     "RETURN",
	"END":        "END",
	"REM":        "REM comment",
	"BEEP":       "BEEP",
	"SOUND":      "SOUND frequency, duration",
	"SAY":        "SAY \"text\" or SAY stringVar$",
	"SPEAK":      "SPEAK \"text\" or SPEAK stringVar$",
	"CLS":        "CLS",
	"LOAD":       "LOAD \"filename\"",
	"SAVE":       "SAVE \"filename\"",
	"DIR":        "DIR",
	"LIST":       "LIST [startLine][-endLine]",
	"RUN":        "RUN",
	"PLOT":       "PLOT x, y",
	"DRAW":       "DRAW x1, y1, x2, y2",
	"CIRCLE":     "CIRCLE x, y, radius",
	"RECT":       "RECT x, y, width, height",
	"FILL":       "FILL x, y",
	"INK":        "INK color",
	"POLY":       "POLY x1, y1, x2, y2, ...",
	"OPEN":       "OPEN \"filename\" FOR INPUT|OUTPUT AS #handle",
	"CLOSE":      "CLOSE #handle",
	"LINE INPUT": "LINE INPUT #handle, var$",
	"DATA":       "DATA item1, item2, ...",
	"READ":       "READ var1, var2, ...",
	"RESTORE":    "RESTORE",
}

// GetFriendlyErrorText retrieves a user-friendly error message.
// Hilfsfunktion: Benutzerfreundlichen Fehlertext holen
func GetFriendlyErrorText(category, code string) string {
	categoryErrors, categoryExists := FriendlyErrorTexts[category]
	if !categoryExists {
		// Debug: Log unknown category
		fmt.Printf("DEBUG: Unknown error category: '%s', code: '%s'\n", category, code)
		return fmt.Sprintf("UNKNOWN ERROR CATEGORY: %s (code: %s)", category, code)
	}
	errorMessage, errorExists := categoryErrors[code]
	if !errorExists {
		// Debug: Log unknown error code
		fmt.Printf("DEBUG: Unknown error code in category '%s': '%s'\n", category, code)
		fmt.Printf("DEBUG: Available codes in category '%s': %v\n", category, getKeysFromMap(categoryErrors))
		return fmt.Sprintf("UNKNOWN ERROR CODE: %s", code)
	}
	return errorMessage
}

// Helper function to get keys from a map for debugging
func getKeysFromMap(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// WrapError wandelt einen beliebigen Fehler in einen BASICError um (mit optionalem Befehl, Modus, Zeile)
func WrapError(err error, command string, directMode bool, lineNumber int) *BASICError {
	if be, ok := err.(*BASICError); ok {
		if command != "" {
			be.Command = command
		}
		return be
	}
	return &BASICError{
		Category:   ErrCategoryExecution,
		Message:    err.Error(),
		Command:    command,
		DirectMode: directMode,
		LineNumber: lineNumber,
		Detail:     err.Error(),
	}
}

// FormatErrorAsMessages converts an error into a Message array (for output)
func FormatErrorAsMessages(err error) []shared.Message {
	if err == nil {
		return nil
	}
	if help, ok := err.(*helpLinesAsError); ok {
		return help.lines
	}
	lines := strings.Split(err.Error(), "\n")
	msgs := make([]shared.Message, 0, len(lines))
	for _, l := range lines {
		msgs = append(msgs, shared.Message{Type: shared.MessageTypeText, Content: l})
	}
	return msgs
}

// ErrorTexts maps error codes to human-readable messages.
var ErrorTexts = map[string]string{
	// Syntax Errors
	"EXPECTED_COMMAND":    "COMMAND EXPECTED",
	"EXPECTED_VARIABLE":   "VARIABLE EXPECTED",
	"EXPECTED_STRING":     "STRING EXPECTED",
	"EXPECTED_NUMBER":     "NUMBER EXPECTED",
	"EXPECTED_EQUALS":     "EQUALS SIGN (=) EXPECTED",
	"EXPECTED_OPEN_PAREN": "OPEN PARENTHESIS (() EXPECTED",
	"INVALID_LINE_NUMBER": "INVALID LINE NUMBER (MUST BE 1-9999)", "INVALID_VARIABLE_NAME": "INVALID VARIABLE NAME FORMAT",
	"MISSING_FILENAME":           "FILENAME EXPECTED",
	"INVALID_FILE_TYPE":          "ONLY .BAS FILES ARE SUPPORTED",
	"UNEXPECTED_CHARACTER":       "UNEXPECTED CHARACTER IN EXPRESSION",
	"UNEXPECTED_TOKEN":           "UNEXPECTED TOKEN",
	"INVALID_EXPRESSION":         "INVALID EXPRESSION",
	"INVALID_ARRAY_INDEX":        "INVALID ARRAY INDEX",
	"INVALID_DIM_STATEMENT":      "INVALID DIM STATEMENT",
	"ARRAY_ALREADY_DIM":          "ARRAY ALREADY DIMENSIONED",
	"EXPECTED_TO":                "TO KEYWORD EXPECTED IN FOR LOOP",
	"EXPECTED_THEN":              "THEN KEYWORD EXPECTED AFTER IF CONDITION",
	"INVALID_NUMBER":             "INVALID NUMBER FORMAT",
	"MISSING_QUOTES":             "STRING LITERAL REQUIRES QUOTES",
	"MISSING_PARENTHESIS":        "MISSING CLOSING PARENTHESIS",
	"INVALID_ARGUMENT":           "INVALID ARGUMENT FOR COMMAND",
	"MISSING_ARGUMENT":           "MISSING ARGUMENT FOR COMMAND",
	"TOO_MANY_ARGUMENTS":         "TOO MANY ARGUMENTS FOR COMMAND",
	"MISSING_TEXT":               "TEXT EXPECTED",
	"MISSING_HANDLE":             "FILE HANDLE NUMBER EXPECTED",
	"COMMA_EXPECTED":             "COMMA EXPECTED",
	"INVALID_FILE_MODE":          "INVALID FILE MODE (USE INPUT OR OUTPUT)",
	"MISSING_AS":                 "AS KEYWORD EXPECTED",
	"INVALID_PARAMETER_COUNT":    "INVALID NUMBER OF PARAMETERS",
	"INVALID_COLOR_FORMAT":       "INVALID COLOR FORMAT",
	"INVALID_PARAMETER_VALUE":    "INVALID PARAMETER VALUE",
	"SYNTAX_ERROR":               "SYNTAX ERROR",
	"INVALID_LOGICAL_EXPRESSION": "INVALID LOGICAL EXPRESSION",
	"VIRTUAL_SPRITE_COUNT_ERROR": "INVALID VIRTUAL SPRITE COUNT",
	// Runtime Errors
	"TYPE_MISMATCH":           "TYPE MISMATCH",
	"ARRAY_INDEX_NOT_NUMERIC": "ARRAY INDEX NOT NUMERIC",
	"DIVISION_BY_ZERO":        "DIVISION BY ZERO",
	"OUT_OF_DATA":             "OUT OF DATA",
	"NEXT_WITHOUT_FOR":        "NEXT WITHOUT FOR",
	"RETURN_WITHOUT_GOSUB":    "RETURN WITHOUT GOSUB",
	"ARRAY_OUT_OF_BOUNDS":     "ARRAY INDEX OUT OF BOUNDS",
	"UNKNOWN_VARIABLE":        "UNKNOWN VARIABLE",
	"LINE_NOT_FOUND":          "PROGRAM LINE NOT FOUND",
	"FOR_NEXT_MISMATCH":       "FOR/NEXT VARIABLE MISMATCH",
	"READ_MISSING_VARIABLE":   "READ STATEMENT IS MISSING A VARIABLE",
	"GOSUB_DEPTH_EXCEEDED":    "MAXIMUM GOSUB NESTING DEPTH EXCEEDED",
	"FOR_LOOP_DEPTH_EXCEEDED": "MAXIMUM FOR LOOP NESTING DEPTH EXCEEDED",
	"MISSING_NEXT":            "MISSING NEXT STATEMENT FOR FOR LOOP",
	"NEXT_VARIABLE_MISMATCH":  "NEXT VARIABLE DOES NOT MATCH FOR VARIABLE",
	"FOR_DEPTH":               "FOR LOOP STACK OVERFLOW",
	"GOSUB_DEPTH":             "GOSUB STACK OVERFLOW",

	// Evaluation Errors
	"OVERFLOW":      "ARITHMETIC OVERFLOW",
	"OUT_OF_RANGE":  "VALUE OUT OF RANGE",
	"NEGATIVE_SQRT": "NEGATIVE VALUE IN SQUARE ROOT",
	"INVALID_LOG":   "NON-POSITIVE VALUE IN LOGARITHM",
	"INVALID_VALUE": "INVALID VALUE FOR OPERATION",

	// File Errors
	"FILE_NOT_FOUND":      "FILE NOT FOUND",
	"FILE_IO_ERROR":       "FILE INPUT/OUTPUT ERROR",
	"FILE_ALREADY_EXISTS": "FILE ALREADY EXISTS",
	"INVALID_FILENAME":    "INVALID FILENAME",
	"FILE_SYSTEM_ERROR":   "FILE SYSTEM ERROR",
	"FILE_ALREADY_OPEN":   "FILE ALREADY OPEN",
	"FILE_NOT_OPEN":       "FILE NOT OPEN",
	"WRONG_FILE_MODE":     "FILE NOT OPEN IN REQUIRED MODE",
	"END_OF_FILE":         "END OF FILE REACHED",
	"INVALID_FILE_HANDLE": "INVALID FILE HANDLE",
	"FILE_READ_ERROR":     "ERROR READING FROM FILE",
	"FILE_WRITE_ERROR":    "ERROR WRITING TO FILE",
	"PERMISSION_DENIED":   "PERMISSION DENIED",

	// Graphics Errors
	"GFX_INVALID_MODE":    "INVALID GRAPHICS MODE",
	"GFX_INVALID_COORD":   "INVALID GRAPHICS COORDINATE",
	"GFX_INVALID_COLOR":   "INVALID GRAPHICS COLOR",
	"GFX_INVALID_COMMAND": "INVALID GRAPHICS COMMAND",

	// Sprite Errors
	"SPRITE_INVALID_ID":                "INVALID SPRITE ID",
	"SPRITE_INVALID_DEF_ID":            "INVALID SPRITE DEFINITION ID",
	"SPRITE_INVALID_DATA":              "INVALID SPRITE DATA FORMAT",
	"SPRITE_NOT_DEFINED":               "SPRITE DEFINITION NOT FOUND",
	"SPRITE_MAX_REACHED":               "MAXIMUM NUMBER OF SPRITES REACHED",
	"SPRITE_DEF_MAX_REACHED":           "MAXIMUM NUMBER OF SPRITE DEFINITIONS REACHED",
	"SPRITE_INVALID_PARAM":             "INVALID SPRITE PARAMETER",
	"SPRITE_COMP_INVALID_ID":           "INVALID VIRTUAL SPRITE COMPOSITION ID",
	"SPRITE_COMP_MAX_REACHED":          "MAXIMUM NUMBER OF VIRTUAL SPRITE COMPOSITIONS REACHED",
	"SPRITE_COMP_NOT_DEFINED":          "VIRTUAL SPRITE COMPOSITION NOT FOUND",
	"SPRITE_ID_PARAM_ERROR":            "SPRITE ID PARAMETER ERROR",
	"SPRITE_ID_TYPE_ERROR":             "SPRITE ID TYPE ERROR",
	"SPRITE_ID_RANGE_ERROR":            "SPRITE ID OUT OF RANGE",
	"SPRITE_PIXELDATA_PARAM_ERROR":     "SPRITE PIXEL DATA PARAMETER ERROR",
	"SPRITE_PIXELDATA_TYPE_ERROR":      "SPRITE PIXEL DATA TYPE ERROR",
	"SPRITE_DATA_SIZE_ERROR":           "SPRITE DATA SIZE ERROR",
	"SPRITE_PIXEL_VALUE_ERROR":         "SPRITE PIXEL VALUE ERROR",
	"SPRITE_PIXEL_RANGE_ERROR":         "SPRITE PIXEL RANGE ERROR",
	"SPRITE_PARAM_ERROR":               "SPRITE PARAMETER ERROR",
	"SPRITE_PARAM_TYPE_ERROR":          "SPRITE PARAMETER TYPE ERROR",
	"SPRITE_INSTANCE_ID_RANGE_ERROR":   "SPRITE INSTANCE ID OUT OF RANGE",
	"SPRITE_DEFINITION_ID_RANGE_ERROR": "SPRITE DEFINITION ID OUT OF RANGE",
	"VIRTUAL_SPRITE_ID_PARAM_ERROR":    "VIRTUAL SPRITE ID PARAMETER ERROR",
	"VIRTUAL_SPRITE_ID_TYPE_ERROR":     "VIRTUAL SPRITE ID TYPE ERROR",
	"VIRTUAL_SPRITE_ID_RANGE_ERROR":    "VIRTUAL SPRITE ID OUT OF RANGE",
	"LAYOUT_PARAM_ERROR":               "LAYOUT PARAMETER ERROR",
	"LAYOUT_TYPE_ERROR":                "LAYOUT TYPE ERROR",
	"LAYOUT_VALUE_ERROR":               "LAYOUT VALUE ERROR",
	"BASE_SPRITE_ID_PARAM_ERROR":       "BASE SPRITE ID PARAMETER ERROR",
	"BASE_SPRITE_ID_TYPE_ERROR":        "BASE SPRITE ID TYPE ERROR",
	"BASE_SPRITE_ID_RANGE_ERROR":       "BASE SPRITE ID OUT OF RANGE",

	// Vector Errors
	"VECTOR_INVALID_ID":             "INVALID VECTOR ID",
	"VECTOR_INVALID_DATA":           "INVALID VECTOR DATA FORMAT",
	"VECTOR_MAX_REACHED":            "MAXIMUM NUMBER OF VECTORS REACHED",
	"VECTOR_INVALID_COMMAND":        "INVALID VECTOR COMMAND",
	"VECTOR_INVALID_PARAM":          "INVALID VECTOR PARAMETER",
	"VECTOR_ID_PARAM_ERROR":         "VECTOR ID PARAMETER ERROR",
	"VECTOR_ID_TYPE_ERROR":          "VECTOR ID TYPE ERROR",
	"VECTOR_ID_RANGE_ERROR":         "VECTOR ID OUT OF RANGE",
	"VECTOR_BRIGHTNESS_PARAM_ERROR": "VECTOR BRIGHTNESS PARAMETER ERROR",
	"VECTOR_BRIGHTNESS_TYPE_ERROR":  "VECTOR BRIGHTNESS TYPE ERROR",
	"NUMERIC_PARAM_ERROR":           "NUMERIC PARAMETER ERROR",

	// Sound Errors
	"SOUND_INVALID_COMMAND":   "INVALID SOUND COMMAND",
	"SOUND_INVALID_PARAM":     "INVALID SOUND PARAMETER",
	"SOUND_PLAY_ERROR":        "ERROR PLAYING SOUND",
	"INVALID_NOISE_PARAMETER": "INVALID NOISE PARAMETER",

	// System/Interpreter Errors
	"UNKNOWN_COMMAND":       "UNKNOWN COMMAND",
	"INTERNAL_ERROR":        "INTERNAL INTERPRETER ERROR",
	"NOT_IMPLEMENTED":       "FEATURE NOT IMPLEMENTED",
	"PROGRAM_TOO_LARGE":     "PROGRAM TOO LARGE",
	"MEMORY_FULL":           "MEMORY FULL",
	"COMMAND_NOT_IN_DIRECT": "COMMAND NOT ALLOWED IN DIRECT MODE",
	"COMMAND_NOT_IN_PROG":   "COMMAND NOT ALLOWED IN PROGRAM MODE", "EXECUTION_CANCELLED": "EXECUTION CANCELLED",
	"MESSAGE_SEND_FAILED": "FAILED TO SEND MESSAGE TO CLIENT",
	"TEXT_TOO_LONG":       "TEXT TOO LONG FOR COMMAND", "RATE_LIMIT_EXCEEDED": "COMMAND RATE LIMIT EXCEEDED",
	"NOISE_RATE_LIMIT_EXCEEDED":   "NOISE COMMAND RATE LIMIT EXCEEDED",
	"NO_PROGRAM_LINES":            "NO PROGRAM LINES TO EXECUTE",
	"NO_PROGRAM_LOADED":           "NO PROGRAM LOADED",
	"STACK_OVERFLOW":              "STACK OVERFLOW",
	"unterminated string literal": "SYNTAX ERROR: MISSING CLOSING QUOTE",
}
