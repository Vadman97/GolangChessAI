package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/transposition_table"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"runtime"
	"sync/atomic"
	"time"
)

func (ab *ABDADA) ABDADA(root *board.Board, depth, alpha, beta int, exclusiveProbe bool, currentPlayer color.Color, previousMove *board.LastMove) ScoredMove {
	if depth == 0 {
		return ScoredMove{
			Score: ab.player.EvaluateBoard(root, currentPlayer).TotalScore,
		}
	} else {
		var best ScoredMove
		best.Score = NegInf

		answerChan := ab.asyncTTRead(root, currentPlayer, uint16(depth), alpha, beta, exclusiveProbe)
		// generate moves while waiting for the answer ...
		movesArr := root.GetAllMoves(currentPlayer, previousMove)

		if len(*movesArr) == 0 {
			return best
		}

		// block and grab the answer
		ttAnswer := <-answerChan
		alpha, beta = ttAnswer.alpha, ttAnswer.beta
		best.Score, best.Move = ttAnswer.score, ttAnswer.bestMove

		/* The current move is not evaluated if causing u a cutoff or
		if we are in exclusive mode and another processor
		is currently evaluating it. */
		if alpha >= beta || best.Score == OnEvaluation {
			atomic.AddUint64(&ab.player.Metrics.MovesPrunedTransposition, uint64(len(*movesArr)))
			return best
		} else {
			iteration := 0
			allDone := false
			for iteration < 2 && alpha < beta && !allDone {
				iteration++
				allDone = true
				firstMove := true
				moves := *movesArr
				move := moves[0]
				moves = moves[1:]
				for alpha < beta {
					// On the first iteration, we want to be the only processor to evaluate young sons
					exclusiveProbe = iteration == 1 && !firstMove

					child, previousMove := ab.player.applyMove(root, &move)
					value := ab.ABDADA(child, depth-1, -beta, -util.MaxScore(alpha, best.Score), exclusiveProbe, currentPlayer^1, previousMove)
					value.Score = -value.Score
					value.Move = move

					if value.Score == -OnEvaluation {
						allDone = false
					} else if value.Score > best.Score {
						best = value
						if best.Score >= beta {
							atomic.AddUint64(&ab.player.Metrics.MovesPrunedAB, uint64(len(moves)))
							ab.syncTTWrite(root, currentPlayer, uint16(depth), alpha, beta, &best)
							return best
						}
					}
					if len(moves) == 0 {
						break
					}
					firstMove = false
					move = moves[0]
					moves = moves[1:]
				}
			}
			ab.syncTTWrite(root, currentPlayer, uint16(depth), alpha, beta, &best)
			return best
		}
	}
}

func (p *AIPlayer) applyMove(root *board.Board, move *location.Move) (child *board.Board, previousMove *board.LastMove) {
	child = root.Copy()
	previousMove = board.MakeMove(move, child)
	p.Metrics.MovesConsidered++
	return
}

type ABDADA struct {
	player             *AIPlayer
	currentSearchDepth int
	lastSearchDepth    int
	lastSearchTime     time.Duration
}

func (ab *ABDADA) GetName() string {
	return fmt.Sprintf("%s,[D:%d;T:%s]", AlgorithmABDADA, ab.lastSearchDepth, ab.lastSearchTime)
}

func (ab *ABDADA) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	ab.player = p

	// TODO(Vadim) make iterative
	var NumThreads = runtime.NumCPU()
	moveChan := make(chan ScoredMove, NumThreads)
	for i := 0; i < NumThreads; i++ {
		go func(moveChan chan ScoredMove) {
			moveChan <- ab.ABDADA(b, p.MaxSearchDepth, NegInf, PosInf, false, p.PlayerColor, previousMove)
		}(moveChan)
	}

	var bestMoves []ScoredMove
	for i := 0; i < NumThreads; i++ {
		bestMoves = append(bestMoves, <-moveChan)
	}

	var bestMove ScoredMove
	bestMove.Score = NegInf
	for i, sm := range bestMoves {
		if sm.Score >= bestMove.Score {
			bestMove = sm
		}
		ab.player.printer <- fmt.Sprintf("Thread #%d best move: %s %d\n", i, sm.Move, sm.Score)
	}

	return &bestMove
}

type TTAnswer struct {
	alpha, beta, score int
	bestMove           location.Move
}

func (ab *ABDADA) syncTTWrite(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta int, sm *ScoredMove) {
	if ab.player.TranspositionTableEnabled {
		// transposition table lookup
		h := root.Hash()
		if e, ok := ab.player.transpositionTable.Read(&h, currentPlayer); ok {
			entry := e.(*transposition_table.TranspositionTableEntryABDADA)
			if entry.Depth <= depth {
				entry.Lock.Lock()

				if entry.Depth == depth {
					entry.NumProcessors--
				} else {
					entry.NumProcessors = 0
				}

				if sm.Score >= beta {
					entry.EntryType = transposition_table.LowerBound
				} else if sm.Score <= alpha {
					entry.EntryType = transposition_table.UpperBound
				} else {
					entry.EntryType = transposition_table.TrueScore
					entry.BestMove = sm.Move // TODO(Vadim) only if TrueScore?
				}
				entry.Score = sm.Score
				entry.Depth = depth

				entry.Lock.Unlock()
			}
		}
	}
}

func (ab *ABDADA) asyncTTRead(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta int, exclusiveProbe bool) chan TTAnswer {
	answerChan := make(chan TTAnswer)
	go func(answerChan chan TTAnswer) {
		answer := TTAnswer{
			alpha: alpha,
			beta:  beta,
			score: NegInf,
		}
		if ab.player.TranspositionTableEnabled {
			// transposition table lookup
			h := root.Hash()
			if e, ok := ab.player.transpositionTable.Read(&h, currentPlayer); ok {
				entry := e.(*transposition_table.TranspositionTableEntryABDADA)
				entry.Lock.Lock()

				if entry.Depth == depth && exclusiveProbe && entry.NumProcessors > 0 {
					// Only one processor allowed if exclusivity is required
					answer.score = OnEvaluation
				} else if entry.Depth >= depth {
					if entry.EntryType == transposition_table.TrueScore {
						answer.score = entry.Score
						answer.alpha = entry.Score
						answer.beta = entry.Score
						answer.bestMove = entry.BestMove // TODO(Vadim) only if TrueScore?
					} else if entry.EntryType == transposition_table.UpperBound && entry.Score < beta {
						answer.score = entry.Score
						answer.beta = entry.Score
					} else if entry.EntryType == transposition_table.LowerBound && entry.Score > alpha {
						answer.score = entry.Score
						answer.alpha = entry.Score
					}

					if entry.Depth == depth && answer.alpha < answer.beta {
						// Increment the number of processors evaluating this node
						entry.NumProcessors++
					}
				} else {
					// This is the first processor to evaluate this node
					// new pass - we've seen node on previous evaluation
					entry.Depth = depth
					entry.EntryType = transposition_table.Unset
					entry.NumProcessors = 1
				}

				entry.Lock.Unlock()
			} else {
				// This is the first processor to ever evaluate this node
				entry := transposition_table.TranspositionTableEntryABDADA{
					Depth:         depth,
					EntryType:     transposition_table.Unset,
					NumProcessors: 1,
				}
				ab.player.transpositionTable.Store(&h, currentPlayer, &entry)
			}
		}
		answerChan <- answer
	}(answerChan)
	return answerChan
}
