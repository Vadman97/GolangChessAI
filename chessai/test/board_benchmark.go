package test

import (
	"ChessAI3/chessai/board"
	"fmt"
	"testing"
)

func BenchmarkSet(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.SetPiece(board.Location{Row: 4, Col: 5}, board.Rook{})
	}
	fmt.Printf("%v\n", board2)
	fmt.Printf("%v\n", board2.GetPiece(board.Location{Row: 4, Col: 5}))
}
func BenchmarkGet(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	board2.SetPiece(board.Location{Row: 4, Col: 5}, board.Rook{})
	for i := 0; i < b.N; i++ {
		board2.GetPiece(board.Location{Row: 4, Col: 5})
	}
	fmt.Printf("%v\n", board2)
	fmt.Printf("%v\n", board2.GetPiece(board.Location{Row: 4, Col: 5}))
}
