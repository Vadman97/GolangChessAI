package ai

import (
	"testing"
	"time"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
)

// uciSquare/uciMove render an engine move in UCI for readable assertions.
// FEN file = 'a' + (7 - col), FEN rank = row + 1.
func uciSquare(l location.Location) string {
	return string(rune('a'+(7-int(l.GetCol())))) + string(rune('1'+int(l.GetRow())))
}
func uciMove(m location.Move) string { return uciSquare(m.Start) + uciSquare(m.End) }

// Position before White's 48th move in lichess game NskVQaIw:
// FEN: 5R2/6pk/1Q2p2p/p4n2/5P2/2N4P/3q2P1/7K w - - 1 48
// White is up a rook. Qxe6 (and several other moves) keep the win; Qc7 — what the
// bot played in the game — abandons control of e1 and allows a forced perpetual
// check (Qe1+/Qg3+) for only a draw. In the game the engine reached depth 5,
// scored Qc7 at +6.2, and had no idea a perpetual was coming because in-search
// repetition was only treated as a draw on the third occurrence. With first-
// recurrence draw detection the perpetual is visible at depth 5.
func loadMove48Position() *board.Board {
	b := &board.Board{}
	// engine row 0 = rank 1, engine col 0 = h-file
	rows := []string{
		"W_K|   |   |   |   |   |   |   ", // rank 1: Kh1
		"   |W_P|   |   |B_Q|   |   |   ", // rank 2: Pg2, qd2
		"W_P|   |   |   |   |W_N|   |   ", // rank 3: Ph3, Nc3
		"   |   |W_P|   |   |   |   |   ", // rank 4: Pf4
		"   |   |B_N|   |   |   |   |B_P", // rank 5: nf5, pa5
		"B_P|   |   |B_P|   |   |W_Q|   ", // rank 6: ph6, pe6, Qb6
		"B_K|B_P|   |   |   |   |   |   ", // rank 7: kh7, pg7
		"   |   |W_R|   |   |   |   |   ", // rank 8: Rf8
	}
	b.LoadBoardFromText(rows)
	b.SetFlag(board.FlagKingMoved, color.White, true)
	b.SetFlag(board.FlagKingMoved, color.Black, true)
	return b
}

const qc7 = "b6c7" // the perpetual-allowing blunder

// TestPerpetualAvoidance: by the depth the real game reached (5), the engine must
// no longer choose Qc7, and must evaluate its choice as winning rather than 0.
func TestPerpetualAvoidance(t *testing.T) {
	for depth := 5; depth <= 7; depth++ {
		b := loadMove48Position()
		player := NewAIPlayer(color.White, NameToAlgorithm[AlgorithmABDADA])
		player.MaxSearchDepth = depth
		player.MaxThinkTime = 60 * time.Second // depth-limited
		sm := player.Algorithm.GetBestMove(player, b, nil)
		assert.NotEqualf(t, qc7, uciMove(sm.Move),
			"depth %d: chose Qc7 which allows a forced perpetual (should win with Qxe6)", depth)
		assert.Truef(t, sm.Score > 200,
			"depth %d: should see a winning score, got %d (treating the win as a draw?)", depth, sm.Score)
	}
}

// TestPerpetualSeenAsDraw: the position AFTER Qc7 is a forced perpetual; the search
// must score it as a draw (~0) rather than as White being up a rook.
func TestPerpetualSeenAsDraw(t *testing.T) {
	b := loadMove48Position()
	board.MakeMove(&location.Move{Start: location.NewLocation(5, 6), End: location.NewLocation(6, 5)}, b) // Qc7
	player := NewAIPlayer(color.Black, NameToAlgorithm[AlgorithmABDADA])
	player.MaxSearchDepth = 6
	player.MaxThinkTime = 60 * time.Second
	sm := player.Algorithm.GetBestMove(player, b, nil)
	assert.Equal(t, 0, sm.Score, "post-Qc7 perpetual should be scored as a draw")
}
