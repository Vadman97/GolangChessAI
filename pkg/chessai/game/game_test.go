package game

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	"github.com/stretchr/testify/assert"
	"runtime"
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
