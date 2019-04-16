package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQueen_GetChar(t *testing.T) {
	queen := PieceFromType(piece.QueenType).(*Queen)
	assert.Equal(t, 'Q', queen.GetChar())
}
