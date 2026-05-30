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
