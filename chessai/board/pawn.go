package board

import (
	"ChessAI3/chessai/board/color"
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

func (r *Pawn) Move(m *Move, b *Board) {
	if r.Color == color.Black {
		if m.End.Row == 7 {
			r.Promote(b)
		}
		r.checkEnPassant(UpMove, b)
	} else if r.Color == color.White {
		if m.End.Row == 0 {
			r.Promote(b)
		}
		r.checkEnPassant(DownMove, b)
	}
}

func (r *Pawn) checkEnPassant(l Location, b *Board) {
	enPassantPawn := b.GetPiece(l)
	if enPassantPawn != nil {
		pawn, ok := enPassantPawn.(*Pawn)
		if ok {
			if r.Color != pawn.GetColor() {
				b.SetPiece(enPassantPawn.GetPosition(), nil)
			}
		}
	}
}

func (r *Pawn) Promote(b *Board) {
	// TODO(Vadim) somehow enable choosing piece
	newPiece := Queen{}
	newPiece.SetColor(r.GetColor())
	newPiece.SetPosition(r.GetPosition())
	b.SetPiece(r.GetPosition(), &newPiece)
}
