package transposition_table

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"sync"
)

type TranspositionTableEntryABMemory struct {
	Lower, Upper int
	BestMove     location.Move
}

const (
	Unset      = byte(iota)
	UpperBound = byte(iota)
	LowerBound = byte(iota)
	TrueScore  = byte(iota)
)

type TranspositionTableEntryABDADA struct {
	Score    int
	BestMove location.Move

	// Flag that identifies the entry
	EntryType byte

	// Length of subtree upon which score is based.
	Depth uint16

	// The number of processors currently evaluating the node related to the transposition table entry
	NumProcessors uint16

	Lock sync.Mutex
}

type TranspositionTableEntryJamboree struct {
	Score    int
	BestMove location.Move
	Depth    uint16
	Lock     sync.Mutex
}
