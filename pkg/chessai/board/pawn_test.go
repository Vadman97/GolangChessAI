package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPawn_GetChar(t *testing.T) {
	pawn := PieceFromType(piece.PawnType).(*Pawn)
	assert.Equal(t, 'P', pawn.GetChar())
}

func TestPawn_hasMovedBlack(t *testing.T) {
	bo1 := &Board{}
	bo1.ResetDefault()
	pawn := bo1.GetPiece(location.NewLocation(1, 1)).(*Pawn)
	assert.False(t, pawn.hasMoved())
}

func TestPawn_hasMovedWhite(t *testing.T) {
	bo1, _ := buildBoardWithInitialMoves(&[]location.Move{
		{
			Start: location.NewLocation(6, 3),
			End:   location.NewLocation(5, 3),
		},
	})
	pawn := bo1.GetPiece(location.NewLocation(5, 3)).(*Pawn)
	assert.True(t, pawn.hasMoved())
}

func TestPawn_hasMovedInvalidColor(t *testing.T) {
	pawn := PieceFromType(piece.PawnType).(*Pawn)
	pawn.SetColor(4)
	assert.True(t, pawn.hasMoved())
}

func TestPawn_forwardWithInvalidColor(t *testing.T) {
	pawn := PieceFromType(piece.PawnType).(*Pawn)
	pawn.SetColor(5)
	assert.Panics(t, func() { pawn.forward(1) })
}

func TestPawn_Promote(t *testing.T) {
	bo1 := &Board{}
	bo1.ResetDefault()
	startLoc := location.NewLocation(6, 3)
	endLoc := location.NewLocation(5, 3)
	endLoc.CreatePawnPromotion(piece.PawnPromotionOptions[0])
	pawn := bo1.GetPiece(startLoc).(*Pawn)
	move := &location.Move{Start: startLoc, End: endLoc}
	assert.Panics(t, func() { pawn.Promote(bo1, move) })
}
