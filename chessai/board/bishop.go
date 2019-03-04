package board

import (
	"ChessAI3/chessai/board/piece"
)

type Bishop struct {
	Location Location
	Color    byte
}

func (r *Bishop) GetChar() rune {
	return piece.BishopChar
}

func (r *Bishop) GetPieceType() byte {
	return piece.BishopType
}

func (r *Bishop) GetColor() byte {
	return r.Color
}

func (r *Bishop) SetColor(color byte) {
	r.Color = color
}

func (r *Bishop) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Bishop) GetPosition() Location {
	return r.Location
}

func (r *Bishop) GetMoves(board *Board) *[]Move {
	return nil
}

func (r *Bishop) Move(m *Move, b *Board) {}
