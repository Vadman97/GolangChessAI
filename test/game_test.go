package test

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TODO(Alex) Stalemate, Checkmate Tests

func performFiftyDrawMoves(bIn *board.Board) *board.Board {
	b := bIn
	if b == nil {
		b = &board.Board{}
	}
	b.ResetDefault()
	for i := 0; i < 25; i++ {
		// Black Knight
		board.MakeMove(&location.Move{
			Start: location.NewLocation(7, 6),
			End: location.NewLocation(6, 5),
		}, b)

		// White Knight
		board.MakeMove(&location.Move{
			Start: location.NewLocation(0, 6),
			End: location.NewLocation(1, 5),
		}, b)

		// Undo Black Knight
		board.MakeMove(&location.Move{
			Start: location.NewLocation(6, 5),
			End: location.NewLocation(7, 6),
		}, b)

		// Undo White Knight
		board.MakeMove(&location.Move{
			Start: location.NewLocation(1, 5),
			End: location.NewLocation(0, 6),
		}, b)
	}

	return b
}

func TestFiftyMoveDraw(t *testing.T) {
	b := performFiftyDrawMoves(nil)
	assert.Equal(t, b.MovesSinceNoDraw, 100)
}

func TestFiftyMoveDrawResetByPawnMove(t *testing.T) {
	b := performFiftyDrawMoves(nil)
	assert.Equal(t, b.MovesSinceNoDraw, 100)

	board.MakeMove(&location.Move{
		Start: location.NewLocation(1, 0),
		End: location.NewLocation(2, 0),
	}, b)
	assert.Equal(t, b.MovesSinceNoDraw, 0)
}

func TestFiftyMoveDrawResetByCapture(t *testing.T) {
	b := &board.Board{}

	// Initialize two pawns in a capture position
	board.MakeMove(&location.Move{
		Start: location.NewLocation(1, 4),
		End: location.NewLocation(3, 4),
	}, b)
	board.MakeMove(&location.Move{
		Start: location.NewLocation(6, 3),
		End: location.NewLocation(4, 3),
	}, b)

	b = performFiftyDrawMoves(b)

	// 100 since pawn moves will reset the counter
	assert.Equal(t, b.MovesSinceNoDraw, 100)

	board.MakeMove(&location.Move{
		Start: location.NewLocation(3, 4),
		End: location.NewLocation(4, 3),
	}, b)

	assert.Equal(t, b.MovesSinceNoDraw, 0)
}
