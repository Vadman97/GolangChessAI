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
	// TODO(Vadim) implement
	return nil
}

func (r *King) Move(m *Move, b *Board) {
	if m.Start.Col == 4 && m.Start.Col-2 == m.End.Col {
		// left castle
		// piece right of king set to the rook from left of dest
		b.SetPiece(m.End.Add(RightMove), b.GetPiece(m.End.Add(LeftMove)))
		b.SetPiece(m.End.Add(LeftMove), nil)
		b.SetFlag(FlagCastled, r.GetColor(), true)
	} else if m.Start.Col == 4 && m.Start.Col+2 == m.End.Col {
		// right castle
		// piece right of king set to the rook from left of dest
		b.SetPiece(m.End.Add(LeftMove), b.GetPiece(m.End.Add(RightMove)))
		b.SetPiece(m.End.Add(RightMove), nil)
		b.SetFlag(FlagCastled, r.GetColor(), true)
	}
	b.SetFlag(FlagKingMoved, r.GetColor(), true)
}
