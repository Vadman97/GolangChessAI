package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
)

func (p *Player) AlphaBetaRecurse(b *board.Board, m location.Move, depth, alpha, beta int, currentPlayer byte) *ScoredMove {
	newBoard := b.Copy()
	board.MakeMove(&m, newBoard)
	p.Metrics.MovesConsidered++
	candidate := p.AlphaBetaWithMemory(newBoard, depth-1, alpha, beta, currentPlayer^1)
	candidate.Move = m
	candidate.MoveSequence = append(candidate.MoveSequence, m)
	return candidate
}

func (p *Player) AlphaBetaWithMemory(b *board.Board, depth, alpha, beta int, currentPlayer byte) *ScoredMove {
	if depth == 0 {
		return &ScoredMove{
			Score: p.EvaluateBoard(b).TotalScore,
		}
	}

	// transposition table lookup
	h := b.Hash()
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
			alpha = entry.Lower
		}
		if entry.Upper < beta {
			beta = entry.Upper
		}
	}
	var maximizingPlayer = currentPlayer == p.PlayerColor
	var best ScoredMove
	if maximizingPlayer {
		best.Score = NegInf
	} else {
		best.Score = PosInf
	}
	moves := b.GetAllMoves(currentPlayer)
	for i, m := range *moves {
		candidate := p.AlphaBetaRecurse(b, m, depth, alpha, beta, currentPlayer)
		if betterMove(maximizingPlayer, &best, candidate) {
			best = *candidate
		}
		if maximizingPlayer {
			if best.Score > alpha {
				alpha = best.Score
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

	return &best
}
