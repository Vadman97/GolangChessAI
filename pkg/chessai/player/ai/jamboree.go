package ai

//int jamboree(CNode n, int α, int β) {
//   if (n is leaf) return static_eval(n);
//   c[] = the childen of n;
//   b = -jamboree(c[0], -β, -α);
//   if (b >= β) return b;
//   if (b >  α) α = b;
//   In Parallel: for (i=1; i < |c[]|; i++) {
//      s = -jamboree(c[i], -α - 1, -α);
//      if (s >  b) b = s;
//      if (s >= β) abort_and_return s;
//      if (s >  α) {
//          /* Wait for completion of all previous iterations of the parallel loop */
//          s = -jamboree(c[i], -β, -α);
//          if (s >= β) abort_and_return s;
//          if (s >  α) α = s;
//          if (s >  b) b = s;
//      }
//      /* Note the completion of the ith iteration of the parallel loop */
//   }
//   return b;
//}

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type Jamboree struct {
	player             *AIPlayer
	threadNumber       int
	activeThreads      int
	activeThreadLock   sync.Mutex
	currentSearchDepth int
}

type TTAnswerJamboree struct {
	Found     bool
	Score     int
	BestMove  location.Move
	Depth     uint16
	EntryType byte
}

func (j *Jamboree) Jamboree(root *board.Board, depth int, alpha int, beta int, currentPlayer color.Color,
	previousMove *board.LastMove, abortFlag *bool) ScoredMove {

	if j.player.abort || *abortFlag {
		return ScoredMove{Score: NegInf}
	}

	if depth == 0 {
		return ScoredMove{Score: AdjustMateScore(j.player.EvaluateBoard(root, currentPlayer).TotalScore, depth)}
	}

	answerChan := j.asyncTTRead(root, currentPlayer)
	moves := root.GetAllMoves(currentPlayer, previousMove)
	ttAnswer := <-answerChan

	if len(*moves) == 0 {
		return ScoredMove{Score: AdjustMateScore(j.player.EvaluateBoard(root, currentPlayer).TotalScore, depth)}
	}

	// Use TT to prune or narrow the search window.
	if ttAnswer.Found && ttAnswer.Depth >= uint16(depth) {
		ttScore := DenormalizeMateScore(ttAnswer.Score, depth)
		switch ttAnswer.EntryType {
		case transposition_table.TrueScore:
			atomic.AddUint64(&j.player.Metrics.MovesPrunedTransposition, uint64(len(*moves)))
			return ScoredMove{Score: ttScore, Move: ttAnswer.BestMove}
		case transposition_table.LowerBound:
			if ttScore >= beta {
				atomic.AddUint64(&j.player.Metrics.MovesPrunedTransposition, uint64(len(*moves)))
				return ScoredMove{Score: ttScore, Move: ttAnswer.BestMove}
			}
			if ttScore > alpha {
				alpha = ttScore
			}
		case transposition_table.UpperBound:
			if ttScore <= alpha {
				atomic.AddUint64(&j.player.Metrics.MovesPrunedTransposition, uint64(len(*moves)))
				return ScoredMove{Score: ttScore, Move: ttAnswer.BestMove}
			}
			if ttScore < beta {
				beta = ttScore
			}
		}
	}

	// Save the (possibly TT-narrowed) window for TT classification on the way back up.
	searchAlpha := alpha

	if j.player.abort || *abortFlag {
		return ScoredMove{Score: NegInf}
	}

	var firstMove location.Move
	if ttAnswer.Found && !ttAnswer.BestMove.Start.Equals(ttAnswer.BestMove.End) && isMoveInList(ttAnswer.BestMove, moves) {
		firstMove = ttAnswer.BestMove
	} else {
		firstMove = (*moves)[0]
	}

	child, prev := j.player.applyMove(root, &firstMove)
	b := j.Jamboree(child, depth-1, -beta, -alpha, currentPlayer^1, prev, abortFlag)
	b.Score *= -1
	b.Move = firstMove

	if b.Score >= beta {
		j.syncTTWrite(root, currentPlayer, b.Score, uint16(depth), b.Move, searchAlpha, beta)
		return b
	}
	if b.Score > alpha {
		alpha = b.Score
	}

	if j.player.abort || *abortFlag {
		return ScoredMove{Score: NegInf}
	}

	nextLevelAbortFlag := false

	var movesToResearchLock sync.Mutex
	var movesToResearch []location.Move

	var childWaitGroup sync.WaitGroup
	var bLock sync.Mutex
	var resultLock sync.Mutex
	result := ScoredMove{ReturnThisMove: false}

	for i := 0; i < len(*moves); i++ {
		if j.player.abort || *abortFlag {
			return ScoredMove{Score: NegInf}
		}

		if (*moves)[i].Equals(&firstMove) {
			continue
		}

		if !nextLevelAbortFlag {
			childWaitGroup.Add(1)
			go func(abortFlag *bool, move location.Move) {
				child, prev := j.player.applyMove(root, &move)
				nextScoredMove := j.Jamboree(child, depth-1, -alpha-1, -alpha, currentPlayer^1, prev, abortFlag)
				nextScoredMove.Score *= -1
				bLock.Lock()
				if nextScoredMove.Score > b.Score || b.Move.Start.Equals(b.Move.End) {
					b.Score = nextScoredMove.Score
					b.Move = move
				}
				bLock.Unlock()
				if nextScoredMove.Score >= beta {
					*abortFlag = true
					resultLock.Lock()
					result.ReturnThisMove = true
					result.Move = move
					result.Score = nextScoredMove.Score
					resultLock.Unlock()
				}
				if nextScoredMove.Score > alpha {
					movesToResearchLock.Lock()
					movesToResearch = append(movesToResearch, move)
					movesToResearchLock.Unlock()
				}
				childWaitGroup.Done()
			}(&nextLevelAbortFlag, (*moves)[i])
			j.threadNumber++
		} else {
			break
		}
	}
	childWaitGroup.Wait()

	if result.ReturnThisMove {
		j.syncTTWrite(root, currentPlayer, result.Score, uint16(depth), result.Move, searchAlpha, beta)
		return result
	}

	// Serially re-search moves whose null-window score beat alpha.
	for i := 0; i < len(movesToResearch); i++ {
		if j.player.abort || *abortFlag {
			return ScoredMove{Score: NegInf}
		}
		child, prev := j.player.applyMove(root, &movesToResearch[i])
		innerAbort := false
		nextScoredMove := j.Jamboree(child, depth-1, -beta, -alpha, currentPlayer^1, prev, &innerAbort)
		nextScoredMove.Score *= -1
		if nextScoredMove.Score >= beta {
			j.syncTTWrite(root, currentPlayer, nextScoredMove.Score, uint16(depth), movesToResearch[i], searchAlpha, beta)
			return ScoredMove{Score: nextScoredMove.Score, Move: movesToResearch[i]}
		}
		if nextScoredMove.Score > alpha {
			alpha = nextScoredMove.Score
		}
		if nextScoredMove.Score > b.Score {
			b.Score = nextScoredMove.Score
			b.Move = movesToResearch[i]
		}
	}

	j.syncTTWrite(root, currentPlayer, b.Score, uint16(depth), b.Move, searchAlpha, beta)
	return b
}

func (j *Jamboree) asyncTTRead(root *board.Board, currentPlayer color.Color) chan TTAnswerJamboree {
	answerChan := make(chan TTAnswerJamboree)
	go func(answerChan chan TTAnswerJamboree) {
		answer := TTAnswerJamboree{Found: false}
		if j.player.TranspositionTableEnabled {
			h := root.Hash()
			if e, ok := j.player.transpositionTable.Read(&h, currentPlayer); ok {
				entry := e.(*transposition_table.TranspositionTableEntryJamboree)
				entry.Lock.Lock()
				answer.Found = true
				answer.Score = entry.Score
				answer.BestMove = entry.BestMove
				answer.Depth = entry.Depth
				answer.EntryType = entry.EntryType
				entry.Lock.Unlock()
			}
		}
		answerChan <- answer
	}(answerChan)
	return answerChan
}

func (j *Jamboree) syncTTWrite(root *board.Board, currentPlayer color.Color, score int, depth uint16, move location.Move, alpha, beta int) {
	if !j.player.TranspositionTableEnabled {
		return
	}
	var entryType byte
	if score >= beta {
		entryType = transposition_table.LowerBound
	} else if score <= alpha {
		entryType = transposition_table.UpperBound
	} else {
		entryType = transposition_table.TrueScore
	}

	normalizedScore := NormalizeMateScore(score, int(depth))
	h := root.Hash()
	e, ok := j.player.transpositionTable.Read(&h, currentPlayer)
	if !ok {
		entry := transposition_table.TranspositionTableEntryJamboree{
			Score:     normalizedScore,
			BestMove:  move,
			Depth:     depth,
			EntryType: entryType,
		}
		j.player.transpositionTable.Store(&h, currentPlayer, &entry)
	} else {
		entry := e.(*transposition_table.TranspositionTableEntryJamboree)
		entry.Lock.Lock()
		if depth >= entry.Depth {
			entry.Score = normalizedScore
			entry.BestMove = move
			entry.Depth = depth
			entry.EntryType = entryType
		}
		entry.Lock.Unlock()
	}
}

func (j *Jamboree) iterativeJamboree(b *board.Board, previousMove *board.LastMove) ScoredMove {
	start := time.Now()
	best := ScoredMove{Score: NegInf}
	iterativeIncrement := config.Get().IterativeIncrement
	for j.currentSearchDepth = iterativeIncrement; j.currentSearchDepth <= j.player.MaxSearchDepth; j.currentSearchDepth += iterativeIncrement {
		thinking, done := make(chan bool), make(chan bool, 1)
		go j.player.trackThinkTime(thinking, done, start)
		rootAbortFlag := false
		j.threadNumber = 0
		newBest := j.Jamboree(b, j.currentSearchDepth, NegInf, PosInf, j.player.PlayerColor, previousMove, &rootAbortFlag)
		close(thinking)
		<-done
		if !j.player.abort {
			best = newBest
			j.player.LastSearchDepth = j.currentSearchDepth
			j.player.printer <- fmt.Sprintf("Best D:%d M:%s\n", j.player.LastSearchDepth, best.Move)
		} else {
			j.player.LastSearchDepth = j.currentSearchDepth - iterativeIncrement
			j.player.printer <- fmt.Sprintf("%s hard abort! evaluated to depth %d\n", j.GetName(), j.player.LastSearchDepth)
			break
		}
	}
	return best
}

func (j *Jamboree) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	j.player = p
	j.player.abort = false
	if b.CacheGetAllMoves || b.CacheGetAllAttackableMoves {
		log.Printf("WARNING: Trying to use %s with move caching enabled.\n", AlgorithmJamboree)
		log.Println("WARNING: Disabling GetAllMoves, GetAllAttackableMoves caching.")
		log.Printf("%s performs better without caching since it generates moves asynchronously\n", AlgorithmJamboree)
		b.CacheGetAllMoves = false
		b.CacheGetAllAttackableMoves = false
	}
	best := j.iterativeJamboree(b, previousMove)
	return &best
}

func (j *Jamboree) GetName() string {
	return fmt.Sprintf("%s", AlgorithmJamboree)
}
