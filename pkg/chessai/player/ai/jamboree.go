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
	player           *AIPlayer
	threadNumber     int
	activeThreads    int
	activeThreadLock sync.Mutex
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

		if len(*moves) == 0 {
			return ScoredMove{Score: j.player.EvaluateBoard(root, currentPlayer).TotalScore}
		}

		// transposition table saved us work
		if ttAnswer.Found && ttAnswer.Depth == uint16(depth) {
			atomic.AddUint64(&j.player.Metrics.MovesPrunedTransposition, uint64(len(*moves)))
			return ScoredMove{Score: ttAnswer.Score}
		}

		if *abortFlag {
			return ScoredMove{Score: NegInf}
		}

		var firstMove location.Move
		if ttAnswer.Found {
			firstMove = ttAnswer.BestMove
		} else {
			firstMove = (*moves)[0]
		}

		child, prev := j.player.applyMove(root, &firstMove)
		b := j.Jamboree(child, depth-1, -beta, -alpha, currentPlayer^1, prev, abortFlag)
		b.Score *= -1
		b.Move = firstMove

		if b.Score >= beta {
			return b
		}
		if b.Score > alpha {
			alpha = b.Score
		}

		if *abortFlag {
			return ScoredMove{Score: NegInf}
		}

		var childrenLock sync.Mutex
		numChildrenSpawned := 0

		moveChan := make(chan ScoredMove, len(*moves)-1)

		nextLevelAbortFlag := false
		foundBadMove := false

		var movesToResearchLock sync.Mutex
		var movesToResearch []location.Move

		someoneAlreadyAborting := false
		var someoneAlreadyAbortingLock sync.Mutex

		for i := 0; i < len(*moves); i++ {
			if *abortFlag {
				return ScoredMove{Score: NegInf}
			}

			if (*moves)[i].Equals(&firstMove) {
				continue
			}
			childrenLock.Lock()
			if !nextLevelAbortFlag {
				go func(moveChan chan ScoredMove, nextLevelAbortFlag *bool, move location.Move, myThreadID int) {
					j.threadNumber++
					fmt.Printf("Thread #%d has started\n", myThreadID)
					child, prev := j.player.applyMove(root, &move)
					nextScoredMove := j.Jamboree(child, depth-1, -alpha-1, -alpha, currentPlayer^1, prev,
						nextLevelAbortFlag)
					nextScoredMove.Score *= -1
					if nextScoredMove.Score > b.Score {
						b.Score = nextScoredMove.Score
						b.Move = move
					}
					if nextScoredMove.Score >= beta {
						fmt.Printf("I am the thread for %s and I found a score better than beta. Depth = %d \n", move, depth-1)

						iGetToAbort := false
						someoneAlreadyAbortingLock.Lock()
						if !someoneAlreadyAborting {
							fmt.Printf("Inside the already aborting lock at depth = %d and %s \n", depth-1, move)
							iGetToAbort = true
							someoneAlreadyAborting = true
							fmt.Printf("I am the thread for %s and I will abort!\n", move)
						}
						someoneAlreadyAbortingLock.Unlock()

						if iGetToAbort {
							childrenLock.Lock()
							fmt.Printf("I am the thread for %s and I am beginning the abort process\n", move)
							*nextLevelAbortFlag = true
							foundBadMove = true
							childrenDeallocated := 0
							for j := 0; j < numChildrenSpawned-1; j++ {
								<-moveChan
								childrenDeallocated++
								fmt.Printf("Children Spawned = %d, Children deallocated = %d\n", numChildrenSpawned, childrenDeallocated)
							}
							fmt.Printf("I am the thread for %s and I have successfully aborted the other threads\n", move)
							childrenLock.Unlock()
						}
					}
					if nextScoredMove.Score > alpha {
						movesToResearchLock.Lock()
						movesToResearch = append(movesToResearch, move)
						movesToResearchLock.Unlock()
					}
					moveChan <- ScoredMove{Score: nextScoredMove.Score, Move: move}
					fmt.Printf("Thread #%d has finished.\n", myThreadID)
				}(moveChan, &nextLevelAbortFlag, (*moves)[i], j.threadNumber)
				fmt.Printf("Thread #%d is associated with child #%d\n", j.threadNumber, numChildrenSpawned)
				j.threadNumber++
				numChildrenSpawned++
			}
			childrenLock.Unlock()
		}
		if foundBadMove {
			fmt.Println("A bad move was found!!!")
			return <-moveChan
		} else {
			// no one was >= beta so collect all data to end goroutines
			childrenDeallocated := 0
			for i := 0; i < numChildrenSpawned; i++ {
				<-moveChan
				childrenDeallocated++
			}
			// now, serially research any specific moves that we could possibly be better than firstChild
			for i := 0; i < len(movesToResearch); i++ {
				if *abortFlag {
					return ScoredMove{Score: NegInf}
				}
				child, prev := j.player.applyMove(root, &movesToResearch[i])
				dummyAbortFlag := false
				nextScoredMove := j.Jamboree(child, depth-1, -beta, -alpha, currentPlayer^1, prev, &dummyAbortFlag)
				nextScoredMove.Score *= -1
				if nextScoredMove.Score >= beta {
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
		}
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
		entry := e.(*transposition_table.TranspositionTableEntryJamboree)
		if !ok || entry.Depth < depth {
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
	dummyAbortFlag := false
	j.threadNumber = 0
	j.activeThreads = 1
	best := j.Jamboree(b, j.player.MaxSearchDepth, NegInf, PosInf, j.player.PlayerColor, previousMove, &dummyAbortFlag)
	return &best
}

func (j *Jamboree) GetName() string {
	return fmt.Sprintf("%s", AlgorithmJamboree)
}
