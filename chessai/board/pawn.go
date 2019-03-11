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
	var moves []Move
	var pawnHasMoved bool

	if r.GetColor() == color.Black {
		pawnHasMoved = r.Location.Row != 1
	} else if r.GetColor() == color.White {
		pawnHasMoved = r.Location.Row != 6
	}

	// check en passant
	for _, m := range []Location{LeftMove, RightMove} {
		move := m.Add(r.Location)
		if eP := r.checkEnPassant(move, board); eP != nil {
			// there is an enemy en passant pawn there
			end := r.GetPosition()
			end = end.Add(r.forward(1))
			end = end.Add(m)
			if end.InBounds() {
				moves = append(moves, Move{
					Start: r.GetPosition(),
					End:   end,
				})
			}
		}
	}

	// first look at attack moves (diagonal ahead)
	for i := -1; i <= 1; i += 2 {
		l := r.GetPosition()
		l = l.Add(Location{0, int8(i)})
		l = l.Add(r.forward(1))
		// can only add if there is an enemy piece there - attacking
		if !board.IsEmpty(l) {
			validMove, _ := CheckLocationForPiece(r.GetColor(), l, board)
			if validMove {
				moves = append(moves, Move{r.GetPosition(), l})
			}
		}
	}

	// now look at forward 2 moves
	forwardThresh := 1
	if !pawnHasMoved {
		forwardThresh = 2
	}
	for i := 1; i <= forwardThresh; i++ {
		l := r.GetPosition()
		l = l.Add(r.forward(i))
		// can only add if empty - no attacking forward with pawns
		if board.IsEmpty(l) {
			validMove, _ := CheckLocationForPiece(r.GetColor(), l, board)
			if validMove {
				moves = append(moves, Move{r.GetPosition(), l})
			}
		}
	}
	return &moves
}

func (r *Pawn) GetAttackableMoves(board *Board) *[]Move {
	//TODO (Devan)
	return nil
}

func (r *Pawn) Move(m *Move, b *Board) {
	if r.Color == color.Black {
		if m.End.Row == 7 {
			r.Promote(b)
		}
		// move put us below enemy (enPassant pawn)
		if eP := r.checkEnPassant(UpMove, b); eP != nil {
			b.SetPiece(eP.GetPosition(), nil)
		}
	} else if r.Color == color.White {
		if m.End.Row == 0 {
			r.Promote(b)
		}
		// move put us above enemy (enPassant pawn)
		if eP := r.checkEnPassant(DownMove, b); eP != nil {
			b.SetPiece(eP.GetPosition(), nil)
		}
	}
}

func (r *Pawn) checkEnPassant(l Location, b *Board) Piece {
	if l.InBounds() {
		enPassantPawn := b.GetPiece(l)
		if enPassantPawn != nil {
			pawn, ok := enPassantPawn.(*Pawn)
			if ok {
				if r.Color != pawn.GetColor() {
					// TODO(Vadim) this is flawed - ensure that it is only if the enemy JUST performed dbl move
					// maybe keep some sort of turn counter - like what turn u made move on?
					/*
						Store column index of pawn last double-moved
						Clear or update on next boardmove - only 3/4 bits? 0-15 pawn ids - store as 4 extra bits
					*/
					if (pawn.GetColor() == color.Black && pawn.GetPosition().Row == 3) ||
						(pawn.GetColor() == color.White && pawn.GetPosition().Row == 4) {
						return enPassantPawn
					}
				}
			}
		}
	}
	return nil
}

func (r *Pawn) Promote(b *Board) {
	// TODO(Vadim) somehow enable choosing piece
	newPiece := Queen{}
	newPiece.SetColor(r.GetColor())
	newPiece.SetPosition(r.GetPosition())
	b.SetPiece(r.GetPosition(), &newPiece)
}

func (r *Pawn) forward(i int) Location {
	if r.GetColor() == color.Black {
		return Location{int8(i), 0}
	} else if r.GetColor() == color.White {
		return Location{int8(-i), 0}
	}
	panic("invalid color provided to forward")
}
