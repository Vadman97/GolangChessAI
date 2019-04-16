package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPawn_GetChar(t *testing.T) {
	pawn := PieceFromType(piece.PawnType).(*Pawn)
	assert.Equal(t, 'P', pawn.GetChar())
}
