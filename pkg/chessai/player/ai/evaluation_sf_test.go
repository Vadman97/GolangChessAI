package ai

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

// The Stockfish-classical eval is symmetric on the starting position, so White-to-move
// should evaluate to ~0 (only tiny PSQT asymmetry, none here since the start is mirrored).
func TestStockfishClassicStartPositionBalanced(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	score := evaluateStockfishClassicScore(b, color.White)
	if score < -10 || score > 10 {
		t.Fatalf("start position should be ~balanced, got %d cp", score)
	}
}

// Evaluation is side-to-move relative: for the same board, evaluating as White must be
// the negation of evaluating as Black.
func TestStockfishClassicSideToMoveSymmetry(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	w := evaluateStockfishClassicScore(b, color.White)
	bl := evaluateStockfishClassicScore(b, color.Black)
	if w != -bl {
		t.Fatalf("side-to-move asymmetry: white=%d black=%d (expected white == -black)", w, bl)
	}
}

// Removing Black's queen must make the position strongly winning for White (a queen is
// ~9 pawns; even at the conservative SF normalization that is several hundred cp).
func TestStockfishClassicMaterialAdvantage(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	// Find and remove Black's queen.
	removed := false
	for row := location.CoordinateType(0); row < board.Height && !removed; row++ {
		for col := location.CoordinateType(0); col < board.Width; col++ {
			p := b.GetPiece(location.NewLocation(row, col))
			if p != nil && p.GetColor() == color.Black && p.GetPieceType() == piece.QueenType {
				b.SetPiece(location.NewLocation(row, col), nil)
				removed = true
				break
			}
		}
	}
	if !removed {
		t.Fatal("could not find Black queen to remove")
	}

	score := evaluateStockfishClassicScore(b, color.White)
	if score < 400 {
		t.Fatalf("White up a queen should be strongly winning, got %d cp", score)
	}
}
