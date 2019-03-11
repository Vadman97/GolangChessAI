package board

import (
	"ChessAI3/chessai/board/piece"
)

var possibleMoves = []Location{
	{-2, 1},
	{-1, 2},
	{1, 2},
	{2, 1},
	{2, -1},
	{1, -2},
	{-2, -1},
	{-1, -2},
}

type Knight struct {
	Location Location
	Color    byte
}

func (r *Knight) GetChar() rune {
	return piece.KnightChar
}

func (r *Knight) GetPieceType() byte {
	return piece.KnightType
}

func (r *Knight) GetColor() byte {
	return r.Color
}

func (r *Knight) SetColor(color byte) {
	r.Color = color
}

func (r *Knight) SetPosition(loc Location) {
	r.Location.Set(loc)
}

func (r *Knight) GetPosition() Location {
	return r.Location
}

func (r *Knight) GetMoves(board *Board) *[]Move {
	var moves []Move
	for _, possibleMove := range possibleMoves {
		l := r.GetPosition()
		l = l.Add(possibleMove)
		validMove, _ := CheckLocationForPiece(r.GetColor(), l, board)
		if validMove {
			moves = append(moves, Move{r.GetPosition(), l})
		}
	}
	return &moves
}

func (r *Knight) GetAttackableMoves(board *Board) *[]Move {
	//TODO (Devan)
	return nil
}

func (r *Knight) Move(m *Move, b *Board) {}
