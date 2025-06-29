package chess

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
)

// Piece types
type PieceType int

const (
	Empty PieceType = iota
	Pawn
	Rook
	Knight
	Bishop
	Queen
	King
)

// Colors
type Color int

const (
	White Color = iota
	Black
)

// Position represents a position on the chessboard
type Position struct {
	Row, Col int
}

// Piece represents a chess piece
type Piece struct {
	Type  PieceType
	Color Color
}

// Move represents a chess move
type Move struct {
	From, To Position
	Piece    Piece
	Captured *Piece // nil if no piece was captured
	Special  string // "castle", "en_passant", "promotion"
}

// ChessEngine represents the chess game engine
type ChessEngine struct {
	Board         [8][8]*Piece
	CurrentPlayer Color
	GameOver      bool
	Winner        *Color
	MoveHistory   []Move
	Difficulty    int // 1=Easy, 2=Medium, 3=Hard
	KingMoved     map[Color]bool
	RookMoved     map[Color]map[Position]bool
}

// NewChessEngine creates a new chess engine with specified difficulty
func NewChessEngine(difficulty int) *ChessEngine {
	if difficulty < 1 || difficulty > 3 {
		difficulty = 2 // Default to medium
	}

	engine := &ChessEngine{
		CurrentPlayer: White,
		Difficulty:    difficulty,
		KingMoved:     make(map[Color]bool),
		RookMoved:     make(map[Color]map[Position]bool),
	}

	// Initialize rook moved tracking
	engine.RookMoved[White] = make(map[Position]bool)
	engine.RookMoved[Black] = make(map[Position]bool)

	engine.InitializeBoard()
	return engine
}

// InitializeBoard sets up the initial chess board position
func (e *ChessEngine) InitializeBoard() {
	// Clear board
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			e.Board[row][col] = nil
		}
	}

	// Place pieces
	pieceOrder := []PieceType{Rook, Knight, Bishop, Queen, King, Bishop, Knight, Rook}

	// Black pieces (top rows)
	for col := 0; col < 8; col++ {
		e.Board[0][col] = &Piece{Type: pieceOrder[col], Color: Black}
		e.Board[1][col] = &Piece{Type: Pawn, Color: Black}
	}

	// White pieces (bottom rows)
	for col := 0; col < 8; col++ {
		e.Board[6][col] = &Piece{Type: Pawn, Color: White}
		e.Board[7][col] = &Piece{Type: pieceOrder[col], Color: White}
	}
}

// IsValidPosition checks if a position is within the board
func IsValidPosition(pos Position) bool {
	return pos.Row >= 0 && pos.Row < 8 && pos.Col >= 0 && pos.Col < 8
}

// GetPiece returns the piece at the given position
func (e *ChessEngine) GetPiece(pos Position) *Piece {
	if !IsValidPosition(pos) {
		return nil
	}
	return e.Board[pos.Row][pos.Col]
}

// IsValidMove checks if a move is valid according to chess rules
func (e *ChessEngine) IsValidMove(from, to Position) bool {
	piece := e.GetPiece(from)
	if piece == nil || piece.Color != e.CurrentPlayer {
		return false
	}

	if !IsValidPosition(to) {
		return false
	}

	targetPiece := e.GetPiece(to)
	if targetPiece != nil && targetPiece.Color == piece.Color {
		return false // Can't capture own piece
	}

	// Check piece-specific movement rules
	switch piece.Type {
	case Pawn:
		return e.isValidPawnMove(from, to, piece.Color)
	case Rook:
		return e.isValidRookMove(from, to)
	case Knight:
		return e.isValidKnightMove(from, to)
	case Bishop:
		return e.isValidBishopMove(from, to)
	case Queen:
		return e.isValidQueenMove(from, to)
	case King:
		return e.isValidKingMove(from, to)
	}

	return false
}

// isValidPawnMove checks pawn movement rules
func (e *ChessEngine) isValidPawnMove(from, to Position, color Color) bool {
	direction := -1 // White pawns move up (decreasing row numbers)
	startRow := 6   // White pawns start at row 6
	if color == Black {
		direction = 1 // Black pawns move down (increasing row numbers)
		startRow = 1  // Black pawns start at row 1
	}

	rowDiff := to.Row - from.Row
	colDiff := to.Col - from.Col

	// Forward move
	if colDiff == 0 {
		// One square forward
		if rowDiff == direction && e.GetPiece(to) == nil {
			return true
		}
		// Two squares forward from starting position
		if from.Row == startRow && rowDiff == 2*direction && e.GetPiece(to) == nil {
			return true
		}
	}

	// Diagonal capture
	if abs(colDiff) == 1 && rowDiff == direction {
		targetPiece := e.GetPiece(to)
		return targetPiece != nil && targetPiece.Color != color
	}

	return false
}

// isValidRookMove checks rook movement rules
func (e *ChessEngine) isValidRookMove(from, to Position) bool {
	if from.Row != to.Row && from.Col != to.Col {
		return false // Rook moves in straight lines
	}

	return e.isPathClear(from, to)
}

// isValidKnightMove checks knight movement rules
func (e *ChessEngine) isValidKnightMove(from, to Position) bool {
	rowDiff := abs(to.Row - from.Row)
	colDiff := abs(to.Col - from.Col)

	return (rowDiff == 2 && colDiff == 1) || (rowDiff == 1 && colDiff == 2)
}

// isValidBishopMove checks bishop movement rules
func (e *ChessEngine) isValidBishopMove(from, to Position) bool {
	rowDiff := abs(to.Row - from.Row)
	colDiff := abs(to.Col - from.Col)

	if rowDiff != colDiff {
		return false // Bishop moves diagonally
	}

	return e.isPathClear(from, to)
}

// isValidQueenMove checks queen movement rules
func (e *ChessEngine) isValidQueenMove(from, to Position) bool {
	return e.isValidRookMove(from, to) || e.isValidBishopMove(from, to)
}

// isValidKingMove checks king movement rules
func (e *ChessEngine) isValidKingMove(from, to Position) bool {
	rowDiff := abs(to.Row - from.Row)
	colDiff := abs(to.Col - from.Col)

	// Normal king move (one square in any direction)
	if rowDiff <= 1 && colDiff <= 1 {
		return true
	}

	// TODO: Add castling logic if needed
	return false
}

// isPathClear checks if the path between two positions is clear
func (e *ChessEngine) isPathClear(from, to Position) bool {
	rowStep := sign(to.Row - from.Row)
	colStep := sign(to.Col - from.Col)

	currentRow := from.Row + rowStep
	currentCol := from.Col + colStep

	for currentRow != to.Row || currentCol != to.Col {
		if e.Board[currentRow][currentCol] != nil {
			return false
		}
		currentRow += rowStep
		currentCol += colStep
	}

	return true
}

// MakeMove executes a move on the board
func (e *ChessEngine) MakeMove(from, to Position) error {
	if !e.IsValidMove(from, to) {
		return errors.New("invalid move")
	}

	piece := e.GetPiece(from)
	captured := e.GetPiece(to)

	// Execute move
	e.Board[to.Row][to.Col] = piece
	e.Board[from.Row][from.Col] = nil

	// Record move
	move := Move{
		From:     from,
		To:       to,
		Piece:    *piece,
		Captured: captured,
	}
	e.MoveHistory = append(e.MoveHistory, move)

	// Switch players
	e.CurrentPlayer = 1 - e.CurrentPlayer

	// Check for game over conditions
	e.checkGameOver()

	return nil
}

// GetComputerMove generates a computer move based on difficulty
func (e *ChessEngine) GetComputerMove() (*Move, error) {
	moves := e.getAllValidMoves(e.CurrentPlayer)
	if len(moves) == 0 {
		return nil, errors.New("no valid moves available")
	}

	switch e.Difficulty {
	case 1: // Easy - random moves
		return e.getRandomMove(moves), nil
	case 2: // Medium - simple evaluation
		return e.getBestMove(moves, 2), nil
	case 3: // Hard - deeper evaluation
		return e.getBestMove(moves, 3), nil
	}

	return e.getRandomMove(moves), nil
}

// getAllValidMoves returns all valid moves for a color
func (e *ChessEngine) getAllValidMoves(color Color) []Move {
	var moves []Move

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			piece := e.Board[row][col]
			if piece == nil || piece.Color != color {
				continue
			}

			from := Position{row, col}
			for toRow := 0; toRow < 8; toRow++ {
				for toCol := 0; toCol < 8; toCol++ {
					to := Position{toRow, toCol}
					if e.IsValidMove(from, to) {
						captured := e.GetPiece(to)
						move := Move{
							From:     from,
							To:       to,
							Piece:    *piece,
							Captured: captured,
						}
						moves = append(moves, move)
					}
				}
			}
		}
	}

	return moves
}

// getRandomMove returns a random move from the list
func (e *ChessEngine) getRandomMove(moves []Move) *Move {
	if len(moves) == 0 {
		return nil
	}
	return &moves[rand.Intn(len(moves))]
}

// getBestMove uses simple minimax to find the best move
func (e *ChessEngine) getBestMove(moves []Move, depth int) *Move {
	bestMove := &moves[0]
	bestScore := -9999.0

	for i := range moves {
		// Make temporary move
		originalPiece := e.GetPiece(moves[i].To)
		e.Board[moves[i].To.Row][moves[i].To.Col] = e.GetPiece(moves[i].From)
		e.Board[moves[i].From.Row][moves[i].From.Col] = nil

		// Evaluate position
		score := e.evaluatePosition()
		if e.CurrentPlayer == Black {
			score = -score
		}

		// Undo move
		e.Board[moves[i].From.Row][moves[i].From.Col] = e.Board[moves[i].To.Row][moves[i].To.Col]
		e.Board[moves[i].To.Row][moves[i].To.Col] = originalPiece

		if score > bestScore {
			bestScore = score
			bestMove = &moves[i]
		}
	}

	return bestMove
}

// evaluatePosition returns a simple evaluation of the current position
func (e *ChessEngine) evaluatePosition() float64 {
	score := 0.0

	pieceValues := map[PieceType]float64{
		Pawn:   1.0,
		Knight: 3.0,
		Bishop: 3.0,
		Rook:   5.0,
		Queen:  9.0,
		King:   100.0,
	}

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			piece := e.Board[row][col]
			if piece == nil {
				continue
			}

			value := pieceValues[piece.Type]
			if piece.Color == White {
				score += value
			} else {
				score -= value
			}
		}
	}

	return score
}

// checkGameOver checks if the game is over
func (e *ChessEngine) checkGameOver() {
	// Simple check - if no pieces left for a side (simplified)
	whiteKing := false
	blackKing := false

	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			piece := e.Board[row][col]
			if piece != nil && piece.Type == King {
				if piece.Color == White {
					whiteKing = true
				} else {
					blackKing = true
				}
			}
		}
	}

	if !whiteKing {
		e.GameOver = true
		winner := Black
		e.Winner = &winner
	} else if !blackKing {
		e.GameOver = true
		winner := White
		e.Winner = &winner
	}
}

// GetBoardString returns a string representation of the board
func (e *ChessEngine) GetBoardString() string {
	var sb strings.Builder

	pieceSymbols := map[PieceType]map[Color]string{
		Pawn:   {White: "♙", Black: "♟"},
		Rook:   {White: "♖", Black: "♜"},
		Knight: {White: "♘", Black: "♞"},
		Bishop: {White: "♗", Black: "♝"},
		Queen:  {White: "♕", Black: "♛"},
		King:   {White: "♔", Black: "♚"},
	}

	sb.WriteString("  a b c d e f g h\n")
	for row := 0; row < 8; row++ {
		sb.WriteString(fmt.Sprintf("%d ", 8-row))
		for col := 0; col < 8; col++ {
			piece := e.Board[row][col]
			if piece == nil {
				sb.WriteString(". ")
			} else {
				sb.WriteString(pieceSymbols[piece.Type][piece.Color] + " ")
			}
		}
		sb.WriteString(fmt.Sprintf("%d\n", 8-row))
	}
	sb.WriteString("  a b c d e f g h\n")

	return sb.String()
}

// ParsePosition converts chess notation (e.g., "e4") to Position
func ParsePosition(notation string) (Position, error) {
	if len(notation) != 2 {
		return Position{}, errors.New("invalid position notation")
	}

	col := int(notation[0] - 'a')
	row := 8 - int(notation[1]-'0')

	if col < 0 || col > 7 || row < 0 || row > 7 {
		return Position{}, errors.New("position out of bounds")
	}

	return Position{row, col}, nil
}

// PositionToNotation converts Position to chess notation
func PositionToNotation(pos Position) string {
	col := rune('a' + pos.Col)
	row := rune('0' + (8 - pos.Row))
	return string(col) + string(row)
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	if x > 0 {
		return 1
	} else if x < 0 {
		return -1
	}
	return 0
}
