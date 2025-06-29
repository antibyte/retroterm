package chess

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/antibyte/retroterm/pkg/logger"
	"github.com/antibyte/retroterm/pkg/shared"
)

// ChessUI handles the chess game user interface
type ChessUI struct {
	Engine         *ChessEngine
	SelectedSquare *Position
	PlayerColor    Color
	ShowCoords     bool
	Messages       []shared.Message
}

// NewChessUI creates a new chess UI
func NewChessUI(difficulty int, playerColor Color) *ChessUI {
	return &ChessUI{
		Engine:      NewChessEngine(difficulty),
		PlayerColor: playerColor,
		ShowCoords:  true,
		Messages:    make([]shared.Message, 0),
	}
}

// GetSpriteFilename returns the filename for a chess piece sprite
func GetSpriteFilename(piece *Piece) string {
	if piece == nil {
		return ""
	}

	colorName := "White"
	if piece.Color == Black {
		colorName = "Black"
	}

	pieceNames := map[PieceType]string{
		Pawn:   "Pawn",
		Rook:   "Rook",
		Knight: "Knight",
		Bishop: "Bishop",
		Queen:  "Queen",
		King:   "King",
	}

	return fmt.Sprintf("%s%s.png", colorName, pieceNames[piece.Type])
}

// LoadBitmap loads a bitmap file and converts it to base64
func LoadBitmap(filename string) (string, error) {
	logger.Debug(logger.AreaChess, "LoadBitmap: Attempting to load bitmap: %s", filename)

	// Try multiple relative paths to find the chess graphics (cross-platform)
	paths := []string{
		filepath.Join("chess_gfx", filename),
		filepath.Join("..", "chess_gfx", filename),
		filepath.Join("..", "..", "chess_gfx", filename),
		filepath.Join("..", "..", "..", "chess_gfx", filename),
		filepath.Join("c:", "Users", "andre", "Desktop", "chess_gfx", filename), // Absolute path for Windows
	}

	var data []byte
	var err error

	for i, path := range paths {
		logger.Debug(logger.AreaChess, "LoadBitmap: Trying path %d: %s", i+1, path)
		data, err = os.ReadFile(path)
		if err == nil {
			logger.Info(logger.AreaChess, "LoadBitmap: Successfully loaded bitmap from path: %s, size: %d bytes", path, len(data))
			break
		}
		logger.Debug(logger.AreaChess, "LoadBitmap: Failed to load from path %s: %v", path, err)
	}

	if err != nil {
		logger.Error(logger.AreaChess, "LoadBitmap: Failed to load bitmap %s from any path: %v", filename, err)
		return "", fmt.Errorf("failed to load bitmap %s from any path: %v", filename, err)
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	logger.Debug(logger.AreaChess, "LoadBitmap: Successfully encoded bitmap %s, base64 length: %d", filename, len(encoded))
	return encoded, nil
}

// LoadSpritePixelData loads a PNG image and converts it to pixel data for the sprite system
func LoadSpritePixelData(filename string) ([]int, error) {
	// Try multiple relative paths to find the chess graphics (cross-platform)
	paths := []string{
		filepath.Join("chess_gfx", filename),
		filepath.Join("..", "chess_gfx", filename),
		filepath.Join("..", "..", "chess_gfx", filename),
		filepath.Join("..", "..", "..", "chess_gfx", filename),
		filepath.Join("c:", "Users", "andre", "Desktop", "chess_gfx", filename), // Absolute path for Windows
	}

	var file *os.File
	var err error

	for _, path := range paths {
		file, err = os.Open(path)
		if err == nil {
			defer file.Close()
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load sprite %s from any path: %v", filename, err)
	}
	// Decode PNG image
	img, err := png.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG %s: %v", filename, err)
	}

	// Convert image to 32x32 pixel data array with 16 brightness levels
	pixelData := make([]int, 1024) // 32x32 = 1024 pixels
	bounds := img.Bounds()
	_ = image.Point{} // Explicit use of image package to avoid unused import error

	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			// Scale coordinates if image is not 32x32
			srcX := x * bounds.Dx() / 32
			srcY := y * bounds.Dy() / 32

			// Get pixel color
			r, g, b, a := img.At(bounds.Min.X+srcX, bounds.Min.Y+srcY).RGBA()

			// Convert to grayscale and quantize to 16 levels (0-15)
			if a == 0 {
				// Transparent pixel
				pixelData[y*32+x] = 0
			} else {
				// Convert RGB to grayscale using standard formula
				gray := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 65535.0
				// Quantize to 16 levels (0-15)
				pixelData[y*32+x] = int(gray * 15)
			}
		}
	}

	return pixelData, nil
}

// RenderBoard creates messages to render the chess board with graphics and positioned text
func (ui *ChessUI) RenderBoard() []shared.Message {
	logger.Debug(logger.AreaChess, "RenderBoard: Starting to render chess board")
	messages := make([]shared.Message, 0)

	// Clear screen first
	messages = append(messages, shared.Message{
		Type: shared.MessageTypeClear,
	}) // Display title centered above the board
	// Board starts at Y=112, so place title a few lines above
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeLocate,
		Content: "36,3", // Center "TinyChess" (9 chars): (80-9)/2 = 35.5 ≈ 36, Y=3 for spacing above board
	})
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeText,
		Content: "TinyChess",
	})

	// Load and display the board bitmap
	logger.Debug(logger.AreaChess, "RenderBoard: Loading board bitmap")
	boardData, err := LoadBitmap("Board.png")
	if err != nil {
		logger.Error(logger.AreaChess, "RenderBoard: Failed to load board bitmap, falling back to text: %v", err)
		// Fallback to text-based board if graphics fail
		return ui.renderTextBoard()
	}
	logger.Debug(logger.AreaChess, "RenderBoard: Successfully loaded board bitmap, sending to client") // Display the board centered both horizontally and vertically
	// Board size: 8 fields × 32 pixels = 256 pixels wide/high
	// Graphics screen: 640x480 pixels
	// Center X = (640 - 256) / 2 = 192
	// Center Y = (480 - 256) / 2 = 112
	boardCenterX := 192
	boardCenterY := 112
	messages = append(messages, shared.Message{
		Type:        shared.MessageTypeBitmap,
		BitmapData:  boardData,
		BitmapX:     boardCenterX,
		BitmapY:     boardCenterY,
		BitmapScale: 1.0, BitmapID: "chessboard",
	}) // Add chess board coordinate labels (A-H, 1-8) as pixel-perfect bitmaps
	// Board is centered at (192, 112) with 16px border offset and 32px fields
	// Column labels (A-H) - placed above the board
	for col := 0; col < 8; col++ {
		letter := string(rune('A' + col))
		// Calculate exact pixel position for each column label
		labelX := boardCenterX + 16 + col*32 + 16 - 4 // Center of each field minus half char width
		labelY := boardCenterY + 16 - 16              // Above the board

		// Create bitmap message for the coordinate label
		labelBitmap := ui.createTextBitmap(letter)
		messages = append(messages, shared.Message{
			Type:        shared.MessageTypeBitmap,
			BitmapData:  labelBitmap,
			BitmapX:     labelX,
			BitmapY:     labelY,
			BitmapScale: 1.0,
			BitmapID:    fmt.Sprintf("coord_col_%d", col),
		})
	}

	// Row labels (1-8) - placed to the left of the board
	for row := 0; row < 8; row++ {
		number := fmt.Sprintf("%d", 8-row) // Chess rows are numbered 8-1 from top to bottom
		// Calculate exact pixel position for each row label
		labelX := boardCenterX + 16 - 16              // Left of the board
		labelY := boardCenterY + 16 + row*32 + 16 - 6 // Center of each field minus half char height

		// Create bitmap message for the coordinate label
		labelBitmap := ui.createTextBitmap(number)
		messages = append(messages, shared.Message{
			Type:        shared.MessageTypeBitmap,
			BitmapData:  labelBitmap,
			BitmapX:     labelX,
			BitmapY:     labelY,
			BitmapScale: 1.0,
			BitmapID:    fmt.Sprintf("coord_row_%d", row),
		})
	}

	// Check if computer should make a move before showing the board
	computerMoveMessages := ui.CheckAndMakeComputerMove()
	if len(computerMoveMessages) > 0 {
		// If computer made a move, add the computer move messages
		messages = append(messages, computerMoveMessages...)
	}

	// Display pieces as sprites (render current board state after any computer move)
	logger.Debug(logger.AreaChess, "RenderBoard: Starting to render chess pieces")
	pieceCount := 0
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			piece := ui.Engine.GetPiece(Position{row, col})
			if piece != nil {
				pieceCount++
				spriteMessages := ui.renderPiece(piece, Position{row, col})
				messages = append(messages, spriteMessages...)
			}
		}
	}
	logger.Debug(logger.AreaChess, "RenderBoard: Rendered %d pieces, total messages: %d", pieceCount, len(messages))

	// Add input prompt at bottom of screen
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeLocate,
		Content: "1,24", // Bottom of screen
	})
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeText,
		Content: "Move (e.g. 'e2 e4'), 'help', or 'quit': ",
	})

	return messages
}

// renderPiece creates bitmap messages for a chess piece (simplified approach)
func (ui *ChessUI) renderPiece(piece *Piece, pos Position) []shared.Message {
	logger.Debug(logger.AreaChess, "renderPiece: Rendering piece type %d at position %d,%d", int(piece.Type), pos.Row, pos.Col)
	messages := make([]shared.Message, 0)

	filename := GetSpriteFilename(piece)
	if filename == "" {
		logger.Debug(logger.AreaChess, "renderPiece: No filename for piece type %d", int(piece.Type))
		return messages
	}
	logger.Debug(logger.AreaChess, "renderPiece: Using filename %s for piece", filename)

	// Load piece bitmap data
	bitmapData, err := LoadBitmap(filename)
	if err != nil {
		logger.Error(logger.AreaChess, "renderPiece: Failed to load bitmap %s: %v", filename, err)
		return messages
	}
	logger.Debug(logger.AreaChess, "renderPiece: Successfully loaded bitmap data, length: %d", len(bitmapData)) // Calculate screen position (board is centered at 192,112)
	// Board size: 8 fields × 32 pixels, plus border offset for the actual playable area
	boardCenterX := 192
	boardCenterY := 112
	fieldSize := 32 // 32x32 pixel fields

	// Border offset: Chess board images typically have a border around the playable area
	// Assuming a ~16 pixel border on each side (adjust if needed based on actual board image)
	borderOffsetX := 16
	borderOffsetY := 16

	// Calculate exact field position including border offset
	screenX := boardCenterX + borderOffsetX + pos.Col*fieldSize
	screenY := boardCenterY + borderOffsetY + pos.Row*fieldSize

	logger.Debug(logger.AreaChess, "renderPiece: Calculated position for piece at board[%d,%d] -> screen[%d,%d]", pos.Row, pos.Col, screenX, screenY)

	// Generate unique bitmap ID for this piece
	bitmapID := fmt.Sprintf("chess_piece_%d_%d", pos.Row, pos.Col)
	// Send bitmap message for the piece with original scaling
	messages = append(messages, shared.Message{
		Type:        shared.MessageTypeBitmap,
		BitmapData:  bitmapData,
		BitmapX:     screenX,
		BitmapY:     screenY,
		BitmapScale: 1.0, // Original size scaling to match board
		BitmapID:    bitmapID,
	})

	return messages
}

// renderTextBoard creates a text-based chess board (fallback) using LOCATE
func (ui *ChessUI) renderTextBoard() []shared.Message {
	messages := make([]shared.Message, 0)

	// Clear screen first
	messages = append(messages, shared.Message{
		Type: shared.MessageTypeClear,
	})
	// Title
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeLocate,
		Content: "30,2",
	})
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeText,
		Content: "TinyOS Chess (Text Mode)",
	})

	// Display ASCII board
	boardLines := strings.Split(ui.Engine.GetBoardString(), "\n")
	startY := 5
	for i, line := range boardLines {
		if strings.TrimSpace(line) != "" {
			messages = append(messages, shared.Message{
				Type:    shared.MessageTypeLocate,
				Content: fmt.Sprintf("10,%d", startY+i),
			})
			messages = append(messages, shared.Message{
				Type:    shared.MessageTypeText,
				Content: line,
			})
		}
	}
	// Game status
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeLocate,
		Content: fmt.Sprintf("10,%d", startY+len(boardLines)+2),
	})
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeText,
		Content: ui.GetStatusMessage(),
	}) // Instructions
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeLocate,
		Content: fmt.Sprintf("10,%d", startY+len(boardLines)+4),
	})
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeText,
		Content: "Enter move (e.g. 'e2 e4'), 'help', or 'quit':",
	})

	return messages
}

// HandleInput processes user input for chess moves
func (ui *ChessUI) HandleInput(input string) []shared.Message {
	messages := make([]shared.Message, 0)

	// Parse move input (e.g., "e2 e4" or "e2-e4")
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)
	if input == "quit" || input == "exit" {
		// Signal that the chess game should be ended
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeClear,
			Content: "",
		})
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "Thanks for playing chess!",
		})
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "CHESS_QUIT_SIGNAL", // Special signal for commands.go to detect quit
		})
		return messages
	}
	if input == "help" {
		// Show help and then return to board
		helpMessages := ui.showHelp()
		boardMessages := ui.RenderBoard()
		return append(helpMessages, boardMessages...)
	}
	// Parse move
	parts := strings.Fields(strings.ReplaceAll(input, "-", " "))
	if len(parts) != 2 {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "Invalid move format. Use: e2 e4 or e2-e4",
		})
		return messages
	}

	fromPos, err := ParsePosition(parts[0])
	if err != nil {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "Invalid from position: " + parts[0],
		})
		return messages
	}

	toPos, err := ParsePosition(parts[1])
	if err != nil {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "Invalid to position: " + parts[1],
		})
		return messages
	}

	// Check if it's the player's turn
	if ui.Engine.CurrentPlayer != ui.PlayerColor {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "It's not your turn!",
		})
		return messages
	}

	// Try to make the move
	err = ui.Engine.MakeMove(fromPos, toPos)
	if err != nil {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "Invalid move: " + err.Error(),
		})
		return messages
	}
	// Move successful
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeText,
		Content: fmt.Sprintf("Move: %s -> %s", parts[0], parts[1]),
	})

	// Render updated board (which will automatically handle computer move if needed)
	boardMessages := ui.RenderBoard()
	messages = append(messages, boardMessages...)

	return messages
}

// CheckAndMakeComputerMove automatically makes a computer move if it's the computer's turn
// Returns messages for the computer move, or empty slice if no move was made
func (ui *ChessUI) CheckAndMakeComputerMove() []shared.Message {
	messages := make([]shared.Message, 0)

	// If game is over, no need to make moves
	if ui.Engine.GameOver {
		return messages
	}

	// Computer's turn - make automatic move
	if ui.Engine.CurrentPlayer != ui.PlayerColor {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: "Computer is thinking...",
		})

		computerMove, err := ui.Engine.GetComputerMove()
		if err != nil {
			messages = append(messages, shared.Message{
				Type:    shared.MessageTypeText,
				Content: "Computer cannot move. Game over!",
			})
			return messages
		}

		err = ui.Engine.MakeMove(computerMove.From, computerMove.To)
		if err != nil {
			messages = append(messages, shared.Message{
				Type:    shared.MessageTypeText,
				Content: "Computer move error: " + err.Error(),
			})
			return messages
		}

		fromNotation := PositionToNotation(computerMove.From)
		toNotation := PositionToNotation(computerMove.To)

		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: fmt.Sprintf("Computer moves: %s -> %s", fromNotation, toNotation),
		})

		// Check if game is over after computer move
		if ui.Engine.GameOver {
			if ui.Engine.Winner != nil {
				winnerName := "White"
				if *ui.Engine.Winner == Black {
					winnerName = "Black"
				}
				messages = append(messages, shared.Message{
					Type:    shared.MessageTypeText,
					Content: fmt.Sprintf("Game Over! %s wins!", winnerName),
				})
			} else {
				messages = append(messages, shared.Message{
					Type:    shared.MessageTypeText,
					Content: "Game Over! It's a draw!",
				})
			}
		}
	}

	return messages
}

// showHelp displays help information using LOCATE for positioning
func (ui *ChessUI) showHelp() []shared.Message {
	messages := make([]shared.Message, 0)

	// Clear screen first
	messages = append(messages, shared.Message{
		Type: shared.MessageTypeClear,
	})
	// Title
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeLocate,
		Content: "30,2",
	})
	messages = append(messages, shared.Message{
		Type:    shared.MessageTypeText,
		Content: "Chess Help",
	})
	// Commands section
	y := 5
	helpLines := []string{
		"Commands:",
		"  Move: e2 e4 or e2-e4 (from square to square)",
		"  help: Show this help",
		"  quit/exit: Quit the game",
		"",
		"Square notation: a1-h8 (letters for columns, numbers for rows)",
		fmt.Sprintf("You are playing as: %s", func() string {
			if ui.PlayerColor == White {
				return "White"
			}
			return "Black"
		}()),
		fmt.Sprintf("Difficulty: %d/3", ui.Engine.Difficulty),
		"",
		"Press any key to continue...",
	}
	for _, line := range helpLines {
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeLocate,
			Content: fmt.Sprintf("5,%d", y),
		})
		messages = append(messages, shared.Message{
			Type:    shared.MessageTypeText,
			Content: line,
		})
		y++
	}

	return messages
}

// GetStatusMessage returns current game status
func (ui *ChessUI) GetStatusMessage() string {
	if ui.Engine.GameOver {
		if ui.Engine.Winner != nil {
			winnerName := "White"
			if *ui.Engine.Winner == Black {
				winnerName = "Black"
			}
			return fmt.Sprintf("Game Over - %s wins!", winnerName)
		}
		return "Game Over - Draw!"
	}

	currentPlayerName := "White"
	if ui.Engine.CurrentPlayer == Black {
		currentPlayerName = "Black"
	}

	return fmt.Sprintf("Current player: %s", currentPlayerName)
}

// createTextBitmap creates a simple monospace bitmap for a single character or short string
// This creates an 8x12 pixel bitmap for each character in monospace font style
func (ui *ChessUI) createTextBitmap(text string) string {
	// Simple 8x12 monospace font bitmap data for characters 0-9, A-H
	// Each character is 8 pixels wide, 12 pixels high
	// Bitmap format: 1 = pixel on (white/green), 0 = pixel off (transparent)

	fontData := map[rune][][]int{
		'A': {
			{0, 0, 1, 1, 1, 1, 0, 0},
			{0, 1, 1, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 1, 1, 1, 1, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'B': {
			{1, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'C': {
			{0, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'D': {
			{1, 1, 1, 1, 1, 0, 0, 0},
			{1, 1, 0, 0, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 1, 1, 0, 0},
			{1, 1, 1, 1, 1, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'E': {
			{1, 1, 1, 1, 1, 1, 1, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 0, 0, 0},
			{1, 1, 1, 1, 1, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 1, 1, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'F': {
			{1, 1, 1, 1, 1, 1, 1, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 0, 0, 0},
			{1, 1, 1, 1, 1, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'G': {
			{0, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 1, 1, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'H': {
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 1, 1, 1, 1, 1, 1},
			{1, 1, 1, 1, 1, 1, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{1, 1, 0, 0, 0, 0, 1, 1},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'1': {
			{0, 0, 1, 1, 0, 0, 0, 0},
			{0, 1, 1, 1, 0, 0, 0, 0},
			{1, 1, 1, 1, 0, 0, 0, 0},
			{0, 0, 1, 1, 0, 0, 0, 0},
			{0, 0, 1, 1, 0, 0, 0, 0},
			{0, 0, 1, 1, 0, 0, 0, 0},
			{0, 0, 1, 1, 0, 0, 0, 0},
			{0, 0, 1, 1, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'2': {
			{0, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 0, 0, 0, 0, 1, 1, 0},
			{0, 0, 0, 0, 1, 1, 0, 0},
			{0, 0, 0, 1, 1, 0, 0, 0},
			{0, 0, 1, 1, 0, 0, 0, 0},
			{0, 1, 1, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 1, 1, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'3': {
			{0, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 0, 0, 0, 0, 1, 1, 0},
			{0, 0, 1, 1, 1, 1, 0, 0},
			{0, 0, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0},
			{0, 0, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'4': {
			{0, 0, 0, 0, 1, 1, 0, 0},
			{0, 0, 0, 1, 1, 1, 0, 0},
			{0, 0, 1, 1, 1, 1, 0, 0},
			{0, 1, 1, 0, 1, 1, 0, 0},
			{1, 1, 0, 0, 1, 1, 0, 0},
			{1, 1, 1, 1, 1, 1, 1, 0},
			{0, 0, 0, 0, 1, 1, 0, 0},
			{0, 0, 0, 0, 1, 1, 0, 0},
			{0, 0, 0, 0, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'5': {
			{1, 1, 1, 1, 1, 1, 1, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 1, 1, 0},
			{0, 0, 0, 0, 0, 1, 1, 0},
			{0, 0, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'6': {
			{0, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'7': {
			{1, 1, 1, 1, 1, 1, 1, 0},
			{0, 0, 0, 0, 0, 1, 1, 0},
			{0, 0, 0, 0, 1, 1, 0, 0},
			{0, 0, 0, 1, 1, 0, 0, 0},
			{0, 0, 1, 1, 0, 0, 0, 0},
			{0, 1, 1, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{1, 1, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
		'8': {
			{0, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 1, 1, 1, 1, 1, 0, 0},
			{0, 1, 1, 1, 1, 1, 0, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{1, 1, 0, 0, 0, 1, 1, 0},
			{0, 1, 1, 1, 1, 1, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
			{0, 0, 0, 0, 0, 0, 0, 0},
		},
	}

	// For simplicity, handle only single character
	if len(text) == 0 {
		return ""
	}

	char := rune(text[0])
	bitmap, exists := fontData[char]
	if !exists {
		// Return empty bitmap for unknown characters
		return ""
	}

	// Create a simple PNG in memory using a minimal approach
	// For now, return a base64-encoded 8x12 monochrome PNG
	// This is a simplified implementation - in a full system you'd use proper PNG encoding

	// Create pixel data array (RGBA format)
	width, height := 8, 12
	pixelData := make([]byte, width*height*4) // RGBA

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := (y*width + x) * 4
			if bitmap[y][x] == 1 {
				// White pixel (will be converted to green in frontend)
				pixelData[index] = 255   // R
				pixelData[index+1] = 255 // G
				pixelData[index+2] = 255 // B
				pixelData[index+3] = 255 // A
			} else {
				// Transparent pixel
				pixelData[index] = 0   // R
				pixelData[index+1] = 0 // G
				pixelData[index+2] = 0 // B
				pixelData[index+3] = 0 // A (transparent)
			}
		}
	}

	// Create a minimal PNG - this is a simplified approach
	// In a production system, you'd use proper PNG encoding libraries

	// For now, create a simple base64 representation that the frontend can handle
	// We'll create a simple bitmap format that the frontend can decode
	// Create PNG using Go's image packages
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if bitmap[y][x] == 1 {
				img.Set(x, y, color.RGBA{255, 255, 255, 255}) // White
			} else {
				img.Set(x, y, color.RGBA{0, 0, 0, 0}) // Transparent
			}
		}
	}

	// Encode to PNG and then to base64
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		logger.Error(logger.AreaChess, "createTextBitmap: Failed to encode PNG: %v", err)
		return ""
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
