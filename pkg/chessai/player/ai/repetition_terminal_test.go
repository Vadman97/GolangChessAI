package ai_test

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

func TestEvaluateBoardMateNotMaskedByPriorRepetitionHistory(t *testing.T) {
	parsed, err := analysis.ParseFEN("r1b5/p1R4Q/2p4k/6p1/3P4/2P5/P1P2PpP/R5K1 b - - 2 28")
	if err != nil {
		t.Fatal(err)
	}
	parsed.Board.PreviousPositionsSeen = 3
	parsed.Board.CurrentPositionRepeats = 0

	player := ai.NewAIPlayer(color.Black, &ai.Random{})
	score := player.EvaluateBoard(parsed.Board, color.Black).TotalScore
	if score > ai.LossScore {
		t.Fatalf("expected checkmate to remain a loss despite prior repetition history, got %d", score)
	}
}

func TestABDADAFinalMateLineNotScoredAsDrawAfterPriorRepetitionHistory(t *testing.T) {
	parsed, err := analysis.ParseFEN("r1b5/p1R2Qk1/2p5/6p1/3P4/2P5/P1P2PpP/R5K1 b - - 0 27")
	if err != nil {
		t.Fatal(err)
	}
	parsed.Board.PreviousPositionsSeen = 3
	parsed.Board.CurrentPositionRepeats = 0

	algorithm := &ai.ABDADA{NumThreads: 1}
	player := ai.NewAIPlayer(color.Black, algorithm)
	scores := algorithm.ScoreRootMoves(player, parsed.Board, parsed.Previous, 3)
	for _, score := range scores {
		if analysis.MoveToUCI(score.Move) == "g7h6" {
			if score.Score > ai.LossScore {
				t.Fatalf("expected g7h6 to score as forced mate loss, got %d", score.Score)
			}
			return
		}
	}
	t.Fatal("expected legal move g7h6")
}
