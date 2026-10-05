package ai_test

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

func TestQuiesceKeepsCaptureDefendedOnlyByPinnedKnight(t *testing.T) {
	// Bxd4 wins a pawn: Ne6 cannot recapture because Re1 pins it to Ke8.
	// Rxe6 is losing since f7 can recapture. The old SEE scored both captures
	// negatively, so quiescence discarded every capture and returned stand pat.
	position, err := analysis.ParseFEN("4k3/5p2/4n3/8/3p4/2B5/8/4R1K1 w - - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	p := ai.NewAIPlayer(position.Active, &ai.ABDADA{NumThreads: 1})
	standPat := p.EvaluateBoard(position.Board, position.Active).TotalScore
	got := p.Quiesce(position.Board, ai.NegInf, ai.PosInf, position.Active, position.Previous)
	if got <= standPat {
		t.Fatalf("quiescence missed the free pawn: score=%d standPat=%d", got, standPat)
	}
}
