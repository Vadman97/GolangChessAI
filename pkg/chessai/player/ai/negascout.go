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
			//Score: n.player.Quiesce(root, alpha.Score, beta.Score, currentPlayer, previousMove),
			Score: n.player.EvaluateBoard(root, currentPlayer).TotalScore,
		}
	} else {
		b := beta
		moves := root.GetAllMoves(currentPlayer, previousMove)
		for i, m := range *moves {
			if n.abort {
				return alpha
			}
			newBoard := root.Copy()
			previousMove = board.MakeMove(&m, newBoard)
			n.player.Metrics.MovesConsidered++
			newAlpha, newBeta := b.NegScore(), alpha.NegScore()
			t := n.NegaScout(newBoard, depth-1, newAlpha, newBeta, currentPlayer^1, previousMove).NegScore()

			if t.Score > alpha.Score && t.Score < beta.Score && i > 0 {
				// re-search
				newAlpha, newBeta := beta.NegScore(), alpha.NegScore()
				t = n.NegaScout(newBoard, depth-1, newAlpha, newBeta, currentPlayer^1, previousMove).NegScore()
			}

			if t.Score > alpha.Score {
				t.Move = m
				alpha = t
			}
			// cut-off
			if alpha.Score >= beta.Score {
				n.player.Metrics.MovesPrunedAB += int64(len(*moves) - i)
				return alpha
			}
			// set new null window
			b = alpha
			b.Score++
		}
		return alpha
	}
}

type NegaScout struct {
	player          *AIPlayer
	abort           bool
	lastSearchDepth int
}

func (n *NegaScout) GetName() string {
	return fmt.Sprintf("%s,[depth:%d]", AlgorithmNegaScout, n.lastSearchDepth)
}

func (n *NegaScout) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	n.player = p
	n.abort = false
	n.lastSearchDepth = p.MaxSearchDepth
	best := n.NegaScout(b, p.MaxSearchDepth, ScoredMove{
		Move:  location.Move{},
		Score: NegInf,
	}, ScoredMove{
		Move:  location.Move{},
		Score: PosInf,
	}, p.PlayerColor, previousMove)
	return &best
}
