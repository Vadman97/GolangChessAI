package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
)

func (p *Player) MTDF(b *board.Board, guess *ScoredMove, depth int, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
	lowerBound := NegInf
	upperBound := PosInf
	for lowerBound < upperBound {
		beta := guess.Score
		if lowerBound+1 > beta {
			beta = lowerBound + 1
		}
		guess = p.AlphaBetaWithMemory(b, depth, beta-1, beta, currentPlayer, previousMove)
		if guess.Score < beta {
			upperBound = guess.Score
		} else {
			lowerBound = guess.Score
		}
	}
	return guess
}
