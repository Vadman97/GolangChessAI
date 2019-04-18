package transposition_table

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
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

type TranspositionTable struct {
	entryMap          map[util.BoardHash]map[color.Color]interface{}
	numStored         int
	numReads, numHits int

	lock sync.RWMutex
}

func NewTranspositionTable() *TranspositionTable {
	var m TranspositionTable
	if m.entryMap == nil {
		m.entryMap = make(map[util.BoardHash]map[color.Color]interface{})
	}
	return &m
}

/*
	Note: Transposition table does not support concurrent read/write at the moment
*/
func (m *TranspositionTable) Store(hash *util.BoardHash, currentTurn color.Color, entry interface{}) {
	m.lock.Lock()
	defer m.lock.Unlock()

	_, ok := m.entryMap[*hash]
	if !ok {
		m.entryMap[*hash] = make(map[color.Color]interface{})
	}
	m.entryMap[*hash][currentTurn] = entry
	m.numStored++
}

func (m *TranspositionTable) Read(hash *util.BoardHash, currentTurn color.Color) (interface{}, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	m.numReads++

	m1, ok := m.entryMap[*hash]
	if ok {
		v, ok := m1[currentTurn]
		if ok {
			m.numHits++
			return v, true
		}
	}
	return nil, false
}

func (m *TranspositionTable) PrintMetrics() (result string) {
	result += fmt.Sprintf("\tTotal entries in transposition table %d\n", m.numStored)
	result += fmt.Sprintf("\tHit ratio %f%% (%d/%d)\n", 100.0*float64(m.numHits)/float64(m.numReads),
		m.numHits, m.numReads)
	return
}
