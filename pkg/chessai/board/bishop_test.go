package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBishop_GetChar(t *testing.T) {
	bishop := PieceFromType(piece.BishopType).(*Bishop)
	assert.Equal(t, 'B', bishop.GetChar())
}
