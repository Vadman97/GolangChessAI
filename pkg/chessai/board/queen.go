package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
)

type Queen struct {
	Location location.Location
	Color    byte
}

func (r *Queen) GetChar() rune {
	return piece.QueenChar
}

func (r *Queen) GetPieceType() byte {
	return piece.QueenType
}

func (r *Queen) GetColor() byte {
	return r.Color
}

func (r *Queen) SetColor(color byte) {
	r.Color = color
}

func (r *Queen) SetPosition(loc location.Location) {
	r.Location.Set(loc)
}

func (r *Queen) GetPosition() location.Location {
	return r.Location
}

/**
 * Calculates all valid moves for this queen.
 */
func (r *Queen) GetMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	for i := 0; i < 8; i++ {
		loc := r.GetPosition()
		var inBounds bool
		for true {
			if i == 0 {
				loc, inBounds = loc.AddRelative(location.UpMove)
			} else if i == 1 {
				loc, inBounds = loc.AddRelative(location.RightUpMove)
			} else if i == 2 {
				loc, inBounds = loc.AddRelative(location.RightMove)
			} else if i == 3 {
				loc, inBounds = loc.AddRelative(location.RightDownMove)
			} else if i == 4 {
				loc, inBounds = loc.AddRelative(location.DownMove)
			} else if i == 5 {
				loc, inBounds = loc.AddRelative(location.LeftDownMove)
			} else if i == 6 {
				loc, inBounds = loc.AddRelative(location.LeftMove)
			} else if i == 7 {
				loc, inBounds = loc.AddRelative(location.LeftUpMove)
			}
			if !inBounds {
				break
			}
			validMove, checkNext := CheckLocationForPiece(r.GetColor(), loc, board)
			if validMove {
				moves = append(moves, location.Move{Start: r.GetPosition(), End: loc})
				if onlyFirstMove {
					return &moves
				}
			}
			if !checkNext {
				break
			}
		}
	}
	return &moves
}

/**
 * Retrieves all squares that this queen can attack.
 */
func (r *Queen) GetAttackableMoves(board *Board) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	for i := 0; i < 8; i++ {
		loc := r.GetPosition()
		var inBounds bool
		for true {
			if i == 0 {
				loc, inBounds = loc.AddRelative(location.UpMove)
			} else if i == 1 {
				loc, inBounds = loc.AddRelative(location.RightUpMove)
			} else if i == 2 {
				loc, inBounds = loc.AddRelative(location.RightMove)
			} else if i == 3 {
				loc, inBounds = loc.AddRelative(location.RightDownMove)
			} else if i == 4 {
				loc, inBounds = loc.AddRelative(location.DownMove)
			} else if i == 5 {
				loc, inBounds = loc.AddRelative(location.LeftDownMove)
			} else if i == 6 {
				loc, inBounds = loc.AddRelative(location.LeftMove)
			} else if i == 7 {
				loc, inBounds = loc.AddRelative(location.LeftUpMove)
			}
			if !inBounds {
				break
			}
			SetLocationAttackable(attackableBoard, loc)
			if !CheckLocationForAttackability(loc, board) {
				break
			}
		}
	}
	return attackableBoard
}

func (r *Queen) Move(m *location.Move, b *Board) {
	//TODO
}
