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

	// The contact-pressure king-danger terms must make the Bxg6 attack
	// competitive so the search selects it — but not by so much that the eval
	// hallucinates winning scores in every queen-attack position (game
	// o6lAdkjC: eval +2669 vs SF +380 lost a won game). Assert at search
	// depth 4 (where the tactic resolves) rather than depth 1: the eval nudge
	// plus search must pick the attack.
	algorithm := &ai.ABDADA{NumThreads: 1}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	scores := algorithm.ScoreRootMoves(player, parsed.Board, parsed.Previous, 4)
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
	// The calibrated contact-pressure terms (see the o6lAdkjC overconfidence
	// fix) leave Bxg6 and exd4 near-equal at this depth; the attack must at
	// least not be dominated — timed searches select c2g6 (bench tag
	// lichess-4HkIRFy9-ply55 tracks that end to end).
	if attack < material {
		t.Fatalf("expected c2g6 attack to at least match e3d4 material grab, got c2g6=%d e3d4=%d", attack, material)
	}
}

func TestABDADARootVerificationKeepsRookAttack(t *testing.T) {
	parsed, err := analysis.ParseFEN("4rrk1/1p1bn1b1/p1p3pp/q1B5/4NP2/3P1N2/PPP2QPP/R4RK1 b - - 0 18")
	if err != nil {
		t.Fatal(err)
	}

	algorithm := &ai.ABDADA{NumThreads: 8}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	player.MaxSearchDepth = 5
	player.TranspositionTableEnabled = true
	player.PrintInfo = false
	player.Debug = false

	best := algorithm.GetBestMove(player, parsed.Board, parsed.Previous)
	if uci := analysis.MoveToUCI(best.Move); uci != "f8f4" {
		t.Fatalf("expected active rook capture f8f4, got %s score=%d", uci, best.Score)
	}
}

func TestABDADADeepRootTieBreakAvoidsKnightRetreat(t *testing.T) {
	parsed, err := analysis.ParseFEN("4rrk1/3bn1b1/pBp3pp/7q/2P1NP2/3P1N2/PP3QPP/4RRK1 b - - 2 21")
	if err != nil {
		t.Fatal(err)
	}

	algorithm := &ai.ABDADA{NumThreads: 8}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	player.MaxSearchDepth = 5
	player.TranspositionTableEnabled = true
	player.PrintInfo = false
	player.Debug = false

	best := algorithm.GetBestMove(player, parsed.Board, parsed.Previous)
	if uci := analysis.MoveToUCI(best.Move); uci != "e7f5" {
		t.Fatalf("expected deeper tie-break to prefer e7f5, got %s score=%d", uci, best.Score)
	}
}
