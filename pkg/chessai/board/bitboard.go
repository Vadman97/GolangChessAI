package board

import "github.com/Vadman97/GolangChessAI/pkg/chessai/location"

type BitBoard uint64

/**
 * Performs a logical OR of each row in two boards (making a new board of the results).
 */
func (bb BitBoard) CombineBitBoards(other BitBoard) BitBoard {
	return bb | other
}

/**
 * Performs a logical AND of each row in two boards (making a new board of the results).
 */
func (bb BitBoard) IntersectBitBoards(other BitBoard) BitBoard {
	return bb & other
}

/**
 * Makes a specific square set on a BitBoard.
 */
func (bb *BitBoard) SetLocation(location location.Location) {
	row, col := location.Get()
	*bb |= 1 << (Width*uint(row) + uint(col))
}

/**
 * Returns a boolean indicating if a specific location on a board is set.
 */
func (bb *BitBoard) IsLocationSet(location location.Location) bool {
	row, col := location.Get()
	return (*bb & (1 << (Width*uint(row) + uint(col)))) != 0
}
