package ai

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
)

func (p *Player) MiniMaxRecurse(b *board.Board, m board.Move, depth int, currentPlayer byte) *ScoredMove {
	newBoard := b.Copy()
	board.MakeMove(&m, newBoard)
	candidate := p.MiniMax(newBoard, depth-1, (currentPlayer+1)%color.NumColors)
	candidate.Move = m
	candidate.MoveSequence = append(candidate.MoveSequence, m)
	return candidate
}

func (p *Player) MiniMax(b *board.Board, depth int, currentPlayer byte) *ScoredMove {
	if depth == 0 {
		eval := p.EvaluateBoard(b)
		return &ScoredMove{
			Score: eval.TotalScore,
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
		if compare(currentPlayer == p.PlayerColor, &best, candidate) {
			best = *candidate
		}
	}

	return &best
}
