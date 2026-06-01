package game

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	"github.com/stretchr/testify/assert"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TODO(Vadim) test game.go

func TestNewGame(t *testing.T) {
	p1 := ai.NewAIPlayer(color.Black, &ai.MiniMax{})
	p1.MaxSearchDepth = 100
	p1.MaxThinkTime = 500 * time.Millisecond
	p2 := ai.NewAIPlayer(color.White, &ai.Random{})
	p2.MaxSearchDepth = 2
	g := NewGame(p1, p2)
	assert.NotNil(t, g.PerformanceLogger)
	assert.NotNil(t, g.Players)
	assert.NotNil(t, g.TotalMoveTime)
	assert.NotNil(t, g.CurrentMoveTime)
	assert.NotNil(t, g.LastMoveTime)
}

// TestGameStopTerminatesBackgroundGoroutines guards against the cross-game memory
// leak: each NewGame spawns memoryThread + printThread that loop while the game is
// Active. Abandoning a game (as the Lichess server does between back-to-back games)
// must let those goroutines exit, or they pin the Game and its players' caches in
// memory forever. Stop() flips the status so they terminate.
func TestGameStopTerminatesBackgroundGoroutines(t *testing.T) {
	baseline := runtime.NumGoroutine()
	const n = 25
	games := make([]*Game, 0, n)
	for i := 0; i < n; i++ {
		games = append(games, NewGame(
			ai.NewAIPlayer(color.White, &ai.Random{}),
			ai.NewAIPlayer(color.Black, &ai.Random{}),
		))
	}
	assert.True(t, runtime.NumGoroutine() > baseline,
		"expected NewGame to spawn background goroutines")

	for _, g := range games {
		g.Stop()
	}

	// memoryThread sleeps up to 1s before re-checking status; poll with margin.
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > baseline+2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	assert.True(t, runtime.NumGoroutine() <= baseline+2,
		"background goroutines leaked after Stop() — Game would never be GC'd")
}

func TestGameDoesNotDrawOnCumulativeDifferentRepetitions(t *testing.T) {
	g := NewGame(
		ai.NewAIPlayer(color.White, &ai.Random{}),
		ai.NewAIPlayer(color.Black, &ai.Random{}),
	)
	defer g.Stop()

	// After these six plies, three different positions have each repeated once:
	// after White's knight move, after Black's knight move, and after both return.
	// That is not a legal threefold claim; the current position has occurred only
	// once before. The old cumulative PreviousPositionsSeen >= 3 check falsely
	// ended games here, matching the 6kouplD3 false draw claim pattern.
	for _, move := range strings.Split("g1f3 g8f6 f3g1 f6g8 g1f3 g8f6", " ") {
		g.PlayTurnMove(parseTestUCIMove(move))
	}

	assert.Equal(t, Active, g.GameStatus)
	assert.Equal(t, 3, g.CurrentBoard.PreviousPositionsSeen)
	assert.Equal(t, 1, g.CurrentBoard.CurrentPositionRepeats)
}

func parseTestUCIMove(uci string) *location.Move {
	sCol := 7 - (uci[0] - 'a')
	sRow := uci[1] - '0' - 1
	fCol := 7 - (uci[2] - 'a')
	fRow := uci[3] - '0' - 1
	return &location.Move{
		Start: location.NewLocation(sRow, sCol),
		End:   location.NewLocation(fRow, fCol),
	}
}
