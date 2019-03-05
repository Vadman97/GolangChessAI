package test

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"github.com/stretchr/testify/assert"
	"testing"
)

func benchMoveCount(b *testing.B, l board.Location, initialMove *board.Move, expectedMoves int) {
	bo1 := board.Board{}
	bo1.ResetDefault()
	if initialMove != nil {
		board.MakeMove(initialMove, &bo1)
	}
	if l.Row == 0 {
		assert.Equal(b, color.Black, bo1.GetPiece(l).GetColor())
	} else if l.Row == 7 {
		assert.Equal(b, color.White, bo1.GetPiece(l).GetColor())
	}
	b.ResetTimer()
	var moves *[]board.Move
	for i := 0; i < b.N; i++ {
		moves = bo1.GetPiece(l).GetMoves(&bo1)
	}
	assert.NotNil(b, moves)
	if moves != nil {
		assert.Equal(b, expectedMoves, len(*moves))
	}
}

func BenchmarkBishopGetMovesNone(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 7, Col: 2}, nil, 0)
}

func BenchmarkBishopGetMoves(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 5, Col: 4}, &board.Move{
		Start: board.Location{Row: 7, Col: 2},
		End:   board.Location{Row: 5, Col: 4},
	}, 7)
}

func BenchmarkBishopGetMovesNoneBlack(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 0, Col: 2}, nil, 0)
}

func BenchmarkBishopGetMovesBlack(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 2, Col: 4}, &board.Move{
		Start: board.Location{Row: 0, Col: 2},
		End:   board.Location{Row: 2, Col: 4},
	}, 7)
}

func BenchmarkQueenGetMovesNone(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 7, Col: 3}, nil, 0)
}

func BenchmarkQueenGetMoves(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 5, Col: 3}, &board.Move{
		Start: board.Location{Row: 7, Col: 3},
		End:   board.Location{Row: 5, Col: 3},
	}, 18)
}

func BenchmarkQueenGetMovesNoneBlack(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 0, Col: 3}, nil, 0)
}

func BenchmarkQueenGetMovesBlack(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 2, Col: 3}, &board.Move{
		Start: board.Location{Row: 0, Col: 3},
		End:   board.Location{Row: 2, Col: 3},
	}, 18)
}
