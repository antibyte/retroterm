package tinybasic

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antibyte/retroterm/pkg/shared"
)

// cmdLoad loads a program. Assumes lock is held.
func (b *TinyBASIC) cmdLoad(args string) error {
	filenameExpr := strings.TrimSpace(args)
	if filenameExpr == "" {
		return NewBASICError(ErrCategorySyntax, "MISSING_FILENAME", b.currentLine == 0, b.currentLine).WithCommand("LOAD")
	}
	filenameVal, err := b.evalExpression(filenameExpr)
	if err != nil || filenameVal.IsNumeric {
		return NewBASICError(ErrCategoryEvaluation, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("LOAD")
	}
	filename := filenameVal.StrValue
	if !strings.Contains(filename, ".") {
		filename += ".bas"
	}
	if b.fs == nil {
		return NewBASICError(ErrCategoryFileSystem, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("LOAD")
	}
	content, err := b.fs.ReadFile(filename, b.sessionID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return NewBASICError(ErrCategoryFileSystem, "FILE_NOT_FOUND", b.currentLine == 0, b.currentLine).WithCommand("LOAD")
		}
		return NewBASICError(ErrCategoryFileSystem, "FILE_READ_ERROR", b.currentLine == 0, b.currentLine).WithCommand("LOAD")
	}
	b.program = make(map[int]string)
	b.variables = make(map[string]BASICValue)

	// Tastaturkonstanten nach Reset wiederherstellen
	b.initializeKeyConstants()

	b.gosubStack = b.gosubStack[:0]
	b.forLoops = b.forLoops[:0]
	b.data = make([]string, 0)
	b.dataPointer = 0
	b.currentLine = 0
	b.running = false
	b.inputVar = ""
	b.closeAllFiles()

	// Clean the content to remove non-printable characters that could cause parsing issues
	content = cleanCodeForLoading(content)

	lines := strings.Split(content, "\n")
	linesLoaded := 0
	for _, line := range lines {
		if num, code, isLine := parseProgramLine(line); isLine && code != "" {
			b.program[num] = code
			linesLoaded++
		}
	}
	b.rebuildProgramLines()
	b.rebuildData()
	b.sendMessage(shared.MessageTypeText, fmt.Sprintf("Loaded %d lines.", linesLoaded))
	return nil
}

// cmdSave saves the current program. Assumes lock is held.
func (b *TinyBASIC) cmdSave(args string) error {
	filenameExpr := strings.TrimSpace(args)
	if filenameExpr == "" {
		return NewBASICError(ErrCategorySyntax, "MISSING_FILENAME", b.currentLine == 0, b.currentLine).WithCommand("SAVE")
	}
	filenameVal, err := b.evalExpression(filenameExpr)
	if err != nil || filenameVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("SAVE")
	}
	filename := filenameVal.StrValue
	if !strings.Contains(filename, ".") {
		filename += ".bas"
	}
	if b.fs == nil {
		return NewBASICError(ErrCategoryFileSystem, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("SAVE")
	}
	var sb strings.Builder
	for _, num := range b.programLines {
		sb.WriteString(fmt.Sprintf("%d %s\n", num, b.program[num]))
	}
	content := sb.String()
	err = b.fs.WriteFile(filename, content, b.sessionID)
	if err != nil {
		return NewBASICError(ErrCategoryFileSystem, "FILE_WRITE_ERROR", b.currentLine == 0, b.currentLine).WithCommand("SAVE")
	}
	b.sendMessage(shared.MessageTypeText, fmt.Sprintf("Saved %d lines.", len(b.programLines)))
	return nil
}

// cmdDir lists all files. Returns listing string. Assumes lock is held.
func (b *TinyBASIC) cmdDir(args string) (string, error) {
	if args != "" {
		return "", NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("DIR")
	}
	if b.fs == nil {
		return "", NewBASICError(ErrCategoryFileSystem, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("DIR")
	}
	// Get all files using the same logic as ListDirProgramFiles but without filtering
	files, err := b.listAllFiles()
	if err != nil {
		return "", NewBASICError(ErrCategoryFileSystem, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("DIR")
	}
	if len(files) == 0 {
		return "No files found.", nil
	}

	// Create a compact list like at BASIC startup
	result := "Files: " + strings.Join(files, ", ")

	// Add storage usage information
	if b.os != nil && b.os.Vfs != nil {
		username := b.os.Username()
		if storageInfo, err := b.os.Vfs.GetUserStorageInfo(username); err == nil {
			result += fmt.Sprintf("\n%d of %d KB used", storageInfo.UsedKB, storageInfo.TotalKB)
		}
	}

	return result, nil
}

// cmdOpen opens a file. Assumes lock is held.
func (b *TinyBASIC) cmdOpen(args string) error {
	// Syntax: OPEN <filename_expr> FOR INPUT|OUTPUT AS #<handle_number>
	parts := splitRespectingQuotes(args)
	if len(parts) < 5 || !strings.EqualFold(parts[1], "FOR") || !strings.EqualFold(parts[3], "AS") {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
	}

	filenameExpr := parts[0]
	modeStr := strings.ToUpper(parts[2])
	handleStr := parts[4]

	// Evaluate filename expression
	filenameVal, err := b.evalExpression(filenameExpr)
	if err != nil || filenameVal.IsNumeric {
		return NewBASICError(ErrCategorySyntax, "INVALID_EXPRESSION", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
	}
	filename := filenameVal.StrValue // Already trimmed by eval if it was quoted string

	if modeStr != "INPUT" && modeStr != "OUTPUT" {
		return NewBASICError(ErrCategorySyntax, "INVALID_FILE_MODE", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
	}
	if !strings.HasPrefix(handleStr, "#") {
		return NewBASICError(ErrCategorySyntax, "MISSING_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
	}
	handle, err := strconv.Atoi(handleStr[1:])
	if err != nil || handle <= 0 {
		return NewBASICError(ErrCategoryIO, "INVALID_FILE_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
	}

	if b.fs == nil {
		return NewBASICError(ErrCategoryIO, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
	}
	if _, exists := b.openFiles[handle]; exists {
		return NewBASICError(ErrCategoryIO, "FILE_ALREADY_OPEN", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
	}

	if modeStr == "INPUT" {
		content, err := b.fs.ReadFile(filename, b.sessionID)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				return NewBASICError(ErrCategoryIO, "FILE_NOT_FOUND", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
			}
			return NewBASICError(ErrCategoryIO, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("OPEN")
		}
		content = strings.ReplaceAll(content, "\r\n", "\n") // Normalize newlines.
		lines := strings.Split(content, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		b.openFiles[handle] = &OpenFile{Name: filename, Mode: modeStr, Lines: lines, Pos: 0}
	} else { // OUTPUT
		b.openFiles[handle] = &OpenFile{Name: filename, Mode: modeStr, WriteBuf: []string{}}
	}
	return nil
}

// cmdClose closes a file. Assumes lock is held.
func (b *TinyBASIC) cmdClose(args string) error {
	// Syntax: CLOSE #<handle_number>
	handleStr := strings.TrimSpace(args)
	if !strings.HasPrefix(handleStr, "#") {
		return NewBASICError(ErrCategorySyntax, "MISSING_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("CLOSE")
	}
	handle, err := strconv.Atoi(handleStr[1:])
	if err != nil || handle <= 0 {
		return NewBASICError(ErrCategoryIO, "INVALID_FILE_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("CLOSE")
	}

	of, ok := b.openFiles[handle]
	if !ok {
		return NewBASICError(ErrCategoryIO, "FILE_NOT_OPEN", b.currentLine == 0, b.currentLine).WithCommand("CLOSE")
	}

	// If output mode, write the buffer content to the file.
	if of.Mode == "OUTPUT" && len(of.WriteBuf) > 0 {
		if b.fs == nil {
			delete(b.openFiles, handle)
			return NewBASICError(ErrCategoryIO, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("CLOSE")
		}
		content := strings.Join(of.WriteBuf, "\n") + "\n"
		err := b.fs.WriteFile(of.Name, content, b.sessionID)
		if err != nil {
			delete(b.openFiles, handle)
			return NewBASICError(ErrCategoryIO, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("CLOSE")
		}
	}

	delete(b.openFiles, handle)
	return nil
}

// cmdInputFile reads data from a file into variables. Assumes lock is held.
func (b *TinyBASIC) cmdInputFile(args string) error {
	// Syntax: INPUT #<handle>, <var1> [, <var2>...]
	parts := strings.SplitN(args, ",", 2)
	if len(parts) != 2 {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
	}

	handlePart := strings.TrimSpace(parts[0]) // Should be "#n"
	varListStr := strings.TrimSpace(parts[1])

	if !strings.HasPrefix(handlePart, "#") {
		return NewBASICError(ErrCategorySyntax, "MISSING_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
	}
	handle, err := strconv.Atoi(handlePart[1:])
	if err != nil || handle <= 0 {
		return NewBASICError(ErrCategoryIO, "INVALID_FILE_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
	}
	if varListStr == "" {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
	}

	of, ok := b.openFiles[handle]
	if !ok {
		return NewBASICError(ErrCategoryIO, "FILE_NOT_OPEN", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
	}
	if of.Mode != "INPUT" {
		return NewBASICError(ErrCategoryIO, "WRONG_FILE_MODE", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
	}

	// Split variable list (basic split, assumes no commas in var names).
	varNames := strings.Split(varListStr, ",")

	// Read one line and assign values. INPUT # reads fields from a line, unlike LINE INPUT #.
	// Standard INPUT # might read comma-separated values from a single line, or one value per line.
	// Let's implement simple one-value-per-non-empty-line behavior.
	for _, varNameRaw := range varNames {
		varName := strings.ToUpper(strings.TrimSpace(varNameRaw))
		if varName == "" {
			return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
		}

		// Find the next non-empty line.
		line := ""
		foundLine := false
		for of.Pos < len(of.Lines) {
			currentLine := of.Lines[of.Pos]
			if strings.TrimSpace(currentLine) != "" {
				line = currentLine // Use the line content.
				foundLine = true
				of.Pos++ // Consume the line.
				break
			}
			of.Pos++ // Skip empty/whitespace lines.
		}

		if !foundLine {
			return NewBASICError(ErrCategoryIO, "END_OF_FILE", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
		}

		// Assign value (handle type conversion).
		trimmedLine := strings.TrimSpace(line)
		if strings.HasSuffix(varName, "$") { // Target is string.
			b.variables[varName] = BASICValue{StrValue: trimmedLine, IsNumeric: false}
		} else {
			val, err := strconv.ParseFloat(trimmedLine, 64)
			if err != nil {
				return NewBASICError(ErrCategoryEvaluation, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("INPUT #")
			}
			b.variables[varName] = BASICValue{NumValue: val, IsNumeric: true}
		}
		// Entfernt: Trace-Logausgabe
	}
	return nil
}

// cmdLineInputFile reads an entire line from a file. Assumes lock is held.
func (b *TinyBASIC) cmdLineInputFile(args string) error {
	// Syntax after parsing command: #<handle>, <string_var$>
	parts := strings.SplitN(args, ",", 2)
	if len(parts) != 2 {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("LINE INPUT #")
	}

	handlePart := strings.TrimSpace(parts[0])
	varName := strings.TrimSpace(parts[1])

	if !strings.HasPrefix(handlePart, "#") {
		return NewBASICError(ErrCategorySyntax, "MISSING_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("LINE INPUT #")
	}
	handle, err := strconv.Atoi(handlePart[1:])
	if err != nil || handle <= 0 {
		return NewBASICError(ErrCategoryIO, "INVALID_FILE_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("LINE INPUT #")
	}
	if varName == "" {
		return NewBASICError(ErrCategorySyntax, "EXPECTED_VARIABLE", b.currentLine == 0, b.currentLine).WithCommand("LINE INPUT #")
	}
	if !strings.HasSuffix(varName, "$") {
		return NewBASICError(ErrCategorySyntax, "TYPE_MISMATCH", b.currentLine == 0, b.currentLine).WithCommand("LINE INPUT #")
	}

	of, ok := b.openFiles[handle]
	if !ok {
		return NewBASICError(ErrCategoryIO, "FILE_NOT_OPEN", b.currentLine == 0, b.currentLine).WithCommand("LINE INPUT #")
	}
	if of.Mode != "INPUT" {
		return NewBASICError(ErrCategoryIO, "WRONG_FILE_MODE", b.currentLine == 0, b.currentLine).WithCommand("LINE INPUT #")
	}

	// Read the *next* available line, including empty ones.
	if of.Pos >= len(of.Lines) {
		return NewBASICError(ErrCategoryIO, "END_OF_FILE", b.currentLine == 0, b.currentLine).WithCommand("LINE INPUT #")
	}

	line := of.Lines[of.Pos]
	of.Pos++ // Consume the line.

	varNameUpper := strings.ToUpper(varName)
	b.variables[varNameUpper] = BASICValue{StrValue: line, IsNumeric: false}
	return nil
}

// cmdPrintFile writes expression results to a file. Assumes lock is held.
func (b *TinyBASIC) cmdPrintFile(args string) error {
	// Syntax: PRINT #<handle>, <expr1> [,|<expr2>...]
	commaIdx := strings.Index(args, ",")
	if commaIdx == -1 {
		return NewBASICError(ErrCategorySyntax, "UNEXPECTED_TOKEN", b.currentLine == 0, b.currentLine).WithCommand("PRINT #")
	}

	handlePart := strings.TrimSpace(args[:commaIdx])
	exprListStr := strings.TrimSpace(args[commaIdx+1:])

	if !strings.HasPrefix(handlePart, "#") {
		return NewBASICError(ErrCategorySyntax, "MISSING_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("PRINT #")
	}
	handle, err := strconv.Atoi(handlePart[1:])
	if err != nil || handle <= 0 {
		return NewBASICError(ErrCategoryIO, "INVALID_FILE_HANDLE", b.currentLine == 0, b.currentLine).WithCommand("PRINT #")
	}

	of, ok := b.openFiles[handle]
	if !ok {
		return NewBASICError(ErrCategoryIO, "FILE_NOT_OPEN", b.currentLine == 0, b.currentLine).WithCommand("PRINT #")
	}
	if of.Mode != "OUTPUT" {
		return NewBASICError(ErrCategoryIO, "WRONG_FILE_MODE", b.currentLine == 0, b.currentLine).WithCommand("PRINT #")
	}

	// Process expressions similarly to console PRINT.
	var outputBuilder strings.Builder
	items, separators := tokenizePrintArgs(exprListStr) // Reuse print tokenizer.

	for i, itemStr := range items {
		itemStr = strings.TrimSpace(itemStr)
		if itemStr == "" {
			// Handle empty item -> affects spacing if comma was used.
			// File output often doesn't use tabbing, just space/nothing.
			// Assume comma just adds a space for file output? Or nothing? Let's add nothing.
			continue
		}

		val, err := b.evalExpression(itemStr)
		if err != nil {
			if strings.HasPrefix(itemStr, "\"") && strings.HasSuffix(itemStr, "\"") {
				val = BASICValue{StrValue: itemStr[1 : len(itemStr)-1], IsNumeric: false}
				err = nil
			} else {
				return WrapError(err, "PRINT #", b.currentLine == 0, b.currentLine)
			}
		}

		// Format value for file output (no leading spaces usually).
		var currentItemOutput string
		if val.IsNumeric {
			currentItemOutput = strconv.FormatFloat(val.NumValue, 'g', -1, 64)
		} else {
			currentItemOutput = val.StrValue
		}

		// Handle separator: comma might add delimiter (like actual comma?), semicolon nothing.
		sep := ""
		if i < len(separators) {
			if separators[i] == ',' {
				sep = "," // Output comma separator for files? Or tab? Let's use comma.
			} // Semicolon adds nothing.
		}

		outputBuilder.WriteString(currentItemOutput)
		outputBuilder.WriteString(sep)
	}

	// Each PRINT # writes one line to the buffer.
	outputLine := outputBuilder.String()
	of.WriteBuf = append(of.WriteBuf, outputLine)
	return nil
}

// listAllFiles returns all files in the user's home directory (similar to ListDirProgramFiles but without filtering)
func (b *TinyBASIC) listAllFiles() ([]string, error) {
	// Use the new VFS method to get all files
	return b.fs.ListDirAllFiles(b.sessionID)
}
