package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
)

type Bishop struct {
	Location location.Location
	Color    byte
}

func (r *Bishop) GetChar() rune {
	return piece.BishopChar
}

func (r *Bishop) GetPieceType() byte {
	return piece.BishopType
}

func (r *Bishop) GetColor() byte {
	return r.Color
}

func (r *Bishop) SetColor(color byte) {
	r.Color = color
}

func (r *Bishop) SetPosition(loc location.Location) {
	r.Location.Set(loc)
}

func (r *Bishop) GetPosition() location.Location {
	return r.Location
}

func (r *Bishop) GetMoves(board *Board) *[]location.Move {
	var moves []location.Move
	for i := 0; i < 4; i++ {
		loc := r.GetPosition()
		for true {
			if i == 0 {
				loc = loc.Add(location.RightUpMove)
			} else if i == 1 {
				loc = loc.Add(location.RightDownMove)
			} else if i == 2 {
				loc = loc.Add(location.LeftUpMove)
			} else if i == 3 {
				loc = loc.Add(location.LeftDownMove)
			}
			validMove, checkNext := CheckLocationForPiece(r.GetColor(), loc, board)
			if validMove {
				moves = append(moves, location.Move{Start: r.GetPosition(), End: loc})
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
		for true {
			if i == 0 {
				loc = loc.Add(location.RightUpMove)
			} else if i == 1 {
				loc = loc.Add(location.RightDownMove)
			} else if i == 2 {
				loc = loc.Add(location.LeftUpMove)
			} else if i == 3 {
				loc = loc.Add(location.LeftDownMove)
			}
			attackable, checkNext := CheckLocationForAttackability(loc, board)
			if attackable {
				SetLocationAttackable(attackableBoard, loc)
			}
			if !checkNext {
				break
			}
		}
	}
	return attackableBoard
}

func (r *Bishop) Move(m *location.Move, b *Board) {}
