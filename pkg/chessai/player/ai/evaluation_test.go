package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
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
	fileData, err := ioutil.ReadFile(path.Join(boardsDirectory, fileName))
	if err != nil {
		log.Fatal(err)
	}
	fileStr := strings.ReplaceAll(string(fileData), "\r", "")
	lines := strings.Split(fileStr, "\n")
	if strings.Contains(lines[0], "skip") {
		log.Printf("WARNING Skipping test %s\n", fileName)
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
	parseBoard(&myBoard, lines[1:])
	evaluateAndCompare(t, playerColor, int(score), &myBoard)
}

func parseBoard(b *board.Board, boardRows []string) {
	for r := location.CoordinateType(0); r < board.Height; r++ {
		pieces := strings.Split(boardRows[r], "|")
		for c, pStr := range pieces {
			l := location.NewLocation(r, location.CoordinateType(c))
			var p board.Piece
			if pStr != "   " && len(pStr) == 3 {
				d := strings.Split(pStr, "_")
				cChar, pChar := rune(d[0][0]), rune(d[1][0])
				p = board.PieceFromType(piece.NameToType[pChar])
				if p == nil {
					panic("piece should not be nil - invalid template")
				}
				p.SetColor(board.ColorFromChar(cChar))
				p.SetPosition(l)
			}
			b.SetPiece(l, p)
		}
	}
}

func evaluateAndCompare(t *testing.T, color byte, score int, b *board.Board) {
	p := NewAIPlayer(color, &Random{})
	eval := p.EvaluateBoard(b)
	fmt.Printf("Expected %d Evaluated %d\n", score, eval.TotalScore)
	assert.Equal(t, score, eval.TotalScore)
}
