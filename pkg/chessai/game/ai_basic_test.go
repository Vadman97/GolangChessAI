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
	const TimeToPlay = 2 * time.Minute

	rand.Seed(config.Get().TestRandSeed)
	aiPlayerSmart := ai.NewAIPlayer(color.Black, &ai.MTDf{})
	aiPlayerSmart.MaxSearchDepth = 100
	aiPlayerSmart.MaxThinkTime = 1 * time.Second
	aiPlayerDumb := ai.NewAIPlayer(color.White, &ai.Random{})
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
	smartScore := aiPlayerSmart.EvaluateBoard(g.CurrentBoard).TotalScore
	dumbScore := aiPlayerDumb.EvaluateBoard(g.CurrentBoard).TotalScore
	fmt.Printf("Good AI %s Evaluation %d.\n", aiPlayerSmart.Repr(), smartScore)
	fmt.Printf("Bad AI %s Evaluation %d.\n", aiPlayerDumb.Repr(), dumbScore)
	assert.True(t, smartScore > dumbScore)
}
