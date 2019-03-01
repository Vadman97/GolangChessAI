package board

import (
	"ChessAI3/chessai/board/piece"
)

type Queen struct {
	Location Location
	Color    byte
}

func (r *Queen) GetChar() rune {
	return piece.QueenChar
}

func (r *Queen) GetPieceType() byte {
	return piece.QueenType
}

func (r *Queen) GetColor() byte {
	return r.Color
}

func (r *Queen) SetColor(color byte) {
	r.Color = color
}

func (r *Queen) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Queen) GetPosition() Location {
	return r.Location
}

func (r *Queen) GetMoves(board *Board) *[]Move {
	return nil
}
