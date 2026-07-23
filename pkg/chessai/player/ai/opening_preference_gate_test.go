package ai

import (
	"testing"
	"time"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fixedOpeningRegressionAlgorithm struct {
	move location.Move
}

func (a *fixedOpeningRegressionAlgorithm) GetName() string {
	return "fixed opening regression"
}

func (a *fixedOpeningRegressionAlgorithm) GetBestMove(_ *AIPlayer, _ *board.Board, _ *board.LastMove) *ScoredMove {
	return &ScoredMove{Move: a.move}
}

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

// Reproduces game DW9hIjlh: after 1.d4 Nf6 2.Nf3 g6 3.Bf4 d6 4.e3 Nh5 5.Bd3 Nxf4, the
// London System book's next move (6.O-O) is legal and its own destination (g1) is safe,
// so the destination-safety gate doesn't catch it. But the book has no way to notice that
// the opponent's last move (Nxf4) just captured our bishop on f4: exf4 recaptures the
// knight for an even trade, while O-O abandons the piece for nothing. The bot played O-O
// live and lost a clean bishop.
func TestOpeningBookDefersWhenBetterCaptureAvailable(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()
	var prev *board.LastMove
	prev = gateApply(t, b, prev, color.White, "d2d4")
	prev = gateApply(t, b, prev, color.Black, "g8f6")
	prev = gateApply(t, b, prev, color.White, "g1f3")
	prev = gateApply(t, b, prev, color.Black, "g7g6")
	prev = gateApply(t, b, prev, color.White, "c1f4")
	prev = gateApply(t, b, prev, color.Black, "d7d6")
	prev = gateApply(t, b, prev, color.White, "e2e3")
	prev = gateApply(t, b, prev, color.Black, "f6h5")
	prev = gateApply(t, b, prev, color.White, "f1d3")
	prev = gateApply(t, b, prev, color.Black, "h5f4")

	p := NewAIPlayer(color.White, &ABDADA{NumThreads: 1})
	p.MaxSearchDepth = 100
	p.MaxThinkTime = 500 * time.Millisecond
	p.Opening = 1
	p.TurnCount = 5

	bookMove := OpeningMoves[color.White][1][5]
	require.Equal(t, gateSquare(t, "e1"), bookMove.Start, "guard: book move 5 should be O-O")
	require.Equal(t, gateSquare(t, "g1"), bookMove.End, "guard: book move 5 should be O-O")

	assert.True(t, p.bookMoveMissesFreeCapture(b, prev, *bookMove),
		"exf4 recaptures the knight for free; the book must recognize O-O leaves it hanging")

	move := p.GetBestMove(b, prev, nil)
	assert.False(t, move.Start.Equals(bookMove.Start) && move.End.Equals(bookMove.End),
		"bot castled instead of recapturing on f4, giving away the bishop for nothing (game DW9hIjlh)")
	assert.Equal(t, OpeningNone, p.Opening, "book must be abandoned once a better capture is found")
}

// Game rtA24XeS reached this position after 1.e4 c5 2.Nf3 Nc6 3.Bc4 e6
// 4.c3 d5. The fixed Italian script then forced 5.d3, ignoring Black's central
// break; 5...dxc4 left White roughly five pawns worse. Normal search finds exd5.
func TestPositionBlindBookStopsBeforeOpponentCentralBreak(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()
	var prev *board.LastMove
	prev = gateApply(t, b, prev, color.White, "e2e4")
	prev = gateApply(t, b, prev, color.Black, "c7c5")
	prev = gateApply(t, b, prev, color.White, "g1f3")
	prev = gateApply(t, b, prev, color.Black, "b8c6")
	prev = gateApply(t, b, prev, color.White, "f1c4")
	prev = gateApply(t, b, prev, color.Black, "e7e6")
	prev = gateApply(t, b, prev, color.White, "c2c3")
	prev = gateApply(t, b, prev, color.Black, "d7d5")

	exd5 := location.Move{Start: gateSquare(t, "e4"), End: gateSquare(t, "d5")}
	p := NewAIPlayer(color.White, &fixedOpeningRegressionAlgorithm{move: exd5})
	p.Opening = 0
	p.TurnCount = 4

	bookMove := OpeningMoves[color.White][0][4]
	require.Equal(t, gateSquare(t, "d2"), bookMove.Start, "guard: scripted move should be d3")
	require.Equal(t, gateSquare(t, "d3"), bookMove.End, "guard: scripted move should be d3")

	got := p.GetBestMove(b, prev, nil)
	assert.Equal(t, exd5.Start, got.Start)
	assert.Equal(t, exd5.End, got.End)
	assert.Equal(t, OpeningNone, p.Opening, "position-blind script must stop before move three")
}
