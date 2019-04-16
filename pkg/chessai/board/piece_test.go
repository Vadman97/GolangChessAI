package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetColorTypeRepr(t *testing.T) {
	whiteRook := PieceFromType(piece.RookType).(*Rook)
	whiteRook.SetColor(color.White)
	assert.Equal(t, "W_R", GetColorTypeRepr(whiteRook))
	blackRook := PieceFromType(piece.RookType).(*Rook)
	blackRook.SetColor(color.Black)
	assert.Equal(t, "B_R", GetColorTypeRepr(blackRook))
	whiteKnight := PieceFromType(piece.KnightType).(*Knight)
	whiteKnight.SetColor(color.White)
	assert.Equal(t, "W_N", GetColorTypeRepr(whiteKnight))
	blackKnight := PieceFromType(piece.KnightType).(*Knight)
	blackKnight.SetColor(color.Black)
	assert.Equal(t, "B_N", GetColorTypeRepr(blackKnight))
	whitePawn := PieceFromType(piece.PawnType).(*Pawn)
	whitePawn.SetColor(color.White)
	assert.Equal(t, "W_P", GetColorTypeRepr(whitePawn))
	blackPawn := PieceFromType(piece.PawnType).(*Pawn)
	blackPawn.SetColor(color.Black)
	assert.Equal(t, "B_P", GetColorTypeRepr(blackPawn))
	whiteBishop := PieceFromType(piece.BishopType).(*Bishop)
	whiteBishop.SetColor(color.White)
	assert.Equal(t, "W_B", GetColorTypeRepr(whiteBishop))
	blackBishop := PieceFromType(piece.BishopType).(*Bishop)
	blackBishop.SetColor(color.Black)
	assert.Equal(t, "B_B", GetColorTypeRepr(blackBishop))
	whiteQueen := PieceFromType(piece.QueenType).(*Queen)
	whiteQueen.SetColor(color.White)
	assert.Equal(t, "W_Q", GetColorTypeRepr(whiteQueen))
	blackQueen := PieceFromType(piece.QueenType).(*Queen)
	blackQueen.SetColor(color.Black)
	assert.Equal(t, "B_Q", GetColorTypeRepr(blackQueen))
	whiteKing := PieceFromType(piece.KingType).(*King)
	whiteKing.SetColor(color.White)
	assert.Equal(t, "W_K", GetColorTypeRepr(whiteKing))
	blackKing := PieceFromType(piece.KingType).(*King)
	blackKing.SetColor(color.Black)
	assert.Equal(t, "B_K", GetColorTypeRepr(blackKing))
}
