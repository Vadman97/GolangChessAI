package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKnight_GetChar(t *testing.T) {
	knight := PieceFromType(piece.KnightType).(*Knight)
	assert.Equal(t, 'N', knight.GetChar())
}
