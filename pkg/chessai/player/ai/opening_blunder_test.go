package ai_test

import (
	"sort"
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

// Game KRDDYqY7: after 1.e4 d5 2.exd5 Nf6 3.Nf3 e6 4.Bb5+ the engine blundered
// ...Nc6 (legal block, but d5 pawn takes it). The search must not pick b8c6.
func TestBb5CheckDoesNotHangKnight(t *testing.T) {
	parsed, err := analysis.ParseFEN("rnbqkb1r/ppp2ppp/4pn2/1B1P4/8/5N2/PPPP1PPP/RNBQK2R b KQkq - 1 4")
	if err != nil {
		t.Fatal(err)
	}
	alg := &ai.ABDADA{NumThreads: 1}
	player := ai.NewAIPlayer(parsed.Active, alg)
	scores := alg.ScoreRootMoves(player, parsed.Board, parsed.Previous, 5)
	sort.Slice(scores, func(i, j int) bool { return scores[i].Score > scores[j].Score })
	t.Logf("top moves:")
	for i, s := range scores {
		if i >= 6 {
			break
		}
		t.Logf("  %s = %d", analysis.MoveToUCI(s.Move), s.Score)
	}
	best := analysis.MoveToUCI(scores[0].Move)
	if best == "b8c6" {
		t.Fatalf("engine still hangs the knight with Nc6 (b8c6)")
	}
}
