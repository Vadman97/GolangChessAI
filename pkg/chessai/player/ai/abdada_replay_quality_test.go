package ai_test

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

func TestABDADAGameReplayPly44ActivatesKingAgainstPasser(t *testing.T) {
	parsed, err := analysis.ParseFEN("1r4k1/2p4p/2P2p2/R3p1p1/2Nr4/2K5/5PPP/R7 b - - 1 22")
	if err != nil {
		t.Fatal(err)
	}

	algorithm := &ai.ABDADA{NumThreads: 1}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	player.MaxSearchDepth = 3

	best := algorithm.GetBestMove(player, parsed.Board, parsed.Previous)
	uci := analysis.MoveToUCI(best.Move)
	switch uci {
	case "g8g7", "g8f7", "d4d8":
	default:
		t.Fatalf("expected active king or immediate rook blockade, got %s score=%d", uci, best.Score)
	}
}

func TestABDADASeesQueenMinorAttackAfterBxg6(t *testing.T) {
	parsed, err := analysis.ParseFEN("4r1k1/p2nrp2/b1pp2p1/3p2Q1/N1Pp4/qP2P3/P1B3PP/R4RK1 w - - 0 27")
	if err != nil {
		t.Fatal(err)
	}

	algorithm := &ai.ABDADA{NumThreads: 1}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	scores := algorithm.ScoreRootMoves(player, parsed.Board, parsed.Previous, 1)
	scoreByMove := map[string]int{}
	for _, score := range scores {
		scoreByMove[analysis.MoveToUCI(score.Move)] = score.Score
	}

	attack, ok := scoreByMove["c2g6"]
	if !ok {
		t.Fatal("expected legal attacking move c2g6")
	}
	material, ok := scoreByMove["e3d4"]
	if !ok {
		t.Fatal("expected legal material move e3d4")
	}
	if attack <= material {
		t.Fatalf("expected c2g6 attack to beat e3d4 material grab, got c2g6=%d e3d4=%d", attack, material)
	}
}
