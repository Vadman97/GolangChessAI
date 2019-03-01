package board

import (
	"ChessAI3/chessai/board/color"
	"log"
)

type Piece interface {
	GetChar() rune
	GetColor() byte
	SetColor(byte)
	GetPosition() Location
	SetPosition(Location)
	GetMoves(*Board) *[]Move
	GetPieceType() byte
}

func MakeMove(m *Move, b *Board) {
	// no UnMove function because we delete the piece we destroy
	// easier to store copy of board before making move
	end := m.GetEnd()
	start := m.GetStart()
	// TODO(Vadim) verify that you can take the piece based on Color - here or in getMoves?
	if end.Equals(start) {
		log.Fatalf("Invalid move attempted! Start and End same: %+v", start)
	} else {
		// piece holds information about its location for convenience
		// game tree stores as compressed game board -> have way to hash compressed game board fast
		// location stored in board coordinates but can be expanded to piece objects
		b.move(m)
		p := b.GetPiece(end)
		rook, ok := p.(*Rook)
		if ok {
			if rook.IsRightRook() {
				b.SetFlag(FlagRightRookMoved, rook.GetColor(), true)
			}
			if rook.IsLeftRook() {
				b.SetFlag(FlagLeftRookMoved, rook.GetColor(), true)
			}
		}
	}
}

func GetColorTypeRepr(p Piece) string {
	var result string
	if p.GetColor() == color.White {
		result += "W_"
	} else if p.GetColor() == color.Black {
		result += "B_"
	}
	return result + string(p.GetChar())
}
