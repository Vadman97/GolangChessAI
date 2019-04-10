package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"log"
)

type Piece interface {
	GetChar() rune
	GetColor() byte
	SetColor(byte)
	GetPosition() location.Location
	SetPosition(location.Location)
	GetMoves(*Board) *[]location.Move
	GetAttackableMoves(*Board) AttackableBoard
	GetPieceType() byte
	Move(m *location.Move, b *Board)
}

func MakeMove(m *location.Move, b *Board) *LastMove {
	// no UnMove function because we delete the piece we destroy
	// easier to store copy of board before making move
	end := m.GetEnd()
	start := m.GetStart()
	// TODO(Vadim) verify that you can take the piece based on Color - here or in getMoves?
	if end.Equals(start) {
		log.Fatalf("Invalid move attempted! Start and End same: %+v", start)
		return nil
	} else {
		// piece holds information about its location for convenience
		// game tree stores as compressed game board -> have way to hash compressed game board fast
		// location stored in board coordinates but can be expanded to piece objects
		pieceCaptured := b.GetPiece(end)
		b.move(m)
		pieceMoved := b.GetPiece(end)
		pieceMoved.Move(m, b)

		return &(LastMove{Piece: &pieceMoved, Move: m, IsCapture: pieceCaptured != nil})
	}
}

func CheckLocationForPiece(pieceColor byte, l location.Location, b *Board) (validMove bool, checkNext bool) {
	if p := b.GetPiece(l); p != nil {
		if p.GetColor() != pieceColor {
			return true, false
		}
		return false, false
	}
	return true, true
}

/**
 * Determines if the position is attackable. Attackable means that this position can be seized,
 * regardless of the color of piece on it. This is necessary since King may take a piece, but
 * put itself into check.  This is less strict than CheckLocationForPiece.
 */
func CheckLocationForAttackability(l location.Location, b *Board) (checkNext bool) {
	p := b.GetPiece(l)
	if p == nil {
		return true
	}
	return false
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
