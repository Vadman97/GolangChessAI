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
	scoreMap       [NumSlices]map[uint64]map[uint64]map[uint64]map[uint64]map[byte]interface{}
	locks          [NumSlices]sync.RWMutex
	lockUsage      [NumSlices]uint64
	entriesWritten [NumSlices]uint64
}

func NewConcurrentBoardMap() *ConcurrentBoardMap {
	var m ConcurrentBoardMap
	for i := 0; i < NumSlices; i++ {
		if m.scoreMap[i] == nil {
			m.scoreMap[i] = make(map[uint64]map[uint64]map[uint64]map[uint64]map[byte]interface{})
		}
	}
	return &m
}

func HashToMapKey(hash *[33]byte) (idx [4]uint64) {
	for x := 0; x < 32; x += 8 {
		idx[x/8] = binary.BigEndian.Uint64((*hash)[x : x+8])
	}
	return idx
}

func (m *ConcurrentBoardMap) getLock(hash *[33]byte) (*sync.RWMutex, uint32) {
	var s uint32
	for i := 0; i < 28; i += 4 {
		s += (binary.BigEndian.Uint32(hash[i:i+4]) / NumSlices) % NumSlices
	}
	s = (s + uint32(hash[32])) % NumSlices
	atomic.AddUint64(&m.lockUsage[s], 1)
	return &m.locks[s], s
}

func (m *ConcurrentBoardMap) Store(hash *[33]byte, value interface{}) {
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
	m.entriesWritten[lockIdx]++
	m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]][(*hash)[32]] = value
}

func (m *ConcurrentBoardMap) Read(hash *[33]byte) (interface{}, bool) {
	idx := HashToMapKey(hash)

	lock, lockIdx := m.getLock(hash)
	lock.Lock()
	defer lock.Unlock()

	m1, ok := m.scoreMap[lockIdx][idx[0]]
	if ok {
		m2, ok := m1[idx[1]]
		if ok {
			m3, ok := m2[idx[2]]
			if ok {
				m4, ok := m3[idx[3]]
				if ok {
					v, ok := m4[(*hash)[32]]
					return v, ok
				}
			}
		}
	}

	return 0, false
}

func (m *ConcurrentBoardMap) PrintMetrics() {
	//fmt.Printf("Lock Usages: \n")
	totalLocks, totalEntries := uint64(0), uint64(0)
	for i := 0; i < NumSlices; i++ {
		//fmt.Printf("Slice #%d, Used #%d times\n", i, m.lockUsage[i])
		totalLocks += m.lockUsage[i]
		totalEntries += m.entriesWritten[i]
	}
	fmt.Printf("\tLock usages in map %d\n", totalLocks)
	fmt.Printf("\tTotal entries in map %d\n", totalEntries)
}
