package test

import (
	"ChessAI3/chessai/board"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var start = board.Location{Row: 2, Col: 5}
var end = board.Location{Row: 4, Col: 5}

func TestBoardMove(t *testing.T) {
	board2 := board.Board{}
	board2.SetPiece(end, board.Rook{})
	board2.SetPiece(start, board.Rook{})
	startPiece := board2.GetPiece(start)
	board.MakeMove(board2.GetPiece(start), &board.Move{
		Start: start,
		End:   end,
	}, &board2)
	assert.Nil(t, board2.GetPiece(start))
	assert.Equal(t, board2.GetPiece(end), startPiece)
	// TODO(Vadim) figure out why position not being set
	assert.Equal(t, board2.GetPiece(end).GetPosition(), end)
}

func BenchmarkSet(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.SetPiece(start, board.Rook{})
	}
	fmt.Printf("%+v\n", board2.GetPiece(start))
}

func BenchmarkGet(b *testing.B) {
	board2 := board.Board{}
	board2.SetPiece(start, board.Rook{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.GetPiece(start)
	}
	fmt.Printf("%+v\n", board2.GetPiece(start))
}

func BenchmarkBoardMove(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.SetPiece(end, board.Rook{})
		board2.SetPiece(start, board.Rook{})
		board.MakeMove(board2.GetPiece(start), &board.Move{
			Start: start,
			End:   end,
		}, &board2)
	}
	fmt.Printf("Start %+v\n", board2.GetPiece(start))
	fmt.Printf("End %+v\n", board2.GetPiece(end))
}
