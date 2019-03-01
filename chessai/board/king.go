package board

import (
	"ChessAI3/chessai/board/piece"
)

type King struct {
	Location Location
	Color    byte
}

func (r *King) GetChar() rune {
	return piece.KingChar
}

func (r *King) GetPieceType() byte {
	return piece.KingType
}

func (r *King) GetColor() byte {
	return r.Color
}

func (r *King) SetColor(color byte) {
	r.Color = color
}

func (r *King) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *King) GetPosition() Location {
	return r.Location
}

func (r *King) GetMoves(board *Board) *[]Move {
	return nil
}
