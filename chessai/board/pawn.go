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

	moves = append(moves, *r.getCaptureMoves(board)...)
	moves = append(moves, *r.getForwardMoves(board)...)
	return &moves
}

/**
 * Returns all diagonal attack moves - any position protected by this pawn.
 */
func (r *Pawn) GetAttackableMoves(board *Board) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	locations := r.getAttackLocations(board)
	for _, location := range *locations {
		if location.InBounds() {
			SetLocationAttackable(attackableBoard, location)
		}
	}
	return attackableBoard
}

/**
 * Determines if a pawn has moved based on its color (black in row one, white in row six).
 */
func (r *Pawn) hasMoved() bool {
	if r.GetColor() == color.Black {
		return r.Location.Row != 1
	} else if r.GetColor() == color.White {
		return r.Location.Row != 6
	}
	return true
}

/**
 * Determines possible attack locations (diagonal ahead to left or right).
 */
func (r *Pawn) getAttackLocations(board *Board) *[]Location {
	var locations []Location
	for i := -1; i <= 1; i += 2 {
		location := r.GetPosition()
		location = location.Add(Location{0, int8(i)})
		location = location.Add(r.forward(1))
		locations = append(locations, location)
	}
	return &locations
}

/**
 * Determines possible capture moves (diagonal ahead with a piece there).
 */
func (r *Pawn) getCaptureMoves(board *Board) *[]Move {
	var moves []Move
	locations := r.getAttackLocations(board)
	for _, location := range *locations {
		if location.InBounds() && !board.IsEmpty(location) {
			pieceOnLocation := board.GetPiece(location)
			if pieceOnLocation != nil && pieceOnLocation.GetColor() != r.Color {
				moves = append(moves, Move{r.GetPosition(), location})
			}
		}
	}
	return &moves
}

/**
 * Determine forward moves.
 */
func (r *Pawn) getForwardMoves(board *Board) *[]Move {
	var moves []Move
	forwardThresh := 1
	if !r.hasMoved() {
		forwardThresh = 2
	}
	for i := 1; i <= forwardThresh; i++ {
		l := r.GetPosition()
		l = l.Add(r.forward(i))
		// can only add if empty - no attacking forward with pawns
		if board.IsEmpty(l) {
			moves = append(moves, Move{r.GetPosition(), l})
		}
	}
	return &moves
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
