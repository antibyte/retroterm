package board

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// BoardUI manages the user interface for the board system
type BoardUI struct {
	manager   *BoardManager
	sessionID string
	username  string
	isGuest   bool
	state     BoardUIState
	
	// Navigation state
	currentCategory   *BoardCategory
	currentMessages   []BoardMessage
	currentMessage    *BoardMessage
	currentPage       int
	messagesPerPage   int
	terminalWidth     int
	terminalHeight    int
	
	// Message composition state
	newMessageSubject string
	newMessageContent string
	compositionStep   int // 0=subject, 1=content, 2=confirm
	
	// Pagination state
	currentDisplayLines []string
	paginationIndex     int
	linesPerPage        int
	isShowingPagination bool
	previousState       BoardUIState
}

// BoardUIState represents the current state of the board UI
type BoardUIState int

const (
	BoardStateCategories BoardUIState = iota
	BoardStateMessages
	BoardStateViewMessage
	BoardStateNewMessage
	BoardStateNewMessageContent
	BoardStateNewMessageConfirm
	BoardStatePagination
)

// NewBoardUI creates a new board UI instance
func NewBoardUI(manager *BoardManager, sessionID, username string, isGuest bool, terminalWidth, terminalHeight int) *BoardUI {
	return &BoardUI{
		manager:         manager,
		sessionID:       sessionID,
		username:        username,
		isGuest:         isGuest,
		state:           BoardStateCategories,
		messagesPerPage: 10,
		terminalWidth:   terminalWidth,
		terminalHeight:  terminalHeight,
		linesPerPage:    20, // Leave some space for prompts
	}
}

// Start initializes the board UI and returns the initial display
func (ui *BoardUI) Start() ([]shared.Message, error) {
	return ui.showCategories()
}

// ProcessInput handles user input and returns the appropriate response
func (ui *BoardUI) ProcessInput(input string) ([]shared.Message, error) {
	// For ViewMessage state, handle raw input (don't trim)
	if ui.state == BoardStateViewMessage {
		return ui.handleViewMessageInput(input)
	}
	
	// For Pagination state, handle raw input (don't trim)
	if ui.state == BoardStatePagination {
		return ui.handlePaginationInput(input)
	}
	
	// For all other states, trim input
	input = strings.TrimSpace(input)
	
	switch ui.state {
	case BoardStateCategories:
		return ui.handleCategoriesInput(input)
	case BoardStateMessages:
		return ui.handleMessagesInput(input)
	case BoardStateNewMessage:
		return ui.handleNewMessageSubjectInput(input)
	case BoardStateNewMessageContent:
		return ui.handleNewMessageContentInput(input)
	case BoardStateNewMessageConfirm:
		return ui.handleNewMessageConfirmInput(input)
	default:
		return ui.showCategories()
	}
}

// showCategories displays the category list
func (ui *BoardUI) showCategories() ([]shared.Message, error) {
	categories, err := ui.manager.GetCategories()
	if err != nil {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: fmt.Sprintf("Error loading categories: %v", err),
		}}, err
	}
	
	ui.state = BoardStateCategories
	
	lines := ui.manager.FormatCategoryList(categories, ui.terminalWidth)
	
	// Use pagination if content is too long
	return ui.showWithPagination(lines, BoardStateCategories)
}

// handleCategoriesInput handles input when viewing categories
func (ui *BoardUI) handleCategoriesInput(input string) ([]shared.Message, error) {
	if input == "q" || input == "quit" {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Goodbye!",
		}}, nil
	}
	
	// Try to parse as category number
	categoryNum, err := strconv.Atoi(input)
	if err != nil {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Invalid input. Please enter a category number or 'q' to quit.",
		}}, nil
	}
	
	// Get categories to validate the number
	categories, err := ui.manager.GetCategories()
	if err != nil {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: fmt.Sprintf("Error loading categories: %v", err),
		}}, err
	}
	
	if categoryNum < 1 || categoryNum > len(categories) {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Invalid category number. Please try again.",
		}}, nil
	}
	
	// Select the category
	selectedCategory := categories[categoryNum-1]
	ui.currentCategory = &selectedCategory
	ui.currentPage = 1
	
	return ui.showMessages()
}

// showMessages displays the message list for the current category
func (ui *BoardUI) showMessages() ([]shared.Message, error) {
	if ui.currentCategory == nil {
		return ui.showCategories()
	}
	
	// Calculate offset for pagination
	offset := (ui.currentPage - 1) * ui.messagesPerPage
	
	messages, err := ui.manager.GetMessages(ui.currentCategory.ID, ui.messagesPerPage, offset)
	if err != nil {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: fmt.Sprintf("Error loading messages: %v", err),
		}}, err
	}
	
	ui.currentMessages = messages
	ui.state = BoardStateMessages
	
	// Calculate total pages
	totalMessages, err := ui.manager.GetMessageCount(ui.currentCategory.ID)
	if err != nil {
		totalMessages = 0
	}
	totalPages := (totalMessages + ui.messagesPerPage - 1) / ui.messagesPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	
	lines := ui.manager.FormatMessageList(messages, ui.currentCategory.Title, 
		ui.currentPage, totalPages, ui.terminalWidth)
	
	// Use pagination if content is too long
	return ui.showWithPagination(lines, BoardStateMessages)
}

// handleMessagesInput handles input when viewing messages list
func (ui *BoardUI) handleMessagesInput(input string) ([]shared.Message, error) {
	switch input {
	case "q", "quit":
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Goodbye!",
		}}, nil
		
	case "b", "back":
		ui.currentCategory = nil
		ui.currentMessages = nil
		return ui.showCategories()
		
	case "n", "new":
		if ui.isGuest {
			return []shared.Message{{
				Type:    shared.MessageTypeText,
				Content: "Sorry, guests cannot post messages. Please log in to post.",
			}}, nil
		}
		return ui.startNewMessage()
		
	case "prev", "previous":
		if ui.currentPage > 1 {
			ui.currentPage--
			return ui.showMessages()
		}
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Already on the first page.",
		}}, nil
		
	case "next":
		// Check if there are more pages
		totalMessages, err := ui.manager.GetMessageCount(ui.currentCategory.ID)
		if err == nil {
			totalPages := (totalMessages + ui.messagesPerPage - 1) / ui.messagesPerPage
			if ui.currentPage < totalPages {
				ui.currentPage++
				return ui.showMessages()
			}
		}
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Already on the last page.",
		}}, nil
	}
	
	// Try to parse as message number
	messageNum, err := strconv.Atoi(input)
	if err != nil {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Invalid input. Enter message number, 'n' for new, 'b' for back, 'next'/'prev' for navigation, or 'q' to quit.",
		}}, nil
	}
	
	if messageNum < 1 || messageNum > len(ui.currentMessages) {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Invalid message number. Please try again.",
		}}, nil
	}
	
	// Select the message
	selectedMessage := ui.currentMessages[messageNum-1]
	ui.currentMessage = &selectedMessage
	
	return ui.showMessage()
}

// showMessage displays a single message
func (ui *BoardUI) showMessage() ([]shared.Message, error) {
	if ui.currentMessage == nil {
		return ui.showMessages()
	}
	
	ui.state = BoardStateViewMessage
	
	lines := ui.manager.FormatMessage(*ui.currentMessage, ui.terminalWidth)
	
	// Use pagination if content is too long
	return ui.showWithPagination(lines, BoardStateViewMessage)
}

// handleViewMessageInput handles input when viewing a single message
func (ui *BoardUI) handleViewMessageInput(input string) ([]shared.Message, error) {
	// ANY input returns to message list
	// This includes empty input, BOARD_CONTINUE signal, and any other input
	ui.currentMessage = nil
	return ui.showMessages()
}

// startNewMessage starts the new message composition process
func (ui *BoardUI) startNewMessage() ([]shared.Message, error) {
	ui.state = BoardStateNewMessage
	ui.newMessageSubject = ""
	ui.newMessageContent = ""
	ui.compositionStep = 0
	
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: createFrameBorder("top")},
		{Type: shared.MessageTypeText, Content: formatFrameLine(centerPad("New Message", CONTENT_WIDTH))},
		{Type: shared.MessageTypeText, Content: createFrameBorder("middle")},
		{Type: shared.MessageTypeText, Content: formatFrameLine(fmt.Sprintf("Category: %s", ui.currentCategory.Title))},
		{Type: shared.MessageTypeText, Content: formatFrameLine(fmt.Sprintf("Author: %s", ui.username))},
		{Type: shared.MessageTypeText, Content: formatFrameLine("")},
		{Type: shared.MessageTypeText, Content: formatFrameLine("Enter message subject:")},
		{Type: shared.MessageTypeText, Content: createFrameBorder("bottom")},
		{Type: shared.MessageTypeText, Content: ""},
	}, nil
}

// handleNewMessageSubjectInput handles subject input for new messages
func (ui *BoardUI) handleNewMessageSubjectInput(input string) ([]shared.Message, error) {
	if input == "q" || input == "quit" {
		return ui.showMessages()
	}
	
	subject := strings.TrimSpace(input)
	if subject == "" {
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Subject cannot be empty. Please enter a subject or 'q' to cancel.",
		}}, nil
	}
	
	ui.newMessageSubject = subject
	ui.state = BoardStateNewMessageContent
	
	return []shared.Message{
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: createFrameBorder("top")},
		{Type: shared.MessageTypeText, Content: formatFrameLine(centerPad("New Message", CONTENT_WIDTH))},
		{Type: shared.MessageTypeText, Content: createFrameBorder("middle")},
		{Type: shared.MessageTypeText, Content: formatFrameLine(fmt.Sprintf("Category: %s", ui.currentCategory.Title))},
		{Type: shared.MessageTypeText, Content: formatFrameLine(fmt.Sprintf("Author: %s", ui.username))},
		{Type: shared.MessageTypeText, Content: formatFrameLine(fmt.Sprintf("Subject: %s", ui.newMessageSubject))},
		{Type: shared.MessageTypeText, Content: formatFrameLine("")},
		{Type: shared.MessageTypeText, Content: formatFrameLine("Enter message content (type 'END' on a new line to finish):")},
		{Type: shared.MessageTypeText, Content: createFrameBorder("bottom")},
		{Type: shared.MessageTypeText, Content: ""},
	}, nil
}

// handleNewMessageContentInput handles content input for new messages
func (ui *BoardUI) handleNewMessageContentInput(input string) ([]shared.Message, error) {
	if input == "q" || input == "quit" {
		return ui.showMessages()
	}
	
	if strings.ToUpper(strings.TrimSpace(input)) == "END" {
		if strings.TrimSpace(ui.newMessageContent) == "" {
			return []shared.Message{{
				Type:    shared.MessageTypeText,
				Content: "Message content cannot be empty. Continue typing or 'q' to cancel.",
			}}, nil
		}
		
		ui.state = BoardStateNewMessageConfirm
		return ui.showMessageConfirmation()
	}
	
	// Append the line to the content
	if ui.newMessageContent == "" {
		ui.newMessageContent = input
	} else {
		ui.newMessageContent += "\n" + input
	}
	
	return []shared.Message{{
		Type:    shared.MessageTypeText,
		Content: "> " + input,
	}}, nil
}

// showMessageConfirmation shows the message confirmation screen
func (ui *BoardUI) showMessageConfirmation() ([]shared.Message, error) {
	contentLines := wrapText(ui.newMessageContent, 72)
	
	lines := []string{}
	lines = append(lines, "")
	lines = append(lines, createFrameBorder("top"))
	lines = append(lines, formatFrameLine(centerPad("Message Preview", CONTENT_WIDTH)))
	lines = append(lines, createFrameBorder("middle"))
	lines = append(lines, formatFrameLine(fmt.Sprintf("From: %s Date: %s", 
		ui.username, time.Now().Format("2006-01-02 15:04:05"))))
	lines = append(lines, formatFrameLine(fmt.Sprintf("Subject: %s", ui.newMessageSubject)))
	lines = append(lines, createFrameBorder("middle"))
	
	for _, line := range contentLines {
		lines = append(lines, formatFrameLine(line))
	}
	
	lines = append(lines, createFrameBorder("bottom"))
	lines = append(lines, "")
	lines = append(lines, "Send this message? (y/n)")
	lines = append(lines, "")
	
	// Use pagination if content is too long
	return ui.showWithPagination(lines, BoardStateNewMessageConfirm)
}

// handleNewMessageConfirmInput handles confirmation input for new messages
func (ui *BoardUI) handleNewMessageConfirmInput(input string) ([]shared.Message, error) {
	input = strings.ToLower(strings.TrimSpace(input))
	
	switch input {
	case "y", "yes":
		return ui.sendMessage()
	case "n", "no", "q", "quit":
		return ui.showMessages()
	default:
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: "Please enter 'y' to send or 'n' to cancel.",
		}}, nil
	}
}

// sendMessage sends the composed message
func (ui *BoardUI) sendMessage() ([]shared.Message, error) {
	err := ui.manager.AddMessage(ui.currentCategory.ID, ui.username, 
		ui.newMessageSubject, ui.newMessageContent, "")
	
	if err != nil {
		logger.Error(logger.AreaGeneral, "Error sending message: %v", err)
		return []shared.Message{{
			Type:    shared.MessageTypeText,
			Content: fmt.Sprintf("Error sending message: %v", err),
		}}, err
	}
	
	// Reset composition state
	ui.newMessageSubject = ""
	ui.newMessageContent = ""
	ui.compositionStep = 0
	
	// Show success message and return to messages list
	messages := []shared.Message{
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: createFrameBorder("top")},
		{Type: shared.MessageTypeText, Content: formatFrameLine(centerPad("SUCCESS", CONTENT_WIDTH))},
		{Type: shared.MessageTypeText, Content: createFrameBorder("middle")},
		{Type: shared.MessageTypeText, Content: formatFrameLine("Your message has been posted successfully!")},
		{Type: shared.MessageTypeText, Content: createFrameBorder("bottom")},
		{Type: shared.MessageTypeText, Content: ""},
		{Type: shared.MessageTypeText, Content: "Press Enter to continue..."},
	}
	
	// Return to messages list after a brief delay
	ui.state = BoardStateMessages
	return messages, nil
}

// GetState returns the current UI state
func (ui *BoardUI) GetState() BoardUIState {
	return ui.state
}

// IsViewingMessage returns true if currently viewing a message
func (ui *BoardUI) IsViewingMessage() bool {
	return ui.state == BoardStateViewMessage
}

// IsActive returns true if the board UI is still active
func (ui *BoardUI) IsActive() bool {
	return ui.state != BoardStateCategories // Could be enhanced with quit state
}

// showWithPagination shows content with pagination if it exceeds screen height
func (ui *BoardUI) showWithPagination(lines []string, returnState BoardUIState) ([]shared.Message, error) {
	if len(lines) <= ui.linesPerPage {
		// Content fits on screen, show directly
		messages := []shared.Message{}
		for _, line := range lines {
			messages = append(messages, shared.Message{
				Type:    shared.MessageTypeText,
				Content: line,
			})
		}
		return messages, nil
	}
	
	// Content needs pagination
	ui.currentDisplayLines = lines
	ui.paginationIndex = 0
	ui.isShowingPagination = true
	ui.state = BoardStatePagination
	
	// Store the state to return to after pagination
	ui.previousState = returnState
	
	return ui.showCurrentPage()
}

// showCurrentPage displays the current page of paginated content
func (ui *BoardUI) showCurrentPage() ([]shared.Message, error) {
	if !ui.isShowingPagination || len(ui.currentDisplayLines) == 0 {
		return []shared.Message{}, nil
	}
	
	start := ui.paginationIndex
	end := start + ui.linesPerPage
	
	if end > len(ui.currentDisplayLines) {
		end = len(ui.currentDisplayLines)
	}
	
	messages := []shared.Message{}
	
	// Show current page content
	for i := start; i < end; i++ {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: ui.currentDisplayLines[i],
		})
	}
	
	// Add pagination prompt if there's more content
	if end < len(ui.currentDisplayLines) {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "",
		})
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "Press Enter to continue or q to stop",
		})
	} else {
		// End of content reached
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "",
		})
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "End of content. Press Enter to continue",
		})
	}
	
	return messages, nil
}

// handlePaginationInput handles input during pagination
func (ui *BoardUI) handlePaginationInput(input string) ([]shared.Message, error) {
	input = strings.TrimSpace(input)
	
	if input == "q" || input == "quit" {
		// Exit pagination and return to previous state
		ui.isShowingPagination = false
		ui.state = ui.previousState
		return ui.returnToPreviousState()
	}
	
	// Any other input (including Enter) continues pagination
	if input == "" || input == "BOARD_CONTINUE" {
		// Continue to next page
		ui.paginationIndex += ui.linesPerPage
		
		if ui.paginationIndex >= len(ui.currentDisplayLines) {
			// End of content reached, return to previous state
			ui.isShowingPagination = false
			ui.state = ui.previousState
			return ui.returnToPreviousState()
		}
		
		// Show next page
		return ui.showCurrentPage()
	}
	
	// Any other input continues pagination
	ui.paginationIndex += ui.linesPerPage
	if ui.paginationIndex >= len(ui.currentDisplayLines) {
		// End of content reached, return to previous state
		ui.isShowingPagination = false
		ui.state = ui.previousState
		return ui.returnToPreviousState()
	}
	
	// Show next page
	return ui.showCurrentPage()
}

// returnToPreviousState returns to the appropriate state after pagination
func (ui *BoardUI) returnToPreviousState() ([]shared.Message, error) {
	switch ui.previousState {
	case BoardStateCategories:
		return ui.showCategories()
	case BoardStateMessages:
		return ui.showMessages()
	case BoardStateViewMessage:
		// Return to message list instead of viewing the message again
		ui.currentMessage = nil
		return ui.showMessages()
	default:
		return ui.showCategories()
	}
}

