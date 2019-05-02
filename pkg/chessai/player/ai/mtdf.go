package ai

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"log"
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
		thinking, done := make(chan bool), make(chan bool, 1)
		go m.player.trackThinkTime(thinking, done, start)
		newGuess := m.MTDf(b, guess, previousMove)
		close(thinking)
		<-done
		// MTDf returns a good move (did not abort search)
		if !m.player.abort {
			guess = newGuess
			m.player.LastSearchDepth = m.currentSearchDepth
			m.player.printer <- fmt.Sprintf("Best D:%d M:%s\n", m.player.LastSearchDepth, guess.Move)
		} else {
			// -1 due to discard of current level due to hard abort
			m.player.LastSearchDepth = m.currentSearchDepth - iterativeIncrement
			m.player.printer <- fmt.Sprintf("MTDf hard abort! evaluated to depth %d\n", m.player.LastSearchDepth)
			break
		}
	}
	return guess
}

type MTDf struct {
	player             *AIPlayer
	ab                 AlphaBetaWithMemory
	currentSearchDepth int
}

func (m *MTDf) GetName() string {
	return AlgorithmMTDf
}

func (m *MTDf) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	m.player = p
	m.player.abort = false
	m.ab = AlphaBetaWithMemory{player: p}

	if !b.CacheGetAllMoves || !b.CacheGetAllAttackableMoves {
		log.Printf("Trying to use %s without move caching enabled.\n", m.GetName())
		log.Println("Enabling GetAllMoves, GetAllAttackableMoves caching.")
		b.CacheGetAllMoves = true
		b.CacheGetAllAttackableMoves = true
	}

	return m.IterativeMTDf(b, nil, previousMove)
}
