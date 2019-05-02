package bench

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	"github.com/stretchr/testify/assert"
	"testing"
)

func BenchmarkEvaluate(b *testing.B) {
	p := ai.AIPlayer{}
	bo1 := board.Board{}
	bo1.ResetDefault()
	var eval *ai.Evaluation
	for i := 0; i < b.N; i++ {
		eval = p.EvaluateBoard(&bo1, color.Black)
	}
	assert.NotNil(b, eval)
	if eval != nil {
		assert.Equal(b, 0, eval.TotalScore)
	}
}

func BenchmarkEvaluateParallel(b *testing.B) {
	b.SetParallelism(8)
	b.RunParallel(func(pb *testing.PB) {
		p := ai.AIPlayer{}
		bo1 := board.Board{}
		bo1.ResetDefault()
		for pb.Next() {
			p.EvaluateBoard(&bo1, color.Black)
		}
	})
}
