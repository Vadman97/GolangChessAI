package transposition_table

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTranspositionTable_StoreUpdateReadABDADA(t *testing.T) {
	tt := NewTranspositionTable()

	b := board.Board{}
	b.ResetDefault()
	b.RandomizeIllegal()
	h := b.Hash()

	ttEntry := TranspositionTableEntryABDADA{}
	tt.Store(&h, color.Black, &ttEntry)
	ttEntry.NumProcessors++

	assert.Equal(t, uint16(1), ttEntry.NumProcessors)

	e, _ := tt.Read(&h, color.Black)
	readEntry := e.(*TranspositionTableEntryABDADA)

	assert.Equal(t, uint16(1), readEntry.NumProcessors)
	readEntry.NumProcessors++

	assert.Equal(t, uint16(2), readEntry.NumProcessors)
	assert.Equal(t, uint16(2), ttEntry.NumProcessors)

	newEntry := TranspositionTableEntryABDADA{}
	tt.Store(&h, color.Black, &newEntry)

	assert.Equal(t, uint16(2), readEntry.NumProcessors)
	assert.Equal(t, uint16(2), ttEntry.NumProcessors)

	e, _ = tt.Read(&h, color.Black)
	readEntry = e.(*TranspositionTableEntryABDADA)

	assert.Equal(t, uint16(0), readEntry.NumProcessors)
}
