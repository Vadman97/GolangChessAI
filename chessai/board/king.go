package board

import (
	"ChessAI3/chessai/board/color"
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

/**
 * Gets all possible moves for the King.
 */
func (r *King) GetMoves(board *Board) *[]Move {
	var moves []Move
	moves = append(moves, *r.GetNormalMoves(board)...)
	moves = append(moves, *r.GetCastleMoves(board)...)
	return &moves
}

/*
 * Determines possible "normal" moves for a king (move in any direction a distance of one).
 */
func (r *King) GetNormalMoves(board *Board) *[]Move {
	var moves []Move
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 {
				l := r.GetPosition()
				l = l.Add(Location{int8(i), int8(j)})
				if !r.underAttack(l, board) {
					validMove, _ := CheckLocationForPiece(r.GetColor(), l, board)
					if validMove {
						moves = append(moves, Move{r.GetPosition(), l})
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
func (r *King) GetCastleMoves(board *Board) *[]Move {
	var moves []Move
	if !board.GetFlag(FlagCastled, r.GetColor()) && !board.GetFlag(FlagKingMoved, r.GetColor()) {
		right, left := r.GetPosition(), r.GetPosition()
		for i := 0; i < 2; i++ {
			right = right.Add(RightMove)
			left = left.Add(LeftMove)
		}
		rightM, leftM := Move{r.GetPosition(), right}, Move{r.GetPosition(), left}
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
				location := r.GetPosition()
				location = location.Add(Location{int8(i), int8(j)})
				if location.InBounds() {
					SetSquareAttackable(attackableBoard, location)
				}
			}
		}
	}
	return attackableBoard
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

/**
 * Verifies that a king does not castle out of, through, or into check.  Also verifies that
 * all squares between a king and rook are empty.
 */
func (r *King) canCastle(m *Move, b *Board) bool {
	if m.End.InBounds() {
		// rook can be under attack - only need to check two spaces where king will move
		var leftLocation, rightLocation Location
		if m.End.Col < m.Start.Col {
			leftLocation = m.End
			rightLocation = m.Start
		} else {
			leftLocation = m.Start
			rightLocation = m.End
		}
		for c := leftLocation.Col; c <= rightLocation.Col; c++ {
			location := Location{leftLocation.Row, c}
			if r.underAttack(location, b) {
				return false
			}
			if !b.IsEmpty(location) {
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
func (r *King) underAttack(location Location, b *Board) bool {
	var potentialAttackMoves AttackableBoard

	if r.Color == color.Black {
		potentialAttackMoves = b.GetAllAttackableMoves(color.White)
	} else if r.Color == color.White {
		potentialAttackMoves = b.GetAllAttackableMoves(color.Black)
	} else {
		return false
	}

	return IsLocationAttackable(potentialAttackMoves, location)
}
