package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
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
func (r *King) GetMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	moves = append(moves, *r.GetNormalMoves(board, onlyFirstMove)...)
	if onlyFirstMove && len(moves) > 0 {
		return &moves
	}
	moves = append(moves, *r.GetCastleMoves(board, onlyFirstMove)...)
	return &moves
}

/*
 * Determines possible "normal" moves for a king (move in any direction a distance of one).
 */
func (r *King) GetNormalMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	for i := int8(-1); i <= 1; i++ {
		for j := int8(-1); j <= 1; j++ {
			if i != 0 || j != 0 {
				l := r.GetPosition()
				l, inBounds := l.AddRelative(location.RelativeLocation{Row: i, Col: j})
				if inBounds {
					pieceOnLocation := board.GetPiece(l)
					if (pieceOnLocation == nil) || (pieceOnLocation.GetColor() != r.Color) {
						possibleMove := location.Move{Start: r.GetPosition(), End: l}
						if !board.willMoveLeaveKingInCheck(r.Color, possibleMove) {
							moves = append(moves, possibleMove)
							if onlyFirstMove {
								return &moves
							}
						}
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
func (r *King) GetCastleMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	if !board.GetFlag(FlagCastled, r.GetColor()) && !board.GetFlag(FlagKingMoved, r.GetColor()) {
		right, left := r.GetPosition(), r.GetPosition()
		var rightIn, leftIn bool
		for i := 0; i < 2; i++ {
			var rin, lin bool
			right, rin = right.AddRelative(location.RightMove)
			left, lin = left.AddRelative(location.LeftMove)
			rightIn, leftIn = rightIn || rin, leftIn || lin
		}
		rightM, leftM := location.Move{Start: r.GetPosition(), End: right},
			location.Move{Start: r.GetPosition(), End: left}
		if rightIn && r.canCastle(&rightM, board) && !board.GetFlag(FlagRightRookMoved, r.GetColor()) {
			if !board.willMoveLeaveKingInCheck(r.Color, rightM) {
				moves = append(moves, rightM)
				if onlyFirstMove {
					return &moves
				}
			}
		}
		if leftIn && r.canCastle(&leftM, board) && !board.GetFlag(FlagLeftRookMoved, r.GetColor()) {
			if !board.willMoveLeaveKingInCheck(r.Color, leftM) {
				moves = append(moves, leftM)
				if onlyFirstMove {
					return &moves
				}
			}
		}
	}
	return &moves
}

/**
 * Retrieves all squares that this king can attack.
 */
func (r *King) GetAttackableMoves(board *Board) BitBoard {
	attackableBoard := BitBoard(0)
	for i := int8(-1); i <= 1; i++ {
		for j := int8(-1); j <= 1; j++ {
			if i != 0 || j != 0 {
				loc := r.GetPosition()
				loc, inBounds := loc.AddRelative(location.RelativeLocation{Row: i, Col: j})
				if inBounds {
					attackableBoard.SetLocation(loc)
				}
			}
		}
	}
	return attackableBoard
}

func (r *King) Move(m *location.Move, b *Board) {
	startCol := m.Start.GetCol()
	endCol := m.End.GetCol()
	right, _ := m.End.AddRelative(location.RightMove)
	left, _ := m.End.AddRelative(location.LeftMove)
	if startCol == 3 && startCol-2 == endCol {
		// left castle
		b.SetPiece(right, b.GetPiece(left))
		b.SetPiece(left, nil)
		b.SetFlag(FlagCastled, r.GetColor(), true)
	} else if startCol == 3 && startCol+2 == endCol {
		// right castle
		rightTwo, _ := right.AddRelative(location.RightMove)
		b.SetPiece(left, b.GetPiece(rightTwo))
		b.SetPiece(rightTwo, nil)
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
	// rook can be under attack - only need to check two spaces where king will move
	var leftLocation, rightLocation location.Location
	if m.End.GetCol() < m.Start.GetCol() {
		leftLocation = m.End
		rightLocation = m.Start
		leftLocation, _ = leftLocation.AddRelative(location.LeftMove)
	} else {
		leftLocation = m.Start
		rightLocation = m.End
		rightLocation, _ = rightLocation.AddRelative(location.RightMove)
	}
	llRow, llCol := leftLocation.Get()
	for c := llCol; c <= rightLocation.GetCol(); c++ {
		loc := location.NewLocation(llRow, c)
		if r.underAttack(loc, b) {
			return false
		}
		if !b.IsEmpty(loc) {
			if b.GetPiece(loc).GetPieceType() != piece.KingType && b.GetPiece(loc).GetPieceType() != piece.RookType {
				return false
			}
		}
	}
	return true
}

/**
 * Determines if a specific location is under attack on the board (can be moved into by any piece
 * of the opposing color).
 */
func (r *King) underAttack(location location.Location, b *Board) bool {
	var potentialAttackMoves BitBoard
	potentialAttackMoves = b.GetAllAttackableMoves(r.Color ^ 1)
	return potentialAttackMoves.IsLocationSet(location)
}
