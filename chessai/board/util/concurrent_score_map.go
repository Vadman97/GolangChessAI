package util

import (
	"encoding/binary"
	"errors"
	"sync"
)

type ConcurrentScoreMap struct {
	// TODO(Vadim) optimize via locks for each map to avoid collisions...
	scoreMap map[uint64]map[uint64]map[uint64]map[uint64]map[byte]uint32
	lock     sync.RWMutex
}

func getIdx(hash *[33]byte) *[]uint64 {
	idx := make([]uint64, 4)
	for x := 0; x < 32; x += 8 {
		idx[x/8] = binary.BigEndian.Uint64((*hash)[x : x+8])
	}
	return &idx
}

func (m *ConcurrentScoreMap) Store(hash *[33]byte, score uint32) {
	m.lock.Lock()
	defer m.lock.Unlock()

	idx := *getIdx(hash)

	if m.scoreMap == nil {
		m.scoreMap = make(map[uint64]map[uint64]map[uint64]map[uint64]map[byte]uint32)
	}

	_, ok := m.scoreMap[idx[0]]
	if !ok {
		m.scoreMap[idx[0]] = make(map[uint64]map[uint64]map[uint64]map[byte]uint32)
	}
	_, ok = m.scoreMap[idx[0]][idx[1]]
	if !ok {
		m.scoreMap[idx[0]][idx[1]] = make(map[uint64]map[uint64]map[byte]uint32)
	}
	_, ok = m.scoreMap[idx[0]][idx[1]][idx[2]]
	if !ok {
		m.scoreMap[idx[0]][idx[1]][idx[2]] = make(map[uint64]map[byte]uint32)
	}
	_, ok = m.scoreMap[idx[0]][idx[1]][idx[2]][idx[3]]
	if !ok {
		m.scoreMap[idx[0]][idx[1]][idx[2]][idx[3]] = make(map[byte]uint32)
	}

	m.scoreMap[idx[0]][idx[1]][idx[2]][idx[3]][hash[32]] = score
}

func (m *ConcurrentScoreMap) Read(hash *[33]byte) (uint32, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	idx := *getIdx(hash)
	m1, ok := m.scoreMap[idx[0]]
	if ok {
		m2, ok := m1[idx[1]]
		if ok {
			m3, ok := m2[idx[2]]
			if ok {
				m4, ok := m3[idx[3]]
				if ok {
					return m4[hash[32]], nil
				}
			}
		}
	}

	return 0, errors.New("hash not found")
}
