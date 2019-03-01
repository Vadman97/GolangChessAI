package board

import (
	"ChessAI3/chessai/board/piece"
)

type Knight struct {
	Location Location
	Color    byte
}

func (r *Knight) GetChar() rune {
	return piece.KnightChar
}

func (r *Knight) GetPieceType() byte {
	return piece.KnightType
}

func (r *Knight) GetColor() byte {
	return r.Color
}

func (r *Knight) SetColor(color byte) {
	r.Color = color
}

func (r *Knight) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Knight) GetPosition() Location {
	return r.Location
}

func (r *Knight) GetMoves(board *Board) *[]Move {
	return nil
}
