package bench

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
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
		board.MakeMove(&location.Move{
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
	scoreMap := util.NewConcurrentBoardMap()
	bo1 := board.Board{}
	bo1.ResetDefault()
	b.ResetTimer()
	hash := bo1.Hash()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		hash = bo1.Hash()
		bo1.RandomizeIllegal()
		b.StartTimer()
		scoreMap.Store(&hash, 0, rand.Int31())
	}
	b.StopTimer()
}

func BenchmarkBoardParallelHashLookup(b *testing.B) {
	scoreMap := util.NewConcurrentBoardMap()
	b.SetParallelism(8)
	b.RunParallel(func(pb *testing.PB) {
		bo1 := board.Board{}
		randGen := rand.New(rand.NewSource(time.Now().UnixNano()))
		for pb.Next() {
			bo1.RandomizeIllegal()
			hash := bo1.Hash()
			r := randGen.Int31()

			scoreMap.Store(&hash, 0, r)

			val, ok := scoreMap.Read(&hash, 0)
			assert.True(b, ok)
			assert.Equal(b, r, val)
		}
	})
}
