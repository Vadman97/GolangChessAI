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
	aiPlayerSmart.MaxSearchDepth = 40
	aiPlayerSmart.MaxThinkTime = 5 * time.Second
	aiPlayerDumb := ai.NewAIPlayer(color.White, &ai.MiniMax{})
	aiPlayerDumb.MaxSearchDepth = 2
	g := NewGame(aiPlayerDumb, aiPlayerSmart)
	g.MoveLimit = MovesToPlay
	g.TimeLimit = TimeToPlay

	fmt.Println("Before moves:")
	fmt.Println(g.CurrentBoard.Print())
	start := time.Now()
	for i := 0; i < MovesToPlay; i++ {
		active := g.PlayTurn()
		fmt.Println(g.Print())
		util.PrintMemStats()
		if !active {
			break
		}
	}
	fmt.Println("After moves:")
	fmt.Println(g.Print())
	fmt.Printf("Played %d moves in %d ms.\n", g.MovesPlayed, time.Now().Sub(start)/time.Millisecond)

	smartScore := aiPlayerSmart.EvaluateBoard(g.CurrentBoard).TotalScore
	dumbScore := aiPlayerDumb.EvaluateBoard(g.CurrentBoard).TotalScore
	fmt.Printf("Good AI %s Evaluation %d.\n", aiPlayerSmart.Repr(), smartScore)
	fmt.Printf("Bad AI %s Evaluation %d.\n", aiPlayerDumb.Repr(), dumbScore)
	assert.True(t, smartScore > dumbScore)
}
