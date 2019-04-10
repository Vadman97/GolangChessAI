package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"time"
)

func (m *MTDf) MTDf(root *board.Board, guess *ScoredMove, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
	lowerBound := NegInf
	upperBound := PosInf
	for true {
		var beta int
		if guess.Score == lowerBound {
			beta = guess.Score + 1
		} else {
			beta = guess.Score
		}
		guess = m.ab.AlphaBetaWithMemory(root, m.currentSearchDepth, beta-1, beta, currentPlayer, previousMove)
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

func (m *MTDf) trackThinkTime(stop chan bool, start time.Time) {
	if m.player.MaxThinkTime != 0 {
		for {
			select {
			case <-stop:
				return
			default:
				thinkTime := time.Now().Sub(start)
				if thinkTime > m.player.MaxThinkTime {
					m.ab.abort = true
					fmt.Println("MTDf requesting AB hard abort, out of time!")
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (m *MTDf) IterativeMTDf(b *board.Board, guess *ScoredMove, previousMove *board.LastMove) *ScoredMove {
	if guess == nil {
		guess = &ScoredMove{
			Score: 0,
		}
	}
	start := time.Now()
	for m.currentSearchDepth = 1; m.currentSearchDepth <= m.player.MaxSearchDepth; m.currentSearchDepth++ {
		thinking := make(chan bool)
		go m.trackThinkTime(thinking, start)
		newGuess := m.MTDf(b, guess, m.player.PlayerColor, previousMove)
		close(thinking)
		// MTDf returns a good move (did not abort search)
		if !m.ab.abort {
			guess = newGuess
			m.lastSearchDepth = m.currentSearchDepth
		} else {
			// -1 due to discard of current level due to hard abort
			m.lastSearchDepth = m.currentSearchDepth - 1
			fmt.Printf("MTDf hard abort! evaluated to depth %d\n", m.lastSearchDepth)
			break
		}
	}
	m.lastSearchTime = time.Now().Sub(start)
	return guess
}

type MTDf struct {
	player             *Player
	ab                 AlphaBetaWithMemory
	currentSearchDepth int
	lastSearchDepth    int
	lastSearchTime     time.Duration
}

func (m *MTDf) GetName() string {
	return fmt.Sprintf("%s,[depth:%d]", AlgorithmMTDf, m.lastSearchDepth)
}

func (m *MTDf) GetBestMove(p *Player, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	m.player = p
	m.ab = AlphaBetaWithMemory{player: p}
	return m.IterativeMTDf(b, nil, previousMove)
}
