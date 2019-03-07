package bench

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/player"
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkEvaluate(b *testing.B) {
	p := player.AIPlayer{}
	bo1 := board.Board{}
	bo1.ResetDefault()
	var eval *board.Evaluation
	for i := 0; i < b.N; i++ {
		eval = p.EvaluateBoard(&bo1)
	}
	assert.NotNil(b, eval)
	if eval != nil {
		assert.Equal(b, 0, eval.TotalScore)
	}
}

func BenchmarkEvaluateParallel(b *testing.B) {
	b.SetParallelism(8)
	b.RunParallel(func(pb *testing.PB) {
		p := player.AIPlayer{}
		bo1 := board.Board{}
		bo1.ResetDefault()
		for pb.Next() {
			p.EvaluateBoard(&bo1)
		}
	})
}
