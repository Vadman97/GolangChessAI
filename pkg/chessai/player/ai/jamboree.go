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
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/transposition_table"
	"log"
	"sync"
	"sync/atomic"
)

type Jamboree struct {
	player                 *AIPlayer
	threadNumber           int
	activeThreads          int
	currentSearchDepth     int
	currentSearchDepthLock sync.Mutex
	activeThreadLock       sync.Mutex
}

type TTAnswerJamboree struct {
	Found    bool
	Score    int
	BestMove location.Move
	Depth    uint16
}

func (j *Jamboree) Jamboree(root *board.Board, depth int, alpha int, beta int, currentPlayer color.Color,
	previousMove *board.LastMove, abortFlag *bool) ScoredMove {
	if depth == 0 {
		return ScoredMove{Score: j.player.EvaluateBoard(root, currentPlayer).TotalScore}
	} else {
		answerChan := j.asyncTTRead(root, currentPlayer)
		moves := root.GetAllMoves(currentPlayer, previousMove)
		ttAnswer := <-answerChan

		if j.player.terminalNode(root, moves) {
			return ScoredMove{Score: j.player.EvaluateBoard(root, currentPlayer).TotalScore}
		}

		// transposition table saved us work
		if ttAnswer.Found && (ttAnswer.Depth == uint16(depth)) {
			atomic.AddUint64(&j.player.Metrics.MovesPrunedTransposition, uint64(len(*moves)))
			return ScoredMove{Score: ttAnswer.Score, Move: ttAnswer.BestMove}
		}

		var firstMove location.Move
		if ttAnswer.Found {
			firstMove = ttAnswer.BestMove
		} else {
			firstMove = (*moves)[0]
		}

		child, prev := j.player.applyMove(root, &firstMove)

		score := j.Jamboree(child, depth-1, -beta, -alpha, currentPlayer^1, prev, abortFlag).NegScore().Score

		b := ScoredMove{Score: score, Move: firstMove}

		if b.Score >= beta {
			return b
		}
		if b.Score > alpha {
			alpha = b.Score
		}

		if *abortFlag || j.player.abort {
			return b
		}

		nextLevelAbortFlag := false

		var childWaitGroup sync.WaitGroup
		var largeWindowWaitGroup sync.WaitGroup
		var bLock sync.Mutex
		var resultLock sync.Mutex
		var alphaLock sync.Mutex
		result := ScoredMove{ReturnThisMove: false}

		for i := 0; i < len(*moves); i++ {
			if (*moves)[i].Equals(&firstMove) {
				continue
			}

			if *abortFlag || j.player.abort {
				return b
			}

			if !nextLevelAbortFlag {
				childWaitGroup.Add(1)
				largeWindowWaitGroup.Add(1)
				go func(abortFlag *bool, move location.Move) {
					child, prev := j.player.applyMove(root, &move)
					score := j.Jamboree(child, depth-1, -alpha-1, -alpha, currentPlayer^1, prev, abortFlag).NegScore().Score
					bLock.Lock()
					if score > b.Score {
						b = ScoredMove{Score: score, Move: move}
					}
					bLock.Unlock()
					if score >= beta {
						*abortFlag = true
						resultLock.Lock()
						result = ScoredMove{Score: score, Move: move, ReturnThisMove: true}
						resultLock.Unlock()
					}
					largeWindowWaitGroup.Done()
					if score > alpha {
						//wait till all previous iterations are complete
						largeWindowWaitGroup.Wait()
						//check again that we should not abort, then proceed
						if !(*abortFlag) {
							child, prev := j.player.applyMove(root, &move)
							score = j.Jamboree(child, depth-1, -beta, -alpha, currentPlayer^1, prev, abortFlag).NegScore().Score
							if score >= beta {
								*abortFlag = true
								resultLock.Lock()
								result = ScoredMove{Score: score, Move: move, ReturnThisMove: true}
								resultLock.Unlock()
							}

							alphaLock.Lock()
							if score > alpha {
								alpha = score
							}
							alphaLock.Unlock()

							bLock.Lock()
							if score > b.Score {
								b = ScoredMove{Score: score, Move: move}
							}
							bLock.Unlock()
						}
					}
					childWaitGroup.Done()
				}(&nextLevelAbortFlag, (*moves)[i])
			} else {
				break
			}
		}
		childWaitGroup.Wait()

		if result.ReturnThisMove {
			j.syncTTWrite(root, currentPlayer, result.Score, uint16(depth), result.Move)
			return result
		}
		j.syncTTWrite(root, currentPlayer, b.Score, uint16(depth), b.Move)
		return b
	}
}

/**
 * Looks to the transposition table for an entry based on the board hash + color.  Returns the value if it exists or a
 * nil value if it does not exist.
 */
func (j *Jamboree) asyncTTRead(root *board.Board, currentPlayer color.Color) chan TTAnswerJamboree {
	answerChan := make(chan TTAnswerJamboree)
	go func(answerChan chan TTAnswerJamboree) {
		answer := TTAnswerJamboree{Found: false}
		if j.player.TranspositionTableEnabled {
			h := root.Hash()
			if e, ok := j.player.transpositionTable.Read(&h, currentPlayer); ok {
				entry := e.(*transposition_table.TranspositionTableEntryJamboree)
				entry.Lock.Lock()
				// make a deep copy of the TTAnswerJamboree data
				answer.Found = true
				answer.Score, answer.BestMove, answer.Depth = entry.Score, entry.BestMove, entry.Depth
				entry.Lock.Unlock()
			}
		}
		answerChan <- answer
	}(answerChan)
	return answerChan
}

/**
 * Performs a write to the transposition table if the depth being written is > depth currently written OR if there is no
 * entry currently in the table.
 */
func (j *Jamboree) syncTTWrite(root *board.Board, currentPlayer color.Color, score int, depth uint16, move location.Move) {
	if j.player.TranspositionTableEnabled {
		h := root.Hash()
		e, ok := j.player.transpositionTable.Read(&h, currentPlayer)
		if !ok {
			entry := transposition_table.TranspositionTableEntryJamboree{
				Score:    score,
				BestMove: move,
				Depth:    depth,
			}
			j.player.transpositionTable.Store(&h, currentPlayer, &entry)
		} else {
			entry := e.(*transposition_table.TranspositionTableEntryJamboree)
			entry.Lock.Lock()
			// make a deep copy into the transposition table
			entry.Score = score
			entry.BestMove = move
			entry.Depth = depth
			entry.Lock.Unlock()
		}
	}
}

func (j *Jamboree) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	j.player = p
	if b.CacheGetAllMoves || b.CacheGetAllAttackableMoves {
		log.Printf("WARNING: Trying to use %s with move caching enabled.\n", AlgorithmJamboree)
		log.Println("WARNING: Disabling GetAllMoves, GetAllAttackableMoves caching.")
		log.Printf("%s performs better without caching since it generates moves asynchronously\n", AlgorithmJamboree)
		b.CacheGetAllMoves = false
		b.CacheGetAllAttackableMoves = false
	}
	j.currentSearchDepth = j.player.MaxSearchDepth
	j.player.abort = false
	dummyAbortFlag := false
	j.threadNumber = 0
	j.activeThreads = 1
	best := j.Jamboree(b, j.player.MaxSearchDepth, NegInf, PosInf, j.player.PlayerColor, previousMove, &dummyAbortFlag)
	j.player.abort = true
	return &best
}

func (j *Jamboree) GetName() string {
	return fmt.Sprintf("%s", AlgorithmJamboree)
}
