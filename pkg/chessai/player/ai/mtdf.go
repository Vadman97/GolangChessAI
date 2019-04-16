package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"time"
)

func (m *MTDf) MTDf(root *board.Board, guess *ScoredMove, previousMove *board.LastMove) *ScoredMove {
	lowerBound := NegInf
	upperBound := PosInf
	for true {
		var beta int
		if guess.Score == lowerBound {
			beta = guess.Score + 1
		} else {
			beta = guess.Score
		}
		guess = m.ab.AlphaBetaWithMemory(root, m.currentSearchDepth, beta-1, beta, m.player.PlayerColor, previousMove)
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

func (m *MTDf) IterativeMTDf(b *board.Board, guess *ScoredMove, previousMove *board.LastMove) *ScoredMove {
	if guess == nil {
		guess = &ScoredMove{
			Score: 0,
		}
	}
	start := time.Now()
	iterativeIncrement := config.Get().IterativeIncrement
	for m.currentSearchDepth = iterativeIncrement; m.currentSearchDepth <= m.player.MaxSearchDepth; m.currentSearchDepth += iterativeIncrement {
		thinking := make(chan bool)
		go m.player.trackThinkTime(thinking, start)
		newGuess := m.MTDf(b, guess, previousMove)
		close(thinking)
		if !m.player.abort {
			guess = newGuess
			m.lastSearchDepth = m.currentSearchDepth
		} else {
			// -1 due to discard of current level due to hard abort
			m.lastSearchDepth = m.currentSearchDepth - iterativeIncrement
			m.player.printer <- fmt.Sprintf("MTDf hard abort! evaluated to depth %d\n", m.lastSearchDepth)
			break
		}
	}
	m.lastSearchTime = time.Now().Sub(start)
	return guess
}

type MTDf struct {
	player             *AIPlayer
	ab                 AlphaBetaWithMemory
	currentSearchDepth int
	lastSearchDepth    int
	lastSearchTime     time.Duration
}

func (m *MTDf) GetName() string {
	return fmt.Sprintf("%s,[D:%d;T:%s]", AlgorithmMTDf, m.lastSearchDepth, m.lastSearchTime)
}

func (m *MTDf) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	m.player = p
	m.ab = AlphaBetaWithMemory{player: p}
	return m.IterativeMTDf(b, nil, previousMove)
}
