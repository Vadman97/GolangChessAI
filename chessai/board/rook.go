package board

import (
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/piece"
	"log"
)

type Rook struct {
	Location Location
	Color    byte
}

func (r *Rook) GetChar() rune {
	return piece.RookChar
}

func (r *Rook) GetPieceType() byte {
	return piece.RookType
}

func (r *Rook) GetColor() byte {
	return r.Color
}

func (r *Rook) SetColor(color byte) {
	r.Color = color
}

func (r *Rook) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Rook) GetPosition() Location {
	return r.Location
}

func (r *Rook) GetMoves(board *Board) *[]Move {
	return nil
}

func (r *Rook) Move(m *Move, b *Board) {
	if r.IsRightRook() {
		b.SetFlag(FlagRightRookMoved, r.GetColor(), true)
	}
	if r.IsLeftRook() {
		b.SetFlag(FlagLeftRookMoved, r.GetColor(), true)
	}
}

func (r *Rook) IsRightRook() bool {
	return r.Location.Col == 7
}

func (r *Rook) IsLeftRook() bool {
	return r.Location.Col == 0
}

func (r *Rook) IsStartingRow() bool {
	if r.Color == color.Black {
		return r.Location.Row == 0
	} else if r.Color == color.White {
		return r.Location.Row == 7
	} else {
		log.Fatal("Invalid Color")
	}
	return false
}
