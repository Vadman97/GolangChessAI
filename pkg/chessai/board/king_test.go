package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKing_GetChar(t *testing.T) {
	king := PieceFromType(piece.KingType).(*King)
	assert.Equal(t, 'K', king.GetChar())
}

func TestKing_GetCastleMovesLeftCastleOnlyFirstMove(t *testing.T) {
	bo1, _ := buildBoardWithInitialMoves(&[]location.Move{
		{
			Start: location.NewLocation(7, 1),
			End:   location.NewLocation(5, 1),
		},
		{
			Start: location.NewLocation(7, 2),
			End:   location.NewLocation(5, 2),
		},
	})
	whiteKing := bo1.GetPiece(location.NewLocation(7, 3)).(*King)
	moves := whiteKing.GetCastleMoves(bo1, true)
	assert.Equal(t, 1, len(*moves))
}

func TestKing_GetCastleMovesRightCastleOnlyFirstMove(t *testing.T) {
	bo1, _ := buildBoardWithInitialMoves(&[]location.Move{
		{
			Start: location.NewLocation(7, 6),
			End:   location.NewLocation(5, 6),
		},
		{
			Start: location.NewLocation(7, 5),
			End:   location.NewLocation(5, 5),
		},
		{
			Start: location.NewLocation(7, 4),
			End:   location.NewLocation(5, 4),
		},
	})
	whiteKing := bo1.GetPiece(location.NewLocation(7, 3)).(*King)
	moves := whiteKing.GetCastleMoves(bo1, true)
	assert.Equal(t, 1, len(*moves))
}

func TestKing_GetCastleMoves(t *testing.T) {
	bo1, _ := buildBoardWithInitialMoves(&[]location.Move{
		{
			Start: location.NewLocation(7, 6),
			End:   location.NewLocation(5, 6),
		},
		{
			Start: location.NewLocation(7, 5),
			End:   location.NewLocation(5, 5),
		},
		{
			Start: location.NewLocation(7, 4),
			End:   location.NewLocation(5, 4),
		},
		{
			Start: location.NewLocation(7, 1),
			End:   location.NewLocation(5, 1),
		},
		{
			Start: location.NewLocation(7, 2),
			End:   location.NewLocation(5, 2),
		},
	})
	whiteKing := bo1.GetPiece(location.NewLocation(7, 3)).(*King)
	moves := whiteKing.GetCastleMoves(bo1, false)
	assert.Equal(t, 2, len(*moves))
}

func TestKing_underAttackInvalidColor(t *testing.T) {
	bo1 := &Board{}
	bo1.ResetDefault()
	king := bo1.GetPiece(location.NewLocation(7, 3)).(*King)
	king.SetColor(4)
	assert.False(t, king.underAttack(location.NewLocation(2, 3), bo1))
}
