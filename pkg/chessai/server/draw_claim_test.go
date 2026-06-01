package server

import (
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/game"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	"github.com/stretchr/testify/assert"
)

// recordingClient captures every request URL and returns 200 {"ok":true}.
type recordingClient struct{ urls []string }

func (c *recordingClient) Do(req *http.Request) (*http.Response, error) {
	c.urls = append(c.urls, req.URL.String())
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"ok":true}`)),
		Header:     make(http.Header),
	}, nil
}

func randomAI(c color.Color) *ai.AIPlayer {
	return ai.NewAIPlayer(c, &ai.Random{Rand: rand.New(rand.NewSource(1))})
}

func TestThinkTimeForClockHonorsConfiguredCap(t *testing.T) {
	assert.Equal(t, 3*time.Second, thinkTimeForClock(180*time.Second, 0, 15))
}

// TestClaimsDrawOnOpponentRepetition reproduces the NskVQaIw failure: the opponent's
// move completes a threefold repetition, so the local engine flips to a draw status
// while it is our turn. The bot must NOT go idle (which flagged it on time in the real
// game) — it must play on and post its move with offeringDraw=true to claim the draw.
func TestClaimsDrawOnOpponentRepetition(t *testing.T) {
	rec := &recordingClient{}
	base, _ := url.Parse("http://test.local")
	l := &Lichess{
		Client: &Client{BaseURL: base, APIKey: "x", HttpClient: rec},
		GameID: "TESTID",
		Player: randomAI(color.White),
		Game:   game.NewGame(randomAI(color.White), randomAI(color.Black)),
	}
	defer l.Game.Stop()

	// Knight shuffle from the start position: after 10 plies, the position with
	// both knights developed has occurred for the third time. The last move is
	// Black's, leaving White (the bot) to move.
	moves := "g1f3 g8f6 f3g1 f6g8 g1f3 g8f6 f3g1 f6g8 g1f3 g8f6"
	l.Mutex.Lock()
	err := l.handleBoardUpdateLocked(&GameEvent{
		Type:        StateTypeGame,
		Moves:       moves,
		Status:      "started",
		WhiteTimeMS: 30000,
		BlackTimeMS: 30000,
	})
	l.Mutex.Unlock()
	assert.NoError(t, err)

	// The bot must have posted a move (not gone idle) and offered the draw.
	var moveURL string
	for _, u := range rec.urls {
		if strings.Contains(u, "/move/") {
			moveURL = u
		}
	}
	assert.NotEmpty(t, moveURL, "bot went idle on a claimable draw instead of playing on (would flag on time)")
	assert.Contains(t, moveURL, "offeringDraw=true", "bot moved but failed to claim/offer the draw")
}

// TestPlaysOnAfterOwnMoveLeftClaimableDraw reproduces the DpqEDBdP loss: our own
// move completes a threefold, leaving the local game in a claimable-draw status, and
// then the opponent moves AGAIN (Lichess plays on through a claimable draw). The
// opponent's slot is a HumanPlayer — exactly as in production — so if the opponent's
// move is dropped (PlayTurnMove no-ops on a non-Active status) the board desyncs,
// it stays the opponent's turn locally, and PlayTurn blocks forever in WaitForMove
// while our clock runs out. The bot must instead apply the move and respond.
func TestPlaysOnAfterOwnMoveLeftClaimableDraw(t *testing.T) {
	rec := &recordingClient{}
	base, _ := url.Parse("http://test.local")
	l := &Lichess{
		Client: &Client{BaseURL: base, APIKey: "x", HttpClient: rec},
		GameID: "TESTID",
		Player: randomAI(color.White),
		// Opponent is a HumanPlayer, like production — a desync makes PlayTurn block
		// in WaitForMove (an AIPlayer opponent would just search and hide the bug).
		Game: game.NewGame(randomAI(color.White), player.NewHumanPlayer(color.Black)),
	}
	defer l.Game.Stop()

	// Drive a knight shuffle until our (White) move completes a threefold, leaving
	// Black (the opponent) to move. Force the status Active before each ply to mimic
	// reaching this state through PlayTurn, which reactivates a claimable draw before
	// playing on (PlayTurnMove alone would start dropping moves at the threefold).
	setup := strings.Split("g1f3 g8f6 f3g1 f6g8 g1f3 g8f6 f3g1 f6g8 g1f3", " ")
	for _, m := range setup {
		l.Game.GameStatus = game.Active
		l.Game.PlayTurnMove(parseUCIMove(m))
	}
	l.movesApplied = len(setup)
	assert.Equal(t, game.RepeatedActionThreeTimeDraw, l.Game.GameStatus,
		"setup: our move should have left the local game in a claimable draw")
	assert.Equal(t, color.Black, l.Game.CurrentTurnColor,
		"setup: it should be the opponent's turn after our move")

	// Opponent plays on through the claimable draw (knight back home). Lichess keeps
	// the game going and now it is our turn again.
	event := &GameEvent{
		Type:        StateTypeGame,
		Moves:       "g1f3 g8f6 f3g1 f6g8 g1f3 g8f6 f3g1 f6g8 g1f3 g8f6",
		Status:      "started",
		WhiteTimeMS: 30000,
		BlackTimeMS: 30000,
	}
	done := make(chan error, 1)
	go func() {
		l.Mutex.Lock()
		defer l.Mutex.Unlock()
		done <- l.handleBoardUpdateLocked(event)
	}()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("bot hung after opponent played on through a claimable draw (blocked in WaitForMove, would lose on time)")
	}

	var moveURL string
	for _, u := range rec.urls {
		if strings.Contains(u, "/move/") {
			moveURL = u
		}
	}
	assert.NotEmpty(t, moveURL, "bot went idle instead of replying to the opponent's move")
}

// TestMakeMoveOffersDrawForClaimableStatuses locks in that MakeMove offers a draw for
// both claimable draw statuses and not otherwise.
func TestMakeMoveOffersDrawForClaimableStatuses(t *testing.T) {
	cases := []struct {
		status     byte
		wantOffer  bool
		statusName string
	}{
		{game.RepeatedActionThreeTimeDraw, true, "threefold"},
		{game.FiftyMoveDraw, true, "fifty-move"},
		{game.Active, false, "active"},
		{game.BlackWin, false, "loss"},
	}
	for _, tc := range cases {
		rec := &recordingClient{}
		base, _ := url.Parse("http://test.local")
		l := &Lichess{
			Client: &Client{BaseURL: base, APIKey: "x", HttpClient: rec},
			GameID: "TESTID",
			Player: randomAI(color.White),
			Game:   game.NewGame(randomAI(color.White), randomAI(color.Black)),
		}
		// Make a legal opening move (while Active) so PreviousMove is set, then force
		// the status under test before posting.
		l.Game.PlayTurnMove(parseUCIMove("g1f3"))
		l.Game.GameStatus = tc.status
		err := l.MakeMove(l.GameID, l.Game.PreviousMove)
		assert.NoError(t, err)
		offered := strings.Contains(rec.urls[len(rec.urls)-1], "offeringDraw=true")
		assert.Equalf(t, tc.wantOffer, offered, "status %s: offeringDraw mismatch", tc.statusName)
		l.Game.Stop()
	}
}
