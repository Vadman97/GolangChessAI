package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"time"
)

/**
 * Based on https://www.chessprogramming.org/NegaScout#Alternative
 */
func (n *NegaScout) NegaScout(root *board.Board, depth int, alpha, beta ScoredMove, currentPlayer color.Color, previousMove *board.LastMove) ScoredMove {
	if depth == 0 {
		// leaf node
		return ScoredMove{
			Score: n.player.Quiesce(root, alpha.Score, beta.Score, currentPlayer, previousMove),
			//Score: n.player.EvaluateBoard(root, currentPlayer).TotalScore,
		}
	} else {
		a := alpha
		b := beta
		moves := root.GetAllMoves(currentPlayer, previousMove)
		for i, m := range *moves {
			if n.player.abort {
				return a
			}
			newBoard := root.Copy()
			previousMove = board.MakeMove(&m, newBoard)
			n.player.Metrics.MovesConsidered++

			// search
			t := n.NegaScout(newBoard, depth-1, b.NegScore(), a.NegScore(), currentPlayer^1, previousMove).NegScore()
			t.Move = m

			if t.Score > a.Score && t.Score < beta.Score && i > 0 && depth < n.startDepth-1 {
				// re-search
				a = n.NegaScout(newBoard, depth-1, beta.NegScore(), t.NegScore(), currentPlayer^1, previousMove).NegScore()
				a.Move = m
			}

			if t.Score > a.Score {
				a = t
			}

			// cut-off
			if a.Score >= beta.Score {
				n.player.Metrics.MovesPrunedAB += int64(len(*moves) - i)
				break
			}
			// set new null window
			b = a
			b.Score++
		}
		return a
	}
}

func (n *NegaScout) IterativeNegaScout(b *board.Board, previousMove *board.LastMove) ScoredMove {
	start := time.Now()
	best := ScoredMove{}
	for n.currentSearchDepth = 1; n.currentSearchDepth <= n.player.MaxSearchDepth; n.currentSearchDepth += 1 {
		thinking, done := make(chan bool), make(chan bool, 1)
		go n.player.trackThinkTime(thinking, done, start)
		newBest := n.NegaScout(b, n.currentSearchDepth, ScoredMove{
			Move:  location.Move{},
			Score: NegInf,
		}, ScoredMove{
			Move:  location.Move{},
			Score: PosInf,
		}, n.player.PlayerColor, previousMove)
		close(thinking)
		<-done
		// did not abort search, good value
		if !n.player.abort {
			best = newBest
			n.lastSearchDepth = n.currentSearchDepth
			n.player.printer <- fmt.Sprintf("Best D:%d M:%s\n", n.lastSearchDepth, best.Move.Print())
		} else {
			// -1 due to discard of current level due to hard abort
			n.lastSearchDepth = n.currentSearchDepth - 1
			n.player.printer <- fmt.Sprintf("NegaScout hard abort! evaluated to depth %d\n", n.lastSearchDepth)
			break
		}
	}
	n.lastSearchTime = time.Now().Sub(start)
	return best
}

type NegaScout struct {
	player             *AIPlayer
	startDepth         int
	currentSearchDepth int
	lastSearchDepth    int
	lastSearchTime     time.Duration
}

func (n *NegaScout) GetName() string {
	return fmt.Sprintf("%s,[D:%d;T:%s]", AlgorithmNegaScout, n.lastSearchDepth, n.lastSearchTime)
}

func (n *NegaScout) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	n.player = p
	n.player.abort = false

	n.startDepth = p.MaxSearchDepth
	best := n.IterativeNegaScout(b, previousMove)

	return &best
}
