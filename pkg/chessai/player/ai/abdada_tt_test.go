package ai

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
)

func TestABDADATTWriteUsesOriginalAlphaForExactScore(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	p := NewAIPlayer(color.White, &ABDADA{})
	p.TranspositionTableEnabled = true
	ab := &ABDADA{player: p}

	best := &ScoredMove{
		Move: location.Move{
			Start: location.NewLocation(1, 4),
			End:   location.NewLocation(3, 4),
		},
		Score: 42,
	}

	ab.syncTTWrite(b, color.White, 3, -100, 100, best)

	h := b.Hash()
	raw, ok := p.transpositionTable.Read(&h, color.White)
	if !ok {
		t.Fatal("expected TT entry")
	}
	entry := raw.(*transposition_table.TranspositionTableEntryABDADA)
	if entry.EntryType != transposition_table.TrueScore {
		t.Fatalf("expected exact TT score, got entry type %d", entry.EntryType)
	}
}

func TestABDADATTWriteDoesNotUnderflowProcessorCount(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	p := NewAIPlayer(color.White, &ABDADA{})
	p.TranspositionTableEnabled = true
	ab := &ABDADA{player: p}
	h := b.Hash()
	p.transpositionTable.Store(&h, color.White, &transposition_table.TranspositionTableEntryABDADA{
		Depth:         3,
		EntryType:     transposition_table.Unset,
		NumProcessors: 0,
	})

	best := &ScoredMove{
		Move: location.Move{
			Start: location.NewLocation(1, 4),
			End:   location.NewLocation(3, 4),
		},
		Score: 42,
	}

	ab.syncTTWrite(b, color.White, 3, -100, 100, best)

	raw, ok := p.transpositionTable.Read(&h, color.White)
	if !ok {
		t.Fatal("expected TT entry")
	}
	entry := raw.(*transposition_table.TranspositionTableEntryABDADA)
	if entry.NumProcessors != 0 {
		t.Fatalf("expected processor count to stay at 0, got %d", entry.NumProcessors)
	}
}

func TestNewPonderPlayerUsesRequestedColorAndIsolatedAlgorithm(t *testing.T) {
	p := NewAIPlayer(color.White, &ABDADA{NumThreads: 2})
	ponder := p.NewPonderPlayer(color.Black)

	if ponder.PlayerColor != color.Black {
		t.Fatalf("expected ponder player to search as black, got %d", ponder.PlayerColor)
	}
	if ponder == p {
		t.Fatal("expected distinct ponder player")
	}
	if ponder.Algorithm == p.Algorithm {
		t.Fatal("expected distinct algorithm instance")
	}
	if ponder.transpositionTable != p.transpositionTable {
		t.Fatal("expected ponder player to share transposition table")
	}
	if ponder.evaluationMap != p.evaluationMap {
		t.Fatal("expected ponder player to share evaluation cache")
	}
	abdada, ok := ponder.Algorithm.(*ABDADA)
	if !ok {
		t.Fatalf("expected ABDADA ponder algorithm, got %T", ponder.Algorithm)
	}
	if abdada.NumThreads != 2 {
		t.Fatalf("expected copied thread count, got %d", abdada.NumThreads)
	}
}

func TestABDADATTWriteSkippedOnAbort(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	p := NewAIPlayer(color.White, &ABDADA{})
	p.TranspositionTableEnabled = true
	p.abort = true
	ab := &ABDADA{player: p}

	best := &ScoredMove{
		Move: location.Move{
			Start: location.NewLocation(1, 4),
			End:   location.NewLocation(3, 4),
		},
		Score: 42,
	}

	ab.syncTTWrite(b, color.White, 3, -100, 100, best)

	h := b.Hash()
	if _, ok := p.transpositionTable.Read(&h, color.White); ok {
		t.Fatal("did not expect aborted search to write TT entry")
	}
}

func TestABDADATTWriteSkippedForOnEvaluationSentinel(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	p := NewAIPlayer(color.White, &ABDADA{})
	p.TranspositionTableEnabled = true
	ab := &ABDADA{player: p}

	best := &ScoredMove{
		Move: location.Move{
			Start: location.NewLocation(1, 4),
			End:   location.NewLocation(3, 4),
		},
		Score: OnEvaluation,
	}

	ab.syncTTWrite(b, color.White, 3, -100, 100, best)

	h := b.Hash()
	if _, ok := p.transpositionTable.Read(&h, color.White); ok {
		t.Fatal("did not expect OnEvaluation sentinel to be stored in TT")
	}
}

func TestABDADAResetRootSearchHeuristics(t *testing.T) {
	ab := &ABDADA{}
	move := location.Move{
		Start: location.NewLocation(1, 1),
		End:   location.NewLocation(2, 2),
	}
	ab.killers[3][0] = move
	ab.history[squareIdx(move.Start)][squareIdx(move.End)] = 99
	ab.countermove[squareIdx(move.Start)][squareIdx(move.End)] = move

	ab.resetRootSearchHeuristics()

	if !ab.killers[3][0].Start.Equals(ab.killers[3][0].End) {
		t.Fatalf("expected killer table to be reset, got %s", ab.killers[3][0])
	}
	if got := ab.historyScore(move); got != 0 {
		t.Fatalf("expected history table to be reset, got %d", got)
	}
	gotCounter := ab.countermove[squareIdx(move.Start)][squareIdx(move.End)]
	if !gotCounter.Start.Equals(gotCounter.End) {
		t.Fatalf("expected countermove table to be reset, got %s", gotCounter)
	}
}

func TestABDADAStableDepthMoveKeepsPreviousMoveOnLargeRegression(t *testing.T) {
	safeMove := location.Move{
		Start: location.NewLocation(5, 3),
		End:   location.NewLocation(6, 2),
	}
	regressedMove := location.Move{
		Start: location.NewLocation(6, 1),
		End:   location.NewLocation(5, 2),
	}

	got := stableDepthMove(
		ScoredMove{Move: safeMove, Score: -138},
		ScoredMove{Move: regressedMove, Score: -540},
	)

	if !got.Move.Equals(&safeMove) {
		t.Fatalf("expected stable move %s, got %s", safeMove, got.Move)
	}
	if got.Score != -540 {
		t.Fatalf("expected regressed score to be preserved, got %d", got.Score)
	}
}

func TestABDADAStableDepthMoveAcceptsSmallRegression(t *testing.T) {
	prevMove := location.Move{
		Start: location.NewLocation(5, 3),
		End:   location.NewLocation(6, 2),
	}
	nextMove := location.Move{
		Start: location.NewLocation(6, 1),
		End:   location.NewLocation(5, 2),
	}

	got := stableDepthMove(
		ScoredMove{Move: prevMove, Score: -138},
		ScoredMove{Move: nextMove, Score: -250},
	)

	if !got.Move.Equals(&nextMove) {
		t.Fatalf("expected newer move %s, got %s", nextMove, got.Move)
	}
}
