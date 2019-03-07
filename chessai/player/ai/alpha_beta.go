package ai

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/util"
)

func (p *Player) AlphaBetaRecurse(b *board.Board, m *board.Move, depth, alpha, beta int, currentPlayer byte) *ScoredMove {
	newBoard := b.Copy()
	board.MakeMove(m, newBoard)
	candidate := p.AlphaBetaWithMemory(newBoard, depth-1, alpha, beta, (currentPlayer+1)%color.NumColors)
	candidate.Move = m
	return candidate
}

func (p *Player) AlphaBetaWithMemory(b *board.Board, depth, alpha, beta int, currentPlayer byte) *ScoredMove {
	// transposition table lookup
	h := b.Hash()
	if entry, ok := p.alphaBetaTable.Read(&h); ok {
		e := *entry
		if e.Lower >= beta {
			return &ScoredMove{
				Move:  e.BestMove,
				Score: e.Lower,
			}
		} else if e.Upper <= alpha {
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

	if depth == 0 {
		return &ScoredMove{
			Move:  nil,
			Score: p.EvaluateBoard(b).TotalScore,
		}
	}

	var best ScoredMove
	if currentPlayer == p.PlayerColor {
		// maximizing player
		best.Score = NegInf
	} else {
		// minimizing player
		best.Score = PosInf
	}
	moves := b.GetAllMoves(p.PlayerColor)
	for _, m := range *moves {
		candidate := p.AlphaBetaRecurse(b, &m, depth, alpha, beta, currentPlayer)
		best = *compare(currentPlayer == p.PlayerColor, &best, candidate)
		alpha, beta = compareAlphaBeta(currentPlayer == p.PlayerColor, alpha, beta, candidate)
		if alpha >= beta {
			// alpha-beta cutoff
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

func compareAlphaBeta(maximizingP bool, currentAlpha, currentBeta int, candidate *ScoredMove) (int, int) {
	if maximizingP {
		if candidate.Score > currentAlpha {
			return candidate.Score, currentBeta
		} else {
			return currentAlpha, currentBeta
		}
	} else {
		if candidate.Score < currentBeta {
			return currentAlpha, candidate.Score
		} else {
			return currentAlpha, currentBeta
		}
	}
}
