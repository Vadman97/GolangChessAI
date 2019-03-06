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
	var moves []Move

	/*
		for i := -1; i <= 1; i++ {
			for j := -1; j <= 1; j++ {
				if i != j {
					l := r.GetPosition()
					l = l.Add(Location{int8(i), int8(j)})
					if l.InBounds() {
						// TODO(Vadim) check l is not protected
						if board.IsEmpty(l) {

						} else if board.GetPiece(l).GetColor() != r.GetColor() {

						}
					}
				}
			}
		}

		validMove, checkNext := CheckLocationForPiece(r.GetColor(), l, board)
		if validMove {
			moves = append(moves, Move{r.GetPosition(), l})
		}
		if !checkNext {
			break
		}

		/*
		for (loc_t i = -1; i <= 1; ++i) {
		    for (loc_t j = -1; j <= 1; ++j) {
		      if (i == j) { continue; }
		      Location l = getPosition() + Location(i, j);
		      if (!l.inBounds()) { continue; }
		      if (board->isEmpty(l)) {
		        moves.emplace_back(Move(getPosition(), l));
		      } else if (board->getPiece(l)->getColor() != getColor()) {
		        // TODO(Vadim) check not protected
		        moves.emplace_back(Move(getPosition(), l));
		      }
		    }
		  }
		  if (!board->getFlag(c, GameBoard::CASTLED)) {
		    // TODO(Vadim) check not protected in the middle of castle
		    addMoveIfValid(board, getPosition() + right(2));
		    addMoveIfValid(board, getPosition() + left(2));
		  }

	*/

	return &moves
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
