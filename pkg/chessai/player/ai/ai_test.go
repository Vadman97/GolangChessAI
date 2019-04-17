package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestAIBestMovesSame(t *testing.T) {
	var algorithmsToTest = [...]Algorithm{
		&MiniMax{},
		&MTDf{},
		&NegaScout{},
	}

	rand.Seed(config.Get().TestRandSeed)

	gameBoard := &board.Board{}
	gameBoard.ResetDefault()
	for i := 0; i < 50; i++ {
		gameBoard.MakeRandomMove()
	}
	fmt.Println(gameBoard.Print())

	for c := color.White; c < color.NumColors; c++ {
		fmt.Printf("===== EVALUATING %s =====\n\n\n", color.Names[c])
		var moves []location.Move
		for _, algorithm := range algorithmsToTest {
			fmt.Printf("\n===== ALGORITHM %s =====\n", algorithm.GetName())
			moves = append(moves, *getBestMove(gameBoard, c, algorithm))
		}
		for _, move := range moves {
			fmt.Println(move.Print())
		}
		for _, move := range moves[1:] {
			assert.Equal(t, moves[0], move)
		}
	}
}

func getBestMove(gameBoard *board.Board, c color.Color, algorithm Algorithm) *location.Move {
	player := NewAIPlayer(c, algorithm)
	player.MaxSearchDepth = 100
	player.MaxThinkTime = 10000 * time.Millisecond

	return player.GetBestMove(gameBoard, nil, nil)
}
