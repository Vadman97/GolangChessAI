package ai

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/stretchr/testify/assert"
)

// TestBishopPairValuedInMiddlegame guards against surrendering the bishop pair
// "for free" in the opening/middlegame. Regression from lichess game FSIJKLxn,
// where the engine played an early Bxc6 trading bishop for knight because the
// bishop-pair bonus was tapered to ~6cp with a full board of pawns. With a base
// bonus, keeping the pair must be worth a meaningful amount even with many pawns.
func TestBishopPairValuedInMiddlegame(t *testing.T) {
	// White keeps both bishops; Black has traded one bishop for a knight (so Black
	// has knight+bishop, White has the pair). Material is otherwise identical.
	// Many pawns on the board (middlegame), which is exactly where the old taper
	// zeroed the bonus out.
	withPair := &board.Board{}
	withPair.LoadBoardFromText([]string{
		"W_R|W_N|W_B|W_Q|W_K|W_B|   |W_R ", // White: both bishops c1,f1
		"W_P|W_P|W_P|W_P|W_P|W_P|W_P|W_P ",
		"   |   |   |   |   |W_N|   |   ", // White knight developed to f3
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |B_N|   |   ", // Black knight developed
		"B_P|B_P|B_P|B_P|B_P|B_P|B_P|B_P ",
		"B_R|B_N|B_B|B_Q|B_K|   |   |B_R ", // Black: only one bishop (c8); f8 empty
	})

	noPair := &board.Board{}
	noPair.LoadBoardFromText([]string{
		"W_R|W_N|W_B|W_Q|W_K|   |   |W_R ", // White: only one bishop (c1); f1 empty
		"W_P|W_P|W_P|W_P|W_P|W_P|W_P|W_P ",
		"   |   |   |   |   |W_N|   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |B_N|   |   ",
		"B_P|B_P|B_P|B_P|B_P|B_P|B_P|B_P ",
		"B_R|B_N|B_B|B_Q|B_K|   |   |B_R ",
	})

	evWith := EvaluateBoardNoCache(withPair, color.White).TotalScore
	evNo := EvaluateBoardNoCache(noPair, color.White).TotalScore
	// evWith includes +300 bishop material AND the pair bonus; evNo lacks both.
	// The pair bonus alone is (evWith - evNo) - 300 (material) - any PST for f1 bishop.
	// We only need a coarse lower bound: keeping the pair must be worth clearly more
	// than the old ~6cp. Subtract a generous 320 for the extra bishop's material+PST.
	pairWorth := (evWith - evNo) - 320
	assert.Truef(t, pairWorth > 15,
		"bishop pair should be worth >15cp in the middlegame, got ~%d", pairWorth)
}

// TestAdvancedKnightNotOutpostWhenKickable guards the corrected outpost detection.
// A knight on e4 with the enemy f-pawn still on its home square is trivially
// kicked by ...f5, so it must NOT count as an outpost. The old detection only
// looked at the single square an enemy pawn attacked from right now, so it
// wrongly rewarded such unstable sorties (lichess game FSIJKLxn, 3.Ne4).
func TestAdvancedKnightNotOutpostWhenKickable(t *testing.T) {
	// Engine columns: col0=h .. col7=a, so e-file=col3, and its adjacent files are
	// f=col2 and d=col4. White knight on e4 = (row 3, col 3). Black f-pawn on f7
	// = (row 6, col 2): it can play ...f5 to attack e4, so e4 is NOT an outpost.
	b := &board.Board{}
	b.LoadBoardFromText([]string{
		"   |   |   |   |W_K|   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |W_N|   |   |   ", // row 3: white knight on e4 (col3)
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |B_P|   |   |   |   |   ", // row 6: black f-pawn on f7 (col2)
		"   |   |   |   |B_K|   |   |   ",
	})
	assert.Falsef(t, isKnightOutpost(b, 3, 3, color.White),
		"knight on e4 with enemy f-pawn on f7 is kickable by ...f5 and must not be an outpost")

	// Sanity: a knight on e5 = (row 4, col 3) with NO enemy pawn on the d/f files
	// (col4/col2) anywhere ahead IS a genuine outpost.
	b2 := &board.Board{}
	b2.LoadBoardFromText([]string{
		"   |   |   |   |W_K|   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |W_N|   |   |   ", // row 4: white knight on e5 (col3)
		"   |   |   |   |   |   |   |   ",
		"B_P|B_P|   |   |   |B_P|   |B_P", // row 6: black pawns on h,g,c,a (cols 0,1,5,7)
		"   |   |   |   |B_K|   |   |   ",
	})
	assert.Truef(t, isKnightOutpost(b2, 4, 3, color.White),
		"knight on e5 with no enemy d/f pawns ahead should be a real outpost")
}
