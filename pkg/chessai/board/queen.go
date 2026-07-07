package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

type Queen struct {
	Location location.Location
	Color    color.Color
}

func (r *Queen) GetChar() rune {
	return piece.QueenChar
}

func (r *Queen) GetPieceType() byte {
	return piece.QueenType
}

func (r *Queen) GetColor() color.Color {
	return r.Color
}

func (r *Queen) SetColor(color color.Color) {
	r.Color = color
}

func (r *Queen) SetPosition(loc location.Location) {
	r.Location.Set(loc)
}

func (r *Queen) GetPosition() location.Location {
	return r.Location
}

/**
 * Calculates all valid moves for this queen.
 */
func (r *Queen) GetMoves(board *Board, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	if board.appendSlidingMoves(r.Color, r.Location, &orthoDirs, &moves, onlyFirstMove) {
		return &moves
	}
	board.appendSlidingMoves(r.Color, r.Location, &diagDirs, &moves, onlyFirstMove)
	return &moves
}

/**
 * Retrieves all squares that this queen can attack.
 */
func (r *Queen) GetAttackableMoves(board *Board) BitBoard {
	row, col := r.Location.Get()
	return board.slidingAttackBits(int(row), int(col), &orthoDirs) |
		board.slidingAttackBits(int(row), int(col), &diagDirs)
}

func (r *Queen) Move(m *location.Move, b *Board) {
	//TODO
}
