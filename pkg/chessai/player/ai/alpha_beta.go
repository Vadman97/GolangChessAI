package ai

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
	"time"
)

func (ab *AlphaBetaWithMemory) AlphaBetaWithMemory(root *board.Board, depth, alpha, beta int, currentPlayer color.Color, previousMove *board.LastMove) *ScoredMove {
	var h util.BoardHash
	if ab.player.TranspositionTableEnabled {
		// transposition table lookup
		h = root.Hash()
		if entry, ok := ab.player.transpositionTable.Read(&h, currentPlayer); ok {
			abEntry := entry.(*transposition_table.TranspositionTableEntryABMemory)
			lower := DenormalizeMateScore(abEntry.Lower, depth)
			upper := DenormalizeMateScore(abEntry.Upper, depth)
			validMove := !abEntry.BestMove.Start.Equals(abEntry.BestMove.End)
			// Only use hard cutoffs from entries at least as deep as the current search.
			if abEntry.Depth >= depth {
				if lower >= beta && validMove {
					ab.player.Metrics.MovesPrunedTransposition++
					return &ScoredMove{
						Score: lower,
						Move:  abEntry.BestMove,
					}
				} else if upper <= alpha && validMove {
					ab.player.Metrics.MovesPrunedTransposition++
					return &ScoredMove{
						Score: upper,
						Move:  abEntry.BestMove,
					}
				}
			}
			// Use shallower entries to narrow the window (safe: just a hint, not a cutoff).
			if lower > NegInf && lower > alpha {
				ab.player.Metrics.MovesABImprovedTransposition++
				alpha = lower
			}
			if upper < PosInf && upper < beta {
				ab.player.Metrics.MovesABImprovedTransposition++
				beta = upper
			}
		}
	}
	var best ScoredMove
	moves := root.GetAllMoves(currentPlayer, previousMove)
	// max recursion or terminal node
	if depth == 0 || ab.player.terminalNode(root, moves) {
		best = ScoredMove{
			Score: AdjustMateScore(ab.player.EvaluateBoard(root, ab.player.PlayerColor).TotalScore, depth),
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
			if ab.player.abort {
				break
			}
		}
	}

	if !ab.player.abort && ab.player.TranspositionTableEnabled && !best.Move.Start.Equals(best.Move.End) {
		normScore := NormalizeMateScore(best.Score, depth)
		if best.Score >= beta {
			ab.player.transpositionTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    normScore,
				Upper:    PosInf,
				BestMove: best.Move,
				Depth:    depth,
			})
		} else if best.Score > alpha && best.Score < beta {
			ab.player.transpositionTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    normScore,
				Upper:    normScore,
				BestMove: best.Move,
				Depth:    depth,
			})
		} else if best.Score <= alpha {
			ab.player.transpositionTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABMemory{
				Lower:    NegInf,
				Upper:    normScore,
				BestMove: best.Move,
				Depth:    depth,
			})
		}
	}

	return &best
}

type AlphaBetaWithMemory struct {
	player             *AIPlayer
	currentSearchDepth int
}

func (ab *AlphaBetaWithMemory) GetName() string {
	return AlgorithmAlphaBetaWithMemory
}

func (ab *AlphaBetaWithMemory) iterativeAlphaBeta(b *board.Board, previousMove *board.LastMove) *ScoredMove {
	start := time.Now()
	best := &ScoredMove{}
	iterativeIncrement := config.Get().IterativeIncrement
	for ab.currentSearchDepth = iterativeIncrement; ab.currentSearchDepth <= ab.player.MaxSearchDepth; ab.currentSearchDepth += iterativeIncrement {
		thinking, done := make(chan bool), make(chan bool, 1)
		go ab.player.trackThinkTime(thinking, done, start)
		newBest := ab.AlphaBetaWithMemory(b, ab.currentSearchDepth, NegInf, PosInf, ab.player.PlayerColor, previousMove)
		close(thinking)
		<-done
		if !ab.player.abort {
			best = newBest
			ab.player.LastSearchDepth = ab.currentSearchDepth
			ab.player.printer <- fmt.Sprintf("Best D:%d M:%s\n", ab.player.LastSearchDepth, best.Move)
		} else {
			ab.player.LastSearchDepth = ab.currentSearchDepth - iterativeIncrement
			ab.player.printer <- fmt.Sprintf("%s hard abort! evaluated to depth %d\n", ab.GetName(), ab.player.LastSearchDepth)
			break
		}
	}
	return best
}

func (ab *AlphaBetaWithMemory) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	ab.player = p
	ab.player.abort = false
	if p.MaxThinkTime != 0 {
		return ab.iterativeAlphaBeta(b, previousMove)
	}
	ab.player.LastSearchDepth = p.MaxSearchDepth
	return ab.AlphaBetaWithMemory(b, p.MaxSearchDepth, NegInf, PosInf, p.PlayerColor, previousMove)
}
