package util

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	NumSlices = 8
)

type BoardHash = [33]byte

type ConcurrentBoardMap struct {
	scoreMap                       [NumSlices]map[BoardHash]interface{}
	locks                          [NumSlices]sync.RWMutex
	lockUsage                      [NumSlices]uint64
	numHits, numWrites, numQueries [NumSlices]uint64
}

func NewConcurrentBoardMap() *ConcurrentBoardMap {
	var m ConcurrentBoardMap
	for i := 0; i < NumSlices; i++ {
		if m.scoreMap[i] == nil {
			m.scoreMap[i] = make(map[BoardHash]interface{})
		}
	}
	return &m
}

func (m *ConcurrentBoardMap) getLock(hash *BoardHash) (*sync.RWMutex, uint32) {
	var s uint32
	for i := 0; i < 28; i += 4 {
		s += (binary.BigEndian.Uint32(hash[i:i+4]) / NumSlices) % NumSlices
	}
	s = (s + uint32(hash[32])) % NumSlices
	atomic.AddUint64(&m.lockUsage[s], 1)
	return &m.locks[s], s
}

func (m *ConcurrentBoardMap) Store(hash *BoardHash, value interface{}) {
	lock, lockIdx := m.getLock(hash)
	lock.Lock()
	defer lock.Unlock()

	m.numWrites[lockIdx]++
	m.scoreMap[lockIdx][*hash] = value
}

func (m *ConcurrentBoardMap) Read(hash *BoardHash) (interface{}, bool) {
	lock, lockIdx := m.getLock(hash)
	lock.Lock()
	defer lock.Unlock()
	m.numQueries[lockIdx]++

	v, ok := m.scoreMap[lockIdx][*hash]
	if ok {
		m.numHits[lockIdx]++
	}
	return v, ok
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
		totalLockUsage += m.lockUsage[i]
	}
	return totalLockUsage
}

func (m *ConcurrentBoardMap) GetTotalHits() uint64 {
	var totalHits uint64 = 0
	for i := 0; i < NumSlices; i++ {
		totalHits += m.numHits[i]
	}
	return totalHits
}

func (m *ConcurrentBoardMap) GetTotalReads() uint64 {
	var totalReads uint64 = 0
	for i := 0; i < NumSlices; i++ {
		totalReads += m.numQueries[i]
	}
	return totalReads
}

func (m *ConcurrentBoardMap) GetTotalWrites() uint64 {
	var totalWrites uint64 = 0
	for i := 0; i < NumSlices; i++ {
		totalWrites += m.numWrites[i]
	}
	return totalWrites
}

func (m *ConcurrentBoardMap) GetHitRatio() float64 {
	return 100.0 * float64(m.GetTotalHits()) / float64(m.GetTotalReads())
}

func (m *ConcurrentBoardMap) GetReadRatio() float64 {
	return 100.0 * float64(m.GetTotalReads()) / float64(m.GetTotalReads()+m.GetTotalWrites())
}
