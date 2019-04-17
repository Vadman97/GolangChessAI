package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/transposition_table"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
)

func (ab *AlphaBetaWithMemory) Quiesce(root *board.Board, alpha, beta int, currentPlayer byte, previousMove *board.LastMove) int {
	standPat := ab.player.EvaluateBoard(root, currentPlayer).TotalScore
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
			score := -ab.Quiesce(child, -beta, -alpha, currentPlayer^1, previousMove)

			if score >= beta {
				return beta
			} else if score > alpha {
				alpha = score
			}
		}
	}
	return alpha
}

func (ab *AlphaBetaWithMemory) AlphaBetaWithMemory(root *board.Board, depth, alpha, beta int, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
	var h util.BoardHash
	if ab.player.TranspositionTableEnabled {
		// transposition table lookup
		h = root.Hash()
		if entry, ok := ab.player.alphaBetaTable.Read(&h, currentPlayer); ok {
			abEntry := entry.(*transposition_table.TranspositionTableEntryABMemory)
			if abEntry.Lower > NegInf && abEntry.Lower >= beta {
				ab.player.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Score: abEntry.Lower,
					Move:  abEntry.BestMove,
				}
			} else if abEntry.Upper < PosInf && abEntry.Upper <= alpha {
				ab.player.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Score: abEntry.Upper,
					Move:  abEntry.BestMove,
				}
			}
			if abEntry.Lower > NegInf && abEntry.Lower > alpha {
				ab.player.Metrics.MovesABImprovedTransposition++
				alpha = abEntry.Lower
				// TODO(Vadim) first in for loop of moves try abEntry.BestMove, same in other else
			}
			if abEntry.Upper < PosInf && abEntry.Upper < beta {
				ab.player.Metrics.MovesABImprovedTransposition++
				beta = abEntry.Upper
			}
		}
	}
	var best ScoredMove
	if depth == 0 {
		best = ScoredMove{
			Score: ab.player.EvaluateBoard(root, ab.player.PlayerColor).TotalScore,
		}
	} else {
		var maximizingPlayer = currentPlayer == ab.player.PlayerColor
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
					ab.player.Metrics.MovesPrunedAB += int64(len(*moves) - i)
					break
				}
			} else {
				if best.Score <= alpha {
					ab.player.Metrics.MovesPrunedAB += int64(len(*moves) - i)
					break
				}
			}
			newBoard := root.Copy()
			previousMove = board.MakeMove(&m, newBoard)
			ab.player.Metrics.MovesConsidered++
			var candidate *ScoredMove
			if ab.abort {
				break
			}
			if maximizingPlayer {
				candidate = ab.AlphaBetaWithMemory(newBoard, depth-1, a, beta, currentPlayer^1, previousMove)
			} else {
				candidate = ab.AlphaBetaWithMemory(newBoard, depth-1, alpha, b, currentPlayer^1, previousMove)
			}
			candidate.Move = m
			candidate.MoveSequence = append(candidate.MoveSequence, candidate.Move)
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

	if !ab.abort && ab.player.TranspositionTableEnabled {
		if best.Score >= beta {
			ab.player.alphaBetaTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    best.Score,
				Upper:    PosInf,
				BestMove: best.Move,
			})
		} else if best.Score > alpha && best.Score < beta {
			ab.player.alphaBetaTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    best.Score,
				Upper:    best.Score,
				BestMove: best.Move,
			})
		} else if best.Score <= alpha {
			ab.player.alphaBetaTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    NegInf,
				Upper:    best.Score,
				BestMove: best.Move,
			})
		}
	}

	return &best
}

type AlphaBetaWithMemory struct {
	player          *AIPlayer
	abort           bool
	lastSearchDepth int
}

func (ab *AlphaBetaWithMemory) GetName() string {
	return fmt.Sprintf("%s,[depth:%d]", AlgorithmAlphaBetaWithMemory, ab.lastSearchDepth)
}

func (ab *AlphaBetaWithMemory) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	ab.player = p
	ab.abort = false
	ab.lastSearchDepth = p.MaxSearchDepth
	return ab.AlphaBetaWithMemory(b, p.MaxSearchDepth, NegInf, PosInf, p.PlayerColor, previousMove)
}
