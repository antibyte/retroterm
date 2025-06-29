package chess

import (
	"testing"
)

func TestChessEngineBasics(t *testing.T) {
	// Create a new chess engine
	engine := NewChessEngine(2)

	// Test initial state
	if engine.CurrentPlayer != White {
		t.Errorf("Expected current player to be White, got %v", engine.CurrentPlayer)
	}

	if engine.GameOver {
		t.Error("Expected game not to be over initially")
	}

	// Test initial board setup
	piece := engine.GetPiece(Position{0, 0})
	if piece == nil || piece.Type != Rook || piece.Color != Black {
		t.Error("Expected black rook at position a8")
	}

	piece = engine.GetPiece(Position{7, 4})
	if piece == nil || piece.Type != King || piece.Color != White {
		t.Error("Expected white king at position e1")
	}
}

func TestChessPositionParsing(t *testing.T) {
	// Test valid positions
	pos, err := ParsePosition("e4")
	if err != nil {
		t.Errorf("Failed to parse valid position: %v", err)
	}
	if pos.Row != 4 || pos.Col != 4 {
		t.Errorf("Expected position (4,4) for e4, got (%d,%d)", pos.Row, pos.Col)
	}

	// Test invalid positions
	_, err = ParsePosition("z9")
	if err == nil {
		t.Error("Expected error for invalid position z9")
	}

	// Test notation conversion
	notation := PositionToNotation(Position{4, 4})
	if notation != "e4" {
		t.Errorf("Expected e4, got %s", notation)
	}
}

func TestChessMove(t *testing.T) {
	engine := NewChessEngine(2)

	// Test valid pawn move
	err := engine.MakeMove(Position{6, 4}, Position{4, 4}) // e2 to e4
	if err != nil {
		t.Errorf("Valid pawn move failed: %v", err)
	}

	// Test that current player switched
	if engine.CurrentPlayer != Black {
		t.Error("Expected current player to switch to Black after move")
	}

	// Test invalid move
	err = engine.MakeMove(Position{0, 0}, Position{7, 7}) // Invalid rook move
	if err == nil {
		t.Error("Expected error for invalid move")
	}
}

func TestChessUI(t *testing.T) {
	ui := NewChessUI(2, White)

	if ui.Engine == nil {
		t.Error("Expected UI to have a chess engine")
	}

	if ui.PlayerColor != White {
		t.Error("Expected player color to be White")
	}

	// Test input handling
	messages := ui.HandleInput("help")
	if len(messages) == 0 {
		t.Error("Expected help messages")
	}
}
