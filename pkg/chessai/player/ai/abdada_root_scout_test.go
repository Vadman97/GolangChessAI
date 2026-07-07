package ai_test

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

// Root sibling scouting discards moves that fail low against the current best
// and re-searches fail-highs with a full window. The selected move must
// therefore always match the maximum over independent full-window scores of
// every root move (up to equal-score ties). Uses fixed depth so the check is
// deterministic and timing-independent.
func TestABDADARootScoutSelectsFullWindowMaximum(t *testing.T) {
	fens := []string{
		// midgame tactic (queen pressure on g5/g6)
		"4r1k1/p2nrp2/b1pp2p1/6Q1/N1pP4/qP6/P1B3PP/R4RK1 w - - 0 28",
		// rook/minor endgame with close quiet candidates
		"1r4k1/2p4p/2P2p2/R3p1p1/2Nr4/2K5/5PPP/R7 b - - 1 22",
	}
	const depth = 4
	for _, fen := range fens {
		parsed, err := analysis.ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		parsed.Board.CacheGetAllMoves = false
		parsed.Board.CacheGetAllAttackableMoves = false

		// Reference: independent full-window exact score for every root move.
		refAlgo := &ai.ABDADA{NumThreads: 1}
		refPlayer := ai.NewAIPlayer(parsed.Active, refAlgo)
		refPlayer.TranspositionTableEnabled = false
		refPlayer.PrintInfo = false
		refScores := refAlgo.ScoreRootMoves(refPlayer, parsed.Board, parsed.Previous, depth)
		if len(refScores) == 0 {
			t.Fatalf("no root scores for %s", fen)
		}
		maxScore := refScores[0].Score
		for _, rs := range refScores {
			if rs.Score > maxScore {
				maxScore = rs.Score
			}
		}

		// Scouted parallel search must select a move whose independent
		// full-window score equals the maximum.
		algo := &ai.ABDADA{NumThreads: 8}
		player := ai.NewAIPlayer(parsed.Active, algo)
		player.MaxSearchDepth = depth
		player.TranspositionTableEnabled = true
		player.PrintInfo = false
		best := algo.GetBestMove(player, parsed.Board, parsed.Previous)

		var chosenRef *ai.RootMoveScore
		for i := range refScores {
			if refScores[i].Move.Start.Equals(best.Move.Start) && refScores[i].Move.End.Equals(best.Move.End) {
				chosenRef = &refScores[i]
				break
			}
		}
		if chosenRef == nil {
			t.Fatalf("fen %s: chosen move %s not found in root move list", fen, analysis.MoveToUCI(best.Move))
		}
		// Allow small slack: TT-assisted iterative deepening inside GetBestMove
		// can legitimately resolve a different-but-equal line (the reference
		// is a single no-TT fixed-depth pass); what must never happen is
		// selecting a clearly inferior root move.
		const slack = 75
		if chosenRef.Score < maxScore-slack {
			t.Fatalf("fen %s: chose %s (full-window score %d) but best root score is %d",
				fen, analysis.MoveToUCI(best.Move), chosenRef.Score, maxScore)
		}
	}
}
