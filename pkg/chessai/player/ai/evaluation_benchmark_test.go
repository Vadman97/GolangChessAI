package ai

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
)

var evaluationSpeedSink int

func BenchmarkEvaluateBoardCold(b *testing.B) {
	position := &board.Board{}
	position.ResetDefault()
	p := NewAIPlayer(color.White, &ABDADA{NumThreads: 1})
	p.evaluationMap = nil
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluationSpeedSink = p.EvaluateBoard(position, color.White).TotalScore
	}
}
