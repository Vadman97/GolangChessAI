package board

import (
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/piece"
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

func basicMove(p Piece, m *Move, b *Board) {
	// piece holds information about its location for convenience
	// game tree stores as compressed game board -> have way to hash compressed game board fast
	// location stored in board coordinates but can be expanded to piece objects
	p.SetPosition(m.GetEnd())
	b.move(m)
}

func MakeMove(p Piece, m *Move, b *Board) {
	// no UnMove function because we delete the piece we destroy
	// easier to store copy of board before making move
	end := m.GetEnd()
	start := m.GetStart()
	// TODO(Vadim) verify that you can take the piece - here or in getMoves?
	if end.Equals(start) {
		log.Fatal("Invalid move attempted! Start and End same.", start.Print())
	} else {
		basicMove(p, m, b)
		rook, ok := p.(Rook)
		if ok {
			if rook.IsRightRook() {
				b.SetFlag(FlagRightRookMoved, p.GetColor(), true)
			}
			if rook.IsLeftRook() {
				b.SetFlag(FlagLeftRookMoved, p.GetColor(), true)
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

type Rook struct {
	pos   Location
	color byte
}

func (r Rook) GetChar() rune {
	return piece.RookChar
}

func (r Rook) GetPieceType() byte {
	return piece.RookType
}

func (r Rook) GetColor() byte {
	return r.color
}

func (r Rook) SetColor(color byte) {
	r.color = color
}

func (r Rook) SetPosition(loc Location) {
	r.pos.Set(loc)
}

func (r Rook) GetPosition() Location {
	return r.pos
}

func (r Rook) GetMoves(board *Board) *[]Move {
	return nil
}

func (r Rook) IsRightRook() bool {
	return r.pos.Col == 7
}

func (r Rook) IsLeftRook() bool {
	return r.pos.Col == 0
}

func (r Rook) IsStartingRow() bool {
	if r.color == color.Black {
		return r.pos.Row == 0
	} else if r.color == color.White {
		return r.pos.Row == 7
	} else {
		log.Fatal("Invalid color")
	}
	return false
}
