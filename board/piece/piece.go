package piece

import (
	"chessAI/board"
	"chessAI/board/piece/color"
	"log"
)

type Piece interface {
	GetChar() rune
	GetColor() byte
	SetColor(byte)
	GetPosition() board.Location
	SetPosition(board.Location)
	GetMoves(*board.Board) *[]board.Move
}

func basicMove(p Piece, m *board.Move, b *board.Board) {
	// TODO(Vadim) need to this about this more - should piece hold its location or location only in gameboard or store pointer to loc
	p.SetPosition(m.GetEnd())
	// TODO(Vadim)
	/*
		board->setPiece(move.end_l, board->getPiece(move.start_l));
		board->setPiece(move.start_l, nullptr);
		if (board->getPiece(move.end_l) != nullptr) {
			board->getPiece(move.end_l)->pos = move.end_l;
		}
	*/
}

func Move(p Piece, m *board.Move, b *board.Board) {
	if m.GetEnd().Equals(m.GetStart()) {
		log.Fatal("Invalid move attempted! Start and end same.", m.GetStart().Print())
	} else {
		basicMove(p, m, b)
		rook, ok := p.(Rook)
		if ok {
			if rook.IsRightRook() {
				// TODO(Vadim)
				// board->setFlag(c, GameBoard::R_ROOK_MOVED, true);
			}
			if rook.IsLeftRook() {
				// TODO(Vadim)
				// board->setFlag(c, GameBoard::L_ROOK_MOVED, true);
			}
		}
	}
}

func UnMove(p Piece, m *board.Move, b *board.Board) {
	// TODO(Vadim)
	/*
		board->setPiece(move.start_l, board->getPiece(move.end_l));
		board->setPiece(move.end_l, nullptr);
	*/
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
	pos   board.Location
	color byte
}

func (r Rook) GetChar() rune {
	return RookChar
}

func (r Rook) GetColor() byte {
	return r.color
}

func (r Rook) SetColor(color byte) {
	r.color = color
}

func (r Rook) SetPosition(loc board.Location) {
	r.pos.Set(loc)
}

func (r Rook) GetPosition() board.Location {
	return r.pos
}

func (r Rook) GetMoves(board *board.Board) *[]board.Move {
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
}
