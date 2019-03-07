package main

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/util"
	"fmt"
	"math/rand"
	"time"
)

func main() {
	const MovesToPlay = 200
	scoreMap := util.NewConcurrentScoreMap()
	myBoard := board.Board{}
	myBoard.ResetDefault()
	fmt.Println("Before moves")
	fmt.Println(myBoard.Print())

	turnColor := color.White
	start := time.Now()
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < MovesToPlay; i++ {
		moves := *myBoard.GetAllMoves(turnColor)
		if len(moves) == 0 {
			break
		}
		idx := rand.Intn(len(moves))
		board.MakeMove(&moves[idx], &myBoard)
		hash := myBoard.Hash()
		scoreMap.Store(&hash, 0)
		turnColor = (turnColor + 1) % color.NumColors
		fmt.Printf("Move %d\n", i)
		fmt.Println(myBoard.Print())
	}

	fmt.Println("After moves")
	fmt.Println(myBoard.Print())
	// comment out printing inside loop for accurate timing
	fmt.Printf("Played %d moves in %d ms.", MovesToPlay, time.Now().Sub(start)/time.Millisecond)
	// show how score map is filled (hashes hopefully distribute evenly over slices)
	//scoreMap.PrintMetrics()
}
