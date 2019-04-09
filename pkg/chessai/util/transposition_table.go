package util

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
)

type TranspositionTableEntry struct {
	Lower, Upper int
	BestMove     location.Move
}

type TranspositionTable struct {
	entryMap          map[BoardHash]map[byte]*TranspositionTableEntry
	numStored         int
	numReads, numHits int
}

func NewTranspositionTable() *TranspositionTable {
	var m TranspositionTable
	if m.entryMap == nil {
		m.entryMap = make(map[BoardHash]map[byte]*TranspositionTableEntry)
	}
	return &m
}

/*
	Note: Transposition table does not support concurrent read/write at the moment
*/
func (m *TranspositionTable) Store(hash *BoardHash, currentTurn byte, entry *TranspositionTableEntry) {
	_, ok := m.entryMap[*hash]
	if !ok {
		m.entryMap[*hash] = make(map[byte]*TranspositionTableEntry)
	}
	m.entryMap[*hash][currentTurn] = entry
	m.numStored++
}

func (m *TranspositionTable) Read(hash *BoardHash, currentTurn byte) (*TranspositionTableEntry, bool) {
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
