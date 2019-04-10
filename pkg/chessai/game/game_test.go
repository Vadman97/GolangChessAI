package game

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"testing"
)

// TODO(Alex) Stalemate, Checkmate Tests

func simulateGameMove(move *location.Move, b *board.Board, ) {
	lastMove := board.MakeMove(move, b)
	b.UpdateDrawCounter(lastMove)
}

func performFiftyDrawMoves(bIn *board.Board) *board.Board {
	b := bIn
	if b == nil {
		b = &board.Board{}
	}
	b.ResetDefault()
	for i := 0; i < 25; i++ {
		// Black Knight
		simulateGameMove(&location.Move{
			Start: location.NewLocation(7, 6),
			End: location.NewLocation(6, 5),
		}, b)

		// White Knight
		simulateGameMove(&location.Move{
			Start: location.NewLocation(0, 6),
			End: location.NewLocation(1, 5),
		}, b)

		// Undo Black Knight
		simulateGameMove(&location.Move{
			Start: location.NewLocation(6, 5),
			End: location.NewLocation(7, 6),
		}, b)

		// Undo White Knight
		simulateGameMove(&location.Move{
			Start: location.NewLocation(1, 5),
			End: location.NewLocation(0, 6),
		}, b)
	}

	return b
}

func TestFiftyMoveDraw(t *testing.T) {
	b := performFiftyDrawMoves(nil)
	assert.Equal(t, 100, b.MovesSinceNoDraw)
}

func TestFiftyMoveDrawResetByPawnMove(t *testing.T) {
	b := performFiftyDrawMoves(nil)
	assert.Equal(t, 100, b.MovesSinceNoDraw)

	simulateGameMove(&location.Move{
		Start: location.NewLocation(1, 0),
		End: location.NewLocation(2, 0),
	}, b)
	assert.Equal(t, 0, b.MovesSinceNoDraw)
}

func TestFiftyMoveDrawResetByCapture(t *testing.T) {
	b := &board.Board{}

	// Initialize two pawns in a capture position
	simulateGameMove(&location.Move{
		Start: location.NewLocation(1, 4),
		End: location.NewLocation(3, 4),
	}, b)
	simulateGameMove(&location.Move{
		Start: location.NewLocation(6, 3),
		End: location.NewLocation(4, 3),
	}, b)

	b = performFiftyDrawMoves(b)

	// 100 since pawn moves will reset the counter
	assert.Equal(t, 100, b.MovesSinceNoDraw)

	simulateGameMove(&location.Move{
		Start: location.NewLocation(3, 4),
		End: location.NewLocation(4, 3),
	}, b)

	assert.Equal(t, 0, b.MovesSinceNoDraw)
}
