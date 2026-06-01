package ai

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEarlyOpeningPreferenceStabilizesQGABlack(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()
	var previous *board.LastMove
	for _, uci := range []string{"d2d4", "d7d5", "c2c4", "g8f6", "g1f3", "d5c4", "b1c3"} {
		m := mustMatchTestUCI(t, b, previous, uci)
		previous = board.MakeMove(&m, b)
	}

	p := NewAIPlayer(color.Black, &ABDADA{})
	p.TurnCount = 3

	got := p.earlyOpeningPreference(b, previous)
	require.NotNil(t, got)
	assert.Equal(t, "e7e6", testMoveToUCI(*got))
}

func mustMatchTestUCI(t *testing.T, b *board.Board, previous *board.LastMove, uci string) location.Move {
	t.Helper()
	side := color.White
	if previous != nil {
		side = (*previous.Piece).GetColor() ^ 1
	}
	target := testUCIToMove(t, uci)
	for _, move := range *b.GetAllMoves(side, previous) {
		if move.Start.Equals(target.Start) && move.End.Equals(target.End) {
			return move
		}
	}
	t.Fatalf("move %s is not legal for %s", uci, color.Names[side])
	return location.Move{}
}

func testUCIToMove(t *testing.T, uci string) location.Move {
	t.Helper()
	require.Len(t, uci, 4)
	return location.Move{
		Start: testUCISquareToLocation(t, uci[:2]),
		End:   testUCISquareToLocation(t, uci[2:4]),
	}
}

func testUCISquareToLocation(t *testing.T, square string) location.Location {
	t.Helper()
	require.Len(t, square, 2)
	file := square[0]
	rank := square[1]
	require.True(t, file >= 'a' && file <= 'h')
	require.True(t, rank >= '1' && rank <= '8')
	return location.NewLocation(
		location.CoordinateType(rank-'1'),
		location.CoordinateType('h'-file),
	)
}

func testMoveToUCI(m location.Move) string {
	return testLocationToUCI(m.Start) + testLocationToUCI(m.End)
}

func testLocationToUCI(l location.Location) string {
	file := byte('h' - l.GetCol())
	rank := byte('1' + l.GetRow())
	return string([]byte{file, rank})
}
