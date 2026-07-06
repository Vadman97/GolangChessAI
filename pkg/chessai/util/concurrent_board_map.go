package util

import (
	"encoding/binary"
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"sync"
	"sync/atomic"
)

const (
	NumSlices = 256
)

type BoardHash = [33]byte

// perColorEntry holds the stored value for each side to move. A fixed-size
// array instead of an inner map: the old map[color.Color]interface{} allocated
// a fresh map per position (runtime.NewEmptyMap was ~28% of allocation CPU in
// search profiles). A nil slot means "not stored" — callers never store nil.
type perColorEntry = [color.NumColors]interface{}

type ConcurrentBoardMap struct {
	entryMap                       [NumSlices]map[BoardHash]perColorEntry
	locks                          [NumSlices]sync.RWMutex
	lockUsage                      [NumSlices]uint64
	numHits, numWrites, numQueries [NumSlices]uint64
}

func NewConcurrentBoardMap() *ConcurrentBoardMap {
	var m ConcurrentBoardMap
	for i := 0; i < NumSlices; i++ {
		if m.entryMap[i] == nil {
			m.entryMap[i] = make(map[BoardHash]perColorEntry)
		}
	}
	return &m
}

func (m *ConcurrentBoardMap) getLock(hash *BoardHash, currentTurn color.Color) (*sync.RWMutex, uint32) {
	var s uint32
	for i := 0; i < 28; i += 4 {
		s += (binary.BigEndian.Uint32(hash[i:i+4]) / NumSlices) % NumSlices
	}
	s = (s + uint32(hash[32]) + uint32(currentTurn)) % NumSlices
	atomic.AddUint64(&m.lockUsage[s], 1)
	return &m.locks[s], s
}

func (m *ConcurrentBoardMap) Store(hash *BoardHash, currentTurn color.Color, value interface{}) {
	if currentTurn >= color.NumColors {
		// The old inner map tolerated arbitrary color keys; keep invalid
		// colors graceful (drop) instead of indexing past the array.
		return
	}
	lock, lockIdx := m.getLock(hash, currentTurn)
	lock.Lock()
	defer lock.Unlock()

	atomic.AddUint64(&m.numWrites[lockIdx], 1)
	entry := m.entryMap[lockIdx][*hash]
	entry[currentTurn] = value
	m.entryMap[lockIdx][*hash] = entry
}

func (m *ConcurrentBoardMap) Read(hash *BoardHash, currentTurn color.Color) (interface{}, bool) {
	if currentTurn >= color.NumColors {
		return nil, false
	}
	lock, lockIdx := m.getLock(hash, currentTurn)
	lock.RLock()
	defer lock.RUnlock()
	atomic.AddUint64(&m.numQueries[lockIdx], 1)

	entry, ok := m.entryMap[lockIdx][*hash]
	if ok {
		if v := entry[currentTurn]; v != nil {
			atomic.AddUint64(&m.numHits[lockIdx], 1)
			return v, true
		}
	}
	return nil, false
}

func (m ConcurrentBoardMap) String() (result string) {
	totalLockUsage := m.GetTotalLockUsage()
	totalHits := m.GetTotalHits()
	totalReads := m.GetTotalReads()
	totalWrites := m.GetTotalWrites()
	result += fmt.Sprintf("\tTotal entries in map %d. Reads %d. Writes %d\n", totalWrites, totalReads, totalWrites)
	result += fmt.Sprintf("\tHit ratio %f%% (%d/%d)\n", m.GetHitRatio(), totalHits, totalReads)
	result += fmt.Sprintf("\tRead ratio %f%%\n", m.GetReadRatio())
	result += fmt.Sprintf("\tLock usages in map %d\n", totalLockUsage)
	return
}

func (m *ConcurrentBoardMap) GetTotalLockUsage() uint64 {
	var totalLockUsage uint64 = 0
	for i := 0; i < NumSlices; i++ {
		totalLockUsage += atomic.LoadUint64(&m.lockUsage[i])
	}
	return totalLockUsage
}

func (m *ConcurrentBoardMap) GetTotalHits() uint64 {
	var totalHits uint64 = 0
	for i := 0; i < NumSlices; i++ {
		totalHits += atomic.LoadUint64(&m.numHits[i])
	}
	return totalHits
}

func (m *ConcurrentBoardMap) GetTotalReads() uint64 {
	var totalReads uint64 = 0
	for i := 0; i < NumSlices; i++ {
		totalReads += atomic.LoadUint64(&m.numQueries[i])
	}
	return totalReads
}

func (m *ConcurrentBoardMap) GetTotalWrites() uint64 {
	var totalWrites uint64 = 0
	for i := 0; i < NumSlices; i++ {
		totalWrites += atomic.LoadUint64(&m.numWrites[i])
	}
	return totalWrites
}

func (m *ConcurrentBoardMap) GetHitRatio() float64 {
	return 100.0 * float64(m.GetTotalHits()) / float64(m.GetTotalReads())
}

func (m *ConcurrentBoardMap) GetReadRatio() float64 {
	return 100.0 * float64(m.GetTotalReads()) / float64(m.GetTotalReads()+m.GetTotalWrites())
}
