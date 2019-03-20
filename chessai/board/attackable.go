package board

type AttackableBoard *[8]byte

/**
 * Creates an empty array of 8 bytes all filled with 0's - represents that no square
 * is attackable.
 */
func CreateEmptyAttackableBoard() AttackableBoard {
	return &([8]byte{0, 0, 0, 0, 0, 0, 0, 0})
}

/**
 * Creates an AttackableBoard for a set of moves (using End locations for each move).
 */
func CreateAttackableBoardFromMoves(moves *[]Move) AttackableBoard {
	attackableBoard := CreateEmptyAttackableBoard()
	for i := range *moves {
		location := (*moves)[i].End
		SetSquareAttackable(attackableBoard, location)
	}
	return attackableBoard
}

/**
 * Performs a logical OR of each row in two boards (placing result into boardOne).
 */
func CombineAttackableBoards(boardOne AttackableBoard, boardTwo AttackableBoard) AttackableBoard {
	for r := 0; r < Height; r++ {
		(*boardOne)[r] |= (*boardTwo)[r]
	}
	return boardOne
}

/**
 * Makes a specific square attackable on an AttackableBoard.
 */
func SetSquareAttackable(attackableBoard AttackableBoard, location Location) {
	attackableBoard[location.Row] |= 1 << uint(location.Col)
}

/**
 * Returns a boolean indicating if a specific location on a board is attackable.
 */
func IsLocationAttackable(attackableBoard AttackableBoard, location Location) bool {
	return ((*attackableBoard)[location.Row] & (1 << uint(location.Col))) != 0
}
