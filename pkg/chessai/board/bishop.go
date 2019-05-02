package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

type Bishop struct {
	Location location.Location
	Color    color.Color
}

func (r *Bishop) GetChar() rune {
	return piece.BishopChar
}

func (r *Bishop) GetPieceType() byte {
	return piece.BishopType
}

func (r *Bishop) GetColor() color.Color {
	return r.Color
}

func (r *Bishop) SetColor(color color.Color) {
	r.Color = color
}

func (r *Bishop) SetPosition(loc location.Location) {
	r.Location.Set(loc)
}

func (r *Bishop) GetPosition() location.Location {
	return r.Location
}

func (r *Bishop) GetMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	for i := 0; i < 4; i++ {
		loc := r.GetPosition()
		var inBounds bool
		for true {
			if i == 0 {
				loc, inBounds = loc.AddRelative(location.RightUpMove)
			} else if i == 1 {
				loc, inBounds = loc.AddRelative(location.RightDownMove)
			} else if i == 2 {
				loc, inBounds = loc.AddRelative(location.LeftUpMove)
			} else if i == 3 {
				loc, inBounds = loc.AddRelative(location.LeftDownMove)
			}
			if !inBounds {
				break
			}
			validMove, checkNext := CheckLocationForPiece(r.Color, loc, board)
			if validMove {
				possibleMove := location.Move{Start: r.GetPosition(), End: loc}
				if !board.willMoveLeaveKingInCheck(r.Color, possibleMove) {
					moves = append(moves, possibleMove)
					if onlyFirstMove {
						return &moves
					}
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
 * Retrieves all squares that this bishop can attack.
 */
func (r *Bishop) GetAttackableMoves(board *Board) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	for i := 0; i < 4; i++ {
		loc := r.GetPosition()
		var inBounds bool
		for true {
			if i == 0 {
				loc, inBounds = loc.AddRelative(location.RightUpMove)
			} else if i == 1 {
				loc, inBounds = loc.AddRelative(location.RightDownMove)
			} else if i == 2 {
				loc, inBounds = loc.AddRelative(location.LeftUpMove)
			} else if i == 3 {
				loc, inBounds = loc.AddRelative(location.LeftDownMove)
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

func (r *Bishop) Move(m *location.Move, b *Board) {}
