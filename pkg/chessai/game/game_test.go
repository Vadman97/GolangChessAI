package game

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	"github.com/stretchr/testify/assert"
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
