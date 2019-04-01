package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
)

func (p *Player) AlphaBetaWithMemory(b *board.Board, depth, alpha, beta int, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
	if depth == 0 {
		return &ScoredMove{
			Score: p.EvaluateBoard(b).TotalScore,
		}
	}

	var h [33]byte
	if p.TranspositionTableEnabled {
		// transposition table lookup
		h = b.Hash()
		if entry, ok := p.alphaBetaTable.Read(&h); ok {
			e := *entry
			if e.Lower >= beta {
				p.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Move:  e.BestMove,
					Score: e.Lower,
				}
			} else if e.Upper <= alpha {
				p.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Move:  e.BestMove,
					Score: e.Upper,
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
	var maximizingPlayer = currentPlayer == p.PlayerColor
	var best ScoredMove
	if maximizingPlayer {
		best.Score = NegInf
	} else {
		best.Score = PosInf
	}
	moves := b.GetAllMoves(currentPlayer, previousMove)
	for i, m := range *moves {
		newBoard := b.Copy()
		previousMove = board.MakeMove(&m, newBoard)
		p.Metrics.MovesConsidered++
		candidate := p.AlphaBetaWithMemory(newBoard, depth-1, alpha, beta, currentPlayer^1, previousMove)
		candidate.Move = m
		candidate.MoveSequence = append(candidate.MoveSequence, m)
		if betterMove(maximizingPlayer, &best, candidate) {
			best = *candidate
		}
		if maximizingPlayer {
			if best.Score > alpha {
				// TODO(Vadim) why does adding this make ab prune too much
				//alpha = best.Score
			}
		} else {
			if best.Score < beta {
				beta = best.Score
			}
		}
		if alpha >= beta {
			// alpha-beta cutoff
			p.Metrics.MovesPrunedAB += int64(len(*moves) - i)
			break
		}
	}

	if p.TranspositionTableEnabled {
		if best.Score <= alpha {
			p.alphaBetaTable.Store(&h, &util.TranspositionTableEntry{
				Lower:    NegInf,
				Upper:    best.Score,
				BestMove: best.Move,
			})
		}
		if best.Score > alpha && best.Score < beta {
			p.alphaBetaTable.Store(&h, &util.TranspositionTableEntry{
				Lower:    best.Score,
				Upper:    best.Score,
				BestMove: best.Move,
			})
		}
		if best.Score >= beta {
			p.alphaBetaTable.Store(&h, &util.TranspositionTableEntry{
				Lower:    best.Score,
				Upper:    PosInf,
				BestMove: best.Move,
			})
		}
	}

	return &best
}
