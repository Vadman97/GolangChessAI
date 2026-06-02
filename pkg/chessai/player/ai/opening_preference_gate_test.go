package ai

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gateSquare converts an algebraic square (e.g. "e6") to the engine's mirrored location.
func gateSquare(t *testing.T, sq string) location.Location {
	t.Helper()
	require.Len(t, sq, 2)
	file, rank := sq[0], sq[1]
	require.True(t, file >= 'a' && file <= 'h')
	require.True(t, rank >= '1' && rank <= '8')
	return location.NewLocation(location.CoordinateType(rank-'1'), location.CoordinateType('h'-file))
}

func gateApply(t *testing.T, b *board.Board, prev *board.LastMove, side color.Color, uci string) *board.LastMove {
	t.Helper()
	from, to := gateSquare(t, uci[:2]), gateSquare(t, uci[2:])
	for _, m := range *b.GetAllMoves(side, prev) {
		if m.Start.Equals(from) && m.End.Equals(to) {
			mv := m
			return board.MakeMove(&mv, b)
		}
	}
	t.Fatalf("move %s not legal for %s", uci, color.Names[side])
	return nil
}

// After 1.e4 Nf6 2.e5 the f6 knight is attacked. The position-blind preference list for
// turn 1 is [Ng8-f6 (illegal), e7-e6, c7-c5] — both legal options leave the knight hanging
// (3.exf6). The SEE gate must reject them and defer to search (game 2ZNkMEJB).
func TestEarlyOpeningPreferenceDefersWhenPieceHangs(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()
	prev := gateApply(t, b, nil, color.White, "e2e4")
	prev = gateApply(t, b, prev, color.Black, "g8f6")
	prev = gateApply(t, b, prev, color.White, "e4e5")

	p := NewAIPlayer(color.Black, &ABDADA{NumThreads: 1})
	p.TurnCount = 1

	// The gate must flag the hanging developing move and accept a knight retreat.
	assert.True(t, p.preferenceMoveHangsMaterial(b, location.Move{Start: gateSquare(t, "e7"), End: gateSquare(t, "e6")}),
		"...e6 leaves the f6 knight hanging and must be rejected")
	assert.False(t, p.preferenceMoveHangsMaterial(b, location.Move{Start: gateSquare(t, "f6"), End: gateSquare(t, "d5")}),
		"...Nd5 saves the knight and must be allowed")

	// With every turn-1 preference unsafe, the book defers to the search.
	assert.Nil(t, p.earlyOpeningPreference(b, prev),
		"opening preference must return nil when all preferred moves drop material")
}

// An even recapture (1.e4 d5 2.exd5 ...Qxd5 regains the pawn) is SEE 0 and must still be
// allowed as a turn-0 preference, so the gate does not over-reject sound book moves.
func TestEarlyOpeningPreferenceAllowsEvenExchange(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()
	prev := gateApply(t, b, nil, color.White, "e2e4")

	p := NewAIPlayer(color.Black, &ABDADA{NumThreads: 1})
	p.TurnCount = 0

	assert.False(t, p.preferenceMoveHangsMaterial(b, location.Move{Start: gateSquare(t, "d7"), End: gateSquare(t, "d5")}),
		"...d5 against 1.e4 is an even exchange (exd5 Qxd5) and must be allowed")

	got := p.earlyOpeningPreference(b, prev)
	require.NotNil(t, got)
	assert.Equal(t, gateSquare(t, "d7"), got.Start)
	assert.Equal(t, gateSquare(t, "d5"), got.End)
}
