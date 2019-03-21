package test

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestBoardAI(t *testing.T) {
	// TODO(Vadim) skip until it works better
	t.Skip()
	const MovesToPlay = 100
	const TimeToPlay = 5 * time.Second
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
		if i%2 == 0 && time.Now().Sub(start) > TimeToPlay {
			fmt.Printf("Aborting - out of time\n")
			break
		}
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
		fmt.Printf("Move %d\n", i+1)
		fmt.Println(myBoard.Print())
		util.PrintMemStats()
	}

	fmt.Println("After moves")
	fmt.Println(myBoard.Print())
	// comment out printing inside loop for accurate timing
	fmt.Printf("Played %d moves in %d ms.\n", MovesToPlay, time.Now().Sub(start)/time.Millisecond)

	smartScore := aiPlayerSmart.EvaluateBoard(&myBoard).TotalScore
	dumbScore := aiPlayerDumb.EvaluateBoard(&myBoard).TotalScore
	fmt.Printf("Good AI Evaluation %d.\n", smartScore)
	fmt.Printf("Bad AI Evaluation %d.\n", dumbScore)
	assert.True(t, smartScore > dumbScore)
}
