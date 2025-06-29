package tinybasic

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// cmdMCP handles the MCP (Model Context Protocol) commands for AI-assisted code generation
func (b *TinyBASIC) cmdMCP(args string) error {
	if args == "" {
		return NewBASICError(ErrCategorySyntax, "MISSING_ARGUMENTS", b.currentLine == 0, b.currentLine).WithCommand("MCP")
	}

	// Get username from session
	username := ""
	if b.os != nil {
		username = b.os.GetUsernameForSession(b.sessionID)
	}

	// Check if user is guest
	if username == "" || username == "guest" {
		b.sendMessageWrapped(shared.MessageTypeText, "Please register and login to use this command")
		return nil
	}

	// Parse MCP command
	parts := strings.Fields(args)
	if len(parts) < 1 {
		return NewBASICError(ErrCategorySyntax, "MISSING_SUBCOMMAND", b.currentLine == 0, b.currentLine).WithCommand("MCP")
	}

	subCommand := strings.ToUpper(parts[0])

	switch subCommand {
	case "CREATE":
		return b.cmdMCPCreate(strings.Join(parts[1:], " "))
	case "EDIT":
		if len(parts) < 2 {
			return NewBASICError(ErrCategorySyntax, "MISSING_FILENAME", b.currentLine == 0, b.currentLine).WithCommand("MCP EDIT")
		}
		filename := parts[1]
		instruction := strings.Join(parts[2:], " ")
		return b.cmdMCPEdit(filename, instruction)
	default:
		return NewBASICError(ErrCategorySyntax, "UNKNOWN_SUBCOMMAND", b.currentLine == 0, b.currentLine).WithCommand("MCP")
	}
}

// cmdMCPCreate handles "MCP CREATE <description>" command
func (b *TinyBASIC) cmdMCPCreate(description string) error {
	// DEBUG: Log function entry
	logger.Debug(logger.AreaTinyBasic, "[MCP] cmdMCPCreate called with description: %s", description)
	if description == "" {
		return NewBASICError(ErrCategorySyntax, "MISSING_DESCRIPTION", b.currentLine == 0, b.currentLine).WithCommand("MCP CREATE")
	}

	// Check description length (max 512 characters)
	if len(description) > 512 {
		b.sendMessageWrapped(shared.MessageTypeText, "Your request is too long (max 512 chars)")
		return nil
	}

	// Get username
	username := b.os.GetUsernameForSession(b.sessionID)

	// Check rate limits
	allowed, message, _ := b.checkMCPRateLimit(username)
	if !allowed {
		b.sendMessageWrapped(shared.MessageTypeText, message)
		return nil
	} // Show thinking message
	b.sendMessageWrapped(shared.MessageTypeText, "MCP is thinking, please wait...")
	// Get the MCP CREATE prompt from PromptManager with template data
	if b.os == nil || b.os.PromptManager == nil {
		return NewBASICError(ErrCategorySystem, "SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("MCP CREATE")
	}

	templateData := shared.TemplateData{
		Username:    username,
		Description: description,
	}

	prompt, err := b.os.PromptManager.GetMCPCreatePrompt(templateData)
	if err != nil {
		return NewBASICError(ErrCategorySystem, "TEMPLATE_ERROR", b.currentLine == 0, b.currentLine).WithCommand("MCP CREATE")
	}

	// DEBUG: Log before DeepSeek call
	logger.Debug(logger.AreaTinyBasic, "[MCP] About to call DeepSeek with prompt length: %d", len(prompt))

	// Send to DeepSeek using existing chat infrastructure
	if b.os == nil {
		return NewBASICError(ErrCategorySystem, "SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("MCP CREATE")
	}

	// Release the lock before the potentially long-running network call.
	b.mu.Unlock()
	responses := b.os.AskDeepSeek(prompt, b.sessionID)
	// Re-acquire the lock after the network call.
	b.mu.Lock()

	if len(responses) == 0 {
		b.sendMessageWrapped(shared.MessageTypeText, "Error: Failed to generate code")
		return nil
	}

	// Extract text response
	response := ""
	for _, msg := range responses {
		if msg.Type == shared.MessageTypeText {
			response += msg.Content
		}
	}

	// Check if the response is an error message from the API
	if strings.Contains(response, "*System Error*") {
		b.sendMessageWrapped(shared.MessageTypeText, response)
		return nil
	}

	// Apply the same sanitization as used in TinyOS chat
	response = b.sanitizeDeepSeekResponse(response)

	// Clean markdown formatting from response
	response = b.cleanMarkdownCode(response)
	// Clean non-printable characters that could cause parsing issues
	response = cleanCodeForLoading(response)

	// DEBUG: Log the extracted response
	logger.Debug(logger.AreaTinyBasic, "[MCP] Extracted response from DeepSeek (length=%d): '%s'", len(response), response)
	if response == "" {
		logger.Debug(logger.AreaTinyBasic, "[MCP] Error: Empty response from DeepSeek")
		b.sendMessageWrapped(shared.MessageTypeText, "Error: No code generated")
		return nil
	}

	logger.Debug(logger.AreaTinyBasic, "[MCP] About to record MCP usage")
	// Record MCP usage ALWAYS (even for complex programs to prevent quota abuse)
	newUsage := b.recordMCPUsage(username)
	logger.Debug(logger.AreaTinyBasic, "[MCP] Recorded usage, new count: %d", newUsage)
	// Check for complexity abort signal AFTER recording usage
	trimmedResponse := strings.TrimSpace(response)
	logger.Debug(logger.AreaTinyBasic, "[MCP] Trimmed response: '%s'", trimmedResponse)
	if trimmedResponse == "PROGRAM_TOO_COMPLEX" {
		logger.Debug(logger.AreaTinyBasic, "[MCP] DeepSeek aborted due to complexity (usage counted)")
		b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("mcp usage today: %d out of %d times", newUsage, MCPUserDailyLimit))
		b.sendMessageWrapped(shared.MessageTypeText, "Requested program too complex, request aborted.")
		return nil
	}

	// Check for declined request signal AFTER recording usage
	if strings.HasPrefix(trimmedResponse, "MCP_DECLINED:") {
		logger.Debug(logger.AreaTinyBasic, "[MCP] DeepSeek declined request (usage counted)")
		declineMessage := strings.TrimSpace(strings.TrimPrefix(trimmedResponse, "MCP_DECLINED:"))
		if declineMessage == "" {
			declineMessage = "Request declined"
		}
		b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("mcp usage today: %d out of %d times", newUsage, MCPUserDailyLimit))
		b.sendMessageWrapped(shared.MessageTypeText, declineMessage)
		return nil
	}

	logger.Debug(logger.AreaTinyBasic, "[MCP] Recorded usage, sending usage info") // Show usage information
	b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("mcp usage today: %d out of %d times", newUsage, MCPUserDailyLimit))

	logger.Debug(logger.AreaTinyBasic, "[MCP] Sending generated code messages") // Ask for filename and show generated code
	logger.Debug(logger.AreaTinyBasic, "[MCP] About to send first message")
	b.sendMessageWrapped(shared.MessageTypeText, "Generated BASIC program:")
	logger.Debug(logger.AreaTinyBasic, "[MCP] First message sent")

	// Small delay to prevent client overload
	time.Sleep(50 * time.Millisecond)

	logger.Debug(logger.AreaTinyBasic, "[MCP] About to send generated code (length: %d)", len(response))
	b.sendMessageWrapped(shared.MessageTypeText, response)
	logger.Debug(logger.AreaTinyBasic, "[MCP] Generated code sent")
	// Small delay to prevent client overload
	time.Sleep(50 * time.Millisecond)

	logger.Debug(logger.AreaTinyBasic, "[MCP] About to send filename prompt")
	b.sendMessageWrapped(shared.MessageTypeText, "Enter filename to save program:")

	// Send empty input for CREATE operation
	inputMsg := shared.Message{
		Type:      shared.MessageTypeInput,
		InputStr:  "",
		CursorPos: 0,
		SessionID: b.sessionID,
	}
	b.sendMessageObject(inputMsg)

	logger.Debug(logger.AreaTinyBasic, "[MCP] Filename prompt sent")

	logger.Debug(logger.AreaTinyBasic, "[MCP] Setting MCP state variables")
	// Store the generated code and set input flag - do this AFTER sending messages
	// Note: mutex is already held by the calling function (executeStatement)
	logger.Debug(logger.AreaTinyBasic, "[MCP] Setting variables without additional mutex lock")
	b.pendingMCPCode = response
	b.waitingForMCPInput = true
	b.pendingMCPFilename = "" // No default filename for CREATE operation
	logger.Debug(logger.AreaTinyBasic, "[MCP] Set variables: pendingMCPCode length=%d, waitingForMCPInput=%v", len(b.pendingMCPCode), b.waitingForMCPInput)

	// DEBUG: Log that messages were sent
	logger.Debug(logger.AreaTinyBasic, "[MCP] All messages sent successfully")

	return nil
}

// cmdMCPEdit handles "MCP EDIT <filename> <instruction>" command
func (b *TinyBASIC) cmdMCPEdit(filename, instruction string) error {
	if instruction == "" {
		return NewBASICError(ErrCategorySyntax, "MISSING_INSTRUCTION", b.currentLine == 0, b.currentLine).WithCommand("MCP EDIT")
	}

	// Check instruction length (max 512 characters)
	if len(instruction) > 512 {
		b.sendMessageWrapped(shared.MessageTypeText, "Your request is too long (max 512 chars)")
		return nil
	}

	// Check if filename ends with .bas
	if !strings.HasSuffix(strings.ToLower(filename), ".bas") {
		return NewBASICError(ErrCategorySyntax, "INVALID_FILENAME", b.currentLine == 0, b.currentLine).WithCommand("MCP EDIT")
	}

	// Get username
	username := b.os.GetUsernameForSession(b.sessionID)

	// Check rate limits
	allowed, message, _ := b.checkMCPRateLimit(username)
	if !allowed {
		b.sendMessageWrapped(shared.MessageTypeText, message)
		return nil
	}

	// Read existing file
	if b.fs == nil {
		return NewBASICError(ErrCategoryFileSystem, "FILE_SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("MCP EDIT")
	}

	content, err := b.fs.ReadFile(filename, b.sessionID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return NewBASICError(ErrCategoryFileSystem, "FILE_NOT_FOUND", b.currentLine == 0, b.currentLine).WithCommand("MCP EDIT")
		}
		return NewBASICError(ErrCategoryFileSystem, "FILE_READ_ERROR", b.currentLine == 0, b.currentLine).WithCommand("MCP EDIT")
	}

	// Check file size (max 3KB)
	if len(content) > 3*1024 {
		b.sendMessageWrapped(shared.MessageTypeText, "Error: File too large for MCP editing (max 3KB)")
		return nil
	} // Show thinking message
	b.sendMessageWrapped(shared.MessageTypeText, "MCP is thinking, please wait...")
	// Get the MCP EDIT prompt from PromptManager with template data
	if b.os == nil || b.os.PromptManager == nil {
		return NewBASICError(ErrCategorySystem, "SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("MCP EDIT")
	}

	templateData := shared.TemplateData{
		Username:    username,
		Instruction: instruction,
		FileContent: content,
	}

	prompt, err := b.os.PromptManager.GetMCPEditPrompt(templateData)
	if err != nil {
		return NewBASICError(ErrCategorySystem, "TEMPLATE_ERROR", b.currentLine == 0, b.currentLine).WithCommand("MCP EDIT")
	}

	// DEBUG: Log before DeepSeek call
	logger.Debug(logger.AreaTinyBasic, "[MCP] About to call DeepSeek with prompt length: %d", len(prompt))

	// Send to DeepSeek using existing chat infrastructure
	if b.os == nil {
		return NewBASICError(ErrCategorySystem, "SYSTEM_ERROR", b.currentLine == 0, b.currentLine).WithCommand("MCP CREATE")
	}

	// Release the lock before the potentially long-running network call.
	b.mu.Unlock()
	responses := b.os.AskDeepSeek(prompt, b.sessionID)
	// Re-acquire the lock after the network call.
	b.mu.Lock()

	if len(responses) == 0 {
		b.sendMessageWrapped(shared.MessageTypeText, "Error: Failed to generate code")
		return nil
	}

	// Extract text response
	response := ""
	for _, msg := range responses {
		if msg.Type == shared.MessageTypeText {
			response += msg.Content
		}
	}

	// Check if the response is an error message from the API
	if strings.Contains(response, "*System Error*") {
		b.sendMessageWrapped(shared.MessageTypeText, response)
		return nil
	}

	// Apply the same sanitization as used in TinyOS chat
	response = b.sanitizeDeepSeekResponse(response)

	// Clean markdown formatting from response
	response = b.cleanMarkdownCode(response)
	// Clean non-printable characters that could cause parsing issues
	response = cleanCodeForLoading(response)

	// DEBUG: Log the extracted response
	logger.Debug(logger.AreaTinyBasic, "[MCP] Extracted response from DeepSeek (length=%d): '%s'", len(response), response)
	if response == "" {
		logger.Debug(logger.AreaTinyBasic, "[MCP] Error: Empty response from DeepSeek")
		b.sendMessageWrapped(shared.MessageTypeText, "Error: No code generated")
		return nil
	}

	logger.Debug(logger.AreaTinyBasic, "[MCP] About to record MCP usage")
	// Record MCP usage ALWAYS (even for complex programs to prevent quota abuse)
	newUsage := b.recordMCPUsage(username)
	logger.Debug(logger.AreaTinyBasic, "[MCP] Recorded usage, new count: %d", newUsage)
	// Check for complexity abort signal AFTER recording usage
	trimmedResponse := strings.TrimSpace(response)
	logger.Debug(logger.AreaTinyBasic, "[MCP] Trimmed response: '%s'", trimmedResponse)
	if trimmedResponse == "PROGRAM_TOO_COMPLEX" {
		logger.Debug(logger.AreaTinyBasic, "[MCP] DeepSeek aborted due to complexity (usage counted)")
		b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("mcp usage today: %d out of %d times", newUsage, MCPUserDailyLimit))
		b.sendMessageWrapped(shared.MessageTypeText, "Requested program too complex, request aborted.")
		return nil
	}

	// Check for declined request signal AFTER recording usage
	if strings.HasPrefix(trimmedResponse, "MCP_DECLINED:") {
		logger.Debug(logger.AreaTinyBasic, "[MCP] DeepSeek declined request (usage counted)")
		declineMessage := strings.TrimSpace(strings.TrimPrefix(trimmedResponse, "MCP_DECLINED:"))
		if declineMessage == "" {
			declineMessage = "Request declined"
		}
		b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("mcp usage today: %d out of %d times", newUsage, MCPUserDailyLimit))
		b.sendMessageWrapped(shared.MessageTypeText, declineMessage)
		return nil
	}

	logger.Debug(logger.AreaTinyBasic, "[MCP] Recorded usage, sending usage info") // Show usage information
	b.sendMessageWrapped(shared.MessageTypeText, fmt.Sprintf("mcp usage today: %d out of %d times", newUsage, MCPUserDailyLimit))

	logger.Debug(logger.AreaTinyBasic, "[MCP] Sending generated code messages") // Ask for filename and show generated code
	logger.Debug(logger.AreaTinyBasic, "[MCP] About to send first message")
	b.sendMessageWrapped(shared.MessageTypeText, "Generated BASIC program:")
	logger.Debug(logger.AreaTinyBasic, "[MCP] First message sent")

	// Small delay to prevent client overload
	time.Sleep(50 * time.Millisecond)

	logger.Debug(logger.AreaTinyBasic, "[MCP] About to send generated code (length: %d)", len(response))
	b.sendMessageWrapped(shared.MessageTypeText, response)
	logger.Debug(logger.AreaTinyBasic, "[MCP] Generated code sent")
	// Small delay to prevent client overload
	time.Sleep(50 * time.Millisecond)

	logger.Debug(logger.AreaTinyBasic, "[MCP] About to send filename prompt")
	b.sendMessageWrapped(shared.MessageTypeText, "Enter filename to save program:")

	// Send empty input for CREATE operation
	inputMsg := shared.Message{
		Type:      shared.MessageTypeInput,
		InputStr:  "",
		CursorPos: 0,
		SessionID: b.sessionID,
	}
	b.sendMessageObject(inputMsg)

	logger.Debug(logger.AreaTinyBasic, "[MCP] Filename prompt sent")

	logger.Debug(logger.AreaTinyBasic, "[MCP] Setting MCP state variables")
	// Store the generated code and set input flag - do this AFTER sending messages
	// Note: mutex is already held by the calling function (executeStatement)
	logger.Debug(logger.AreaTinyBasic, "[MCP] Setting variables without additional mutex lock")
	b.pendingMCPCode = response
	b.waitingForMCPInput = true
	b.pendingMCPFilename = "" // No default filename for CREATE operation
	logger.Debug(logger.AreaTinyBasic, "[MCP] Set variables: pendingMCPCode length=%d, waitingForMCPInput=%v", len(b.pendingMCPCode), b.waitingForMCPInput)

	// DEBUG: Log that messages were sent
	logger.Debug(logger.AreaTinyBasic, "[MCP] All messages sent successfully")

	return nil
}

// sanitizeDeepSeekResponse removes emojis and unwanted characters from MCP responses
func (b *TinyBASIC) sanitizeDeepSeekResponse(response string) string {
	// regexp for matching emojis and filtering them out
	emojiRegex := regexp.MustCompile(`[\x{1F000}-\x{1FFFF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{FE00}-\x{FEFF}\x{1F900}-\x{1F9FF}\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}]`)
	return emojiRegex.ReplaceAllString(response, "")
}

// cleanMarkdownCode removes markdown formatting from DeepSeek responses
func (b *TinyBASIC) cleanMarkdownCode(response string) string {
	lines := strings.Split(response, "\n")
	var cleanedLines []string
	inCodeBlock := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip markdown code block markers
		if trimmedLine == "```" || strings.HasPrefix(trimmedLine, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		// If we're in a code block, include the line
		if inCodeBlock {
			cleanedLines = append(cleanedLines, line)
		} else {
			// Outside code blocks, skip explanatory text but keep lines that look like BASIC
			// BASIC lines typically start with numbers or REM
			if strings.HasPrefix(trimmedLine, "REM ") ||
				(len(trimmedLine) > 0 && trimmedLine[0] >= '0' && trimmedLine[0] <= '9') {
				cleanedLines = append(cleanedLines, line)
			}
		}
	}

	return strings.Join(cleanedLines, "\n")
}
