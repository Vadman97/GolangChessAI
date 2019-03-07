package ai

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
)

func (p *Player) MiniMaxRecurse(b *board.Board, m *board.Move, depth int, currentPlayer byte) *ScoredMove {
	newBoard := b.Copy()
	board.MakeMove(m, newBoard)
	candidate := p.MiniMax(newBoard, depth-1, (currentPlayer+1)%color.NumColors)
	candidate.Move = m
	return candidate
}

func (p *Player) MiniMax(b *board.Board, depth int, currentPlayer byte) *ScoredMove {
	if depth == 0 {
		return &ScoredMove{
			Move:  nil,
			Score: p.EvaluateBoard(b).TotalScore,
		}
	}

	var best ScoredMove
	if currentPlayer == p.PlayerColor {
		// maximizing player
		best.Score = NegInf
	} else {
		// minimizing player
		best.Score = PosInf
	}
	moves := b.GetAllMoves(p.PlayerColor)
	for _, m := range *moves {
		candidate := p.MiniMaxRecurse(b, &m, depth, currentPlayer)
		best = *compare(currentPlayer == p.PlayerColor, &best, candidate)
	}

	return &best
}
