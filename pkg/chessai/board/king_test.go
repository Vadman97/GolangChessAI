package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKing_GetChar(t *testing.T) {
	king := PieceFromType(piece.KingType).(*King)
	assert.Equal(t, 'K', king.GetChar())
}
