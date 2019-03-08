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
	Move(m *Move, b *Board)
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
		b.GetPiece(end).Move(m, b)
	}
}

func CheckLocationForPiece(pieceColor byte, l Location, b *Board) (validMove bool, checkNext bool) {
	if !l.InBounds() {
		return false, false
	}
	if p := b.GetPiece(l); p != nil {
		if p.GetColor() != pieceColor {
			return true, false
		}
		return false, false
	}
	return true, true
}

func GetColorTypeRepr(p Piece) string {
	var result string
	if p == nil {
		return "   "
	}
	if p.GetColor() == color.White {
		result += string(color.WhiteChar) + "_"
	} else if p.GetColor() == color.Black {
		result += string(color.BlackChar) + "_"
	}
	return result + string(p.GetChar())
}
