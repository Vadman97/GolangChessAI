package bench

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"testing"
)

func BenchmarkMap(b *testing.B) {
	var PieceValue = map[byte]int{
		piece.PawnType:   1,
		piece.BishopType: 3,
		piece.KnightType: 3,
		piece.RookType:   5,
		piece.QueenType:  9,
		piece.KingType:   100,
	}
	for i := 0; i < b.N; i++ {
		_ = PieceValue[byte(i%piece.NumPieces)]
	}
}

func BenchmarkSlice(b *testing.B) {
	var PieceValue = [piece.NumPieces]int{
		piece.PawnType:   1,
		piece.BishopType: 3,
		piece.KnightType: 3,
		piece.RookType:   5,
		piece.QueenType:  9,
		piece.KingType:   100,
	}
	for i := 0; i < b.N; i++ {
		_ = PieceValue[byte(i%piece.NumPieces)]
	}
}
