package util

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
)

const (
	NumSlices = 256
)

type ConcurrentScoreMap struct {
	// TODO(Vadim) optimize via locks for each map to avoid collisions...
	scoreMap  [NumSlices]map[uint64]map[uint64]map[uint64]map[uint64]map[byte]uint32
	locks     [NumSlices]sync.RWMutex
	lockUsage [NumSlices]int
}

func NewConcurrentScoreMap() *ConcurrentScoreMap {
	var m ConcurrentScoreMap
	for i := 0; i < NumSlices; i++ {
		if m.scoreMap[i] == nil {
			m.scoreMap[i] = make(map[uint64]map[uint64]map[uint64]map[uint64]map[byte]uint32)
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
	s := binary.BigEndian.Uint32(hash[:8]) % NumSlices
	// TODO(Vadim) fix bug - pick on of these methods
	m.lockUsage[s]++
	return &m.locks[s], s

	//var lockIdx = (*hash)[0] % NumSlices
	var lockIdx, bitIdx uint32
	for i := uint32(0); i < uint32(math.Log2(NumSlices)); i++ {
		// 11 is arbitrary prime, 264 is length of hash in bits
		bitIdx = (bitIdx + 11*i + 7) % 264
		hashBit := (*hash)[bitIdx/8] & (1 << (bitIdx % 8))
		if hashBit != 0 {
			lockIdx |= 1 << i
		}
	}
	// TODO(Vadim) is there a way to verify that this results in uniformly distributed lockIdx?
	m.lockUsage[lockIdx]++
	return &m.locks[lockIdx], uint32(lockIdx)
}

func (m *ConcurrentScoreMap) Store(hash *[33]byte, score uint32) {
	idx := getIdx(hash)

	lock, lockIdx := m.getLock(hash)
	lock.Lock()
	defer lock.Unlock()

	_, ok := m.scoreMap[lockIdx][idx[0]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]] = make(map[uint64]map[uint64]map[uint64]map[byte]uint32)
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]] = make(map[uint64]map[uint64]map[byte]uint32)
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]] = make(map[uint64]map[byte]uint32)
	}
	_, ok = m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]]
	if !ok {
		m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]] = make(map[byte]uint32)
	}

	m.scoreMap[lockIdx][idx[0]][idx[1]][idx[2]][idx[3]][(*hash)[32]] = score
}

func (m *ConcurrentScoreMap) Read(hash *[33]byte) (uint32, error) {
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
					return m4[(*hash)[32]], nil
				}
			}
		}
	}

	return 0, errors.New("hash not found")
}

func (m *ConcurrentScoreMap) PrintMetrics() {
	fmt.Printf("Lock Usages: \n")
	for i := 0; i < NumSlices; i++ {
		fmt.Printf("Slice #%d, Used #%d times\n", i, m.lockUsage[i])
	}
}
