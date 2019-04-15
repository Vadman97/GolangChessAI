package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"log"
	"path"
	"strconv"
	"strings"
	"testing"
)

const boardsDirectory = "evaluation_boards"

func TestBoardEvaluate(t *testing.T) {
	files, err := ioutil.ReadDir(boardsDirectory)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		testBoard(t, f.Name())
	}
}

func testBoard(t *testing.T, fileName string) {
	lines, skip := util.LoadBoardFile(path.Join(boardsDirectory, fileName))
	if skip {
		return
	}
	player, expectedScore := strings.Split(lines[0], " ")[0], strings.Split(lines[0], " ")[1]
	playerColor := color.Black
	if player == "White" {
		playerColor = color.White
	}
	score, err := strconv.ParseInt(expectedScore, 10, 32)
	if err != nil {
		log.Fatal(err)
	}
	myBoard := board.Board{}
	myBoard.LoadBoardFromText(lines[1:])
	evaluateAndCompare(t, playerColor, int(score), &myBoard)
}

func evaluateAndCompare(t *testing.T, color byte, score int, b *board.Board) {
	p := NewAIPlayer(color, &Random{})
	eval := p.EvaluateBoard(b, color)
	fmt.Printf("Expected %d Evaluated %d\n", score, eval.TotalScore)
	assert.Equal(t, score, eval.TotalScore)
}
