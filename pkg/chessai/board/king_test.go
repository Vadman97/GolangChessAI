package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
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
		{
			Start: location.NewLocation(7, 3),
			End:   location.NewLocation(5, 3),
		},
	})
	whiteKing := bo1.GetPiece(location.NewLocation(7, 4)).(*King)
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
	})
	whiteKing := bo1.GetPiece(location.NewLocation(7, 4)).(*King)
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
			Start: location.NewLocation(7, 1),
			End:   location.NewLocation(5, 1),
		},
		{
			Start: location.NewLocation(7, 2),
			End:   location.NewLocation(5, 2),
		},
		{
			Start: location.NewLocation(7, 3),
			End:   location.NewLocation(5, 3),
		},
	})
	whiteKing := bo1.GetPiece(location.NewLocation(7, 4)).(*King)
	moves := whiteKing.GetCastleMoves(bo1, false)
	assert.Equal(t, 2, len(*moves))
}
