package main

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/player"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestAI(t *testing.T) {
	const MovesToPlay = 20
	myBoard := board.Board{}
	myBoard.ResetDefault()
	fmt.Println("Before moves")
	fmt.Println(myBoard.Print())

	aiPlayer := player.AIPlayer{
		TurnCount:   0,
		PlayerColor: color.Black,
	}

	turnColor := color.White
	start := time.Now()
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < MovesToPlay; i++ {
		if turnColor == color.White {
			// TODO(Vadim) make dummy random player class - player interface
			moves := *myBoard.GetAllMoves(turnColor)
			idx := rand.Intn(len(moves))
			board.MakeMove(&moves[idx], &myBoard)
		} else {
			aiPlayer.MakeMove(&myBoard)
		}
		turnColor = (turnColor + 1) % color.NumColors
		fmt.Printf("Move %d\n", i)
		fmt.Println(myBoard.Print())
	}

	fmt.Println("After moves")
	fmt.Println(myBoard.Print())
	// comment out printing inside loop for accurate timing
	fmt.Printf("Played %d moves in %d ms.", MovesToPlay, time.Now().Sub(start)/time.Millisecond)
}
