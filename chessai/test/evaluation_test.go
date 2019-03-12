package test

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/piece"
	"ChessAI3/chessai/player/ai"
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
	lines := strings.Split(string(fileData), "\n")
	player, expectedScore := strings.Split(lines[0], " ")[0], strings.Split(lines[0], " ")[1]
	playerColor := color.Black
	if player == "W" {
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
	for r := int8(0); r < board.Height; r++ {
		pieces := strings.Split(boardRows[r], "|")
		for c, pStr := range pieces {
			l := board.Location{Row: r, Col: int8(c)}
			var p board.Piece
			if pStr != "   " {
				d := strings.Split(pStr, "_")
				cChar, pChar := rune(d[0][0]), rune(d[1][0])
				p = board.PieceFromType(piece.NameToType[pChar])
				p.SetColor(board.ColorFromChar(cChar))
				p.SetPosition(l)
			}
			b.SetPiece(l, p)
		}
	}
}

func evaluateAndCompare(t *testing.T, color byte, score int, b *board.Board) {
	p := ai.NewAIPlayer(color)
	eval := p.EvaluateBoard(b)
	assert.Equal(t, score, eval.TotalScore)
}