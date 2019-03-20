package board

import (
	"ChessAI3/chessai/board/piece"
)

type Queen struct {
	Location Location
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

func (r *Queen) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Queen) GetPosition() Location {
	return r.Location
}

/**
 * Calculates all valid moves for this queen.
 */
func (r *Queen) GetMoves(board *Board) *[]Move {
	var moves []Move
	for i := 0; i < 8; i++ {
		location := r.GetPosition()
		for true {
			if i == 0 {
				location = location.Add(UpMove)
			} else if i == 1 {
				location = location.Add(RightUpMove)
			} else if i == 2 {
				location = location.Add(RightMove)
			} else if i == 3 {
				location = location.Add(RightDownMove)
			} else if i == 4 {
				location = location.Add(DownMove)
			} else if i == 5 {
				location = location.Add(LeftDownMove)
			} else if i == 6 {
				location = location.Add(LeftMove)
			} else if i == 7 {
				location = location.Add(LeftUpMove)
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
 * Retrieves all squares that this queen can attack.
 */
func (r *Queen) GetAttackableMoves(board *Board) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	for i := 0; i < 8; i++ {
		location := r.GetPosition()
		for true {
			if i == 0 {
				location = location.Add(UpMove)
			} else if i == 1 {
				location = location.Add(RightUpMove)
			} else if i == 2 {
				location = location.Add(RightMove)
			} else if i == 3 {
				location = location.Add(RightDownMove)
			} else if i == 4 {
				location = location.Add(DownMove)
			} else if i == 5 {
				location = location.Add(LeftDownMove)
			} else if i == 6 {
				location = location.Add(LeftMove)
			} else if i == 7 {
				location = location.Add(LeftUpMove)
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

func (r *Queen) Move(m *Move, b *Board) {
	//TODO
}
