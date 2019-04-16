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
		t := ScoredMove{}
		b := beta
		moves := root.GetAllMoves(currentPlayer, previousMove)
		for i, m := range *moves {
			if n.abort {
				break
			}
			newBoard := root.Copy()
			previousMove = board.MakeMove(&m, newBoard)
			n.player.Metrics.MovesConsidered++
			t = n.NegaScout(newBoard, depth-1, ScoredMove{
				Move:  b.Move,
				Score: -b.Score,
			}, ScoredMove{
				Move:  alpha.Move,
				Score: -alpha.Score,
			}, currentPlayer^1, previousMove)
			t.Score = -t.Score

			if t.Score > alpha.Score && t.Score < beta.Score && i > 0 {
				// re-search
				t = n.NegaScout(newBoard, depth-1, ScoredMove{
					Move:  beta.Move,
					Score: -beta.Score,
				}, ScoredMove{
					Move:  alpha.Move,
					Score: -alpha.Score,
				}, currentPlayer^1, previousMove)
				t.Score = -t.Score
			}

			t.Move = m
			t.MoveSequence = append(t.MoveSequence, m)

			if t.Score >= alpha.Score {
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
