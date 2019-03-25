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
	entryMap          map[uint64]map[uint64]map[uint64]map[uint64]map[byte]*TranspositionTableEntry
	numStored         int
	numReads, numHits int
}

func NewTranspositionTable() *TranspositionTable {
	var m TranspositionTable
	if m.entryMap == nil {
		m.entryMap = make(map[uint64]map[uint64]map[uint64]map[uint64]map[byte]*TranspositionTableEntry)
	}
	return &m
}

func (m *TranspositionTable) Store(hash *[33]byte, entry *TranspositionTableEntry) {
	idx := HashToMapKey(hash)

	_, ok := m.entryMap[idx[0]]
	if !ok {
		m.entryMap[idx[0]] = make(map[uint64]map[uint64]map[uint64]map[byte]*TranspositionTableEntry)
	}
	_, ok = m.entryMap[idx[0]][idx[1]]
	if !ok {
		m.entryMap[idx[0]][idx[1]] = make(map[uint64]map[uint64]map[byte]*TranspositionTableEntry)
	}
	_, ok = m.entryMap[idx[0]][idx[1]][idx[2]]
	if !ok {
		m.entryMap[idx[0]][idx[1]][idx[2]] = make(map[uint64]map[byte]*TranspositionTableEntry)
	}
	_, ok = m.entryMap[idx[0]][idx[1]][idx[2]][idx[3]]
	if !ok {
		m.entryMap[idx[0]][idx[1]][idx[2]][idx[3]] = make(map[byte]*TranspositionTableEntry)
	}

	m.entryMap[idx[0]][idx[1]][idx[2]][idx[3]][(*hash)[32]] = entry
	m.numStored++
}

func (m *TranspositionTable) Read(hash *[33]byte) (*TranspositionTableEntry, bool) {
	idx := HashToMapKey(hash)
	m.numReads++

	m1, ok := m.entryMap[idx[0]]
	if ok {
		m2, ok := m1[idx[1]]
		if ok {
			m3, ok := m2[idx[2]]
			if ok {
				m4, ok := m3[idx[3]]
				if ok {
					v, ok := m4[(*hash)[32]]
					if ok {
						m.numHits++
					}
					return v, ok
				}
			}
		}
	}

	return nil, false
}

func (m *TranspositionTable) PrintMetrics() {
	fmt.Printf("\tTotal entries in transposition table %d\n", m.numStored)
	fmt.Printf("\tHit ratio %f%% (%d/%d)\n", 100.0*float64(m.numHits)/float64(m.numReads),
		m.numHits, m.numReads)
}
