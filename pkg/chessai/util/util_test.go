package util

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

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
