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
func SetLocationAttackable(attackableBoard AttackableBoard, location Location) {
	attackableBoard[location.Row] |= 1 << uint(location.Col)
}

/**
 * Returns a boolean indicating if a specific location on a board is attackable.
 */
func IsLocationUnderAttack(attackableBoard AttackableBoard, location Location) bool {
	return ((*attackableBoard)[location.Row] & (1 << uint(location.Col))) != 0
}
