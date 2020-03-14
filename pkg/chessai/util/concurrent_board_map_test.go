package util

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestConcurrentBoardMap_StoreUpdateReadABDADA(t *testing.T) {
	tt := NewConcurrentBoardMap()

	var h BoardHash
	r := rand.New(rand.NewSource(config.Get().TestRandSeed))
	r.Read(h[:])

	ttEntry := transposition_table.TranspositionTableEntryABDADA{}
	tt.Store(&h, color.Black, &ttEntry)
	ttEntry.NumProcessors++

	assert.Equal(t, uint16(1), ttEntry.NumProcessors)

	e, _ := tt.Read(&h, color.Black)
	readEntry := e.(*transposition_table.TranspositionTableEntryABDADA)

	assert.Equal(t, uint16(1), readEntry.NumProcessors)
	readEntry.NumProcessors++

	assert.Equal(t, uint16(2), readEntry.NumProcessors)
	assert.Equal(t, uint16(2), ttEntry.NumProcessors)

	newEntry := transposition_table.TranspositionTableEntryABDADA{}
	tt.Store(&h, color.Black, &newEntry)

	assert.Equal(t, uint16(2), readEntry.NumProcessors)
	assert.Equal(t, uint16(2), ttEntry.NumProcessors)

	e, _ = tt.Read(&h, color.Black)
	readEntry = e.(*transposition_table.TranspositionTableEntryABDADA)

	assert.Equal(t, uint16(0), readEntry.NumProcessors)
}
