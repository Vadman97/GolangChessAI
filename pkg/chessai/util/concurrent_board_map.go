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

type ConcurrentBoardMap struct {
	scoreMap                       [NumSlices]map[uint64]map[uint64]map[uint64]map[uint64]map[byte]interface{}
	locks                          [NumSlices]sync.RWMutex
	lockUsage                      [NumSlices]uint64
	numHits, numWrites, numQueries [NumSlices]uint64
}

type BoardHash = [33]byte

func NewConcurrentBoardMap() *ConcurrentBoardMap {
	var m ConcurrentBoardMap
	for i := 0; i < NumSlices; i++ {
		if m.scoreMap[i] == nil {
			m.scoreMap[i] = make(map[uint64]map[uint64]map[uint64]map[uint64]map[byte]interface{})
		}
	}
	return &m
}

func HashToMapKey(hash *BoardHash) (idx [4]uint64) {
	for x := 0; x < 32; x += 8 {
		idx[x/8] = binary.BigEndian.Uint64((*hash)[x : x+8])
	}
	return idx
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
	idx := HashToMapKey(hash)

	lock, lockIdx := m.getLock(hash)
	lock.Lock()
	defer lock.Unlock()

	_, ok := m.scoreMap[lockIdx][idx[0]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]] = make(map[uint64]map[uint64]map[uint64]map[byte]interface{})
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]] = make(map[uint64]map[uint64]map[byte]interface{})
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]] = make(map[uint64]map[byte]interface{})
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]] = make(map[byte]interface{})
	}
	m.numWrites[lockIdx]++
	m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]][(*hash)[32]] = value
}

func (m *ConcurrentBoardMap) Read(hash *BoardHash) (interface{}, bool) {
	idx := HashToMapKey(hash)

	lock, lockIdx := m.getLock(hash)
	lock.Lock()
	defer lock.Unlock()
	m.numQueries[lockIdx]++

	m1, ok := m.scoreMap[lockIdx][idx[0]]
	if ok {
		m2, ok := m1[idx[1]]
		if ok {
			m3, ok := m2[idx[2]]
			if ok {
				m4, ok := m3[idx[3]]
				if ok {
					v, ok := m4[(*hash)[32]]
					if ok {
						m.numHits[lockIdx]++
					}
					return v, ok
				}
			}
		}
	}

	return 0, false
}

func (m *ConcurrentBoardMap) PrintMetrics() (result string) {
	totalLockUsage := uint64(0)
	totalHits := uint64(0)
	totalReads := uint64(0)
	totalWrites := uint64(0)
	for i := 0; i < NumSlices; i++ {
		//result += fmt.Sprintf("Slice #%d, Used #%d times\n", i, m.lockUsage[i])
		totalLockUsage += m.lockUsage[i]
		totalHits += m.numHits[i]
		totalReads += m.numQueries[i]
		totalWrites += m.numWrites[i]
	}
	result += fmt.Sprintf("\tTotal entries in map %d. Reads %d. Writes %d\n", totalWrites, totalReads, totalWrites)
	result += fmt.Sprintf("\tHit ratio %f%% (%d/%d)\n", 100.0*float64(totalHits)/float64(totalReads),
		totalHits, totalReads)
	result += fmt.Sprintf("\tRead ratio %f%%\n", 100.0*float64(totalReads)/float64(totalReads+totalWrites))
	result += fmt.Sprintf("\tLock usages in map %d\n", totalLockUsage)
	return
}
