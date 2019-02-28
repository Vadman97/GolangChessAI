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
	assert.Equal(t, end, board2.GetPiece(end).GetPosition())
}

func TestBoardFlags(t *testing.T) {
	board2 := board.Board{}
	for i := 0; i < board.FlagRightRookMoved; i++ {
		for c := 0; c < 2; c++ {
			assert.False(t, board2.GetFlag(byte(i), byte(c)))
		}
	}
	for i := 0; i < board.FlagRightRookMoved; i++ {
		for c := 0; c < 2; c++ {
			board2.SetFlag(byte(i), byte(c), true)
			assert.True(t, board2.GetFlag(byte(i), byte(c)))
			for i2 := 0; i2 < board.FlagRightRookMoved; i2++ {
				for c2 := 0; c2 < 2; c2++ {
					if i != i2 && c != c2 {
						assert.False(t, board2.GetFlag(byte(i2), byte(c2)))
					}
				}
			}
			board2.SetFlag(byte(i), byte(c), false)
			assert.False(t, board2.GetFlag(byte(i), byte(c)))
		}
	}
}

func TestBoardSetAndCopy(t *testing.T) {
	t.Fail()
}

func BenchmarkCopy(b *testing.B) {
	board2 := board.Board{}
	bNew := board2.Copy()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bNew = board2.Copy()
	}
	b.StopTimer()
	bNew.Copy()
}

func BenchmarkSetPiece(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.SetPiece(start, &board.Rook{})
	}
}

func BenchmarkGetPiece(b *testing.B) {
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
