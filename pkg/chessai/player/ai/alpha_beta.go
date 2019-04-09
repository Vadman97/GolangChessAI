package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
)

func (p *Player) Quiesce(root *board.Board, alpha, beta int, currentPlayer byte, previousMove *board.LastMove) int {
	standPat := p.EvaluateBoard(root).TotalScore
	if standPat >= beta {
		return beta
	} else if alpha < standPat {
		alpha = standPat
	}
	// until every capture has been examined
	moves := root.GetAllMoves(currentPlayer, previousMove)
	for _, m := range *moves {
		// capture move
		if !root.IsEmpty(m.End) {
			child := root.Copy()
			board.MakeMove(&m, child)
			score := p.Quiesce(child, alpha, beta, currentPlayer^1, previousMove)

			if score >= beta {
				return beta
			} else if score > alpha {
				alpha = score
			}
		}
	}
	return alpha
}

func (p *Player) AlphaBetaWithMemory(root *board.Board, depth, alpha, beta int, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
	var h util.BoardHash
	if p.TranspositionTableEnabled {
		// transposition table lookup
		h = root.Hash()
		if entry, ok := p.alphaBetaTable.Read(&h, currentPlayer); ok {
			if entry.Lower >= beta {
				p.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Move:  entry.BestMove,
					Score: entry.Lower,
				}
			} else if entry.Upper <= alpha {
				p.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Move:  entry.BestMove,
					Score: entry.Upper,
				}
			}
			if entry.Lower > alpha {
				p.Metrics.MovesABImprovedTransposition++
				alpha = entry.Lower
			}
			if entry.Upper < beta {
				p.Metrics.MovesABImprovedTransposition++
				beta = entry.Upper
			}
		}
	}
	var best ScoredMove
	if depth == 0 {
		best = ScoredMove{
			Score: p.EvaluateBoard(root).TotalScore,
			// TODO(Vadim) compare quiescence with none
			//Score: p.Quiesce(root, alpha, beta, currentPlayer, previousMove),
		}
	} else {
		var maximizingPlayer = currentPlayer == p.PlayerColor
		var a, b int
		if maximizingPlayer {
			best.Score = NegInf
			a = alpha
		} else {
			best.Score = PosInf
			b = beta
		}
		moves := root.GetAllMoves(currentPlayer, previousMove)
		for i, m := range *moves {
			if maximizingPlayer {
				if best.Score >= beta {
					p.Metrics.MovesPrunedAB += int64(len(*moves) - i)
					break
				}
			} else {
				if best.Score <= alpha {
					p.Metrics.MovesPrunedAB += int64(len(*moves) - i)
					break
				}
			}
			newBoard := root.Copy()
			previousMove = board.MakeMove(&m, newBoard)
			p.Metrics.MovesConsidered++
			var candidate *ScoredMove
			if maximizingPlayer {
				candidate = p.AlphaBetaWithMemory(newBoard, depth-1, a, beta, currentPlayer^1, previousMove)
			} else {
				candidate = p.AlphaBetaWithMemory(newBoard, depth-1, alpha, b, currentPlayer^1, previousMove)
			}
			candidate.Move = m
			candidate.MoveSequence = append(candidate.MoveSequence, m)
			if betterMove(maximizingPlayer, &best, candidate) {
				best = *candidate
			}
			if maximizingPlayer {
				a = util.MaxScore(best.Score, a)
			} else {
				b = util.MinScore(best.Score, b)
			}
		}
	}

	if p.TranspositionTableEnabled {
		if best.Score <= alpha {
			p.alphaBetaTable.Store(&h, currentPlayer, &util.TranspositionTableEntry{
				Lower:    NegInf,
				Upper:    best.Score,
				BestMove: best.Move,
			})
		}
		if best.Score > alpha && best.Score < beta {
			p.alphaBetaTable.Store(&h, currentPlayer, &util.TranspositionTableEntry{
				Lower:    best.Score,
				Upper:    best.Score,
				BestMove: best.Move,
			})
		}
		if best.Score >= beta {
			p.alphaBetaTable.Store(&h, currentPlayer, &util.TranspositionTableEntry{
				Lower:    best.Score,
				Upper:    PosInf,
				BestMove: best.Move,
			})
		}
	}

	return &best
}
