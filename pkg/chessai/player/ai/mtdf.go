package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"log"
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

func (m *MTDf) IterativeMTDf(b *board.Board, guess *ScoredMove, previousMove *board.LastMove) *ScoredMove {
	if guess == nil {
		guess = &ScoredMove{
			Score: 0,
		}
	}
	start := time.Now()
	for m.currentSearchDepth = 1; m.currentSearchDepth <= m.player.MaxSearchDepth; m.currentSearchDepth++ {
		thinking := true
		if m.player.MaxThinkTime != 0 {
			go func() {
				for thinking {
					thinkTime := time.Now().Sub(start)
					if thinkTime > m.player.MaxThinkTime {
						m.ab.abort = true
						fmt.Println("MTDf requesting AB hard abort, out of time!")
					}
					time.Sleep(100 * time.Millisecond)
				}
			}()
		}
		newGuess := m.MTDf(b, guess, m.player.PlayerColor, previousMove)
		thinking = false
		// MTDf returns a good move (did not abort search)
		if !m.ab.abort {
			guess = newGuess
		} else {
			// -1 due to discard of current level due to hard abort
			fmt.Printf("MTDf hard abort! evaluated to depth %d\n", m.currentSearchDepth-1)
			break
		}
	}
	return guess
}

type MTDf struct {
	player             *Player
	ab                 AlphaBetaWithMemory
	currentSearchDepth int
}

func (m *MTDf) GetName() string {
	return AlgorithmMTDf
}

func (m *MTDf) GetBestMove(p *Player, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	m.player = p
	m.ab = AlphaBetaWithMemory{player: p}
	return m.IterativeMTDf(b, nil, previousMove)
}
