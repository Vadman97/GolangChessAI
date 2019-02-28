package test

import (
	"ChessAI3/chessai/board"
	"github.com/stretchr/testify/assert"
	"testing"
)

var start = board.Location{Row: 2, Col: 5}
var end = board.Location{Row: 4, Col: 5}

func TestBoardMove(t *testing.T) {
	board2 := board.Board{}
	board2.SetPiece(end, &board.Rook{})
	board2.SetPiece(start, &board.Rook{})
	startPiece := board2.GetPiece(start)
	startPiece.SetPosition(end)
	board.MakeMove(&board.Move{
		Start: start,
		End:   end,
	}, &board2)
	assert.Nil(t, board2.GetPiece(start))
	assert.Equal(t, startPiece, board2.GetPiece(end))
	// TODO(Vadim) figure out why position not being set
	assert.Equal(t, end, board2.GetPiece(end).GetPosition())
}

func BenchmarkSet(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.SetPiece(start, &board.Rook{})
	}
}

func BenchmarkGet(b *testing.B) {
	board2 := board.Board{}
	board2.SetPiece(start, &board.Rook{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.GetPiece(start)
	}
}

func BenchmarkBoardMove(b *testing.B) {
	board2 := board.Board{}
	board2.SetPiece(end, &board.Rook{})
	board2.SetPiece(start, &board.Rook{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board.MakeMove(&board.Move{
			Start: start,
			End:   end,
		}, &board2)
	}
}
