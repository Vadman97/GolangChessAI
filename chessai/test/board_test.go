package test

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/util"
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

var start = board.Location{Row: 2, Col: 5}
var end = board.Location{Row: 4, Col: 6}

func TestBoardMove(t *testing.T) {
	board2 := board.Board{}
	board2.SetPiece(end, &board.Rook{})
	board2.SetPiece(start, &board.Rook{})
	startPiece := board2.GetPiece(start)
	startPiece.SetPosition(end)
	board.MakeMove(&board.Move{
		Start: start,
		End:   end,
	}, &board2)
	assert.Nil(t, board2.GetPiece(start))
	assert.Equal(t, startPiece, board2.GetPiece(end))
	assert.Equal(t, end, board2.GetPiece(end).GetPosition())
}

func TestBoardMoveClear(t *testing.T) {
	board2 := board.Board{}
	assert.Panics(t, func() {
		for i := 0; i < 3; i++ {
			board.MakeMove(&board.Move{
				Start: start,
				End:   end,
			}, &board2)
		}
	})
}

func TestBoardFlags(t *testing.T) {
	board2 := board.Board{}
	for i := 0; i < board.FlagRightRookMoved; i++ {
		for c := 0; c < 2; c++ {
			assert.False(t, board2.GetFlag(byte(i), byte(c)))
		}
	}
	for i := 0; i < board.FlagRightRookMoved; i++ {
		for c := 0; c < 2; c++ {
			board2.SetFlag(byte(i), byte(c), true)
			assert.True(t, board2.GetFlag(byte(i), byte(c)))
			for i2 := 0; i2 < board.FlagRightRookMoved; i2++ {
				for c2 := 0; c2 < 2; c2++ {
					if i != i2 && c != c2 {
						assert.False(t, board2.GetFlag(byte(i2), byte(c2)))
					}
				}
			}
			board2.SetFlag(byte(i), byte(c), false)
			assert.False(t, board2.GetFlag(byte(i), byte(c)))
		}
	}
}

func TestBoardSetAndCopy(t *testing.T) {
	bo1 := board.Board{}
	bo2 := board.Board{}
	bo1.ResetDefault()
	bo1.SetFlag(board.FlagCastled, color.Black, true)
	bo1.SetFlag(board.FlagRightRookMoved, color.Black, true)
	bo1.SetFlag(board.FlagRightRookMoved, color.White, true)
	bo1.SetFlag(board.FlagLeftRookMoved, color.White, true)
	assert.False(t, bo1.Equals(&bo2))
	assert.False(t, bo2.Equals(&bo1))
	bo2 = *bo1.Copy()
	assert.True(t, bo1.Equals(&bo2))
	assert.True(t, bo2.Equals(&bo1))
}

func TestBoardResetDefault(t *testing.T) {
	bo1 := board.Board{}
	bo2 := board.Board{}
	bo1.ResetDefault()
	bo2.ResetDefaultSlow()
	assert.True(t, bo1.Equals(&bo2))
	assert.True(t, bo2.Equals(&bo1))
	bo2.SetFlag(board.FlagCastled, color.Black, true)
	assert.False(t, bo1.Equals(&bo2))
	assert.False(t, bo2.Equals(&bo1))
}

func TestBoardHash(t *testing.T) {
	bo1 := board.Board{}
	bo2 := board.Board{}
	bo1.ResetDefault()
	bo1.SetFlag(board.FlagCastled, color.Black, true)
	bo1.SetFlag(board.FlagRightRookMoved, color.Black, true)
	bo1.SetFlag(board.FlagRightRookMoved, color.White, true)
	bo1.SetFlag(board.FlagLeftRookMoved, color.White, true)
	assert.False(t, reflect.DeepEqual(bo1.Hash(), bo2.Hash()))
	bo2 = *bo1.Copy()
	assert.True(t, reflect.DeepEqual(bo1.Hash(), bo2.Hash()))
}

func TestBoardHashLookupParallel(t *testing.T) {
	const (
		NumThreads = 8
		NumOps     = 10000
	)
	scoreMap := util.NewConcurrentScoreMap()

	done := make([]chan int, NumThreads)
	for tIdx := 0; tIdx < NumThreads; tIdx++ {
		done[tIdx] = make(chan int)
		go func(thread int) {
			bo1 := board.Board{}
			bo1.TestRandGen = rand.New(rand.NewSource(time.Now().UnixNano() + int64(thread)))
			numStores := 0
			for i := 0; i < NumOps; i++ {
				bo1.RandomizeIllegal()
				hash := bo1.Hash()
				r := bo1.TestRandGen.Uint32()
				_, ok := scoreMap.Read(&hash)
				if !ok {
					scoreMap.Store(&hash, r)
					score, _ := scoreMap.Read(&hash)
					assert.Equal(t, r, score)
					numStores++
				}
			}
			done[thread] <- numStores
		}(tIdx)
	}
	start := time.Now()
	totalNumStores := 0
	for tIdx := 0; tIdx < NumThreads; tIdx++ {
		totalNumStores += <-done[tIdx]
	}
	duration := time.Now().Sub(start)
	timePerOp := duration.Nanoseconds() / int64(totalNumStores)
	pSuccess := 100.0 * float64(totalNumStores) / (NumOps * NumThreads)
	log.Printf("Parallel randomize,hash,write,read %d ops with %d us/loop. %.1f%% stores successful (%d)\n",
		NumOps*NumThreads, timePerOp, pSuccess, totalNumStores)
	//scoreMap.PrintMetrics()
}

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
		board2.SetPiece(start, &board.Rook{})
	}
}

func BenchmarkGetPiece(b *testing.B) {
	board2 := board.Board{}
	board2.SetPiece(start, &board.Rook{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.GetPiece(start)
	}
}

func BenchmarkBoardMove(b *testing.B) {
	board2 := board.Board{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		board2.SetPiece(end, &board.Rook{})
		board2.SetPiece(start, &board.Rook{})
		board.MakeMove(&board.Move{
			Start: start,
			End:   end,
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
