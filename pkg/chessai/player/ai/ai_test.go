package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

var algorithmsToTest = [...]string{
	AlgorithmMiniMax,
	AlgorithmMTDf,
	AlgorithmNegaScout,
}

func TestAIBestMovesSame(t *testing.T) {

	rand.Seed(config.Get().TestRandSeed)

	gameBoard := &board.Board{}
	gameBoard.ResetDefault()
	for i := 0; i < 42; i++ {
		gameBoard.MakeRandomMove()
	}
	fmt.Println(gameBoard)

	for c := color.White; c < color.NumColors; c++ {
		fmt.Printf("===== EVALUATING %s =====\n\n\n", color.Names[c])
		moves := map[string]location.Move{}
		for _, algorithmName := range algorithmsToTest {
			algorithm := NameToAlgorithm[algorithmName]
			fmt.Printf("\n\n===== ALGORITHM %s =====\n", algorithmName)
			moves[algorithmName] = *getBestMove(gameBoard, c, algorithm)
			fmt.Printf("===== ALGORITHM %s =====\n\n", algorithm.GetName())
		}
		for _, move := range moves {
			fmt.Println(move)
		}
		evaluateScores(t, c, gameBoard, moves)
		// TODO(Vadim) check that score actually improves by comparing to other outcomes of moves

		/*for algorithmName, move := range moves {
			if algorithmName != AlgorithmMiniMax {
				// this is used to ensure all algorithms get the same moves
				assert.Equal(t, moves[AlgorithmMiniMax], move)
			}
			// TODO(Vadim) add this once we can ensure ai is better
		}*/
	}
}

func evaluateScores(t *testing.T, c color.Color, gameBoard *board.Board, moves map[string]location.Move) {
	eval := NewAIPlayer(c, nil)
	scores := map[string]int{}
	for algorithmName, move := range moves {
		newBoard := gameBoard.Copy()
		board.MakeMove(&move, newBoard)
		scores[algorithmName] = eval.EvaluateBoard(newBoard, c).TotalScore
		fmt.Printf("\n===== ALGORITHM:%s \t\t %d =====\n", algorithmName, scores[algorithmName])
	}
	minScore, maxScore := PosInf, NegInf
	for _, score := range scores {
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}
	diff := maxScore - minScore
	fmt.Printf("Difference: %d\n", diff)
	// test that the moves are all good within a pawn
	// TODO(Vadim) make more aggressive
	assert.True(t, diff <= PieceValueWeight*PieceValue[piece.QueenType])
}

func getBestMove(gameBoard *board.Board, c color.Color, algorithm Algorithm) *location.Move {
	player := NewAIPlayer(c, algorithm)
	player.MaxSearchDepth = 16
	player.MaxThinkTime = 5000 * time.Millisecond

	return player.GetBestMove(gameBoard, nil, nil)
}
