package board

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/antibyte/retroterm/pkg/logger"
)

// BoardCategory represents a message board category
type BoardCategory struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	CreatedBy   string    `json:"created_by"`
	MessageCount int      `json:"message_count"`
}

// BoardMessage represents a message in the board
type BoardMessage struct {
	ID         int       `json:"id"`
	CategoryID int       `json:"category_id"`
	Author     string    `json:"author"`
	Subject    string    `json:"subject"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	IPAddress  string    `json:"ip_address"`
}

// BoardManager manages the message board system
type BoardManager struct {
	db *sql.DB
}

// NewBoardManager creates a new board manager
func NewBoardManager(db *sql.DB) *BoardManager {
	return &BoardManager{db: db}
}

// InitializeDatabase creates the necessary tables for the board system
func (bm *BoardManager) InitializeDatabase() error {
	// Create categories table
	createCategoriesTable := `
		CREATE TABLE IF NOT EXISTS board_categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			description TEXT,
			created_at INTEGER NOT NULL,
			created_by TEXT NOT NULL
		);
	`
	
	// Create messages table
	createMessagesTable := `
		CREATE TABLE IF NOT EXISTS board_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category_id INTEGER NOT NULL,
			author TEXT NOT NULL,
			subject TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			ip_address TEXT,
			FOREIGN KEY (category_id) REFERENCES board_categories(id) ON DELETE CASCADE
		);
	`
	
	// Create indices for better performance
	createIndices := `
		CREATE INDEX IF NOT EXISTS idx_messages_category ON board_messages(category_id);
		CREATE INDEX IF NOT EXISTS idx_messages_created_at ON board_messages(created_at);
		CREATE INDEX IF NOT EXISTS idx_messages_author ON board_messages(author);
	`
	
	// Execute all SQL statements
	if _, err := bm.db.Exec(createCategoriesTable); err != nil {
		return fmt.Errorf("failed to create board_categories table: %v", err)
	}
	
	if _, err := bm.db.Exec(createMessagesTable); err != nil {
		return fmt.Errorf("failed to create board_messages table: %v", err)
	}
	
	if _, err := bm.db.Exec(createIndices); err != nil {
		return fmt.Errorf("failed to create indices: %v", err)
	}
	
	// Create default "news" category if it doesn't exist
	if err := bm.createDefaultCategory(); err != nil {
		return fmt.Errorf("failed to create default category: %v", err)
	}
	
	logger.Info(logger.AreaGeneral, "Board system database initialized successfully")
	return nil
}

// createDefaultCategory creates the default "news" category
func (bm *BoardManager) createDefaultCategory() error {
	// Check if news category already exists
	var count int
	err := bm.db.QueryRow("SELECT COUNT(*) FROM board_categories WHERE name = 'news'").Scan(&count)
	if err != nil {
		return err
	}
	
	if count > 0 {
		return nil // Category already exists
	}
	
	// Create the news category
	_, err = bm.db.Exec(`
		INSERT INTO board_categories (name, title, description, created_at, created_by)
		VALUES (?, ?, ?, ?, ?)
	`, "news", "News & Announcements", "General news and system announcements", time.Now().Unix(), "system")
	
	if err != nil {
		return err
	}
	
	logger.Info(logger.AreaGeneral, "Default 'news' category created")
	return nil
}

// GetCategories returns all available categories
func (bm *BoardManager) GetCategories() ([]BoardCategory, error) {
	query := `
		SELECT bc.id, bc.name, bc.title, bc.description, bc.created_at, bc.created_by,
		       COUNT(bm.id) as message_count
		FROM board_categories bc
		LEFT JOIN board_messages bm ON bc.id = bm.category_id
		GROUP BY bc.id, bc.name, bc.title, bc.description, bc.created_at, bc.created_by
		ORDER BY bc.name
	`
	
	rows, err := bm.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var categories []BoardCategory
	for rows.Next() {
		var cat BoardCategory
		var createdAt int64
		
		err := rows.Scan(&cat.ID, &cat.Name, &cat.Title, &cat.Description, 
			&createdAt, &cat.CreatedBy, &cat.MessageCount)
		if err != nil {
			return nil, err
		}
		
		cat.CreatedAt = time.Unix(createdAt, 0)
		categories = append(categories, cat)
	}
	
	return categories, nil
}

// GetCategoryByName returns a category by its name
func (bm *BoardManager) GetCategoryByName(name string) (*BoardCategory, error) {
	query := `
		SELECT bc.id, bc.name, bc.title, bc.description, bc.created_at, bc.created_by,
		       COUNT(bm.id) as message_count
		FROM board_categories bc
		LEFT JOIN board_messages bm ON bc.id = bm.category_id
		WHERE bc.name = ?
		GROUP BY bc.id, bc.name, bc.title, bc.description, bc.created_at, bc.created_by
	`
	
	row := bm.db.QueryRow(query, name)
	
	var cat BoardCategory
	var createdAt int64
	
	err := row.Scan(&cat.ID, &cat.Name, &cat.Title, &cat.Description, 
		&createdAt, &cat.CreatedBy, &cat.MessageCount)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("category not found: %s", name)
		}
		return nil, err
	}
	
	cat.CreatedAt = time.Unix(createdAt, 0)
	return &cat, nil
}

// GetMessages returns messages for a specific category
func (bm *BoardManager) GetMessages(categoryID int, limit int, offset int) ([]BoardMessage, error) {
	query := `
		SELECT id, category_id, author, subject, content, created_at, ip_address
		FROM board_messages
		WHERE category_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := bm.db.Query(query, categoryID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var messages []BoardMessage
	for rows.Next() {
		var msg BoardMessage
		var createdAt int64
		var ipAddress sql.NullString
		
		err := rows.Scan(&msg.ID, &msg.CategoryID, &msg.Author, &msg.Subject, 
			&msg.Content, &createdAt, &ipAddress)
		if err != nil {
			return nil, err
		}
		
		msg.CreatedAt = time.Unix(createdAt, 0)
		if ipAddress.Valid {
			msg.IPAddress = ipAddress.String
		}
		
		messages = append(messages, msg)
	}
	
	return messages, nil
}

// AddMessage adds a new message to a category
func (bm *BoardManager) AddMessage(categoryID int, author, subject, content, ipAddress string) error {
	// Validate input
	if strings.TrimSpace(author) == "" {
		return fmt.Errorf("author cannot be empty")
	}
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("subject cannot be empty")
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("content cannot be empty")
	}
	
	// Check if category exists
	var count int
	err := bm.db.QueryRow("SELECT COUNT(*) FROM board_categories WHERE id = ?", categoryID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("category not found")
	}
	
	// Insert the message
	_, err = bm.db.Exec(`
		INSERT INTO board_messages (category_id, author, subject, content, created_at, ip_address)
		VALUES (?, ?, ?, ?, ?, ?)
	`, categoryID, author, subject, content, time.Now().Unix(), ipAddress)
	
	if err != nil {
		return err
	}
	
	logger.Info(logger.AreaGeneral, "Message added to board by %s: %s", author, subject)
	return nil
}

// GetMessageCount returns the total number of messages in a category
func (bm *BoardManager) GetMessageCount(categoryID int) (int, error) {
	var count int
	err := bm.db.QueryRow("SELECT COUNT(*) FROM board_messages WHERE category_id = ?", categoryID).Scan(&count)
	return count, err
}

// Constants for consistent frame formatting
const (
	FRAME_WIDTH = 76  // Total frame width including borders
	CONTENT_WIDTH = 72  // Content width (FRAME_WIDTH - 4 for borders and padding)
)

// formatFrameLine creates a properly formatted frame line with content
func formatFrameLine(content string) string {
	if len(content) > CONTENT_WIDTH {
		content = content[:CONTENT_WIDTH-3] + "..."
	}
	return fmt.Sprintf("║ %-*s ║", CONTENT_WIDTH, content)
}

// createFrameBorder creates frame border lines
func createFrameBorder(borderType string) string {
	switch borderType {
	case "top":
		return "╔" + strings.Repeat("═", FRAME_WIDTH-2) + "╗"
	case "middle":
		return "╠" + strings.Repeat("═", FRAME_WIDTH-2) + "╣"
	case "bottom":
		return "╚" + strings.Repeat("═", FRAME_WIDTH-2) + "╝"
	default:
		return ""
	}
}

// FormatCategoryList formats the category list for display
func (bm *BoardManager) FormatCategoryList(categories []BoardCategory, terminalWidth int) []string {
	if len(categories) == 0 {
		return []string{"No categories available."}
	}
	
	lines := []string{}
	lines = append(lines, "")
	lines = append(lines, createFrameBorder("top"))
	lines = append(lines, formatFrameLine(centerPad("RETROTERM BBS", CONTENT_WIDTH)))
	lines = append(lines, formatFrameLine(centerPad("Message Board System", CONTENT_WIDTH)))
	lines = append(lines, createFrameBorder("middle"))
	lines = append(lines, formatFrameLine("Categories:"))
	lines = append(lines, createFrameBorder("middle"))
	
	for i, cat := range categories {
		msgCountStr := fmt.Sprintf("(%d messages)", cat.MessageCount)
		
		// Calculate dynamic widths for category display
		// Available space: CONTENT_WIDTH - number prefix - spaces - message count
		numPrefix := fmt.Sprintf("%d. ", i+1)
		availableSpace := CONTENT_WIDTH - len(numPrefix) - len(msgCountStr) - 1 // -1 for space before message count
		
		// Split available space between title and description
		titleWidth := 20
		descWidth := availableSpace - titleWidth - 1 // -1 for space between title and description
		
		// Ensure minimum widths
		if descWidth < 10 {
			descWidth = 10
			titleWidth = availableSpace - descWidth - 1
		}
		
		// Truncate if necessary
		title := cat.Title
		if len(title) > titleWidth {
			title = title[:titleWidth-3] + "..."
		}
		
		desc := cat.Description
		if len(desc) > descWidth {
			desc = desc[:descWidth-3] + "..."
		}
		
		// Format the line with precise width control
		content := fmt.Sprintf("%s%-*s %-*s %s", 
			numPrefix, titleWidth, title, descWidth, desc, msgCountStr)
		
		lines = append(lines, formatFrameLine(content))
	}
	
	lines = append(lines, createFrameBorder("bottom"))
	lines = append(lines, "")
	lines = append(lines, "Enter category number to view messages, or 'q' to quit.")
	lines = append(lines, "")
	
	return lines
}

// FormatMessageList formats the message list for display
func (bm *BoardManager) FormatMessageList(messages []BoardMessage, categoryTitle string, 
	page int, totalPages int, terminalWidth int) []string {
	
	lines := []string{}
	lines = append(lines, "")
	lines = append(lines, createFrameBorder("top"))
	lines = append(lines, formatFrameLine(centerPad(categoryTitle, CONTENT_WIDTH)))
	
	// Format page info with proper padding
	pageInfo := fmt.Sprintf("Page %d of %d", page, totalPages)
	lines = append(lines, formatFrameLine(pageInfo))
	lines = append(lines, createFrameBorder("middle"))
	
	if len(messages) == 0 {
		lines = append(lines, formatFrameLine("No messages in this category."))
	} else {
		for i, msg := range messages {
			timeStr := msg.CreatedAt.Format("2006-01-02 15:04")
			
			// Calculate dynamic widths for message display
			numPrefix := fmt.Sprintf("%d. ", i+1)
			availableSpace := CONTENT_WIDTH - len(numPrefix) - len(msg.Author) - len(timeStr) - 2 // -2 for spaces
			
			// Use remaining space for subject
			subjectWidth := availableSpace
			if subjectWidth < 10 {
				subjectWidth = 10
			}
			
			subject := msg.Subject
			if len(subject) > subjectWidth {
				subject = subject[:subjectWidth-3] + "..."
			}
			
			// Format the line with precise width control
			content := fmt.Sprintf("%s%-*s %s %s", 
				numPrefix, subjectWidth, subject, msg.Author, timeStr)
			
			lines = append(lines, formatFrameLine(content))
		}
	}
	
	lines = append(lines, createFrameBorder("bottom"))
	lines = append(lines, "")
	lines = append(lines, "Enter message number to read, 'n' for new message, 'b' for back, or 'q' to quit.")
	lines = append(lines, "")
	
	return lines
}

// FormatMessage formats a single message for display
func (bm *BoardManager) FormatMessage(message BoardMessage, terminalWidth int) []string {
	lines := []string{}
	lines = append(lines, "")
	lines = append(lines, createFrameBorder("top"))
	
	// Format header information with proper spacing
	dateStr := message.CreatedAt.Format("2006-01-02 15:04:05")
	headerInfo := fmt.Sprintf("From: %s Date: %s", message.Author, dateStr)
	lines = append(lines, formatFrameLine(headerInfo))
	
	// Format subject line
	subjectInfo := fmt.Sprintf("Subject: %s", message.Subject)
	lines = append(lines, formatFrameLine(subjectInfo))
	
	lines = append(lines, createFrameBorder("middle"))
	
	// Word wrap the content
	contentLines := wrapText(message.Content, CONTENT_WIDTH)
	for _, line := range contentLines {
		lines = append(lines, formatFrameLine(line))
	}
	
	lines = append(lines, createFrameBorder("bottom"))
	lines = append(lines, "")
	lines = append(lines, "Press Enter to continue...")
	lines = append(lines, "")
	
	return lines
}

// Helper functions

// centerPad centers text within a given width
func centerPad(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}
	
	padding := width - len(text)
	leftPad := padding / 2
	rightPad := padding - leftPad
	
	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}

// truncateString truncates a string to a maximum length
func truncateString(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// wrapText wraps text to fit within a given width
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	
	lines := []string{}
	currentLine := ""
	
	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}
	
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	
	return lines
}

// CreateCategory creates a new category (admin function)
func (bm *BoardManager) CreateCategory(name, title, description, createdBy string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("category name cannot be empty")
	}
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("category title cannot be empty")
	}
	
	// Check if category already exists
	var count int
	err := bm.db.QueryRow("SELECT COUNT(*) FROM board_categories WHERE name = ?", name).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("category already exists: %s", name)
	}
	
	// Insert the category
	_, err = bm.db.Exec(`
		INSERT INTO board_categories (name, title, description, created_at, created_by)
		VALUES (?, ?, ?, ?, ?)
	`, name, title, description, time.Now().Unix(), createdBy)
	
	if err != nil {
		return err
	}
	
	logger.Info(logger.AreaGeneral, "Category created: %s by %s", name, createdBy)
	return nil
}