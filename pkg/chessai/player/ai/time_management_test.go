package ai

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

// place sets a colored piece of the given type at (row, col) and returns nothing.
func place(b *board.Board, row, col location.CoordinateType, c color.Color, pieceType byte) location.Location {
	loc := location.NewLocation(row, col)
	p := board.PieceFromType(pieceType)
	p.SetColor(c)
	p.SetPosition(loc)
	b.SetPiece(loc, p)
	if pieceType == piece.KingType {
		b.KingLocations[c] = loc
	}
	return loc
}

func TestSearchUnstable(t *testing.T) {
	mv := func(sr, sc, er, ec location.CoordinateType) location.Move {
		return location.Move{Start: location.NewLocation(sr, sc), End: location.NewLocation(er, ec)}
	}
	a := mv(1, 4, 3, 4)
	b := mv(1, 3, 3, 3)

	// No prior completed depth → never unstable.
	if searchUnstable(ScoredMove{Score: NegInf}, ScoredMove{Move: a, Score: 100}) {
		t.Error("first iteration should be treated as stable")
	}
	// Same move, steady score → stable.
	if searchUnstable(ScoredMove{Move: a, Score: 100}, ScoredMove{Move: a, Score: 90}) {
		t.Error("same move with a small score change should be stable")
	}
	// Best move changed → unstable.
	if !searchUnstable(ScoredMove{Move: a, Score: 100}, ScoredMove{Move: b, Score: 100}) {
		t.Error("a changed best move should be unstable")
	}
	// Same move but score dropped past the threshold → unstable.
	if !searchUnstable(ScoredMove{Move: a, Score: 100}, ScoredMove{Move: a, Score: 40}) {
		t.Error("a >50cp score drop should be unstable")
	}
}

// TestIterativeABDADAEasyMove builds a position with exactly one legal move
// (the white king is checked along its file and has a single escape) and
// verifies the search returns that move immediately at depth 0 without
// burning the clock.
func TestIterativeABDADAEasyMove(t *testing.T) {
	b := &board.Board{}
	// Geometry in (row, col): white king cornered at (0,7), black rook on the
	// same file at (7,7) gives check, a black knight at (2,4) covers (1,6), so
	// the only legal move is K(0,7)->(0,6).
	place(b, 0, 7, color.White, piece.KingType)
	place(b, 7, 7, color.Black, piece.RookType)
	place(b, 2, 4, color.Black, piece.KnightType)
	place(b, 7, 0, color.Black, piece.KingType)

	moves := b.GetAllMoves(color.White, nil)
	if got := len(*moves); got != 1 {
		t.Fatalf("test precondition failed: expected exactly 1 legal move, got %d", got)
	}
	only := (*moves)[0]

	p := NewAIPlayer(color.White, &ABDADA{})
	p.MaxSearchDepth = 8
	ab := &ABDADA{player: p}

	got := ab.iterativeABDADA(b, nil)
	if !got.Move.Start.Equals(only.Start) || !got.Move.End.Equals(only.End) {
		t.Fatalf("easy move: got %s, want the only legal move %s", got.Move, only)
	}
	if p.LastSearchDepth != 0 {
		t.Errorf("easy move should not deepen the search; LastSearchDepth = %d, want 0", p.LastSearchDepth)
	}
}
