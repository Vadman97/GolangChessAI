package board

import (
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

func (r *King) GetMoves(board *Board) *[]Move {
	var moves []Move

	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i != 0 || j != 0 {
				l := r.GetPosition()
				l = l.Add(Location{int8(i), int8(j)})
				if l.InBounds() {
					if !r.underAttack(l, board) {
						if board.IsEmpty(l) {
							validMove, _ := CheckLocationForPiece(r.GetColor(), l, board)
							if validMove {
								moves = append(moves, Move{r.GetPosition(), l})
							}
						} else if board.GetPiece(l).GetColor() != r.GetColor() {
							validMove, _ := CheckLocationForPiece(r.GetColor(), l, board)
							if validMove {
								moves = append(moves, Move{r.GetPosition(), l})
							}
						}
					}
				}
			}
		}
	}

	if !board.GetFlag(FlagCastled, r.GetColor()) && !board.GetFlag(FlagKingMoved, r.GetColor()) {
		right, left := r.GetPosition(), r.GetPosition()
		for i := 0; i < 2; i++ {
			right = right.Add(RightMove)
			left = left.Add(RightMove)
		}
		rightM, leftM := Move{r.GetPosition(), right}, Move{r.GetPosition(), left}
		if r.canCastle(&rightM, board) {
			moves = append(moves, rightM)
		}
		if r.canCastle(&leftM, board) {
			moves = append(moves, leftM)
		}
	}

	return &moves
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

func (r *King) canCastle(m *Move, b *Board) bool {
	if m.End.InBounds() {
		// rook can be under attack - only need to check two spaces where king will move
		for c := m.Start.Col; c <= m.End.Col; c++ {
			if r.underAttack(Location{m.End.Row, c}, b) {
				return false
			}
		}
		for c := m.Start.Col; c >= m.End.Col; c-- {
			if r.underAttack(Location{m.End.Row, c}, b) {
				return false
			}
		}
		if b.IsEmpty(m.End) {
			return true
		}
	}
	return false
}

func (r *King) underAttack(l Location, b *Board) bool {
	// TODO(Vadim) check space not under attack - efficient algo?
	return false
}
