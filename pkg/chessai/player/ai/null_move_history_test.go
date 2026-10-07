package ai

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/stretchr/testify/require"
)

func TestNullMoveClearsOppositeTurnRepetition(t *testing.T) {
	root := &board.Board{}
	root.ResetDefault()
	// Keep a material imbalance so a false path draw changes the search score.
	root.SetPiece(gateSquare(t, "d1"), nil)
	initial := root.Hash()
	gateApply(t, root, nil, color.Black, "b8c6")
	require.Len(t, root.PreviousPositions, 1)

	// A black move after White's artificial pass restores the placement, but
	// White is now to move whereas Black was to move at the initial position.
	broken := root.Copy()
	gateApply(t, broken, nil, color.Black, "c6b8")
	require.Equal(t, initial, broken.Hash())
	require.Equal(t, 1, broken.CurrentPositionRepeats)

	child := boardAfterNullMove(root)
	require.Empty(t, child.PreviousPositions)
	require.Zero(t, child.CurrentPositionRepeats)
	require.Zero(t, child.PreviousPositionsSeen)
	last := gateApply(t, child, nil, color.Black, "c6b8")
	require.Equal(t, initial, child.Hash())
	require.Zero(t, child.CurrentPositionRepeats)
	require.Len(t, root.PreviousPositions, 1)
	require.Zero(t, root.CurrentPositionRepeats)

	p := NewAIPlayer(color.White, &ABDADA{NumThreads: 1})
	ab := &ABDADA{player: p, NumThreads: 1}
	falseDraw := ab.ABDADA(broken, 0, NegInf, PosInf, false, color.White, last, false, 2, 0, false)
	require.True(t, falseDraw.PathDependentDraw)
	require.Zero(t, falseDraw.Score)
	score := ab.ABDADA(child, 0, NegInf, PosInf, false, color.White, last, false, 2, 0, false)
	require.False(t, score.PathDependentDraw)
	require.True(t, score.Score < -500, "material disadvantage must not become a draw: %d", score.Score)

	// Ordinary same-turn cycles entirely after the pass still count normally.
	last = gateApply(t, child, last, color.White, "g1f3")
	last = gateApply(t, child, last, color.Black, "b8c6")
	last = gateApply(t, child, last, color.White, "f3g1")
	gateApply(t, child, last, color.Black, "c6b8")
	require.Equal(t, 1, child.CurrentPositionRepeats)
}

func TestNullMovePreservesParentHistoryAndCounters(t *testing.T) {
	root := &board.Board{}
	root.ResetDefault()
	gateApply(t, root, nil, color.White, "g1f3")
	root.CurrentPositionRepeats = 1
	root.PreviousPositionsSeen = 3
	root.MovesSinceNoDraw = 42
	hash := root.Hash()
	child := boardAfterNullMove(root)
	require.Equal(t, hash, child.Hash())
	require.Equal(t, root.MovesSinceNoDraw, child.MovesSinceNoDraw)
	require.Zero(t, child.CurrentPositionRepeats)
	require.Zero(t, child.PreviousPositionsSeen)
	require.Len(t, root.PreviousPositions, 1)
	require.Equal(t, 1, root.CurrentPositionRepeats)
	require.Equal(t, 3, root.PreviousPositionsSeen)
}
