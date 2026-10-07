package analysis

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/game"
)

// Keep the complete-game evidence replayable, including promotions and draw
// history. A capped active game must never silently become a scored draw.
func TestRecordedStrengthGamesHaveLegalTerminalResults(t *testing.T) {
	for _, budget := range []string{"200ms", "400ms"} {
		t.Run(budget, func(t *testing.T) { testRecordedStrengthGames(t, budget) })
	}
}

func testRecordedStrengthGames(t *testing.T, budget string) {
	data, err := os.ReadFile("../../../testdata/strength_2026_10_05_games_" + budget + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		PlyCap int `json:"plyCap"`
		Games  []struct {
			Prefix       string   `json:"prefix"`
			ImprovedSide int      `json:"improvedSide"`
			Status       string   `json:"status"`
			Winner       *int     `json:"winner"`
			Score        *float64 `json:"score"`
			Moves        string   `json:"moves"`
		} `json:"games"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Games) != 20 {
		t.Fatalf("expected 20 paired games, got %d", len(report.Games))
	}
	pairs := make(map[string]byte)
	for i, recorded := range report.Games {
		if recorded.ImprovedSide < 0 || recorded.ImprovedSide > 1 || (recorded.Winner != nil && (*recorded.Winner < 0 || *recorded.Winner > 1)) {
			t.Fatalf("game %d: invalid side", i+1)
		}
		mask := byte(1 << recorded.ImprovedSide)
		if pairs[recorded.Prefix]&mask != 0 {
			t.Fatalf("game %d: duplicate opening/color combination", i+1)
		}
		pairs[recorded.Prefix] |= mask
		if !strings.HasPrefix(recorded.Moves, recorded.Prefix+" ") {
			t.Fatalf("game %d: opening prefix differs", i+1)
		}
		states, err := ReplayUCIMoves(recorded.Moves)
		if err != nil {
			t.Fatalf("game %d: %v", i+1, err)
		}
		for _, state := range states[:len(states)-1] {
			if state.Status != game.Active {
				t.Fatalf("game %d continued after terminal ply %d", i+1, state.Ply)
			}
		}
		last := states[len(states)-1]
		if recorded.Status == "ply-cap" {
			if last.Ply != report.PlyCap || last.Status != game.Active || recorded.Score != nil || recorded.Winner != nil {
				t.Fatalf("game %d: capped game must remain active and unscored", i+1)
			}
			continue
		}
		if recorded.Score == nil {
			t.Fatalf("game %d: completed game is unscored", i+1)
		}
		wantScore := 0.5
		if recorded.Status == "mate" {
			if recorded.Winner == nil {
				t.Fatalf("game %d: missing mate winner", i+1)
			}
			wantStatus := byte(game.WhiteWin)
			if *recorded.Winner == 1 {
				wantStatus = game.BlackWin
			}
			if last.Status != wantStatus || last.LegalMoveCount != 0 {
				t.Fatalf("game %d: recorded winner differs from replay", i+1)
			}
			wantScore = 0
			if *recorded.Winner == recorded.ImprovedSide {
				wantScore = 1
			}
		} else {
			draws := map[string]byte{"material": game.InsufficientMaterialDraw, "repetition": game.RepeatedActionThreeTimeDraw, "fifty-move": game.FiftyMoveDraw, "stalemate": game.Stalemate}
			wantStatus, ok := draws[recorded.Status]
			if !ok || last.Status != wantStatus || recorded.Winner != nil {
				t.Fatalf("game %d: unrecognized or mismatched draw", i+1)
			}
		}
		if *recorded.Score != wantScore {
			t.Fatalf("game %d: result scored for the wrong side", i+1)
		}
	}
	if len(pairs) != 10 {
		t.Fatalf("expected ten opening pairs, got %d", len(pairs))
	}
	for prefix, mask := range pairs {
		if mask != 3 {
			t.Fatalf("opening %s was not played with both colors", prefix)
		}
	}
}
