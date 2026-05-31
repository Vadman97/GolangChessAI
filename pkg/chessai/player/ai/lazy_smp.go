package ai

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// threadRootMove records a thread's best move found at a given search depth.
type threadRootMove struct {
	move  location.Move
	score int
	depth int
}

type LazySMP struct {
	player             *AIPlayer
	currentSearchDepth int
	numThreads         int

	// helperAbort is set atomically to 1 to signal helper threads to stop.
	helperAbort int32

	rootMoves   []threadRootMove
	rootMovesMu []sync.Mutex
}

func (smp *LazySMP) GetName() string { return AlgorithmLazySMP }

func (smp *LazySMP) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	smp.player = p
	if b.CacheGetAllMoves || b.CacheGetAllAttackableMoves {
		log.Printf("Disabling move caching for %s\n", smp.GetName())
		b.CacheGetAllMoves = false
		b.CacheGetAllAttackableMoves = false
	}
	best := smp.iterativeLazySMP(b, previousMove)
	return &best
}

func (smp *LazySMP) iterativeLazySMP(b *board.Board, previousMove *board.LastMove) ScoredMove {
	start := time.Now()
	if smp.numThreads == 0 {
		smp.numThreads = runtime.NumCPU()
		log.Printf("LazySMP defaulting to %d threads\n", smp.numThreads)
	}

	smp.rootMoves = make([]threadRootMove, smp.numThreads)
	smp.rootMovesMu = make([]sync.Mutex, smp.numThreads)

	best := ScoredMove{Score: NegInf}
	iterativeIncrement := config.Get().IterativeIncrement

	// Depth 1: main thread only, full window — provides initial score for aspiration.
	smp.currentSearchDepth = iterativeIncrement
	{
		thinking, done := make(chan bool), make(chan bool, 1)
		go smp.player.trackThinkTime(thinking, done, start)
		result := smp.search(b, smp.currentSearchDepth, NegInf, PosInf, smp.player.PlayerColor, previousMove, 0)
		close(thinking)
		<-done
		if !smp.player.isAborted() {
			best = result
			smp.player.LastSearchDepth = smp.currentSearchDepth
			smp.rootMoves[0] = threadRootMove{move: best.Move, score: best.Score, depth: smp.currentSearchDepth}
		}
	}

	for smp.currentSearchDepth = iterativeIncrement * 2; smp.currentSearchDepth <= smp.player.MaxSearchDepth; smp.currentSearchDepth += iterativeIncrement {
		if smp.player.isAborted() {
			break
		}

		// Launch helper threads. Per Cheng's spec, odd 0-based helpers search at depth+1.
		atomic.StoreInt32(&smp.helperAbort, 0)
		var helperWg sync.WaitGroup
		for i := 1; i < smp.numThreads; i++ {
			helperWg.Add(1)
			helperIdx := i
			helperDepth := smp.currentSearchDepth
			if helperIdx%2 == 1 {
				helperDepth++
			}
			rootCopy := b.Copy()
			go func() {
				defer helperWg.Done()
				smp.helperSearch(rootCopy, helperDepth, previousMove, helperIdx)
			}()
		}

		// Main thread: aspiration window search.
		alpha := best.Score - aspirationDelta
		beta := best.Score + aspirationDelta
		delta := aspirationDelta
		var newBest ScoredMove
		aspirationFailed := false
		for {
			if smp.player.isAborted() {
				aspirationFailed = true
				break
			}
			thinking, done := make(chan bool), make(chan bool, 1)
			go smp.player.trackThinkTime(thinking, done, start)
			result := smp.search(b, smp.currentSearchDepth, alpha, beta, smp.player.PlayerColor, previousMove, 0)
			close(thinking)
			<-done

			if smp.player.isAborted() {
				aspirationFailed = true
				break
			}

			if result.Score <= alpha {
				beta = (alpha + beta) / 2
				alpha -= delta
				if alpha < NegInf/2 {
					alpha = NegInf
				}
				delta *= 2
			} else if result.Score >= beta {
				alpha = (alpha + beta) / 2
				beta += delta
				if beta > PosInf/2 {
					beta = PosInf
				}
				delta *= 2
			} else {
				newBest = result
				break
			}

			if delta > aspirationMaxDelta {
				alpha = NegInf
				beta = PosInf
			}
		}

		// Stop helpers and wait.
		atomic.StoreInt32(&smp.helperAbort, 1)
		helperWg.Wait()

		if !aspirationFailed && !smp.player.isAborted() {
			smp.rootMovesMu[0].Lock()
			smp.rootMoves[0] = threadRootMove{move: newBest.Move, score: newBest.Score, depth: smp.currentSearchDepth}
			smp.rootMovesMu[0].Unlock()

			best = smp.threadVote()
			smp.player.LastSearchDepth = smp.currentSearchDepth
			smp.player.printer <- fmt.Sprintf("Best D:%d M:%s Score:%d\n", smp.player.LastSearchDepth, best.Move, best.Score)
		} else {
			smp.player.LastSearchDepth = smp.currentSearchDepth - iterativeIncrement
			smp.player.printer <- fmt.Sprintf("%s hard abort! evaluated to depth %d\n", smp.GetName(), smp.player.LastSearchDepth)
			break
		}
	}

	return best
}

// helperSearch runs iterative deepening from depth 1 up to targetDepth, sharing
// the transposition table with the main thread to warm it up ahead of the main search.
func (smp *LazySMP) helperSearch(b *board.Board, targetDepth int, previousMove *board.LastMove, threadIdx int) {
	iterativeIncrement := config.Get().IterativeIncrement
	var lastBest threadRootMove
	for d := iterativeIncrement; d <= targetDepth; d += iterativeIncrement {
		if atomic.LoadInt32(&smp.helperAbort) != 0 || smp.player.isAborted() {
			break
		}
		result := smp.search(b, d, NegInf, PosInf, smp.player.PlayerColor, previousMove, threadIdx)
		if atomic.LoadInt32(&smp.helperAbort) != 0 || smp.player.isAborted() {
			break
		}
		lastBest = threadRootMove{move: result.Move, score: result.Score, depth: d}
	}
	if lastBest.depth > 0 {
		smp.rootMovesMu[threadIdx].Lock()
		smp.rootMoves[threadIdx] = lastBest
		smp.rootMovesMu[threadIdx].Unlock()
	}
}

// threadVote selects the best move across all threads using Berserk-style weighted voting.
// Votes are weighted by (score - worstScore + 10) * depth, so deeper, higher-scoring
// moves receive more weight.
func (smp *LazySMP) threadVote() ScoredMove {
	worstScore := PosInf
	for i := range smp.rootMoves {
		if smp.rootMoves[i].depth > 0 && smp.rootMoves[i].score < worstScore {
			worstScore = smp.rootMoves[i].score
		}
	}

	type moveKey struct{ from, to location.Location }
	votes := make(map[moveKey]int)
	for i := range smp.rootMoves {
		rm := smp.rootMoves[i]
		if rm.depth == 0 {
			continue
		}
		key := moveKey{rm.move.Start, rm.move.End}
		weight := (rm.score - worstScore + 10) * max(1, rm.depth)
		votes[key] += weight
	}

	// Main thread result is the baseline; other threads can override via voting.
	smp.rootMovesMu[0].Lock()
	bestRM := smp.rootMoves[0]
	smp.rootMovesMu[0].Unlock()

	bestVotes := votes[moveKey{bestRM.move.Start, bestRM.move.End}]

	const tbWinBound = 900_000_000
	for i := 1; i < smp.numThreads; i++ {
		smp.rootMovesMu[i].Lock()
		rm := smp.rootMoves[i]
		smp.rootMovesMu[i].Unlock()
		if rm.depth == 0 {
			continue
		}
		v := votes[moveKey{rm.move.Start, rm.move.End}]

		if abs(bestRM.score) >= tbWinBound {
			// Fastest mate / longest avoidance always wins.
			if rm.score > bestRM.score {
				bestRM, bestVotes = rm, v
			}
		} else if rm.score > -tbWinBound && v > bestVotes {
			bestRM, bestVotes = rm, v
		}
	}

	return ScoredMove{Move: bestRM.move, Score: bestRM.score}
}

// search is a standard negamax alpha-beta search with TT used by all LazySMP threads.
// threadIdx==0 is the main thread; helpers pass their 1-based index so they respect helperAbort.
func (smp *LazySMP) search(root *board.Board, depth, alpha, beta int, currentPlayer color.Color, previousMove *board.LastMove, threadIdx int) ScoredMove {
	if depth == 0 {
		return ScoredMove{
			Score: smp.player.Quiesce(root, alpha, beta, currentPlayer, previousMove),
		}
	}

	var ttBestMove location.Move

	// TT lookup — read score bounds and best move hint.
	if smp.player.TranspositionTableEnabled {
		h := root.Hash()
		if e, ok := smp.player.transpositionTable.Read(&h, currentPlayer); ok {
			entry := e.(*transposition_table.TranspositionTableEntryABDADA)
			entry.Lock.Lock()
			ttScore := DenormalizeMateScore(entry.Score, depth)
			ttMove := entry.BestMove
			deepEnough := entry.Depth >= uint16(depth)
			entryType := entry.EntryType
			entry.Lock.Unlock()

			ttBestMove = ttMove
			if deepEnough {
				switch entryType {
				case transposition_table.TrueScore:
					return ScoredMove{Score: ttScore, Move: ttMove}
				case transposition_table.LowerBound:
					if ttScore > alpha {
						alpha = ttScore
					}
				case transposition_table.UpperBound:
					if ttScore < beta {
						beta = ttScore
					}
				}
				if alpha >= beta {
					atomic.AddUint64(&smp.player.Metrics.MovesPrunedTransposition, 1)
					return ScoredMove{Score: ttScore, Move: ttMove}
				}
			}
		}
	}

	movesArr := root.GetAllMoves(currentPlayer, previousMove)
	if smp.player.terminalNode(root, movesArr) {
		return ScoredMove{
			Score: AdjustMateScore(smp.player.EvaluateBoard(root, currentPlayer).TotalScore, depth),
		}
	}

	orderedMoves := orderMoves(*movesArr, ttBestMove, [2]location.Move{}, nil, root, previousMove)

	var best ScoredMove
	best.Score = NegInf

	for _, m := range orderedMoves {
		if smp.player.isAborted() || (threadIdx != 0 && atomic.LoadInt32(&smp.helperAbort) != 0) {
			return best
		}
		child, pm := smp.player.applyMove(root, &m)
		value := smp.search(child, depth-1, -beta, -util.MaxScore(alpha, best.Score), currentPlayer^1, pm, threadIdx)
		value.Score = -value.Score
		value.Move = m

		if value.Score > best.Score || best.Move.Start.Equals(best.Move.End) {
			best = value
			if best.Score >= beta {
				atomic.AddUint64(&smp.player.Metrics.MovesPrunedAB, uint64(len(orderedMoves)))
				break
			}
			if best.Score > alpha {
				alpha = best.Score
			}
		}
	}

	// TT write — only store if the search was not aborted mid-way.
	if !smp.player.isAborted() && smp.player.TranspositionTableEnabled && !best.Move.Start.Equals(best.Move.End) {
		h := root.Hash()
		entryType := transposition_table.TrueScore
		if best.Score >= beta {
			entryType = transposition_table.LowerBound
		} else if best.Score <= alpha {
			entryType = transposition_table.UpperBound
		}
		normalizedScore := NormalizeMateScore(best.Score, depth)

		if e, ok := smp.player.transpositionTable.Read(&h, currentPlayer); ok {
			entry := e.(*transposition_table.TranspositionTableEntryABDADA)
			entry.Lock.Lock()
			if entry.Depth <= uint16(depth) {
				entry.Score = normalizedScore
				entry.EntryType = entryType
				entry.BestMove = best.Move
				entry.Depth = uint16(depth)
				entry.NumProcessors = 0
			}
			entry.Lock.Unlock()
		} else {
			smp.player.transpositionTable.Store(&h, currentPlayer, &transposition_table.TranspositionTableEntryABDADA{
				Score:     normalizedScore,
				EntryType: entryType,
				BestMove:  best.Move,
				Depth:     uint16(depth),
			})
		}
	}

	return best
}
