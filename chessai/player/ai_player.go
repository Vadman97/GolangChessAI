package player

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/piece"
)

var PieceValue = map[byte]int{
	piece.PawnType:   1,
	piece.BishopType: 3,
	piece.KnightType: 3,
	piece.RookType:   5,
	piece.QueenType:  9,
	piece.KingType:   100,
}

type ScoredMove struct {
	Move  *board.Move
	Score int
}

type AIPlayer struct {
	TurnCount   int
	PlayerColor byte
}

func (p *AIPlayer) MakeMove() {
	// TODO(Vadim)
	p.TurnCount++
}

func (p *AIPlayer) EvaluateBoard(b *board.Board) *board.Evaluation {
	// TODO(Vadim) make more intricate
	eval := board.Evaluation{
		PieceCounts: map[byte]map[byte]uint8{
			color.Black: {},
			color.White: {},
		},
	}

	for r := int8(0); r < board.Width; r++ {
		for c := int8(0); c < board.Height; c++ {
			if p := b.GetPiece(board.Location{Row: r, Col: c}); p != nil {
				_, ok := eval.PieceCounts[p.GetColor()][p.GetPieceType()]
				if !ok {
					eval.PieceCounts[p.GetColor()][p.GetPieceType()] = 1
				} else {
					eval.PieceCounts[p.GetColor()][p.GetPieceType()]++
				}
			}
		}
	}

	for c := byte(0); c < color.NumColors; c++ {
		for pieceType, value := range PieceValue {
			cMult := 1
			if c != p.PlayerColor {
				cMult = -1
			}
			eval.TotalScore += value * int(eval.PieceCounts[c][pieceType]) * cMult
		}
	}

	return &eval
}
