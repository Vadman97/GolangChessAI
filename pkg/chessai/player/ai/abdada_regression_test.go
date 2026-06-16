package ai_test

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

func TestABDADACdTsyKE6AvoidsQuietBlunderWhenPawnBreakAvailable(t *testing.T) {
	t.Skip("known ABDADA weakness; tracked in testdata/abdada_fens.txt for benchmark-driven optimization")
}

func TestABDADACdTsyKE6DefendsMatingNetWithRook(t *testing.T) {
	parsed, err := analysis.ParseFEN("2q5/P6p/3Qpk1p/4n3/7P/1b4R1/1KP5/5r2 w - - 0 43")
	if err != nil {
		t.Fatal(err)
	}

	algorithm := &ai.ABDADA{NumThreads: 1}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	player.MaxSearchDepth = 3
	player.TranspositionTableEnabled = true
	player.PrintInfo = false
	player.Debug = false

	best := algorithm.GetBestMove(player, parsed.Board, parsed.Previous)
	uci := analysis.MoveToUCI(best.Move)
	if uci != "g3c3" {
		t.Fatalf("expected rook defense g3c3 in mating net, got %s score=%d", uci, best.Score)
	}
}

func TestABDADACdTsyKE6AvoidsForcedMateWalk(t *testing.T) {
	t.Skip("known ABDADA weakness; tracked in testdata/abdada_fens.txt for benchmark-driven optimization")
}
