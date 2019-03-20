package test

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/util"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestBoardHashLookup(t *testing.T) {
	const N = 100000
	scoreMap := util.NewConcurrentBoardMap()
	bo1 := board.Board{}
	bo1.ResetDefault()
	hash := bo1.Hash()
	hits := 0
	for i := 0; i < N; i++ {
		//bo1.RandomizeIllegal()
		moves := *bo1.GetAllMoves(byte(rand.Int() % color.NumColors))
		if len(moves) > 0 {
			board.MakeMove(&moves[rand.Int()%len(moves)], &bo1)
		}
		hash = bo1.Hash()
		v, ok := scoreMap.Read(&hash)
		if ok {
			hits++
			bo2 := v.(*board.Board)
			assert.True(t, bo1.Equals(bo2))
			assert.True(t, bo2.Equals(&bo1))
		} else {
			scoreMap.Store(&hash, &bo1)
		}
	}
	fmt.Println(bo1.Print())
	fmt.Printf("Hit ratio %.2f%%\n", float64(hits)/float64(N)*100.0)
	fmt.Printf("Number different boards %d\n", N-hits)
	scoreMap.PrintMetrics()
}
