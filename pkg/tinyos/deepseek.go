package tinyos

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/configuration"

	"github.com/antibyte/retroterm/pkg/shared"
)

// getDeepSeekAPIKey retrieves the API key from configuration or environment
func getDeepSeekAPIKey() string {
	// Check if we should use environment variable
	useEnv := configuration.GetBool("DeepSeek", "use_environment_variable", true)

	if useEnv {
		// Try environment variable first
		if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" {
			return apiKey
		}
	}

	// Fall back to configuration file
	return configuration.GetString("DeepSeek", "api_key", "")
}

// AskDeepSeek sends a request to the DeepSeek AI with chat history and returns the response as []shared.Message
func (os *TinyOS) AskDeepSeek(query string, sessionID string) []shared.Message {
	log.Printf("[DeepSeek] Request: %q", query)

	// Extract username from session
	username := os.GetUsernameForSession(sessionID)
	if username == "" {
		log.Printf("[DeepSeek] Error: not logged in")
		return []shared.Message{{
			Content: "Error: You must be logged in to use the chat.",
			Type:    shared.MessageTypeText,
		}}
	}

	// Get current session to access chat history
	os.sessionMutex.RLock()
	session, exists := os.sessions[sessionID]
	os.sessionMutex.RUnlock()
	if !exists {
		log.Printf("[DeepSeek] Error: session not found")
		return []shared.Message{{
			Content: "Error: Session not found.",
			Type:    shared.MessageTypeText}}
	}

	// Get API key from configuration
	apiKey := getDeepSeekAPIKey()
	if apiKey == "" {
		return []shared.Message{{
			Content: "Error: DeepSeek API key not configured. Please set DEEPSEEK_API_KEY environment variable or configure in settings.cfg",
			Type:    shared.MessageTypeText,
		}}
	}

	// Use the Chat prompt from PromptManager with template data
	templateData := shared.TemplateData{
		Username: username,
	}
	prompt, err := os.PromptManager.GetMCPChatPrompt(templateData)
	if err != nil {
		log.Printf("[DeepSeek] Error generating chat prompt: %v", err)
		return []shared.Message{{
			Content: "Error: Failed to generate system prompt.",
			Type:    shared.MessageTypeText,
		}}
	}

	// Define request/response types
	type deepSeekRequest struct {
		Model    string        `json:"model"`
		Messages []interface{} `json:"messages"`
	}
	type deepSeekMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type deepSeekResponse struct {
		Choices []struct {
			Message deepSeekMessage `json:"message"`
		} `json:"choices"`
	}

	// Build messages array with system prompt and chat history
	messages := []interface{}{
		deepSeekMessage{Role: "system", Content: prompt},
	}

	// Add chat history from session
	os.sessionMutex.RLock()
	log.Printf("[DeepSeek] Current chat history has %d messages", len(session.ChatHistory))
	for _, chatMsg := range session.ChatHistory {
		messages = append(messages, deepSeekMessage{
			Role:    chatMsg.Role,
			Content: chatMsg.Content,
		})
	}
	os.sessionMutex.RUnlock()

	// Add current user message
	userMessage := deepSeekMessage{Role: "user", Content: query}
	messages = append(messages, userMessage)

	// Log number of messages being sent
	log.Printf("[DeepSeek] Sending %d messages to API (including system prompt)", len(messages))

	// Add user message to chat history
	os.sessionMutex.Lock()
	session.ChatHistory = append(session.ChatHistory, ChatMessage{
		Role:    "user",
		Content: query,
		Time:    time.Now(),
	})
	os.sessionMutex.Unlock()

	// Build request
	reqBody := deepSeekRequest{
		Model:    "deepseek-chat",
		Messages: messages,
	}
	body, _ := json.Marshal(reqBody)
	client := &http.Client{Timeout: 30 * time.Second}
	httpReq, _ := http.NewRequest("POST", "https://api.deepseek.com/v1/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	log.Printf("[DeepSeek] Request-Body: %s", string(body))
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Printf("[DeepSeek] HTTP error: %v", err)
		return []shared.Message{{
			Content: "Error communicating with DeepSeek: " + err.Error(),
			Type:    shared.MessageTypeText,
		}}
	}
	defer resp.Body.Close()
	log.Printf("[DeepSeek] HTTP-Status: %d %s", resp.StatusCode, resp.Status)
	bodyBytes, _ := io.ReadAll(resp.Body)
	log.Printf("[DeepSeek] HTTP-Body: %s", string(bodyBytes))
	resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Borrow body for decoding

	var dsResp deepSeekResponse
	if err := json.NewDecoder(resp.Body).Decode(&dsResp); err != nil {
		log.Printf("[DeepSeek] Response decode error: %v", err)
		return []shared.Message{{
			Content: "*System Error* MCP communications link disrupted. The DeepSeek neural network cannot be reached. Please try again later, Program.",
			Type:    shared.MessageTypeText,
		}}
	}
	if len(dsResp.Choices) == 0 {
		log.Printf("[DeepSeek] Empty response received")
		return []shared.Message{{
			Content: "*System Error* MCP received empty response from neural network. The artificial intelligence core may be temporarily offline.",
			Type:    shared.MessageTypeText}}
	}
	response := dsResp.Choices[0].Message.Content
	log.Printf("[DeepSeek] Raw response: %q", response)
	// Add assistant message to chat history
	os.sessionMutex.Lock()
	session.ChatHistory = append(session.ChatHistory, ChatMessage{
		Role:    "assistant",
		Content: response,
		Time:    time.Now(),
	})
	os.sessionMutex.Unlock()

	// Limit chat history to last 20 messages (10 user + 10 assistant)
	os.limitChatHistory(sessionID, 20)

	response = sanitizeDeepSeekResponse(response) // Extract *beep* and *talk:* (case-insensitive)
	resultMessages := []shared.Message{}
	// Check for *BEEP* (case-insensitive)
	lowerResponse := strings.ToLower(response)
	if strings.Contains(lowerResponse, "*beep*") {
		log.Printf("[DeepSeek] *beep* detected")
		// Remove both *beep* and *BEEP*
		beepRegex := regexp.MustCompile(`(?i)\*beep\*`)
		response = beepRegex.ReplaceAllString(response, "")
		resultMessages = append(resultMessages, shared.Message{Type: shared.MessageTypeBeep})
	}
	// Check for *TALK:text* (case-insensitive)
	talkRegex := regexp.MustCompile(`(?i)\*talk:(.*?)\*`)
	matches := talkRegex.FindAllStringSubmatch(response, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			log.Printf("[DeepSeek] *talk* detected: %q", match[1])
			cleanTalk := strings.TrimSpace(match[1])
			resultMessages = append(resultMessages, shared.Message{Type: shared.MessageTypeSpeak, Content: cleanTalk})
			response = strings.Replace(response, match[0], "", 1)
		}
	}
	// Check for *EVIL* (case-insensitive) - MCP's dramatic noise effect
	if strings.Contains(lowerResponse, "*evil*") {
		log.Printf("[DeepSeek] *evil* detected - MCP is being particularly malicious")
		// Remove *evil* from response
		evilRegex := regexp.MustCompile(`(?i)\*evil\*`)
		response = evilRegex.ReplaceAllString(response, "")
		resultMessages = append(resultMessages, shared.Message{Type: shared.MessageTypeEvil}) // Also send EVIL message directly to the terminal frontend (not just chat)
		if os.SendToClientCallback != nil {
			evilMessage := shared.Message{Type: shared.MessageTypeEvil}
			log.Printf("[DeepSeek] About to send EVIL message to terminal frontend for session: %s", sessionID)

			// CRITICAL FIX: Make callback asynchronous to prevent deadlock
			go func() {
				if err := os.SendToClientCallback(sessionID, evilMessage); err != nil {
					log.Printf("[DeepSeek] Failed to send EVIL message to terminal frontend: %v", err)
				} else {
					log.Printf("[DeepSeek] EVIL message sent to terminal frontend successfully")
				}
			}()
		} else {
			log.Printf("[DeepSeek] WARNING: SendToClientCallback is nil, cannot send EVIL message to terminal frontend")
		}
	}
	response = strings.TrimSpace(response)
	if response != "" {
		log.Printf("[DeepSeek] Final response to user: %q", response)
		resultMessages = append(resultMessages, shared.Message{Content: response, Type: shared.MessageTypeText})
	}

	// Log the number of generated messages
	log.Printf("[DeepSeek] Generated %d messages for user", len(resultMessages))

	return resultMessages
}

// limitChatHistory limits the chat history to the last N messages
func (os *TinyOS) limitChatHistory(sessionID string, maxMessages int) {
	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()

	session, exists := os.sessions[sessionID]
	if !exists {
		return
	}

	// Keep only the last maxMessages messages
	if len(session.ChatHistory) > maxMessages {
		// Keep the most recent messages
		session.ChatHistory = session.ChatHistory[len(session.ChatHistory)-maxMessages:]
	}
}

// clearChatHistory clears the entire chat history for a session
func (os *TinyOS) clearChatHistory(sessionID string) {
	os.sessionMutex.Lock()
	defer os.sessionMutex.Unlock()

	session, exists := os.sessions[sessionID]
	if !exists {
		return
	}

	session.ChatHistory = []ChatMessage{}
	log.Printf("[DeepSeek] Chat history cleared for session %s", sessionID)
}

// sanitizeDeepSeekResponse removes emojis and unwanted characters
func sanitizeDeepSeekResponse(response string) string {
	// regexp for matching emojis and filtering them out
	emojiRegex := regexp.MustCompile(`[\x{1F000}-\x{1FFFF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{FE00}-\x{FEFF}\x{1F900}-\x{1F9FF}\x{1F600}-\x{1F64F}\x{1F300}-\x{1F5FF}\x{1F680}-\x{1F6FF}]`)
	return emojiRegex.ReplaceAllString(response, "")
}
