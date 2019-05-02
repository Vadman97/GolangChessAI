package ai

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
	"log"
	"runtime"
	"sync/atomic"
	"time"
)

func (ab *ABDADA) ABDADA(root *board.Board, depth int, alpha, beta Value, exclusiveProbe bool, currentPlayer color.Color, previousMove *board.LastMove) ScoredMove {
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

		// block and grab the answer
		ttAnswer := <-answerChan
		alpha, beta = ttAnswer.alpha, ttAnswer.beta
		best.Score, best.Move = ttAnswer.score, ttAnswer.bestMove

		// this is a terminal node because we have no moves, either we lost or tied
		if ab.player.terminalNode(root, movesArr) {
			return ScoredMove{
				Score: ab.player.EvaluateBoard(root, currentPlayer).TotalScore,
			}
		}

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
				if ab.player.abort || ab.kill {
					return best
				}
				iteration++
				allDone = true
				firstMove := true
				moves := *movesArr
				move := moves[0]
				moves = moves[1:]
				for alpha < beta {
					if ab.player.abort || ab.kill {
						return best
					}
					// On the first iteration, we want to be the only processor to evaluate young sons
					exclusiveProbe = iteration == 1 && !firstMove

					child, previousMove := ab.player.applyMove(root, &move)
					value := ab.ABDADA(child, depth-1, -beta, -MaxScore(alpha, best.Score), exclusiveProbe, currentPlayer^1, previousMove)
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

func (ab *ABDADA) getBestMove(b *board.Board, depth int, alpha, beta Value, previousMove *board.LastMove) ScoredMove {
	ab.player.abort = false
	if ab.NumThreads == 0 {
		ab.NumThreads = runtime.NumCPU()
		log.Printf("ABDADA runs in parallel, defaulting to #%d threads (# cpu cores)\n", ab.NumThreads)
	}
	moveChan := make(chan ScoredMove, ab.NumThreads)
	for i := 0; i < ab.NumThreads; i++ {
		go func(moveChan chan ScoredMove) {
			moveChan <- ab.ABDADA(b, depth, alpha, beta, false, ab.player.PlayerColor, previousMove)
		}(moveChan)
	}

	var bestMoves []ScoredMove

	const AbortAfterFirst = true
	if AbortAfterFirst {
		// abort the other threads after the first move is retrieved
		// get first thread's move
		bestMoves = append(bestMoves, <-moveChan)
		// abort the rest
		ab.kill = true
		for i := 0; i < ab.NumThreads-1; i++ {
			<-moveChan
		}
		ab.kill = false
	} else {
		for i := 0; i < ab.NumThreads; i++ {
			bestMoves = append(bestMoves, <-moveChan)
		}
	}

	var bestMove ScoredMove
	bestMove.Score = NegInf
	for _, sm := range bestMoves {
		if sm.Score >= bestMove.Score {
			bestMove = sm
		}
		//ab.player.printer <- fmt.Sprintf("Thread #%d best move: %s %d\n", i, sm.Move, sm.Score)
	}
	return bestMove
}

func (ab *ABDADA) iterativeABDADA(b *board.Board, previousMove *board.LastMove) ScoredMove {
	start := time.Now()
	best := ScoredMove{
		Score: NegInf,
	}
	iterativeIncrement := config.Get().IterativeIncrement
	for ab.currentSearchDepth = iterativeIncrement; ab.currentSearchDepth <= ab.player.MaxSearchDepth; ab.currentSearchDepth += iterativeIncrement {
		thinking, done := make(chan bool), make(chan bool, 1)
		go ab.player.trackThinkTime(thinking, done, start)
		newGuess := ab.getBestMove(b, ab.currentSearchDepth, NegInf, PosInf, previousMove)
		close(thinking)
		<-done
		// MTDf returns a good move (did not abort search)
		if !ab.player.abort {
			best = newGuess
			ab.player.LastSearchDepth = ab.currentSearchDepth
			ab.player.printer <- fmt.Sprintf("Best D:%d M:%s\n", ab.player.LastSearchDepth, best.Move)
		} else {
			// -1 due to discard of current level due to hard abort
			ab.player.LastSearchDepth = ab.currentSearchDepth - iterativeIncrement
			ab.player.printer <- fmt.Sprintf("%s hard abort! evaluated to depth %d\n", ab.GetName(), ab.player.LastSearchDepth)
			break
		}
	}
	if best.Move.Start.Equals(best.Move.End) {
		log.Printf("%s has no best move: %s", ab.GetName(), best.Move)
	}
	return best
}

func (ab *ABDADA) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	ab.player = p
	if b.CacheGetAllMoves || b.CacheGetAllAttackableMoves {
		log.Printf("Trying to use %s with move caching enabled.\n", ab.GetName())
		log.Println("Disabling GetAllMoves, GetAllAttackableMoves caching.")
		log.Printf("%s performs better without caching since it generates moves asynchronously\n", ab.GetName())
		b.CacheGetAllMoves = false
		b.CacheGetAllAttackableMoves = false
	}
	best := ab.iterativeABDADA(b, previousMove)
	return &best
}

func (p *AIPlayer) applyMove(root *board.Board, move *location.Move) (child *board.Board, previousMove *board.LastMove) {
	child = root.Copy()
	previousMove = board.MakeMove(move, child)
	p.Metrics.MovesConsidered++
	return
}

type ABDADA struct {
	player             *AIPlayer
	kill               bool
	currentSearchDepth int
	NumThreads         int
}

func (ab *ABDADA) GetName() string {
	return AlgorithmABDADA
}

type TTAnswer struct {
	alpha, beta, score Value
	bestMove           location.Move
}

func (ab *ABDADA) syncTTWrite(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta Value, sm *ScoredMove) {
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
				}
				entry.Score = int(sm.Score)
				entry.BestMove = sm.Move
				entry.Depth = depth

				entry.Lock.Unlock()
			}
		}
	}
}

func (ab *ABDADA) asyncTTRead(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta Value, exclusiveProbe bool) chan TTAnswer {
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
						answer.score = Value(entry.Score)
						answer.alpha = Value(entry.Score)
						answer.beta = Value(entry.Score)
					} else if entry.EntryType == transposition_table.UpperBound && Value(entry.Score) < beta {
						answer.score = Value(entry.Score)
						answer.beta = Value(entry.Score)
					} else if entry.EntryType == transposition_table.LowerBound && Value(entry.Score) > alpha {
						answer.score = Value(entry.Score)
						answer.alpha = Value(entry.Score)
					}
					answer.bestMove = entry.BestMove

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
