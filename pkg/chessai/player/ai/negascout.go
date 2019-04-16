package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
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
			if n.abort {
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
				return a
			}
			// set new null window
			b = a
			b.Score++
		}
		return a
	}
}

type NegaScout struct {
	player          *AIPlayer
	abort           bool
	startDepth      int
	lastSearchDepth int
}

func (n *NegaScout) GetName() string {
	return fmt.Sprintf("%s,[depth:%d]", AlgorithmNegaScout, n.lastSearchDepth)
}

func (n *NegaScout) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	n.player = p
	n.abort = false
	n.startDepth = p.MaxSearchDepth
	best := n.NegaScout(b, n.startDepth, ScoredMove{
		Move:  location.Move{},
		Score: NegInf,
	}, ScoredMove{
		Move:  location.Move{},
		Score: PosInf,
	}, p.PlayerColor, previousMove)
	n.lastSearchDepth = n.startDepth
	return &best
}
