package ai

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
)

func (ab *AlphaBetaWithMemory) AlphaBetaWithMemory(root *board.Board, depth int, alpha, beta Value, currentPlayer color.Color, previousMove *board.LastMove) *ScoredMove {
	var h util.BoardHash
	if ab.player.TranspositionTableEnabled {
		// transposition table lookup
		h = root.Hash()
		if entry, ok := ab.player.transpositionTable.Read(&h, currentPlayer); ok {
			abEntry := entry.(*transposition_table.TranspositionTableEntryABMemory)
			if Value(abEntry.Lower) >= beta {
				ab.player.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Score: Value(abEntry.Lower),
					Move:  abEntry.BestMove,
				}
			} else if Value(abEntry.Upper) <= alpha {
				ab.player.Metrics.MovesPrunedTransposition++
				return &ScoredMove{
					Score: Value(abEntry.Upper),
					Move:  abEntry.BestMove,
				}
			}
			if Value(abEntry.Lower) > NegInf && Value(abEntry.Lower) > alpha {
				ab.player.Metrics.MovesABImprovedTransposition++
				alpha = Value(abEntry.Lower)
				// TODO(Vadim) first in for loop of moves try abEntry.BestMove, same in other else
			}
			if Value(abEntry.Upper) < PosInf && Value(abEntry.Upper) < beta {
				ab.player.Metrics.MovesABImprovedTransposition++
				beta = Value(abEntry.Upper)
			}
		}
	}
	var best ScoredMove
	moves := root.GetAllMoves(currentPlayer, previousMove)
	// max recursion or terminal node
	if depth == 0 || ab.player.terminalNode(root, moves) {
		best = ScoredMove{
			Score: ab.player.EvaluateBoard(root, ab.player.PlayerColor).TotalScore,
		}
	} else {
		var maximizingPlayer = currentPlayer == ab.player.PlayerColor
		var a, b Value
		if maximizingPlayer {
			best.Score = NegInf
			a = alpha
		} else {
			best.Score = PosInf
			b = beta
		}
		for i, m := range *moves {
			if maximizingPlayer {
				if best.Score >= beta {
					ab.player.Metrics.MovesPrunedAB += uint64(len(*moves) - i)
					break
				}
			} else {
				if best.Score <= alpha {
					ab.player.Metrics.MovesPrunedAB += uint64(len(*moves) - i)
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
				a = MaxScore(best.Score, a)
			} else {
				b = MinScore(best.Score, b)
			}
		}
	}

	if !ab.player.abort && ab.player.TranspositionTableEnabled {
		if best.Score >= beta {
			ab.player.transpositionTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    int(best.Score),
				Upper:    int(PosInf),
				BestMove: best.Move,
			})
		} else if best.Score > alpha && best.Score < beta {
			ab.player.transpositionTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    int(best.Score),
				Upper:    int(best.Score),
				BestMove: best.Move,
			})
		} else if best.Score <= alpha {
			ab.player.transpositionTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    int(NegInf),
				Upper:    int(best.Score),
				BestMove: best.Move,
			})
		}
	}

	return &best
}

type AlphaBetaWithMemory struct {
	player *AIPlayer
}

func (ab *AlphaBetaWithMemory) GetName() string {
	return AlgorithmAlphaBetaWithMemory
}

func (ab *AlphaBetaWithMemory) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	ab.player = p
	ab.player.abort = false
	ab.player.LastSearchDepth = p.MaxSearchDepth
	return ab.AlphaBetaWithMemory(b, p.MaxSearchDepth, NegInf, PosInf, p.PlayerColor, previousMove)
}
