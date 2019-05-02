package ai

import "github.com/Vadman97/GolangChessAI/pkg/chessai/board"

func (p *AIPlayer) Quiesce(root *board.Board, alpha, beta Value, currentPlayer byte, previousMove *board.LastMove) Value {
	standPat := p.EvaluateBoard(root, currentPlayer).TotalScore
	if standPat >= beta {
		return beta
	} else if alpha < standPat {
		alpha = standPat
	}
	// until every capture has been examined
	moves := root.GetAllMoves(currentPlayer, previousMove)
	for _, m := range *moves {
		if p.abort {
			break
		}
		// capture move
		if !root.IsEmpty(m.End) {
			child := root.Copy()
			board.MakeMove(&m, child)
			p.Metrics.MovesConsidered++
			score := -p.Quiesce(child, -beta, -alpha, currentPlayer^1, previousMove)

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
