package board

import (
	"ChessAI3/chessai/board/piece"
)

type Pawn struct {
	Location Location
	Color    byte
}

func (r *Pawn) GetChar() rune {
	return piece.PawnChar
}

func (r *Pawn) GetPieceType() byte {
	return piece.PawnType
}

func (r *Pawn) GetColor() byte {
	return r.Color
}

func (r *Pawn) SetColor(color byte) {
	r.Color = color
}

func (r *Pawn) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Pawn) GetPosition() Location {
	return r.Location
}

func (r *Pawn) GetMoves(board *Board) *[]Move {
	return nil
}
