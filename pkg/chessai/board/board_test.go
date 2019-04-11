package board

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"log"
	"math/rand"
	"path"
	"reflect"
	"testing"
	"time"
)

func TestBoardMove(t *testing.T) {
	board2 := Board{}
	board2.SetPiece(util.End, &Rook{})
	board2.SetPiece(util.Start, &Rook{})
	startPiece := board2.GetPiece(util.Start)
	startPiece.SetPosition(util.End)
	MakeMove(&location.Move{
		Start: util.Start,
		End:   util.End,
	}, &board2)
	assert.Nil(t, board2.GetPiece(util.Start))
	assert.Equal(t, startPiece, board2.GetPiece(util.End))
	assert.Equal(t, util.End, board2.GetPiece(util.End).GetPosition())
}

func TestBoardMoveClear(t *testing.T) {
	board2 := Board{}
	assert.Panics(t, func() {
		for i := 0; i < 3; i++ {
			MakeMove(&location.Move{
				Start: util.Start,
				End:   util.End,
			}, &board2)
		}
	})
}

func TestBoardFlags(t *testing.T) {
	board2 := Board{}
	for i := 0; i < FlagRightRookMoved; i++ {
		for c := 0; c < 2; c++ {
			assert.False(t, board2.GetFlag(byte(i), byte(c)))
		}
	}
	for i := 0; i < FlagRightRookMoved; i++ {
		for c := 0; c < 2; c++ {
			board2.SetFlag(byte(i), byte(c), true)
			assert.True(t, board2.GetFlag(byte(i), byte(c)))
			for i2 := 0; i2 < FlagRightRookMoved; i2++ {
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
	bo1 := Board{}
	bo2 := Board{}
	bo1.ResetDefault()
	bo1.SetFlag(FlagCastled, color.Black, true)
	bo1.SetFlag(FlagRightRookMoved, color.Black, true)
	bo1.SetFlag(FlagRightRookMoved, color.White, true)
	bo1.SetFlag(FlagLeftRookMoved, color.White, true)
	assert.False(t, bo1.Equals(&bo2))
	assert.False(t, bo2.Equals(&bo1))
	bo2 = *bo1.Copy()
	assert.True(t, bo1.Equals(&bo2))
	assert.True(t, bo2.Equals(&bo1))
}

func TestBoardResetDefault(t *testing.T) {
	bo1 := Board{}
	bo2 := Board{}
	bo1.ResetDefault()
	bo2.ResetDefaultSlow()
	assert.True(t, bo1.Equals(&bo2))
	assert.True(t, bo2.Equals(&bo1))
	bo2.SetFlag(FlagCastled, color.Black, true)
	assert.False(t, bo1.Equals(&bo2))
	assert.False(t, bo2.Equals(&bo1))
}

func TestBoardHash(t *testing.T) {
	bo1 := Board{}
	bo2 := Board{}
	bo1.ResetDefault()
	bo1.SetFlag(FlagCastled, color.Black, true)
	bo1.SetFlag(FlagRightRookMoved, color.Black, true)
	bo1.SetFlag(FlagRightRookMoved, color.White, true)
	bo1.SetFlag(FlagLeftRookMoved, color.White, true)
	assert.False(t, reflect.DeepEqual(bo1.Hash(), bo2.Hash()))
	bo2 = *bo1.Copy()
	assert.True(t, reflect.DeepEqual(bo1.Hash(), bo2.Hash()))
}

func TestBoardHashLookupParallel(t *testing.T) {
	const (
		NumThreads = 8
		NumOps     = 1000
	)
	scoreMap := util.NewConcurrentBoardMap()

	done := make([]chan int, NumThreads)
	for tIdx := 0; tIdx < NumThreads; tIdx++ {
		done[tIdx] = make(chan int)
		go func(thread int) {
			bo1 := Board{}
			bo1.TestRandGen = rand.New(rand.NewSource(time.Now().UnixNano() + int64(thread)))
			numStores := 0
			for i := 0; i < NumOps; i++ {
				bo1.RandomizeIllegal()
				hash := bo1.Hash()
				r := bo1.TestRandGen.Int31()
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

func TestBoardColorFromChar(t *testing.T) {
	assert.Equal(t, color.Black, ColorFromChar('B'))
	assert.Equal(t, color.White, ColorFromChar('W'))
	assert.Equal(t, byte(0xFF), ColorFromChar('a'))
}

const boardsDirectory = "board_test"

func TestBoard_IsInCheckmate(t *testing.T) {
	b := Board{}
	lines, _ := util.LoadBoardFile(path.Join(boardsDirectory, "black_is_in_checkmate.txt"))
	b.LoadBoardFromText(lines)
	fmt.Printf(b.Print())
	assert.False(t, b.IsInCheckmate(color.White, nil))
	assert.True(t, b.IsInCheckmate(color.Black, nil))
}

// TODO(Alex) Stalemate, Checkmate Tests

func simulateGameMove(move *location.Move, b *Board) {
	lastMove := MakeMove(move, b)
	b.UpdateDrawCounter(lastMove)
}

func performFiftyDrawMoves(bIn *Board) *Board {
	b := bIn
	if b == nil {
		b = &Board{}
		b.ResetDefault()
	}

	for i := 0; i < 25; i++ {
		// Black Knight
		simulateGameMove(&location.Move{
			Start: location.NewLocation(0, 6),
			End:   location.NewLocation(2, 5),
		}, b)

		// White Knight
		simulateGameMove(&location.Move{
			Start: location.NewLocation(7, 6),
			End:   location.NewLocation(5, 5),
		}, b)

		// Undo Black Knight
		simulateGameMove(&location.Move{
			Start: location.NewLocation(2, 5),
			End:   location.NewLocation(0, 6),
		}, b)

		// Undo White Knight
		simulateGameMove(&location.Move{
			Start: location.NewLocation(5, 5),
			End:   location.NewLocation(7, 6),
		}, b)
	}

	return b
}

func TestFiftyMoveDraw(t *testing.T) {
	b := performFiftyDrawMoves(nil)
	assert.Equal(t, 100, b.MovesSinceNoDraw)
}

func TestFiftyMoveDrawResetByPawnMove(t *testing.T) {
	b := performFiftyDrawMoves(nil)
	assert.Equal(t, 100, b.MovesSinceNoDraw)

	simulateGameMove(&location.Move{
		Start: location.NewLocation(1, 0),
		End:   location.NewLocation(2, 0),
	}, b)
	assert.Equal(t, 0, b.MovesSinceNoDraw)
}

func TestFiftyMoveDrawResetByCapture(t *testing.T) {
	b := &Board{}
	b.ResetDefault()

	// Initialize two pawns in a capture position
	simulateGameMove(&location.Move{
		Start: location.NewLocation(1, 4),
		End:   location.NewLocation(3, 4),
	}, b)
	simulateGameMove(&location.Move{
		Start: location.NewLocation(6, 3),
		End:   location.NewLocation(4, 3),
	}, b)

	b = performFiftyDrawMoves(b)

	// 100 since pawn moves will reset the counter
	assert.Equal(t, 100, b.MovesSinceNoDraw)

	simulateGameMove(&location.Move{
		Start: location.NewLocation(3, 4),
		End:   location.NewLocation(4, 3),
	}, b)

	assert.Equal(t, 0, b.MovesSinceNoDraw)
}
