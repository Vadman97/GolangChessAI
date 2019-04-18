package game

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestBoardAI(t *testing.T) {
	const MovesToPlay = 100
	const TimeToPlay = 60 * time.Minute

	rand.Seed(config.Get().TestRandSeed)
	aiPlayerSmart := ai.NewAIPlayer(color.Black, &ai.ABDADA{})
	aiPlayerSmart.MaxSearchDepth = 3
	aiPlayerSmart.MaxThinkTime = 1 * time.Second
	aiPlayerDumb := ai.NewAIPlayer(color.White, &ai.Random{
		Rand: rand.New(rand.NewSource(config.Get().TestRandSeed)),
	})
	aiPlayerDumb.MaxSearchDepth = 2
	g := NewGame(aiPlayerDumb, aiPlayerSmart)
	g.MoveLimit = MovesToPlay
	g.TimeLimit = TimeToPlay

	for i := 0; i < MovesToPlay; i++ {
		active := g.PlayTurn()
		fmt.Printf(util.GetMemStatString())
		if !active {
			break
		}
	}
	smartScore := aiPlayerSmart.EvaluateBoard(g.CurrentBoard, aiPlayerSmart.PlayerColor).TotalScore
	dumbScore := aiPlayerDumb.EvaluateBoard(g.CurrentBoard, aiPlayerDumb.PlayerColor).TotalScore
	fmt.Printf("Good AI %s Evaluation %d.\n", aiPlayerSmart, smartScore)
	fmt.Printf("Bad AI %s Evaluation %d.\n", aiPlayerDumb, dumbScore)
	assert.True(t, smartScore > dumbScore)
}

// ABDADA vs. Random, diff depths
// Depth:             1,  2,  3,   4
// Moves (ply):      71, 43, 17,  61
// Total game turns: 36, 22,  9,  31
// Time (sec):        1,  3, 38, 352
