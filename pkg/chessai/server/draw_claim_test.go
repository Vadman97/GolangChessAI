package server

import (
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/game"
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

	// Knight shuffle from the start position: after 6 plies the start position has
	// recurred enough times that PreviousPositionsSeen >= 3 (threefold). The last
	// move is Black's, leaving White (the bot) to move.
	moves := "g1f3 g8f6 f3g1 f6g8 g1f3 g8f6"
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

	// The board really is a threefold repetition.
	assert.Equal(t, game.RepeatedActionThreeTimeDraw, l.Game.GameStatus,
		"setup sanity: position should be a threefold repetition")

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
