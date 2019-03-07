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

type ConcurrentScoreMap struct {
	scoreMap  [NumSlices]map[uint64]map[uint64]map[uint64]map[uint64]map[byte]int32
	locks     [NumSlices]sync.RWMutex
	lockUsage [NumSlices]uint64
}

func NewConcurrentScoreMap() *ConcurrentScoreMap {
	var m ConcurrentScoreMap
	for i := 0; i < NumSlices; i++ {
		if m.scoreMap[i] == nil {
			m.scoreMap[i] = make(map[uint64]map[uint64]map[uint64]map[uint64]map[byte]int32)
		}
	}
	return &m
}

func getIdx(hash *[33]byte) (idx [4]uint64) {
	for x := 0; x < 32; x += 8 {
		idx[x/8] = binary.BigEndian.Uint64((*hash)[x : x+8])
	}
	return idx
}

func (m *ConcurrentScoreMap) getLock(hash *[33]byte) (*sync.RWMutex, uint32) {
	var s uint32
	for i := 0; i < 28; i += 4 {
		s += (binary.BigEndian.Uint32(hash[i:i+4]) / NumSlices) % NumSlices
	}
	s += uint32(hash[32]) % NumSlices
	atomic.AddUint64(&m.lockUsage[s], 1)
	return &m.locks[s], s
}

func (m *ConcurrentScoreMap) Store(hash *[33]byte, score int32) {
	idx := getIdx(hash)

	lock, lockIdx := m.getLock(hash)
	lock.Lock()
	defer lock.Unlock()

	_, ok := m.scoreMap[lockIdx][idx[0]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]] = make(map[uint64]map[uint64]map[uint64]map[byte]int32)
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]] = make(map[uint64]map[uint64]map[byte]int32)
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]] = make(map[uint64]map[byte]int32)
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]] = make(map[byte]int32)
	}

	m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]][(*hash)[32]] = score
}

func (m *ConcurrentScoreMap) Read(hash *[33]byte) (int32, bool) {
	idx := getIdx(hash)

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

func (m *ConcurrentScoreMap) PrintMetrics() {
	fmt.Printf("Lock Usages: \n")
	total := uint64(0)
	for i := 0; i < NumSlices; i++ {
		fmt.Printf("Slice #%d, Used #%d times\n", i, m.lockUsage[i])
		total += m.lockUsage[i]
	}
	fmt.Printf("Total entries in map %d\n", total)
}
