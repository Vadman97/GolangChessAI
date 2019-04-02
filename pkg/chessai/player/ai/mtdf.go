package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
)

func (p *Player) MTDF(b *board.Board, guess *ScoredMove, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
	lowerBound := NegInf
	upperBound := PosInf
	for true {
		beta := util.MaxScore(guess.Score, lowerBound+1)

		guess = p.AlphaBetaWithMemory(b, p.CurrentSearchDepth, beta-1, beta, currentPlayer, previousMove)
		if guess.Score < beta {
			upperBound = guess.Score
		} else {
			lowerBound = guess.Score
		}

		if lowerBound >= upperBound {
			break
		}
	}
	return guess
}

func (p *Player) IterativeMTDF(b *board.Board, guess *ScoredMove, previousMove *board.LastMove) *ScoredMove {
	for p.CurrentSearchDepth = 1; p.CurrentSearchDepth <= p.MaxSearchDepth; p.CurrentSearchDepth++ {
		guess = p.MTDF(b, guess, p.PlayerColor, previousMove)
	}
	return guess
}
