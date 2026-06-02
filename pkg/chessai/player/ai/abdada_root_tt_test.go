package ai_test

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

func TestABDADAParallelTTRootScoresAreExactInPromotionMateRace(t *testing.T) {
	parsed, err := analysis.ParseFEN("8/5p1R/B3pk2/8/2p1n1P1/4P3/1p1r1P2/1K6 w - - 1 41")
	if err != nil {
		t.Fatal(err)
	}
	parsed.Board.CacheGetAllMoves = false
	parsed.Board.CacheGetAllAttackableMoves = false

	algorithm := &ai.ABDADA{NumThreads: 8}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	player.MaxSearchDepth = 8
	player.TranspositionTableEnabled = true
	player.PrintInfo = false
	player.Debug = false

	best := algorithm.GetBestMove(player, parsed.Board, parsed.Previous)
	uci := analysis.MoveToUCI(best.Move)
	if uci == "h7f7" || uci == "h7h6" {
		t.Fatalf("ABDADA+TT chose mate-losing root move %s with score %d", uci, best.Score)
	}
}
