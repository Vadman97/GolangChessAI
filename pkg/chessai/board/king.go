package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
)

type King struct {
	Location location.Location
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

func (r *King) SetPosition(loc location.Location) {
	r.Location.Set(loc)
}

func (r *King) GetPosition() location.Location {
	return r.Location
}

/**
 * Gets all possible moves for the King.
 */
func (r *King) GetMoves(board *Board) *[]location.Move {
	var moves []location.Move
	moves = append(moves, *r.GetNormalMoves(board)...)
	moves = append(moves, *r.GetCastleMoves(board)...)
	return &moves
}

/*
 * Determines possible "normal" moves for a king (move in any direction a distance of one).
 */
func (r *King) GetNormalMoves(board *Board) *[]location.Move {
	var moves []location.Move
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 {
				l := r.GetPosition()
				l = l.Add(location.Location{int8(i), int8(j)})
				if l.InBounds() {
					pieceOnLocation := board.GetPiece(l)
					if (pieceOnLocation == nil) || (pieceOnLocation.GetColor() != r.Color) {
						moves = append(moves, location.Move{r.GetPosition(), l})
					}
				}
			}
		}
	}
	return &moves
}

/**
 * Determines if the king is able to left castle or right castle.
 */
func (r *King) GetCastleMoves(board *Board) *[]location.Move {
	var moves []location.Move
	if !board.GetFlag(FlagCastled, r.GetColor()) && !board.GetFlag(FlagKingMoved, r.GetColor()) {
		right, left := r.GetPosition(), r.GetPosition()
		for i := 0; i < 2; i++ {
			right = right.Add(location.RightMove)
			left = left.Add(location.LeftMove)
		}
		rightM, leftM := location.Move{r.GetPosition(), right}, location.Move{r.GetPosition(), left}
		if r.canCastle(&rightM, board) && !board.GetFlag(FlagRightRookMoved, r.GetColor()) {
			moves = append(moves, rightM)
		}
		if r.canCastle(&leftM, board) && !board.GetFlag(FlagLeftRookMoved, r.GetColor()) {
			moves = append(moves, leftM)
		}
	}
	return &moves
}

/**
 * Retrieves all squares that this king can attack.
 */
func (r *King) GetAttackableMoves(board *Board) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 {
				loc := r.GetPosition()
				loc = loc.Add(location.Location{Row: int8(i), Col: int8(j)})
				if loc.InBounds() {
					SetLocationAttackable(attackableBoard, loc)
				}
			}
		}
	}
	return attackableBoard
}

func (r *King) Move(m *location.Move, b *Board) {
	if m.Start.Col == 4 && m.Start.Col-2 == m.End.Col {
		// left castle
		// piece right of king set to the rook from left of dest
		b.SetPiece(m.End.Add(location.RightMove), b.GetPiece(m.End.Add(location.LeftMove)))
		b.SetPiece(m.End.Add(location.LeftMove), nil)
		b.SetFlag(FlagCastled, r.GetColor(), true)
	} else if m.Start.Col == 4 && m.Start.Col+2 == m.End.Col {
		// right castle
		// piece right of king set to the rook from left of dest
		b.SetPiece(m.End.Add(location.LeftMove), b.GetPiece(m.End.Add(location.RightMove)))
		b.SetPiece(m.End.Add(location.RightMove), nil)
		b.SetFlag(FlagCastled, r.GetColor(), true)
	}
	b.SetFlag(FlagKingMoved, r.GetColor(), true)
	b.KingLocations[r.Color] = m.End
}

/**
 * Verifies that a king does not castle out of, through, or into check.  Also verifies that
 * all squares between a king and rook are empty.
 */
func (r *King) canCastle(m *location.Move, b *Board) bool {
	if m.End.InBounds() {
		// rook can be under attack - only need to check two spaces where king will move
		var leftLocation, rightLocation location.Location
		if m.End.Col < m.Start.Col {
			leftLocation = m.End
			rightLocation = m.Start
		} else {
			leftLocation = m.Start
			rightLocation = m.End
		}
		for c := leftLocation.Col; c <= rightLocation.Col; c++ {
			loc := location.Location{Row: leftLocation.Row, Col: c}
			if r.underAttack(loc, b) {
				return false
			}
			if !b.IsEmpty(loc) && b.GetPiece(loc).GetPieceType() != piece.KingType {
				return false
			}
		}
		return true
	}
	return false
}

/**
 * Determines if a specific location is under attack on the board (can be moved into by any piece
 * of the opposing color).
 */
func (r *King) underAttack(location location.Location, b *Board) bool {
	var potentialAttackMoves AttackableBoard

	if r.Color == color.Black {
		potentialAttackMoves = b.GetAllAttackableMoves(color.White)
	} else if r.Color == color.White {
		potentialAttackMoves = b.GetAllAttackableMoves(color.Black)
	} else {
		return false
	}

	return IsLocationUnderAttack(potentialAttackMoves, location)
}
