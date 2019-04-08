package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"log"
)

func (p *Player) MTDF(root *board.Board, guess *ScoredMove, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
	lowerBound := NegInf
	upperBound := PosInf
	for true {
		var beta int
		if guess.Score == lowerBound {
			beta = guess.Score + 1
		} else {
			beta = guess.Score
		}

		guess = p.AlphaBetaWithMemory(root, p.CurrentSearchDepth, beta-1, beta, currentPlayer, previousMove)
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
	if guess.Move.Start.Equals(guess.Move.End) {
		log.Printf("MTD-f resigns, no best move available.\n")
		return p.RandomMove(b, previousMove)
	}
	return guess
}
