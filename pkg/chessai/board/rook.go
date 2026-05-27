package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
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
func (r *Rook) GetMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	for i := 0; i < 4; i++ {
		l := r.GetPosition()
		var inBounds bool
		for true {
			if i == 0 {
				l, inBounds = l.AddRelative(location.UpMove)
			} else if i == 1 {
				l, inBounds = l.AddRelative(location.RightMove)
			} else if i == 2 {
				l, inBounds = l.AddRelative(location.DownMove)
			} else if i == 3 {
				l, inBounds = l.AddRelative(location.LeftMove)
			}
			if !inBounds {
				break
			}
			validMove, checkNext := CheckLocationForPiece(r.Color, l, board)
			if validMove {
				possibleMove := location.Move{Start: r.GetPosition(), End: l}
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
 * Retrieves all locations that this rook can attack.
 */
func (r *Rook) GetAttackableMoves(board *Board) BitBoard {
	attackableBoard := BitBoard(0)
	for i := 0; i < 4; i++ {
		loc := r.GetPosition()
		var inBounds bool
		for true {
			if i == 0 {
				loc, inBounds = loc.AddRelative(location.UpMove)
			} else if i == 1 {
				loc, inBounds = loc.AddRelative(location.RightMove)
			} else if i == 2 {
				loc, inBounds = loc.AddRelative(location.DownMove)
			} else if i == 3 {
				loc, inBounds = loc.AddRelative(location.LeftMove)
			}
			if !inBounds {
				break
			}
			attackableBoard.SetLocation(loc)
			if !CheckLocationForAttackability(loc, board) {
				break
			}
		}
	}
	return attackableBoard
}

func (r *Rook) Move(m *location.Move, b *Board) {
	// Check the START column, not r.Location: by the time Move() is called the piece
	// is already at the destination, so r.Location reflects the end square. A rook moving
	// from col 7 (a-file) to col 5 (c-file) would incorrectly pass neither check if we
	// used r.Location. The same logic applies when a rook is captured (piece.go calls
	// Move with Start=End=captureSquare) — that still works because the captured rook's
	// position is correct at capture time.
	if m.Start.GetCol() == 7 {
		b.SetFlag(FlagRightRookMoved, r.GetColor(), true)
	}
	if m.Start.GetCol() == 0 {
		b.SetFlag(FlagLeftRookMoved, r.GetColor(), true)
	}
}

func (r *Rook) IsRightRook() bool {
	return r.Location.GetCol() == 7
}

func (r *Rook) IsLeftRook() bool {
	return r.Location.GetCol() == 0
}

func (r *Rook) IsStartingRow() bool {
	if r.Color == color.Black {
		return r.Location.GetRow() == 0
	} else if r.Color == color.White {
		return r.Location.GetRow() == 7
	}
	return false
}
