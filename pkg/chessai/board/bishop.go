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
	board.appendSlidingMoves(r.Color, r.Location, &diagDirs, &moves, onlyFirstMove)
	return &moves
}

/**
 * Retrieves all squares that this bishop can attack.
 */
func (r *Bishop) GetAttackableMoves(board *Board) BitBoard {
	row, col := r.Location.Get()
	return board.slidingAttackBits(int(row), int(col), &diagDirs)
}

func (r *Bishop) Move(m *location.Move, b *Board) {}
