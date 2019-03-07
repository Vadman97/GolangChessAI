package board

import "ChessAI3/chessai/board/color"

type Evaluation struct {
	// [color][pieceType] -> overall piece count
	PieceCounts map[byte]map[byte]uint8
	// [color][pieceType] -> count of pieces off starting position
	PieceAdvanced map[byte]map[byte]uint8
	// [color][column] -> num pawns
	PawnColumns map[byte]map[int8]uint8
	// [color] -> num moves
	NumMoves   map[byte]uint8
	TotalScore int
}

func NewEvaluation() *Evaluation {
	e := Evaluation{
		PieceCounts: map[byte]map[byte]uint8{
			color.Black: {},
			color.White: {},
		},
		PieceAdvanced: map[byte]map[byte]uint8{
			color.Black: {},
			color.White: {},
		},
		PawnColumns: map[byte]map[int8]uint8{
			color.Black: {},
			color.White: {},
		},
	}
	return &e
}
