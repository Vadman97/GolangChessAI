package ai_test

import (
	"sort"
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

func topRootMoves(t *testing.T, fen string, depth int) []ai.RootMoveScore {
	t.Helper()
	parsed, err := analysis.ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	alg := &ai.ABDADA{NumThreads: 1}
	player := ai.NewAIPlayer(parsed.Active, alg)
	scores := alg.ScoreRootMoves(player, parsed.Board, parsed.Previous, depth)
	sort.Slice(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })
	return scores
}

// Game KRDDYqY7: after 1.e4 d5 2.exd5 Nf6 3.Nf3 e6 4.Bb5+ the bot blundered ...Nc6
// (a legal block, but the d5 pawn just takes it). The search must not pick b8c6.
func TestBb5CheckDoesNotHangKnight(t *testing.T) {
	scores := topRootMoves(t, "rnbqkb1r/ppp2ppp/4pn2/1B1P4/8/5N2/PPPP1PPP/RNBQK2R b KQkq - 1 4", 5)
	if best := analysis.MoveToUCI(scores[0].Move); best == "b8c6" {
		t.Fatalf("engine still hangs the knight with Nc6 (b8c6); top move was %s", best)
	}
}

// Game 2ZNkMEJB: after 1.e4 Nf6 2.e5 the knight on f6 is attacked by the e5 pawn. The bot
// played the position-blind preference move ...e6 (leaving the knight hanging, 3.exf6).
// The search ranks the knight-saving moves above ...e6, so once the preference book defers
// to the search (it now must — the move drops material) the chosen move saves the knight.
func TestAlekhineSearchSavesAttackedKnight(t *testing.T) {
	scores := topRootMoves(t, "rnbqkb1r/pppppppp/5n2/4P3/8/8/PPPP1PPP/RNBQKBNR b KQkq - 0 2", 5)
	best := scores[0].Move
	// The f6 knight is on row 5 (rank 6), col 2 (file f). The best move must move it.
	if !(best.Start.GetRow() == 5 && best.Start.GetCol() == 2) {
		t.Fatalf("expected the attacked f6 knight to move; best was %s", analysis.MoveToUCI(best))
	}
}
