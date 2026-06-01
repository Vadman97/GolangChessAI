package ai

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
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

var algorithmsToTest = [...]string{
	AlgorithmMiniMax,
	AlgorithmMTDf,
	AlgorithmABDADA,
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

	boardsToTest["Random"] = competitionBoard{
		board: gb,
	}

	//boardsToTest = loadCompetitionBoards(boardsToTest)

	for boardName, entry := range boardsToTest {
		for c := color.White; c < color.NumColors; c++ {
			fmt.Printf("===== EVALUATING %s =====\n", color.Names[c])
			fmt.Println(boardName)
			fmt.Println(*entry.board)
			fmt.Printf("Best move: %s\n\n\n", entry.bestMove)

			moves := map[string]location.Move{}
			for _, algorithmName := range algorithmsToTest {
				algorithm := NameToAlgorithm[algorithmName]
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

// TestABDADAFindsHangingKnight verifies the AI captures a free knight rather than
// playing a passive quiet move. This position was a real-game blunder where
// a stale TT entry (quiet bishop move) caused the search to miss QxN.
//
// FEN: r2qkb1r/p2bpppp/2p5/8/3Qn3/5N2/PPP2PPP/R1B2RK1 w kq - 0 10
// White queen on d4 can capture a hanging Black knight on e4.
func TestABDADAFindsHangingKnight(t *testing.T) {
	b := &board.Board{}
	// Board rows in engine coordinates (row 0 = rank 1, col 0 = h-file)
	rows := []string{
		"   |W_K|W_R|   |   |W_B|   |W_R", // row 0: rank 1
		"W_P|W_P|W_P|   |   |W_P|W_P|W_P", // row 1: rank 2
		"   |   |W_N|   |   |   |   |   ", // row 2: rank 3
		"   |   |   |B_N|W_Q|   |   |   ", // row 3: rank 4 — B_N on e4, W_Q on d4
		"   |   |   |   |   |   |   |   ", // row 4: rank 5
		"   |   |   |   |   |B_P|   |   ", // row 5: rank 6
		"B_P|B_P|B_P|B_P|B_B|   |   |B_P", // row 6: rank 7
		"B_R|   |B_B|B_K|B_Q|   |   |B_R", // row 7: rank 8
	}
	b.LoadBoardFromText(rows)
	// White king has moved (castled to g1); mark flags accordingly.
	b.SetFlag(board.FlagKingMoved, color.White, true)
	b.SetFlag(board.FlagKingMoved, color.Black, true)

	move := getBestMove(b, color.White, NameToAlgorithm[AlgorithmABDADA])

	// Expected: White queen d4 (row 3, col 4) captures Black knight e4 (row 3, col 3)
	wantFrom := location.NewLocation(3, 4) // d4
	wantTo := location.NewLocation(3, 3)   // e4
	assert.Equal(t, wantFrom, move.Start,
		"AI should capture hanging knight: start square should be d4 (3,4)")
	assert.Equal(t, wantTo, move.End,
		"AI should capture hanging knight: end square should be e4 (3,3)")
}

func TestLazySMPFindsHangingKnight(t *testing.T) {
	b := &board.Board{}
	rows := []string{
		"   |W_K|W_R|   |   |W_B|   |W_R",
		"W_P|W_P|W_P|   |   |W_P|W_P|W_P",
		"   |   |W_N|   |   |   |   |   ",
		"   |   |   |B_N|W_Q|   |   |   ",
		"   |   |   |   |   |   |   |   ",
		"   |   |   |   |   |B_P|   |   ",
		"B_P|B_P|B_P|B_P|B_B|   |   |B_P",
		"B_R|   |B_B|B_K|B_Q|   |   |B_R",
	}
	b.LoadBoardFromText(rows)
	b.SetFlag(board.FlagKingMoved, color.White, true)
	b.SetFlag(board.FlagKingMoved, color.Black, true)

	move := getBestMove(b, color.White, NameToAlgorithm[AlgorithmLazySMP])

	wantFrom := location.NewLocation(3, 4)
	wantTo := location.NewLocation(3, 3)
	assert.Equal(t, wantFrom, move.Start, "LazySMP should capture hanging knight: start d4 (3,4)")
	assert.Equal(t, wantTo, move.End, "LazySMP should capture hanging knight: end e4 (3,3)")
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
	assert.True(t, diff <= PawnValueWeight*PieceValue[piece.QueenType])
}

func getBestMove(gameBoard *board.Board, c color.Color, algorithm Algorithm) *location.Move {
	player := NewAIPlayer(c, algorithm)
	player.MaxSearchDepth = 100
	player.MaxThinkTime = 1000 * time.Millisecond

	return player.GetBestMove(gameBoard, nil, nil)
}

// TestOpeningBookSkipsIllegalMoveWhenInCheck reproduces game WOK8C5yA: the bot was
// following the Sicilian-style book (...c5 Nc6 d6 Nf6 g6) when the opponent deviated
// with 5.Bxc6+ — a check. The book is indexed purely by turn count and ignored the
// check, returning the booked ...g6, which is illegal (it does not address the check).
// Lichess rejected it and the bot abandoned the game. GetBestMove must instead detect
// the book move is illegal, abandon the book, and return a legal (check-evading) move.
func TestOpeningBookSkipsIllegalMoveWhenInCheck(t *testing.T) {
	b := &board.Board{}
	// After 1.Nc3 c5 2.Nf3 Nc6 3.e4 d6 4.Bb5 Nf6 5.Bxc6+ — Black to move, in check
	// from the bishop on c6 along c6-d7-e8. (row 0 = rank 1, col 0 = h-file.)
	rows := []string{
		"W_R|   |   |W_K|W_Q|W_B|   |W_R", // rank 1
		"W_P|W_P|W_P|   |W_P|W_P|W_P|W_P", // rank 2
		"   |   |W_N|   |   |W_N|   |   ", // rank 3: Nf3, Nc3
		"   |   |   |W_P|   |   |   |   ", // rank 4: e4
		"   |   |   |   |   |B_P|   |   ", // rank 5: c5
		"   |   |B_N|   |B_P|W_B|   |   ", // rank 6: Nf6, d6, Bc6
		"B_P|B_P|B_P|B_P|   |   |B_P|B_P", // rank 7
		"B_R|   |B_B|B_K|B_Q|B_B|   |B_R", // rank 8
	}
	b.LoadBoardFromText(rows)
	assert.True(t, b.IsKingInCheck(color.Black), "setup sanity: Black should be in check from Bc6")

	player := NewAIPlayer(color.Black, NameToAlgorithm[AlgorithmABDADA])
	player.MaxSearchDepth = 100
	player.MaxThinkTime = 500 * time.Millisecond
	// Force the exact book state from the game: Sicilian-style line (index 1), whose
	// move #4 (0-indexed) is ...g6 — the illegal booked move.
	player.Opening = 1
	player.TurnCount = 4
	bookG6 := OpeningMoves[color.Black][1][4]
	assert.Equal(t, location.NewLocation(6, 1), bookG6.Start, "guard: book move 4 should be g7-g6")
	assert.Equal(t, location.NewLocation(5, 1), bookG6.End, "guard: book move 4 should be g7-g6")

	move := player.GetBestMove(b, nil, nil)

	// It must not play the illegal book move...
	assert.False(t, move.Start.Equals(bookG6.Start) && move.End.Equals(bookG6.End),
		"bot played the illegal booked ...g6 while in check (Lichess rejects this and the game is abandoned)")
	// ...and the move it does play must be legal: applying it leaves the king safe.
	legal := false
	for _, m := range *b.GetAllMoves(color.Black, nil) {
		if m.Start.Equals(move.Start) && m.End.Equals(move.End) {
			legal = true
			break
		}
	}
	assert.True(t, legal, "bot returned a move that is not in the legal move list")
	assert.Equal(t, OpeningNone, player.Opening, "broken book should be abandoned for the rest of the game")
}
