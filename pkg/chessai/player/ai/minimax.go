package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
)

func (p *Player) MiniMaxRecurse(b *board.Board, m location.Move, depth int, currentPlayer byte) *ScoredMove {
	newBoard := b.Copy()
	board.MakeMove(&m, newBoard)
	p.Metrics.MovesConsidered++
	candidate := p.MiniMax(newBoard, depth-1, currentPlayer^1)
	candidate.Move = m
	candidate.MoveSequence = append(candidate.MoveSequence, m)
	return candidate
}

func (p *Player) MiniMax(b *board.Board, depth int, currentPlayer byte) *ScoredMove {
	if depth == 0 {
		return &ScoredMove{
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
	moves := b.GetAllMoves(currentPlayer)
	for _, m := range *moves {
		candidate := p.MiniMaxRecurse(b, m, depth, currentPlayer)
		if betterMove(currentPlayer == p.PlayerColor, &best, candidate) {
			best = *candidate
		}
	}

	return &best
}
