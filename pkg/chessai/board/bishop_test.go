package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBishop_GetChar(t *testing.T) {
	bishop := PieceFromType(piece.BishopType).(*Bishop)
	assert.Equal(t, 'B', bishop.GetChar())
}

func TestBishop_GetMovesOnlyFirstMove(t *testing.T) {
	bo1, _ := buildBoardWithInitialMoves(&[]location.Move{{
		Start: location.NewLocation(6, 3),
		End:   location.NewLocation(4, 3),
	}})
	whiteBishop := bo1.GetPiece(location.NewLocation(7, 2))
	moves := whiteBishop.GetMoves(bo1, true)
	assert.Equal(t, 1, len(*moves))
}
