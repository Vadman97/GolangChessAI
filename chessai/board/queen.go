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
 * Explores a board using canMove, a function that determines how much to explore.
 */
func (r *Queen) exploreMoves(board *Board,
	canMove func(pieceColor byte, l Location, b *Board) (validMove bool, checkNext bool)) *[]Move {
	var moves []Move
	for i := 0; i < 8; i++ {
		l := r.GetPosition()
		for l.InBounds() {
			if i == 0 {
				l = l.Add(UpMove)
			} else if i == 1 {
				l = l.Add(RightUpMove)
			} else if i == 2 {
				l = l.Add(RightMove)
			} else if i == 3 {
				l = l.Add(RightDownMove)
			} else if i == 4 {
				l = l.Add(DownMove)
			} else if i == 5 {
				l = l.Add(LeftDownMove)
			} else if i == 6 {
				l = l.Add(LeftMove)
			} else if i == 7 {
				l = l.Add(LeftUpMove)
			}
			validMove, checkNext := canMove(r.GetColor(), l, board)
			if validMove {
				moves = append(moves, Move{r.GetPosition(), l})
			}
			if !checkNext {
				break
			}
		}
	}
	return &moves
}

func (r *Queen) GetMoves(board *Board) *[]Move {
	return r.exploreMoves(board, CheckLocationForPiece)
}

/**
 * Retrieves all squares that this queen can attack.
 */
func (r *Queen) GetAttackableMoves(board *Board) AttackableBoard {
	moves := r.exploreMoves(board, CheckLocationForAttackability)
	return CreateAttackableBoardFromMoves(moves)
}

func (r *Queen) Move(m *Move, b *Board) {
	//TODO
}
