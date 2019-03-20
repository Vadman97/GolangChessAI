package board

import (
	"ChessAI3/chessai/board/piece"
)

type Bishop struct {
	Location Location
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

func (r *Bishop) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Bishop) GetPosition() Location {
	return r.Location
}

func (r *Bishop) GetMoves(board *Board) *[]Move {
	var moves []Move
	for i := 0; i < 4; i++ {
		location := r.GetPosition()
		for true {
			if i == 0 {
				location = location.Add(RightUpMove)
			} else if i == 1 {
				location = location.Add(RightDownMove)
			} else if i == 2 {
				location = location.Add(LeftUpMove)
			} else if i == 3 {
				location = location.Add(LeftDownMove)
			}
			validMove, checkNext := CheckLocationForPiece(r.GetColor(), location, board)
			if validMove {
				moves = append(moves, Move{r.GetPosition(), location})
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
		location := r.GetPosition()
		for true {
			if i == 0 {
				location = location.Add(RightUpMove)
			} else if i == 1 {
				location = location.Add(RightDownMove)
			} else if i == 2 {
				location = location.Add(LeftUpMove)
			} else if i == 3 {
				location = location.Add(LeftDownMove)
			}
			attackable, checkNext := CheckLocationForAttackability(location, board)
			if attackable {
				SetLocationAttackable(attackableBoard, location)
			}
			if !checkNext {
				break
			}
		}
	}
	return attackableBoard
}

func (r *Bishop) Move(m *Move, b *Board) {}
