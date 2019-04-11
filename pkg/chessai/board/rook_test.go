package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRook_IsStartingRow(t *testing.T) {
	rook := PieceFromType(piece.RookType).(*Rook)
	rook.SetColor(color.Black)
	rook.SetPosition(location.NewLocation(1, 5))
	assert.False(t, rook.IsStartingRow())
	rook.SetPosition(location.NewLocation(0, 5))
	assert.True(t, rook.IsStartingRow())

	rook.SetColor(color.White)
	rook.SetPosition(location.NewLocation(6, 5))
	assert.False(t, rook.IsStartingRow())
	rook.SetPosition(location.NewLocation(7, 5))
	assert.True(t, rook.IsStartingRow())
	rook.SetPosition(location.NewLocation(7, 5))
	assert.True(t, rook.IsStartingRow())

	rook.SetColor(2)
	assert.False(t, rook.IsStartingRow())
}

func TestRook_IsLeftRook(t *testing.T) {
	rook := PieceFromType(piece.RookType).(*Rook)
	rook.SetColor(color.Black)
	rook.SetPosition(location.NewLocation(5, 5))
	assert.False(t, rook.IsLeftRook())
	rook.SetPosition(location.NewLocation(5, 0))
	assert.True(t, rook.IsLeftRook())
	rook.SetPosition(location.NewLocation(5, Width))
	assert.False(t, rook.IsLeftRook())
}

func TestRook_IsRightRook(t *testing.T) {
	rook := PieceFromType(piece.RookType).(*Rook)
	rook.SetColor(color.Black)
	rook.SetPosition(location.NewLocation(5, 5))
	assert.False(t, rook.IsLeftRook())
	rook.SetPosition(location.NewLocation(5, 0))
	assert.False(t, rook.IsLeftRook())
	rook.SetPosition(location.NewLocation(5, Width))
	assert.True(t, rook.IsLeftRook())
}
