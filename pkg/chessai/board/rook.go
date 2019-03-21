package board

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"log"
)

type Rook struct {
	Location location.Location
	Color    byte
}

func (r *Rook) GetChar() rune {
	return piece.RookChar
}

func (r *Rook) GetPieceType() byte {
	return piece.RookType
}

func (r *Rook) GetColor() byte {
	return r.Color
}

func (r *Rook) SetColor(color byte) {
	r.Color = color
}

func (r *Rook) SetPosition(loc location.Location) {
	r.Location.Set(loc)
}

func (r *Rook) GetPosition() location.Location {
	return r.Location
}

/**
 * Gets all valid next moves for this rook.
 */
func (r *Rook) GetMoves(board *Board) *[]location.Move {
	var moves []location.Move
	for i := 0; i < 4; i++ {
		l := r.GetPosition()
		for true {
			if i == 0 {
				l = l.Add(location.UpMove)
			} else if i == 1 {
				l = l.Add(location.RightMove)
			} else if i == 2 {
				l = l.Add(location.DownMove)
			} else if i == 3 {
				l = l.Add(location.LeftMove)
			}
			validMove, checkNext := CheckLocationForPiece(r.GetColor(), l, board)
			if validMove {
				moves = append(moves, location.Move{r.GetPosition(), l})
			}
			if !checkNext {
				break
			}
		}
	}
	return &moves
}

/**
 * Retrieves all locations that this rook can attack.
 */
func (r *Rook) GetAttackableMoves(board *Board) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	for i := 0; i < 4; i++ {
		loc := r.GetPosition()
		for true {
			if i == 0 {
				loc = loc.Add(location.UpMove)
			} else if i == 1 {
				loc = loc.Add(location.RightMove)
			} else if i == 2 {
				loc = loc.Add(location.DownMove)
			} else if i == 3 {
				loc = loc.Add(location.LeftMove)
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

func (r *Rook) Move(m *location.Move, b *Board) {
	if r.IsRightRook() {
		b.SetFlag(FlagRightRookMoved, r.GetColor(), true)
	}
	if r.IsLeftRook() {
		b.SetFlag(FlagLeftRookMoved, r.GetColor(), true)
	}
}

func (r *Rook) IsRightRook() bool {
	return r.Location.Col == 7
}

func (r *Rook) IsLeftRook() bool {
	return r.Location.Col == 0
}

func (r *Rook) IsStartingRow() bool {
	if r.Color == color.Black {
		return r.Location.Row == 0
	} else if r.Color == color.White {
		return r.Location.Row == 7
	} else {
		log.Fatal("Invalid Color")
	}
	return false
}
