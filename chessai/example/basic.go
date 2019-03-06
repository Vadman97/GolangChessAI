package main

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"fmt"
	"math/rand"
	"time"
)

func main() {
	const MovesToPlay = 100
	myBoard := board.Board{}
	myBoard.ResetDefault()

	turnColor := color.White
	start := time.Now()
	for i := 0; i < MovesToPlay; i++ {
		blackMoves, whiteMoves := myBoard.GetAllMoves(turnColor == color.Black, turnColor == color.White)
		var moves []board.Move
		if turnColor == color.Black {
			moves = *blackMoves
		} else if turnColor == color.White {
			moves = *whiteMoves
		}
		if len(moves) == 0 {
			break
		}
		idx := rand.Intn(len(moves))
		board.MakeMove(&moves[idx], &myBoard)
		turnColor = (turnColor + 1) % color.NumColors
	}

	fmt.Println(myBoard.Print())
	fmt.Printf("Played %d moves in %d ms.", MovesToPlay, time.Now().Sub(start)/time.Millisecond)
}
