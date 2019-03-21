package test

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestBoardAI(t *testing.T) {
	const MovesToPlay = 100
	const TimeToPlay = 60 * time.Second

	aiPlayerSmart := ai.NewAIPlayer(color.Black)
	aiPlayerSmart.Algorithm = ai.AlgorithmMiniMax
	aiPlayerDumb := ai.NewAIPlayer(color.White)
	aiPlayerDumb.Algorithm = ai.AlgorithmAlphaBetaWithMemory
	g := game.NewGame(aiPlayerDumb, aiPlayerSmart)

	fmt.Println("Before moves:")
	fmt.Println(g.CurrentBoard.Print())
	start := time.Now()
	for i := 0; i < MovesToPlay; i++ {
		if i%2 == 0 && time.Now().Sub(start) > TimeToPlay {
			fmt.Printf("Aborting - out of time\n")
			break
		}
		g.PlayTurn()
		fmt.Printf("Move %d\n", g.MovesPlayed)
		fmt.Println(g.CurrentBoard.Print())
		fmt.Printf("White %s has thought for %s\n", g.Players[color.White].Repr(), g.PlayTime[color.White])
		fmt.Printf("Black %s has thought for %s\n", g.Players[color.Black].Repr(), g.PlayTime[color.Black])
		util.PrintMemStats()
	}

	fmt.Println("After moves:")
	fmt.Println(g.CurrentBoard.Print())
	// comment out printing inside loop for accurate timing
	fmt.Printf("Played %d moves in %d ms.\n", g.MovesPlayed, time.Now().Sub(start)/time.Millisecond)

	smartScore := aiPlayerSmart.EvaluateBoard(g.CurrentBoard).TotalScore
	dumbScore := aiPlayerDumb.EvaluateBoard(g.CurrentBoard).TotalScore
	fmt.Printf("Good AI Evaluation %d.\n", smartScore)
	fmt.Printf("Bad AI Evaluation %d.\n", dumbScore)
	assert.True(t, smartScore > dumbScore)
}
