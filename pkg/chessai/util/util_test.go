package util

import (
	"github.com/stretchr/testify/assert"
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
