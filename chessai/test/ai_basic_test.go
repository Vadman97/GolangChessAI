package test

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/player/ai"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestAI(t *testing.T) {
	t.Skip()
	const MovesToPlay = 100
	myBoard := board.Board{}
	myBoard.ResetDefault()
	fmt.Println("Before moves")
	fmt.Println(myBoard.Print())

	aiPlayerSmart := ai.NewAIPlayer(color.Black)
	aiPlayerSmart.Algorithm = ai.AlgorithmAlphaBetaWithMemory
	aiPlayerDumb := ai.NewAIPlayer(color.White)
	aiPlayerDumb.Algorithm = ai.AlgorithmMiniMax

	turnColor := color.White
	start := time.Now()
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < MovesToPlay; i++ {
		if turnColor == color.White {
			aiPlayerDumb.MakeMove(&myBoard)
			// TODO(Vadim) make dummy random player class - player interface
			//moves := *myBoard.GetAllMoves(turnColor)
			//idx := rand.Intn(len(moves))
			//board.MakeMove(&moves[idx], &myBoard)
		} else {
			aiPlayerSmart.MakeMove(&myBoard)
		}
		turnColor = (turnColor + 1) % color.NumColors
		fmt.Printf("Move %d\n", i)
		fmt.Println(myBoard.Print())
	}

	fmt.Println("After moves")
	fmt.Println(myBoard.Print())
	// comment out printing inside loop for accurate timing
	fmt.Printf("Played %d moves in %d ms.", MovesToPlay, time.Now().Sub(start)/time.Millisecond)

	assert.True(t, aiPlayerSmart.EvaluateBoard(&myBoard).TotalScore > aiPlayerDumb.EvaluateBoard(&myBoard).TotalScore)
}
