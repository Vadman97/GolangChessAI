package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"math/rand"
	"path"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

var algorithmsToTest = map[string]Algorithm{
	AlgorithmMiniMax: &MiniMax{},
	AlgorithmMTDf:    &MTDf{},
	AlgorithmABDADA:  &ABDADA{},
}

type competitionBoard struct {
	board         *board.Board
	bestMove      location.Move
	bestMoveColor color.Color
}

func TestAIBestMovesSame(t *testing.T) {
	rand.Seed(config.Get().TestRandSeed)

	boardsToTest := map[string]competitionBoard{}

	gb := &board.Board{}
	gb.ResetDefault()
	for i := 0; i < 42; i++ {
		gb.MakeRandomMove()
	}
	//boardsToTest["Random"] = competitionBoard{
	//	board: gb,
	//}

	boardsToTest = loadCompetitionBoards(boardsToTest)

	for boardName, entry := range boardsToTest {
		for c := color.White; c < color.NumColors; c++ {
			fmt.Printf("===== EVALUATING %s =====\n", color.Names[c])
			fmt.Println(boardName)
			fmt.Println(*entry.board)
			fmt.Printf("Best move: %s\n\n\n", entry.bestMove)

			moves := map[string]location.Move{}
			for algorithmName, algorithm := range algorithmsToTest {
				fmt.Printf("\n\n===== ALGORITHM %s =====\n", algorithmName)
				moves[algorithmName] = *getBestMove(entry.board, c, algorithm)
				fmt.Printf("===== ALGORITHM %s =====\n\n", algorithm.GetName())

				runtime.GC()
				fmt.Println(util.GetMemStatString())
			}
			for _, move := range moves {
				fmt.Println(move)
			}

			// TODO(Vadim) check that score actually improves by comparing to other outcomes of moves after n turns
			evaluateScores(t, c, entry.board, moves)

			for algorithmName, move := range moves {
				// TODO(Vadim) add this once we can ensure ai is better
				/*if algorithmName != AlgorithmMiniMax {
					// this is used to ensure all algorithms get the same moves
					assert.Equal(t, moves[AlgorithmMiniMax], move)
				}*/
				// if there is an expected best move, compare
				if !entry.bestMove.Start.Equals(entry.bestMove.End) {
					fmt.Printf("===== ALGORITHM %s =====\n", algorithmName)
					fmt.Printf("== EXPECTED BEST MOVE %s ==\n", entry.bestMove)
					assert.Equal(t, entry.bestMove, move)
				}
			}
		}
	}
}

const historicBoardsDirectory = "competition_boards"

/**
 * Load historic competition games and their corresponding true best moves
 */
func loadCompetitionBoards(boards map[string]competitionBoard) map[string]competitionBoard {
	files, err := ioutil.ReadDir(historicBoardsDirectory)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		fileName := f.Name()
		lines, skip := util.LoadBoardFile(path.Join(historicBoardsDirectory, fileName))
		if skip {
			continue
		}
		player, bestMoveStr := strings.Split(lines[0], " ")[0], strings.Split(lines[0], " ")[1]
		playerColor := color.Black
		if player == "White" {
			playerColor = color.White
		}
		bestMoveStart, bestMoveEnd := strings.Split(bestMoveStr, "-")[0], strings.Split(bestMoveStr, "-")[1]
		var locations []location.Location
		for _, str := range []string{bestMoveStart, bestMoveEnd} {
			rowS, colS := strings.Split(str, ",")[0], strings.Split(str, ",")[1]
			row, _ := strconv.ParseInt(rowS, 10, 32)
			col, _ := strconv.ParseInt(colS, 10, 32)
			locations = append(locations, location.NewLocation(location.CoordinateType(row), location.CoordinateType(col)))
		}
		gameBoard := board.Board{}
		gameBoard.LoadBoardFromText(lines[1:])
		boards[fileName] = competitionBoard{
			board: &gameBoard,
			bestMove: location.Move{
				Start: locations[0],
				End:   locations[1],
			},
			bestMoveColor: playerColor,
		}
	}
	return boards
}

func evaluateScores(t *testing.T, c color.Color, gameBoard *board.Board, moves map[string]location.Move) {
	eval := NewAIPlayer(c, nil)
	scores := map[string]int{}
	for algorithmName, move := range moves {
		newBoard := gameBoard.Copy()
		board.MakeMove(&move, newBoard)
		scores[algorithmName] = eval.EvaluateBoard(newBoard, c).TotalScore
		fmt.Printf("\n===== ALGORITHM:%10s%5d =====\n", algorithmName, scores[algorithmName])
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
	player.MaxThinkTime = 180000 * time.Millisecond

	return player.GetBestMove(gameBoard, nil, nil)
}
