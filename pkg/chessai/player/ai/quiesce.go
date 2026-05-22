package ai

import "github.com/Vadman97/GolangChessAI/pkg/chessai/board"

func (p *AIPlayer) Quiesce(root *board.Board, alpha, beta int, currentPlayer byte, previousMove *board.LastMove) int {
	// Generate all moves first so terminal detection uses correct previousMove (en passant included).
	moves := root.GetAllMoves(currentPlayer, previousMove)
	if p.terminalNode(root, moves) {
		return AdjustMateScore(p.EvaluateBoard(root, currentPlayer).TotalScore, 0)
	}
	standPat := p.EvaluateBoard(root, currentPlayer).TotalScore
	if standPat >= beta {
		return beta
	} else if alpha < standPat {
		alpha = standPat
	}
	// until every capture has been examined
	for _, m := range *moves {
		if p.abort {
			break
		}
		// capture move
		if !root.IsEmpty(m.End) {
			child := root.Copy()
			lastMove := board.MakeMove(&m, child)
			p.Metrics.MovesConsidered++
			score := -p.Quiesce(child, -beta, -alpha, currentPlayer^1, lastMove)

			if score >= beta {
				p.Metrics.MovesPrunedAB++
				return beta
			} else if score > alpha {
				alpha = score
			}
		}
	}
	return alpha
}
