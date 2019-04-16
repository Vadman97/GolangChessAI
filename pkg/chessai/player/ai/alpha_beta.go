package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
)

func (ab *AlphaBetaWithMemory) AlphaBetaWithMemory(root *board.Board, depth, alpha, beta int, currentPlayer color.Color, previousMove *board.LastMove) *ScoredMove {
	var h util.BoardHash
	if ab.player.TranspositionTableEnabled {
		// transposition table lookup
		h = root.Hash()
		if entry, ok := ab.player.alphaBetaTable.Read(&h, currentPlayer); ok {
			if entry.Lower >= beta {
				ab.player.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Score: entry.Lower,
					Move:  entry.BestMove,
				}
			} else if entry.Upper <= alpha {
				ab.player.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Score: entry.Upper,
					Move:  entry.BestMove,
				}
			}
			if entry.Lower > NegInf && entry.Lower > alpha {
				ab.player.Metrics.MovesABImprovedTransposition++
				alpha = entry.Lower
				// TODO(Vadim) first in for loop of moves try entry.BestMove, same in other else
			}
			if entry.Upper < PosInf && entry.Upper < beta {
				ab.player.Metrics.MovesABImprovedTransposition++
				beta = entry.Upper
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
			if ab.player.abort {
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

	if !ab.player.abort && ab.player.TranspositionTableEnabled {
		if best.Score >= beta {
			ab.player.alphaBetaTable.Store(&h, currentPlayer, &util.TranspositionTableEntry{
				Lower:    best.Score,
				Upper:    PosInf,
				BestMove: best.Move,
			})
		} else if best.Score > alpha && best.Score < beta {
			ab.player.alphaBetaTable.Store(&h, currentPlayer, &util.TranspositionTableEntry{
				Lower:    best.Score,
				Upper:    best.Score,
				BestMove: best.Move,
			})
		} else if best.Score <= alpha {
			ab.player.alphaBetaTable.Store(&h, currentPlayer, &util.TranspositionTableEntry{
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
	lastSearchDepth int
}

func (ab *AlphaBetaWithMemory) GetName() string {
	return fmt.Sprintf("%s,[depth:%d]", AlgorithmAlphaBetaWithMemory, ab.lastSearchDepth)
}

func (ab *AlphaBetaWithMemory) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	ab.player = p
	ab.player.abort = false
	ab.lastSearchDepth = p.MaxSearchDepth
	return ab.AlphaBetaWithMemory(b, p.MaxSearchDepth, NegInf, PosInf, p.PlayerColor, previousMove)
}
