package test

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/util"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

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
		board2.SetPiece(util.Start, &board.Rook{})
	}
}

func BenchmarkGetPiece(b *testing.B) {
	board2 := board.Board{}
	board2.SetPiece(util.Start, &board.Rook{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.GetPiece(util.Start)
	}
}

func BenchmarkBoardMove(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.SetPiece(util.End, &board.Rook{})
		board2.SetPiece(util.Start, &board.Rook{})
		board.MakeMove(&board.Move{
			Start: util.Start,
			End:   util.End,
		}, &board2)
	}
}

func BenchmarkBoardHash(b *testing.B) {
	bo1 := board.Board{}
	bo2 := board.Board{}
	bo1.ResetDefault()
	bo2.ResetDefaultSlow()
	b.ResetTimer()
	for i := 0; i < b.N/2; i++ {
		bo2.Hash()
		bo1.Hash()
	}
}

func BenchmarkBoardResetDefault(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.ResetDefault()
	}
}

func BenchmarkBoardEquals(b *testing.B) {
	bo1 := board.Board{}
	bo2 := board.Board{}
	bo1.ResetDefault()
	bo2.ResetDefaultSlow()
	b.ResetTimer()
	for i := 0; i < b.N/2; i++ {
		bo1.Equals(&bo2)
		bo2.Equals(&bo1)
	}
}

func BenchmarkBoardHashLookup(b *testing.B) {
	scoreMap := util.NewConcurrentScoreMap()
	bo1 := board.Board{}
	bo1.ResetDefault()
	b.ResetTimer()
	hash := bo1.Hash()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		hash = bo1.Hash()
		bo1.RandomizeIllegal()
		b.StartTimer()
		scoreMap.Store(&hash, rand.Uint32())
	}
	b.StopTimer()
}

func BenchmarkBoardParallelHashLookup(b *testing.B) {
	scoreMap := util.NewConcurrentScoreMap()
	b.SetParallelism(8)
	b.RunParallel(func(pb *testing.PB) {
		bo1 := board.Board{}
		randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			bo1.RandomizeIllegal()
			hash := bo1.Hash()
			r := randGen.Uint32()

			scoreMap.Store(&hash, r)

			val, ok := scoreMap.Read(&hash)
			assert.True(b, ok)
			assert.Equal(b, r, val)
		}
	})
}

func benchMoveCount(b *testing.B, l board.Location, initialMove *board.Move, expectedMoves int) {
	bo1 := board.Board{}
	bo1.ResetDefault()
	if initialMove != nil {
		board.MakeMove(initialMove, &bo1)
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
	// todo(Vadim) b & w
	// todo(Vadim) make these into tests
	// todo(Vadim) move these into separate piece bench
}

func BenchmarkBishopGetMoves(b *testing.B) {
	benchMoveCount(b, board.Location{Row: 5, Col: 4}, &board.Move{
		Start: board.Location{Row: 7, Col: 2},
		End:   board.Location{Row: 5, Col: 4},
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
