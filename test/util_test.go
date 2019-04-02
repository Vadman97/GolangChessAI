package test

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMax(t *testing.T) {
	assert.Equal(t, 5, util.MaxScore(5, 3))
	assert.Equal(t, 3, util.MaxScore(3, 3))
	assert.Equal(t, 5, util.MaxScore(3, 5))
	assert.Equal(t, 0, util.MaxScore(0, 0))
	assert.Equal(t, 0, util.MaxScore(0, -1))
	assert.Equal(t, -1, util.MaxScore(-1, -1))
	assert.Equal(t, 0, util.MaxScore(-1, 0))
	assert.Equal(t, -1, util.MaxScore(-1, -1))
	assert.Equal(t, -1, util.MaxScore(-2, -1))
	assert.Equal(t, -1, util.MaxScore(-1, -2))
}

func TestMin(t *testing.T) {
	assert.Equal(t, 3, util.MinScore(5, 3))
	assert.Equal(t, 3, util.MinScore(3, 3))
	assert.Equal(t, 3, util.MinScore(3, 5))
	assert.Equal(t, 0, util.MinScore(0, 0))
	assert.Equal(t, -1, util.MinScore(0, -1))
	assert.Equal(t, -1, util.MinScore(-1, -1))
	assert.Equal(t, -1, util.MinScore(-1, 0))
	assert.Equal(t, -1, util.MinScore(-1, -1))
	assert.Equal(t, -2, util.MinScore(-2, -1))
	assert.Equal(t, -2, util.MinScore(-1, -2))
}
