package player

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/piece"
	"fmt"
	"math"
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

func compare(maximizingP bool, currentBest *ScoredMove, candidate *ScoredMove) *ScoredMove {
	if maximizingP {
		if candidate.Score > currentBest.Score {
			return candidate
		} else {
			return currentBest
		}
	} else {
		if candidate.Score < currentBest.Score {
			return candidate
		} else {
			return currentBest
		}
	}
}

func (p *AIPlayer) MiniMax(b *board.Board, depth int, currentPlayer byte) *ScoredMove {
	if depth == 0 {
		return &ScoredMove{
			Move:  nil,
			Score: p.EvaluateBoard(b).TotalScore,
		}
	}

	var best ScoredMove
	if currentPlayer == p.PlayerColor {
		// maximizing player
		best.Score = math.MinInt32
	} else {
		// minimizing player
		best.Score = math.MaxInt32
	}
	for _, m := range *b.GetAllMoves(p.PlayerColor) {
		newBoard := b.Copy()
		board.MakeMove(&m, newBoard)
		candidate := p.MiniMax(newBoard, depth-1, (currentPlayer+1)%color.NumColors)
		candidate.Move = &m
		best = *compare(currentPlayer == p.PlayerColor, &best, candidate)
	}
	return &best
}

func (p *AIPlayer) GetBestMove(b *board.Board) *board.Move {
	m := p.MiniMax(b, 4, p.PlayerColor)
	fmt.Printf("AI Player best move leads to score %d\n", m.Score)
	return m.Move
}

func (p *AIPlayer) MakeMove(b *board.Board) {
	board.MakeMove(p.GetBestMove(b), b)
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
