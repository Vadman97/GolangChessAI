package util

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

func TestMax(t *testing.T) {
	assert.Equal(t, 5, MaxScore(5, 3))
	assert.Equal(t, 3, MaxScore(3, 3))
	assert.Equal(t, 5, MaxScore(3, 5))
	assert.Equal(t, 0, MaxScore(0, 0))
	assert.Equal(t, 0, MaxScore(0, -1))
	assert.Equal(t, -1, MaxScore(-1, -1))
	assert.Equal(t, 0, MaxScore(-1, 0))
	assert.Equal(t, -1, MaxScore(-1, -1))
	assert.Equal(t, -1, MaxScore(-2, -1))
	assert.Equal(t, -1, MaxScore(-1, -2))
}

func TestMin(t *testing.T) {
	assert.Equal(t, 3, MinScore(5, 3))
	assert.Equal(t, 3, MinScore(3, 3))
	assert.Equal(t, 3, MinScore(3, 5))
	assert.Equal(t, 0, MinScore(0, 0))
	assert.Equal(t, -1, MinScore(0, -1))
	assert.Equal(t, -1, MinScore(-1, -1))
	assert.Equal(t, -1, MinScore(-1, 0))
	assert.Equal(t, -1, MinScore(-1, -1))
	assert.Equal(t, -2, MinScore(-2, -1))
	assert.Equal(t, -2, MinScore(-1, -2))
}

func TestRandShuffleMoves(t *testing.T) {
	rand.Seed(config.Get().TestRandSeed)
	moves := []location.Move{{
		Start: location.NewLocation(3, 5),
		End:   location.NewLocation(2, 4),
	}, {
		Start: location.NewLocation(0, 0),
		End:   location.NewLocation(1, 1),
	}, {
		Start: location.NewLocation(3, 3),
		End:   location.NewLocation(4, 4),
	},
	}
	shuffled := RandShuffleMoves(moves)
	assert.NotEqual(t, moves, shuffled)
}
