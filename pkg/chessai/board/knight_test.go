package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKnight_GetChar(t *testing.T) {
	knight := PieceFromType(piece.KnightType).(*Knight)
	assert.Equal(t, 'N', knight.GetChar())
}
