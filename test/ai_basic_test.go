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
	const MovesToPlay = 40
	const TimeToPlay = 60 * time.Second

	aiPlayerSmart := ai.NewAIPlayer(color.Black)
	aiPlayerSmart.Algorithm = ai.AlgorithmMiniMax
	aiPlayerSmart.Depth = 4
	aiPlayerDumb := ai.NewAIPlayer(color.White)
	aiPlayerDumb.Algorithm = ai.AlgorithmAlphaBetaWithMemory
	aiPlayerDumb.TranspositionTableEnabled = true
	aiPlayerDumb.Depth = 4
	g := game.NewGame(aiPlayerDumb, aiPlayerSmart)

	fmt.Println("Before moves:")
	fmt.Println(g.CurrentBoard.Print())
	start := time.Now()
	for i := 0; i < MovesToPlay; i++ {
		if i%2 == 0 && time.Now().Sub(start) > TimeToPlay {
			fmt.Printf("Aborting - out of time\n")
			break
		}
		fmt.Printf("\nPlayer %s thinking...\n", g.Players[g.CurrentTurnColor].Repr())
		g.PlayTurn()
		fmt.Printf("Move %d\n", g.MovesPlayed)
		fmt.Println(g.CurrentBoard.Print())
		fmt.Println(g.Print())
		util.PrintMemStats()
		if g.GameStatus != game.Active {
			fmt.Printf("Game Over! Result is: %s\n", game.StatusStrings[g.GameStatus])
			break
		}
	}

	fmt.Println("After moves:")
	fmt.Println(g.CurrentBoard.Print())
	fmt.Println(g.Print())
	// comment out printing inside loop for accurate timing
	fmt.Printf("Played %d moves in %d ms.\n", g.MovesPlayed, time.Now().Sub(start)/time.Millisecond)

	smartScore := aiPlayerSmart.EvaluateBoard(g.CurrentBoard).TotalScore
	dumbScore := aiPlayerDumb.EvaluateBoard(g.CurrentBoard).TotalScore
	fmt.Printf("Good AI %s Evaluation %d.\n", aiPlayerSmart.Repr(), smartScore)
	fmt.Printf("Bad AI %s Evaluation %d.\n", aiPlayerDumb.Repr(), dumbScore)
	assert.True(t, smartScore > dumbScore)
}
