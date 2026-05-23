package bench

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
)

// startBoard returns a freshly reset board with no caches (matches ABDADA's CacheGetAllMoves=false).
func startBoard() *board.Board {
	b := &board.Board{}
	b.ResetDefault()
	b.CacheGetAllMoves = false
	b.CacheGetAllAttackableMoves = false
	return b
}

// newABDADAPlayer returns a minimal AIPlayer wired for single-threaded ABDADA.
func newABDADAPlayer(c color.Color) *ai.AIPlayer {
	p := ai.NewAIPlayer(c, &ai.ABDADA{NumThreads: 1})
	p.TranspositionTableEnabled = true
	p.MaxSearchDepth = 4
	return p
}

// BenchmarkABDADADepth3 measures ABDADA nodes-per-second searching depth 3 from the start position.
// This is the core ABDADA hot-path benchmark: it exercises TT reads/writes, move generation,
// board copy, and evaluation all together.
func BenchmarkABDADADepth3(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bo := startBoard()
		p := newABDADAPlayer(color.White)
		p.MaxSearchDepth = 3
		p.GetBestMove(bo, nil, nil)
	}
}

// BenchmarkABDADADepth4 is the same but one ply deeper; shows super-linear cost growth.
func BenchmarkABDADADepth4(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bo := startBoard()
		p := newABDADAPlayer(color.White)
		p.MaxSearchDepth = 4
		p.GetBestMove(bo, nil, nil)
	}
}

// BenchmarkGetAllMoves isolates move generation (including willMoveLeaveKingInCheck for
// each candidate). This is called at every internal ABDADA node.
func BenchmarkGetAllMoves(b *testing.B) {
	bo := startBoard()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bo.GetAllMoves(color.White, nil)
	}
}

// BenchmarkGetAllMovesParallel shows how move generation degrades under contention
// (ABDADA calls this from multiple goroutines on independent board copies, but
// the board's shared ConcurrentBoardMap is the contention point when caching is on).
func BenchmarkGetAllMovesParallel(b *testing.B) {
	b.SetParallelism(8)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		bo := startBoard()
		for pb.Next() {
			_ = bo.GetAllMoves(color.White, nil)
		}
	})
}

// BenchmarkEvaluateNoCache exercises EvaluateBoardNoCache directly, bypassing the
// evaluation map. This isolates the raw eval cost without cache effects.
func BenchmarkEvaluateNoCache(b *testing.B) {
	bo := startBoard()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ai.EvaluateBoardNoCache(bo, color.White)
	}
}

// BenchmarkTTReadParallel measures how the transposition table scales under the parallel
// reads that ABDADA generates. With Lock() (instead of RLock()), all readers serialize.
func BenchmarkTTReadParallel(b *testing.B) {
	tt := util.NewConcurrentBoardMap()
	bo := &board.Board{}
	bo.ResetDefault()
	h := bo.Hash()
	tt.Store(&h, color.White, 42)

	b.SetParallelism(8)
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		bLocal := &board.Board{}
		bLocal.ResetDefault()
		for pb.Next() {
			hLocal := bLocal.Hash()
			tt.Read(&hLocal, color.White)
		}
	})
}

// BenchmarkApplyMove measures the cost of the applyMove inner loop: Copy + MakeMove.
// This happens for every move ABDADA explores.
func BenchmarkApplyMove(b *testing.B) {
	bo := startBoard()
	moves := bo.GetAllMoves(color.White, nil)
	if len(*moves) == 0 {
		b.Fatal("no moves")
	}
	m := (*moves)[0]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		child := bo.Copy()
		board.MakeMove(&m, child)
	}
}
