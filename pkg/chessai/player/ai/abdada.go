package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/transposition_table"
	"time"
)

func (ab *ABDADA) ABDADA(root *board.Board, depth, alpha, beta int, exclusiveProbe bool, currentPlayer color.Color, previousMove *board.LastMove) *ScoredMove {
	if depth == 0 {
		return &ScoredMove{
			Score: ab.player.EvaluateBoard(root, ab.player.PlayerColor).TotalScore,
		}
	} else {
		var best ScoredMove

		answerChan := ab.asyncTTRead(root, currentPlayer, uint16(depth), alpha, beta, exclusiveProbe)
		// generate moves while waiting for the answer ...
		moves := root.GetAllMoves(currentPlayer, previousMove)

		// block and grab the answer
		ttAnswer := <-answerChan
		alpha, beta = ttAnswer.alpha, ttAnswer.beta
		best.Score, best.Move = ttAnswer.score, ttAnswer.bestMove

		/* The current move is not evaluated if causing u a cutoff or
		if we are in exclusive mode and another processor
		is currently evaluating it. */
	}
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
}

type TTAnswer struct {
	alpha, beta, score int
	bestMove           location.Move
	onEvaluation       bool
}

func (ab *ABDADA) syncTTWrite(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta int, sm *ScoredMove) {
	if ab.player.TranspositionTableEnabled {
		// transposition table lookup
		h := root.Hash()
		if e, ok := ab.player.alphaBetaTable.Read(&h, currentPlayer); ok {
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
			if e, ok := ab.player.alphaBetaTable.Read(&h, currentPlayer); ok {
				entry := e.(*transposition_table.TranspositionTableEntryABDADA)
				entry.Lock.Lock()

				if entry.Depth == depth && exclusiveProbe && entry.NumProcessors > 0 {
					answer.onEvaluation = true
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
					entry.Depth = depth
					entry.EntryType = transposition_table.Unset
					entry.NumProcessors = 1
				}

				entry.Lock.Unlock()
			}
		}
		answerChan <- answer
	}(answerChan)
	return answerChan
}
