package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
)

type Pawn struct {
	Location location.Location
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

func (r *Pawn) SetPosition(loc location.Location) {
	r.Location.Set(loc)
}

func (r *Pawn) GetPosition() location.Location {
	return r.Location
}

func (r *Pawn) GetMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move

	moves = append(moves, *r.getCaptureMoves(board, onlyFirstMove)...)
	if onlyFirstMove && len(moves) > 0 {
		return &moves
	}
	moves = append(moves, *r.getForwardMoves(board, onlyFirstMove)...)
	return &moves
}

/**
 * Returns all diagonal attack moves - any position protected by this pawn.
 */
func (r *Pawn) GetAttackableMoves(board *Board) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	locations := r.getAttackLocations(board)
	for _, loc := range *locations {
		SetLocationAttackable(attackableBoard, loc)
	}
	return attackableBoard
}

/**
 * Determines if a pawn has moved based on its color (black in row one, white in row six).
 */
func (r *Pawn) hasMoved() bool {
	if r.GetColor() == color.Black {
		return r.Location.GetRow() != 1
	} else if r.GetColor() == color.White {
		return r.Location.GetRow() != 6
	}
	return true
}

/**
 * Determines possible attack locations (diagonal ahead to left or right). Only returns inBounds locations
 * TODO cache lookups
 */
func (r *Pawn) getAttackLocations(board *Board) *[]location.Location {
	var locations []location.Location
	for i := -1; i <= 1; i += 2 {
		loc := r.GetPosition()
		loc, inBounds := loc.AddRelative(location.RelativeLocation{Col: int8(i)})
		if inBounds {
			loc, inBounds = loc.AddRelative(r.forward(1))
			if inBounds {
				locations = append(locations, loc)
			}
		}
	}
	return &locations
}

/**
 * Determines possible capture moves (diagonal ahead with a piece there).
 */
func (r *Pawn) getCaptureMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	locations := r.getAttackLocations(board)
	for _, loc := range *locations {
		if !board.IsEmpty(loc) {
			pieceOnLocation := board.GetPiece(loc)
			if pieceOnLocation.GetColor() != r.Color {
				if r.canPromote(loc) {
					for _, promotedType := range piece.PawnPromotionOptions {
						loc = loc.CreatePawnPromotion(promotedType)
						possibleMove := location.Move{Start: r.GetPosition(), End: loc}
						if !board.willMoveLeaveKingInCheck(r.Color, possibleMove) {
							moves = append(moves, possibleMove)
							if onlyFirstMove {
								return &moves
							}
						}
					}
				} else {
					possibleMove := location.Move{Start: r.GetPosition(), End: loc}
					if !board.willMoveLeaveKingInCheck(r.Color, possibleMove) {
						moves = append(moves, location.Move{Start: r.GetPosition(), End: loc})
						if onlyFirstMove {
							return &moves
						}
					}
				}
			}
		}
	}
	return &moves
}

/**
 * Determine forward moves.
 */
func (r *Pawn) getForwardMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	forwardThresh := 1
	if !r.hasMoved() {
		forwardThresh = 2
	}
	for i := 1; i <= forwardThresh; i++ {
		l := r.GetPosition()
		l, inBounds := l.AddRelative(r.forward(i))
		if inBounds {
			// can only add if empty - no attacking forward with pawns
			if board.IsEmpty(l) {
				if r.canPromote(l) {
					for _, promotedType := range piece.PawnPromotionOptions {
						l = l.CreatePawnPromotion(promotedType)
						possibleMove := location.Move{Start: r.GetPosition(), End: l}
						if !board.willMoveLeaveKingInCheck(r.Color, possibleMove) {
							moves = append(moves, possibleMove)
							if onlyFirstMove {
								return &moves
							}
						}
					}
				} else {
					possibleMove := location.Move{Start: r.GetPosition(), End: l}
					if !board.willMoveLeaveKingInCheck(r.Color, possibleMove) {
						moves = append(moves, possibleMove)
						if onlyFirstMove {
							return &moves
						}
					}
				}
			} else {
				return &moves
			}
		}
	}
	return &moves
}

func (r *Pawn) canPromote(l location.Location) bool {
	return r.Color == color.Black && l.GetRow() == 7 || r.Color == color.White && l.GetRow() == 0
}

func (r *Pawn) Move(m *location.Move, b *Board) {
	if r.Color == color.Black {
		if m.End.GetRow() == 7 {
			r.Promote(b, m)
		}
		// move put us above enemy (enPassant pawn)
		l, inBounds := r.GetPosition().AddRelative(location.UpMove)
		if inBounds {
			if eP := r.checkEnPassant(l, b); eP != nil {
				b.SetPiece(eP.GetPosition(), nil)
			}
		}
	} else if r.Color == color.White {
		if m.End.GetRow() == 0 {
			r.Promote(b, m)
		}
		// move put us above enemy (enPassant pawn)
		l, inBounds := r.GetPosition().AddRelative(location.DownMove)
		if inBounds {
			if eP := r.checkEnPassant(l, b); eP != nil {
				b.SetPiece(eP.GetPosition(), nil)
			}
		}
	}
}

func (r *Pawn) checkEnPassant(l location.Location, b *Board) Piece {
	enPassantPawn := b.GetPiece(l)
	if enPassantPawn != nil {
		pawn, ok := enPassantPawn.(*Pawn)
		if ok {
			if r.Color != pawn.GetColor() {
				return enPassantPawn
			}
		}
	}
	return nil
}

func (r *Pawn) Promote(b *Board, m *location.Move) {
	// allows chosing a piece  in the location object
	promoted, newType := m.End.GetPawnPromotion()
	if !promoted {
		panic("trying to promote pawn but move was not a promotion")
	}
	newPiece := PieceFromType(newType)
	newPiece.SetColor(r.GetColor())
	newPiece.SetPosition(r.GetPosition())
	b.SetPiece(r.GetPosition(), newPiece)
}

func (r *Pawn) forward(i int) location.RelativeLocation {
	if r.GetColor() == color.Black {
		return location.RelativeLocation{Row: int8(i)}
	} else if r.GetColor() == color.White {
		return location.RelativeLocation{Row: int8(-i)}
	}
	panic("invalid color provided to forward")
}
