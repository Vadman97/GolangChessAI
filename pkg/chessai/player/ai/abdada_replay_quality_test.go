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
	scores := algorithm.ScoreRootMoves(player, parsed.Board, parsed.Previous, 3)
	scoreByMove := map[string]int{}
	for _, score := range scores {
		scoreByMove[analysis.MoveToUCI(score.Move)] = score.Score
	}

	activeKing, ok := scoreByMove["g8f7"]
	if !ok {
		t.Fatal("expected legal active king move g8f7")
	}
	passiveRook, ok := scoreByMove["b8d8"]
	if !ok {
		t.Fatal("expected legal passive rook move b8d8")
	}
	if activeKing <= passiveRook {
		t.Fatalf("expected g8f7 to beat b8d8 in replay rook ending, got g8f7=%d b8d8=%d", activeKing, passiveRook)
	}
}
