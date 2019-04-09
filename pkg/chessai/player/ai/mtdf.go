package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
)

func (p *Player) MTDf(root *board.Board, guess *ScoredMove, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
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

func (p *Player) IterativeMTDf(b *board.Board, guess *ScoredMove, previousMove *board.LastMove) *ScoredMove {
	if guess == nil {
		guess = &ScoredMove{
			Score: 0,
		}
	}
	for p.CurrentSearchDepth = 1; p.CurrentSearchDepth <= p.MaxSearchDepth; p.CurrentSearchDepth++ {
		guess = p.MTDf(b, guess, p.PlayerColor, previousMove)
	}
	return guess
}

type MTDf struct{}

func (m *MTDf) GetName() string {
	return AlgorithmMTDf
}

func (m *MTDf) GetBestMove(p *Player, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	return p.IterativeMTDf(b, nil, previousMove)
}
