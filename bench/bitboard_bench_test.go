package bench

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"testing"
)

func BenchmarkBitboardCombine(b *testing.B) {
	//bb1 := board.BitBoard{0xF, 0x6, 0x3, 0xC, 0xA, 0xF, 0x1, 0xD}
	//bb2 := board.BitBoard{0xA, 0x3, 0x3, 0xF, 0xA, 0x1, 0x4, 0xB}
	bb1 := board.BitBoard(0xF63CAF1D)
	bb2 := board.BitBoard(0xA33FA14B)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bb1.CombineBitBoards(bb2)
	}
	b.StopTimer()
}

func BenchmarkBitboardIntersect(b *testing.B) {
	//bb1 := board.BitBoard{0xF, 0x6, 0x3, 0xC, 0xA, 0xF, 0x1, 0xD}
	//bb2 := board.BitBoard{0xA, 0x3, 0x3, 0xF, 0xA, 0x1, 0x4, 0xB}
	bb1 := board.BitBoard(0xF63CAF1D)
	bb2 := board.BitBoard(0xA33FA14B)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bb1.IntersectBitBoards(bb2)
	}
	b.StopTimer()
}
