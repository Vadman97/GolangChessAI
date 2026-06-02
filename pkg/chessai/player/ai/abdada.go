package ai

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
	"log"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	maxKillerDepth     = 64
	aspirationDelta    = 50  // initial window half-width in centipawns
	aspirationMaxDelta = 500 // full-window fallback after repeated failures
	aspirationWiden    = 4   // multiply delta by this on each failure
	nullMoveMinDepth   = 3   // minimum depth to attempt null move
	nullMoveR          = 2   // null move depth reduction (increases to 3 at depth>=7)
	lmrMinDepth        = 3   // minimum depth before LMR kicks in
	lmrMinMoveIdx      = 3   // LMR applies after this many moves have been searched
	maxExtensions      = 4   // total check-extension budget per branch

	// Futility pruning: skip quiet moves at frontier/pre-frontier nodes when the
	// static eval is far below alpha. Margins indexed by depth (1 or 2).
	futilityMaxDepth = 2
	futilityMargin1  = PawnValueWeight     // 100 cp at depth 1
	futilityMargin2  = 3 * PawnValueWeight // 300 cp at depth 2

	// Razoring: if static eval + margin < alpha at depth 1, drop straight to qsearch.
	// If qsearch also fails low, prune the node.
	razorDepth  = 1
	razorMargin = 2 * PawnValueWeight // 200 cp

)

// squareIdx converts a location to a flat [0,63] index for history/killer tables.
func squareIdx(l location.Location) int {
	return int(l.GetRow())*board.Width + int(l.GetCol())
}

// onlyKingAndPawns returns true if the given side has no pieces other than king+pawns.
// Used to guard null move pruning against zugzwang.
func onlyKingAndPawns(b *board.Board, c color.Color) bool {
	for row := location.CoordinateType(0); row < board.Height; row++ {
		for col := location.CoordinateType(0); col < board.Width; col++ {
			p := b.GetPiece(location.NewLocation(row, col))
			if p == nil || p.GetColor() != c {
				continue
			}
			pt := p.GetPieceType()
			if pt != piece.KingType && pt != piece.PawnType {
				return false
			}
		}
	}
	return true
}

type ABDADA struct {
	player             *AIPlayer
	kill               uint32
	currentSearchDepth int
	NumThreads         int
	DisableNullMove    bool
	DisableLMR         bool
	DisableFutility    bool
	DisableRazoring    bool
	heuristicMu        sync.RWMutex
	// killers[depth%maxKillerDepth][0..1]: last two quiet moves causing beta cutoffs at this depth.
	killers [maxKillerDepth][2]location.Move
	// history[from][to]: accumulated depth^2 bonuses for quiet moves that caused cutoffs.
	// Shared across threads; atomic int32 for lock-free updates.
	history [board.Height * board.Width][board.Height * board.Width]int32
	// countermove[from][to]: best quiet response to the opponent's last move (from→to).
	// Cheap heuristic between killers and history in move ordering.
	countermove [board.Height * board.Width][board.Height * board.Width]location.Move
}

type RootMoveScore struct {
	Move  location.Move
	Score int
}

func (ab *ABDADA) GetName() string {
	return AlgorithmABDADA
}

func (ab *ABDADA) resetRootSearchHeuristics() {
	ab.heuristicMu.Lock()
	defer ab.heuristicMu.Unlock()
	ab.killers = [maxKillerDepth][2]location.Move{}
	ab.history = [board.Height * board.Width][board.Height * board.Width]int32{}
	ab.countermove = [board.Height * board.Width][board.Height * board.Width]location.Move{}
}

func (ab *ABDADA) isKiller(m location.Move, depth int) bool {
	ab.heuristicMu.RLock()
	defer ab.heuristicMu.RUnlock()
	k := &ab.killers[depth%maxKillerDepth]
	return (m.Start.Equals(k[0].Start) && m.End.Equals(k[0].End)) ||
		(m.Start.Equals(k[1].Start) && m.End.Equals(k[1].End))
}

func (ab *ABDADA) killerPair(depth int) [2]location.Move {
	ab.heuristicMu.RLock()
	defer ab.heuristicMu.RUnlock()
	return ab.killers[depth%maxKillerDepth]
}

func (ab *ABDADA) storeKiller(depth int, m location.Move) {
	ab.heuristicMu.Lock()
	defer ab.heuristicMu.Unlock()
	k := &ab.killers[depth%maxKillerDepth]
	if m.Start.Equals(k[0].Start) && m.End.Equals(k[0].End) {
		return
	}
	k[1] = k[0]
	k[0] = m
}

func (ab *ABDADA) updateHistory(m location.Move, depth int) {
	from := squareIdx(m.Start)
	to := squareIdx(m.End)
	atomic.AddInt32(&ab.history[from][to], int32(depth*depth))
}

func (ab *ABDADA) historyScore(m location.Move) int32 {
	return atomic.LoadInt32(&ab.history[squareIdx(m.Start)][squareIdx(m.End)])
}

func (ab *ABDADA) counterMove(prev *board.LastMove, moves *[]location.Move) (location.Move, bool) {
	if prev == nil {
		return location.Move{}, false
	}
	from := squareIdx(prev.Move.Start)
	to := squareIdx(prev.Move.End)
	ab.heuristicMu.RLock()
	cm := ab.countermove[from][to]
	ab.heuristicMu.RUnlock()
	if !cm.Start.Equals(cm.End) && isMoveInList(cm, moves) {
		return cm, true
	}
	return location.Move{}, false
}

func (ab *ABDADA) storeCounterMove(prev *board.LastMove, m location.Move) {
	if prev == nil {
		return
	}
	from := squareIdx(prev.Move.Start)
	to := squareIdx(prev.Move.End)
	ab.heuristicMu.Lock()
	ab.countermove[from][to] = m
	ab.heuristicMu.Unlock()
}

func (ab *ABDADA) isKilled() bool {
	return atomic.LoadUint32(&ab.kill) != 0
}

func (ab *ABDADA) setKilled(v bool) {
	if v {
		atomic.StoreUint32(&ab.kill, 1)
	} else {
		atomic.StoreUint32(&ab.kill, 0)
	}
}

func stableDepthMove(previous, current ScoredMove) ScoredMove {
	const scoreRegressionThreshold = 200
	if previous.Score != NegInf &&
		!previous.Move.Start.Equals(previous.Move.End) &&
		current.Score < previous.Score-scoreRegressionThreshold {
		current.Move = previous.Move
		current.MoveSequence = previous.MoveSequence
	}
	return current
}

// searchUnstable reports whether the root search looks unstable between two
// consecutive completed iterative-deepening depths: either the best move
// changed, or its score dropped meaningfully (a fail-low / worsening). Unstable
// positions are worth extra time — see the soft-bound logic in iterativeABDADA.
func searchUnstable(previous, current ScoredMove) bool {
	if previous.Score == NegInf || previous.Move.Start.Equals(previous.Move.End) {
		return false // no prior completed depth to compare against
	}
	if !previous.Move.Start.Equals(current.Move.Start) || !previous.Move.End.Equals(current.Move.End) {
		return true // best move changed between depths
	}
	const dropThreshold = 50 // centipawns
	return current.Score < previous.Score-dropThreshold
}

// ABDADA is the core parallel alpha-beta search function.
// nullMoveOk: false immediately after a null move (prevents consecutive null moves).
// ply: distance from the root (0 at root), used for killer indexing.
// extensions: remaining check-extension budget (starts at maxExtensions at root).
func (ab *ABDADA) ABDADA(root *board.Board, depth, alpha, beta int, exclusiveProbe bool, currentPlayer color.Color, previousMove *board.LastMove, nullMoveOk bool, ply int, extensions int) ScoredMove {
	// Repetition draw: a position recurring for the first time (against the game
	// history or earlier in this search path) is scored as a draw. A side that can
	// force one repetition can force three, so treating the first recurrence as a
	// draw lets the search recognize a perpetual ~2 plies sooner than waiting for
	// the third occurrence — decisive under blitz time pressure, where the real
	// game only reached depth 5 and walked into a perpetual it scored as +6.
	// Never at the root (ply 0): we still owe a move there.
	if ply > 0 && root.CurrentPositionRepeats >= 1 {
		return ScoredMove{Score: StalemateScore}
	}

	inCheck := root.IsKingInCheck(currentPlayer)

	// Check extension: when the side to move is in check, extend by 1 ply so the
	// search doesn't cut off forced check-evasion sequences at the horizon.
	// Bounded by an extension budget to prevent infinite recursion.
	if inCheck && extensions > 0 {
		depth++
		extensions--
	}

	if depth == 0 {
		return ScoredMove{
			Score: ab.player.Quiesce(root, alpha, beta, currentPlayer, previousMove),
		}
	}

	var best ScoredMove
	best.Score = NegInf

	searchAlpha, searchBeta := alpha, beta
	ttAnswer := ab.ttRead(root, currentPlayer, uint16(depth), alpha, beta, exclusiveProbe)
	movesArr := root.GetAllMoves(currentPlayer, previousMove)
	if !ttAnswer.bestMove.Start.Equals(ttAnswer.bestMove.End) && !isMoveInList(ttAnswer.bestMove, movesArr) {
		ttAnswer.bestMove = location.Move{}
		ttAnswer.score = NegInf
		ttAnswer.alpha = searchAlpha
		ttAnswer.beta = searchBeta
	}
	alpha, beta = ttAnswer.alpha, ttAnswer.beta
	originalAlpha := alpha
	best.Score, best.Move = ttAnswer.score, ttAnswer.bestMove

	if ab.player.terminalNode(root, movesArr) {
		return ScoredMove{
			Score: AdjustMateScore(ab.player.EvaluateBoard(root, currentPlayer).TotalScore, depth),
		}
	}

	// One-reply extension: only one legal move means we're in a forced position —
	// extend search so we don't cut off the resolution of a forcing sequence.
	// Bounded by the extension budget shared with check extensions.
	if len(*movesArr) == 1 && extensions > 0 {
		depth++
		extensions--
	}

	if alpha >= beta || best.Score == OnEvaluation {
		// Only prune if we have a valid best move to return. A zero BestMove in
		// the TT means the entry was written before any move was evaluated (e.g.
		// after an abort). Returning it would propagate a zero-move to the root.
		if !best.Move.Start.Equals(best.Move.End) {
			atomic.AddUint64(&ab.player.Metrics.MovesPrunedTransposition, uint64(len(*movesArr)))
			return best
		}
	}

	// Null Move Pruning: pass the turn and search at reduced depth.
	// Skip at ply=0 (root): pruning here returns ScoredMove{Score:beta, Move:zero}
	// with no valid move, which propagates up and triggers a random-move fallback.
	// Skip when in check, in zugzwang-prone endgames, or after a prior null move.
	if !ab.DisableNullMove && nullMoveOk && !inCheck && depth >= nullMoveMinDepth && !onlyKingAndPawns(root, currentPlayer) && ply > 0 {
		R := nullMoveR
		if depth >= 7 {
			R = 3
		}
		nullVal := ab.ABDADA(root, depth-1-R, -beta, -beta+1, false, currentPlayer^1, nil, false, ply+1, extensions)
		nullVal.Score = -nullVal.Score
		if nullVal.Score >= beta {
			atomic.AddUint64(&ab.player.Metrics.MovesPrunedAB, 1)
			return ScoredMove{Score: beta}
		}
	}

	// Futility pruning + razoring: compute static eval once for frontier nodes.
	// Futility: skip quiet moves where standPat + margin can't reach alpha.
	// Razoring: if standPat + margin < alpha even before any moves, drop to qsearch.
	// Neither applies when in check (must search all evasions).
	var standPat int
	var canFutilityPrune bool
	if !inCheck && depth <= futilityMaxDepth && alpha < WinScore && beta > LossScore && (!ab.DisableFutility || !ab.DisableRazoring) {
		standPat = ab.player.EvaluateBoard(root, currentPlayer).TotalScore
		canFutilityPrune = !ab.DisableFutility

		// Razoring at depth 1: if the static eval is far below alpha, drop to qsearch.
		// If qsearch also fails low, prune this whole branch.
		// Skip at ply=0 (root): returning a null move here is never safe.
		if !ab.DisableRazoring && depth == razorDepth && standPat+razorMargin < alpha && ply > 0 {
			qScore := ab.player.Quiesce(root, alpha-1, alpha, currentPlayer, previousMove)
			if qScore < alpha {
				atomic.AddUint64(&ab.player.Metrics.MovesPrunedAB, uint64(len(*movesArr)))
				return ScoredMove{Score: qScore}
			}
		}
	}

	killerPair := ab.killerPair(ply)
	orderedMoves := orderMoves(*movesArr, ttAnswer.bestMove, killerPair, ab, root, previousMove)

	iteration := 0
	allDone := false
	for iteration < 2 && alpha < beta && !allDone {
		// Don't abort before evaluating at least one move: a zero BestMove
		// would propagate to the root and trigger a random fallback.
		if (ab.player.isAborted() || ab.isKilled()) && !best.Move.Start.Equals(best.Move.End) {
			return best
		}
		iteration++
		allDone = true
		firstMove := true
		moves := orderedMoves
		move := moves[0]
		moves = moves[1:]
		moveIdx := 0
		for alpha < beta {
			if (ab.player.isAborted() || ab.isKilled()) && !firstMove {
				return best
			}
			moveIdx++
			exclusiveProbe = iteration == 1 && !firstMove

			isCapture := root.GetPiece(move.End) != nil || isEnPassantMove(root, move)
			isPromo, _ := move.End.GetPawnPromotion()
			isKiller := ab.isKiller(move, ply)
			isTTMove := move.Start.Equals(ttAnswer.bestMove.Start) && move.End.Equals(ttAnswer.bestMove.End)

			// Near-promotion pawn advances (rank 5 or 6 from own back rank) are tactically
			// critical — the threat of promoting with check or material gain must be seen
			// at full depth. LMR on these moves caused false forced-mate hallucinations and
			// missed Qxd3-type captures in the opponent's response tree.
			isNearPromo := false
			if mp := root.GetPiece(move.Start); mp != nil && mp.GetPieceType() == piece.PawnType {
				endRow := int(move.End.GetRow())
				if currentPlayer == color.White && endRow >= 5 {
					isNearPromo = true
				} else if currentPlayer == color.Black && endRow <= 2 {
					isNearPromo = true
				}
			}

			// Futility pruning: skip quiet moves at frontier/pre-frontier nodes when
			// standPat + margin can't possibly reach alpha. This lets us search deeper
			// on positions that matter without wasting time on hopeless quiet moves.
			// Not applied to: captures, promotions, killers, TT moves, near-promos,
			// the first move (we must evaluate at least one), or when aborting.
			futilityPruned := false
			if canFutilityPrune && !firstMove && !isCapture && !isPromo && !isKiller && !isTTMove && !isNearPromo && iteration == 1 {
				var margin int
				if depth == 1 {
					margin = futilityMargin1
				} else {
					margin = futilityMargin2
				}
				if standPat+margin <= alpha {
					atomic.AddUint64(&ab.player.Metrics.MovesPrunedAB, 1)
					futilityPruned = true
				}
			}

			if !futilityPruned {
				// Late Move Reductions: quietly search less-promising moves at reduced depth.
				// Conditions: not a capture, not a promotion, not a killer, not the TT move,
				// not when in check, not a near-promotion pawn advance, only after lmrMinMoveIdx
				// moves already searched.
				// Guard: best.Score must be a real score (> NegInf). When ABDADA threads
				// return OnEvaluation for the first N moves, best.Score stays NegInf and
				// -(NegInf+1) = PosInf-1 overflows the LMR window into garbage territory.
				doLMR := iteration == 1 &&
					!ab.DisableLMR &&
					depth >= lmrMinDepth &&
					moveIdx > lmrMinMoveIdx &&
					!isCapture && !isPromo && !isKiller && !isTTMove && !inCheck && !isNearPromo &&
					best.Score > NegInf

				var value ScoredMove
				child, pm := ab.player.applyMove(root, &move)

				if doLMR {
					reduction := int(math.Log(float64(depth)) * math.Log(float64(moveIdx)) / 2.0)
					if reduction < 1 {
						reduction = 1
					}
					if reduction > depth-2 {
						reduction = depth - 2
					}
					lmr := ab.ABDADA(child, depth-1-reduction, -(util.MaxScore(alpha, best.Score) + 1), -util.MaxScore(alpha, best.Score), false, currentPlayer^1, pm, true, ply+1, extensions)
					lmr.Score = -lmr.Score
					lmr.Move = move
					if lmr.Score == -OnEvaluation || lmr.Score > util.MaxScore(alpha, best.Score) {
						// LMR failed high or deferred — do full-depth re-search.
						value = ab.ABDADA(child, depth-1, -beta, -util.MaxScore(alpha, best.Score), exclusiveProbe, currentPlayer^1, pm, true, ply+1, extensions)
						value.Score = -value.Score
						value.Move = move
					} else {
						// LMR confirmed: move is not good enough.
						value = lmr
					}
				} else {
					value = ab.ABDADA(child, depth-1, -beta, -util.MaxScore(alpha, best.Score), exclusiveProbe, currentPlayer^1, pm, true, ply+1, extensions)
					value.Score = -value.Score
					value.Move = move
				}

				if value.Score == -OnEvaluation {
					allDone = false
				} else if value.Score > best.Score || best.Move.Start.Equals(best.Move.End) {
					best = value
					if best.Score >= beta {
						atomic.AddUint64(&ab.player.Metrics.MovesPrunedAB, uint64(len(moves)))
						// Update killer, history, and countermove for quiet cutoff moves.
						if !isCapture && !isPromo {
							ab.storeKiller(ply, move)
							ab.updateHistory(move, depth)
							// Countermove: record this move as a good response to the opponent's last move.
							ab.storeCounterMove(previousMove, move)
						}
						ab.syncTTWrite(root, currentPlayer, uint16(depth), originalAlpha, beta, &best)
						return best
					}
					if best.Score > alpha {
						alpha = best.Score
					}
				}
			} // end if !futilityPruned
			if len(moves) == 0 {
				break
			}
			firstMove = false
			move = moves[0]
			moves = moves[1:]
		}
	}
	ab.syncTTWrite(root, currentPlayer, uint16(depth), originalAlpha, beta, &best)
	return best
}

func (ab *ABDADA) getBestMove(b *board.Board, depth, alpha, beta int, previousMove *board.LastMove) ScoredMove {
	ab.player.setAbort(false)
	originalAlpha := alpha
	originalBeta := beta
	if ab.NumThreads == 0 {
		ab.NumThreads = runtime.NumCPU()
		log.Printf("ABDADA runs in parallel, defaulting to #%d threads (# cpu cores)\n", ab.NumThreads)
	}
	if runtime.GOMAXPROCS(0) < ab.NumThreads {
		runtime.GOMAXPROCS(ab.NumThreads)
	}

	movesArr := b.GetAllMoves(ab.player.PlayerColor, previousMove)
	if len(*movesArr) == 0 {
		return ScoredMove{Score: ab.player.EvaluateBoard(b, ab.player.PlayerColor).TotalScore}
	}
	ttAnswer := ab.ttRead(b, ab.player.PlayerColor, uint16(depth), alpha, beta, false)
	orderedMoves := orderMoves(*movesArr, ttAnswer.bestMove, [2]location.Move{}, ab, b, previousMove)

	if ab.NumThreads == 1 {
		best := ScoredMove{Score: NegInf}
		second := ScoredMove{Score: NegInf}
		for _, move := range orderedMoves {
			if ab.player.isAborted() && !best.Move.Start.Equals(best.Move.End) {
				break
			}
			child, pm := ab.player.applyMove(b, &move)
			value := ab.ABDADA(child, depth-1, NegInf, PosInf, false, ab.player.PlayerColor^1, pm, true, 1, maxExtensions)
			value.Score = -value.Score
			value.Move = move
			if value.Score == OnEvaluation || value.Score == -OnEvaluation {
				continue
			}
			best, second = updateRootTopTwo(best, second, value)
		}
		best = ab.verifyCloseRootMoves(b, depth, best, second)
		if !best.Move.Start.Equals(best.Move.End) {
			ab.syncTTWrite(b, ab.player.PlayerColor, uint16(depth), originalAlpha, originalBeta, &best)
		}
		return best
	}

	best := ScoredMove{Score: NegInf}
	second := ScoredMove{Score: NegInf}
	firstRootMove := orderedMoves[0]
	firstValue := ab.searchRootMove(b, firstRootMove, depth, NegInf, PosInf)
	if firstValue.Score != OnEvaluation && firstValue.Score != -OnEvaluation {
		best, second = updateRootTopTwo(best, second, firstValue)
		if best.Score > alpha {
			alpha = best.Score
		}
		if alpha >= beta {
			ab.syncTTWrite(b, ab.player.PlayerColor, uint16(depth), originalAlpha, originalBeta, &best)
			return best
		}
	}
	remainingMoves := orderedMoves[1:]
	if len(remainingMoves) == 0 {
		if !best.Move.Start.Equals(best.Move.End) {
			ab.syncTTWrite(b, ab.player.PlayerColor, uint16(depth), originalAlpha, originalBeta, &best)
		}
		return best
	}

	workerCount := ab.NumThreads
	if workerCount > len(remainingMoves) {
		workerCount = len(remainingMoves)
	}
	jobs := make(chan location.Move, len(remainingMoves))
	results := make(chan ScoredMove, len(remainingMoves))
	for i := 0; i < workerCount; i++ {
		go func() {
			for move := range jobs {
				if ab.player.isAborted() {
					continue
				}
				// Root move scores are used for final move selection, so they must
				// be exact. Searching root siblings with a moving alpha window can
				// return fail-low/fail-high bounds that look comparable but are not;
				// in forcing promotion/mate lines that let TT-bound scores outrank
				// the only non-mating defense. Keep alpha/beta pruning inside the
				// child tree, but score each root move with a full window.
				value := ab.searchRootMove(b, move, depth, NegInf, PosInf)
				results <- value
			}
		}()
	}
	for _, move := range remainingMoves {
		jobs <- move
	}
	close(jobs)

	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
collectResults:
	for completed := 0; completed < len(remainingMoves); completed++ {
		var result ScoredMove
		select {
		case result = <-results:
		case <-ticker.C:
			if ab.player.isAborted() {
				break collectResults
			}
			completed--
			continue
		}
		if result.Score == OnEvaluation || result.Score == -OnEvaluation {
			continue
		}
		best, second = updateRootTopTwo(best, second, result)
	}
	best = ab.verifyCloseRootMoves(b, depth, best, second)
	if !best.Move.Start.Equals(best.Move.End) {
		ab.syncTTWrite(b, ab.player.PlayerColor, uint16(depth), originalAlpha, originalBeta, &best)
	}
	return best
}

func updateRootTopTwo(best, second, candidate ScoredMove) (ScoredMove, ScoredMove) {
	if candidate.Score == OnEvaluation || candidate.Score == -OnEvaluation || candidate.Move.Start.Equals(candidate.Move.End) {
		return best, second
	}
	if best.Move.Start.Equals(best.Move.End) || candidate.Score > best.Score {
		if !sameRootMove(candidate, best) {
			second = best
		}
		best = candidate
		return best, second
	}
	if !sameRootMove(candidate, best) && (second.Move.Start.Equals(second.Move.End) || candidate.Score > second.Score) {
		second = candidate
	}
	return best, second
}

func sameRootMove(a, b ScoredMove) bool {
	return a.Move.Start.Equals(b.Move.Start) && a.Move.End.Equals(b.Move.End)
}

func (ab *ABDADA) verifyCloseRootMoves(b *board.Board, depth int, best, second ScoredMove) ScoredMove {
	const verifyMargin = 150
	if ab.player.isAborted() ||
		depth < 4 ||
		best.Move.Start.Equals(best.Move.End) ||
		second.Move.Start.Equals(second.Move.End) ||
		best.Score-second.Score > verifyMargin {
		return best
	}
	verifiedBest := ab.searchRootMove(b, best.Move, depth, NegInf, PosInf)
	verifiedSecond := ab.searchRootMove(b, second.Move, depth, NegInf, PosInf)
	if verifiedBest.Score == OnEvaluation || verifiedBest.Score == -OnEvaluation {
		return best
	}
	if verifiedSecond.Score != OnEvaluation && verifiedSecond.Score != -OnEvaluation && verifiedSecond.Score > verifiedBest.Score {
		return verifiedSecond
	}
	return verifiedBest
}

func (ab *ABDADA) searchRootMove(b *board.Board, move location.Move, depth, alpha, beta int) ScoredMove {
	child, pm := ab.player.applyMove(b, &move)
	value := ab.ABDADA(child, depth-1, -beta, -alpha, false, ab.player.PlayerColor^1, pm, true, 1, maxExtensions)
	value.Score = -value.Score
	value.Move = move
	return value
}

func raiseRootAlpha(rootAlpha *int64, score int) {
	for {
		current := atomic.LoadInt64(rootAlpha)
		if int64(score) <= current {
			return
		}
		if atomic.CompareAndSwapInt64(rootAlpha, current, int64(score)) {
			return
		}
	}
}

func (ab *ABDADA) ScoreRootMoves(p *AIPlayer, b *board.Board, previousMove *board.LastMove, depth int) []RootMoveScore {
	ab.player = p
	ab.resetRootSearchHeuristics()
	movesArr := b.GetAllMoves(p.PlayerColor, previousMove)
	if len(*movesArr) == 0 {
		return nil
	}
	orderedMoves := orderMoves(*movesArr, location.Move{}, [2]location.Move{}, ab, b, previousMove)
	scores := make([]RootMoveScore, 0, len(orderedMoves))
	for _, move := range orderedMoves {
		child, pm := p.applyMove(b, &move)
		value := ab.ABDADA(child, depth-1, NegInf, PosInf, false, p.PlayerColor^1, pm, true, 1, maxExtensions)
		value.Score = -value.Score
		if value.Score == OnEvaluation || value.Score == -OnEvaluation {
			continue
		}
		scores = append(scores, RootMoveScore{Move: move, Score: value.Score})
	}
	return scores
}

type rootVote struct {
	move  ScoredMove
	count int
}

func selectRootBest(bestMoves []ScoredMove) ScoredMove {
	if len(bestMoves) == 0 {
		return ScoredMove{Score: NegInf}
	}
	votes := make([]rootVote, 0, len(bestMoves))
	for _, sm := range bestMoves {
		found := false
		for i := range votes {
			if sm.Move.Equals(&votes[i].move.Move) {
				votes[i].count++
				if sm.Score > votes[i].move.Score {
					votes[i].move = sm
				}
				found = true
				break
			}
		}
		if !found {
			votes = append(votes, rootVote{move: sm, count: 1})
		}
	}

	bestVote := rootVote{move: ScoredMove{Score: NegInf}}
	for _, vote := range votes {
		if vote.move.Score > bestVote.move.Score ||
			(vote.move.Score == bestVote.move.Score && vote.count > bestVote.count) {
			bestVote = vote
		}
	}
	return bestVote.move
}

func (ab *ABDADA) iterativeABDADA(b *board.Board, previousMove *board.LastMove) ScoredMove {
	start := time.Now()
	best := ScoredMove{Score: NegInf}
	iterativeIncrement := config.Get().IterativeIncrement

	// Easy move: with a single legal move there is nothing to search — play it
	// immediately and bank the clock for positions that actually need thought.
	rootMoves := b.GetAllMoves(ab.player.PlayerColor, previousMove)
	if len(*rootMoves) == 1 {
		m := (*rootMoves)[0]
		score := ab.player.EvaluateBoard(b, ab.player.PlayerColor).TotalScore
		ab.player.LastSearchDepth = 0
		ab.player.printer <- fmt.Sprintf("%s easy move (only legal move): %s\n", ab.GetName(), m)
		return ScoredMove{Move: m, Score: score}
	}

	// Soft/hard time bounds. MaxThinkTime is the HARD ceiling enforced mid-search
	// by trackThinkTime. The SOFT target decides whether to START a new
	// iterative-deepening iteration: each iteration roughly doubles search time,
	// so once we've spent ~half the budget the next iteration almost certainly
	// won't finish before the hard abort and we'd just throw away the partial.
	// We only push past the soft target — up to the hard ceiling — when the
	// position is unstable (best move changing or score dropping), where the
	// extra depth is most likely to change the move we play.
	hardLimit := ab.player.MaxThinkTime
	var softStable, softUnstable time.Duration
	if hardLimit > 0 {
		softStable = hardLimit / 2        // 50% rule: quiet, settled positions
		softUnstable = hardLimit * 9 / 10 // extend toward the hard ceiling when unstable
	}
	unstable := false

	for ab.currentSearchDepth = iterativeIncrement; ab.currentSearchDepth <= ab.player.MaxSearchDepth; ab.currentSearchDepth += iterativeIncrement {
		// Soft-bound check: decide whether to begin THIS iteration. Always run at
		// least the first iteration so we never return a zero-move.
		if hardLimit > 0 && ab.currentSearchDepth > iterativeIncrement {
			limit := softStable
			if unstable {
				limit = softUnstable
			}
			if elapsed := time.Since(start); elapsed >= limit {
				ab.player.printer <- fmt.Sprintf("%s soft stop after depth %d (elapsed %s >= %s, unstable=%v)\n",
					ab.GetName(), ab.player.LastSearchDepth, elapsed, limit, unstable)
				break
			}
		}
		// Aspiration windows: start with a narrow window around the previous score.
		// On failure, widen exponentially until the full window is used.
		alpha, beta := NegInf, PosInf
		delta := aspirationDelta
		// Skip aspiration windows when the previous score was a mate — the window
		// would be centered on a huge value, causing repeated fail-lows as normal
		// evals fall outside it, wasting time on re-searches.
		isMate := best.Score >= WinScore || best.Score <= LossScore
		if ab.currentSearchDepth > iterativeIncrement && best.Score != NegInf && !isMate {
			alpha = best.Score - delta
			beta = best.Score + delta
		}

		var newGuess ScoredMove
		for {
			thinking, done := make(chan bool), make(chan bool, 1)
			go ab.player.trackThinkTime(thinking, done, start)
			newGuess = ab.getBestMove(b, ab.currentSearchDepth, alpha, beta, previousMove)
			close(thinking)
			<-done

			if ab.player.isAborted() {
				break
			}

			if newGuess.Score <= alpha {
				// Fail-low: widen the window downward.
				delta *= aspirationWiden
				alpha = newGuess.Score - delta
				if alpha < NegInf {
					alpha = NegInf
				}
			} else if newGuess.Score >= beta {
				// Fail-high: widen the window upward.
				delta *= aspirationWiden
				beta = newGuess.Score + delta
				if beta > PosInf {
					beta = PosInf
				}
			} else {
				break // search succeeded within the window
			}
		}

		if !ab.player.isAborted() {
			prevForStability := best
			best = stableDepthMove(best, newGuess)
			ab.player.LastSearchDepth = ab.currentSearchDepth
			// Feed the next iteration's soft-bound decision: was this depth stable?
			unstable = searchUnstable(prevForStability, best)
			ab.player.printer <- fmt.Sprintf("Best D:%d M:%s score:%d\n", ab.player.LastSearchDepth, best.Move, best.Score)
		} else {
			// Use the partial result if we haven't found any valid move yet.
			// Without this, a timeout on the very first IDA iteration leaves
			// best as zero-move and the engine falls back to a random move.
			if best.Move.Start.Equals(best.Move.End) && !newGuess.Move.Start.Equals(newGuess.Move.End) {
				best = newGuess
			}
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
	ab.resetRootSearchHeuristics()
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
	atomic.AddUint64(&p.Metrics.MovesConsidered, 1)
	return
}

// orderMoves returns moves ordered for best alpha-beta pruning:
//  1. TT best move (if capture)
//  2. All other captures (MVV-LVA sorted)
//  3. TT best move (if quiet)
//  4. Killer moves
//  5. Remaining quiet moves (sorted by history heuristic score, descending)
//
// mvvLvaScore returns a score for capture ordering: high victim value and low attacker value
// are both good. Multiply victim by 10 to ensure victim dominates regardless of attacker type.
func mvvLvaScore(b *board.Board, m location.Move) int {
	attacker := b.GetPiece(m.Start)
	victim := b.GetPiece(m.End)
	av, vv := 1, 0
	if attacker != nil {
		av = PieceValue[attacker.GetPieceType()]
	}
	if victim != nil {
		vv = PieceValue[victim.GetPieceType()]
	}
	return vv*10 - av
}

// sortCapturesMVVLVA sorts a capture list in place by MVV-LVA score descending.
func sortCapturesMVVLVA(captures []location.Move, b *board.Board) {
	if len(captures) <= 1 {
		return
	}
	scores := make([]int, len(captures))
	for i, m := range captures {
		scores[i] = mvvLvaScore(b, m)
	}
	for i := 1; i < len(captures); i++ {
		for j := i; j > 0 && scores[j] > scores[j-1]; j-- {
			captures[j], captures[j-1] = captures[j-1], captures[j]
			scores[j], scores[j-1] = scores[j-1], scores[j]
		}
	}
}

func isEnPassantMove(b *board.Board, m location.Move) bool {
	p := b.GetPiece(m.Start)
	if p == nil || p.GetPieceType() != piece.PawnType {
		return false
	}
	return b.IsEmpty(m.End) && m.Start.GetCol() != m.End.GetCol()
}

func isMoveInList(m location.Move, moves *[]location.Move) bool {
	for _, lm := range *moves {
		if lm.Start.Equals(m.Start) && lm.End.Equals(m.End) {
			return true
		}
	}
	return false
}

// orderMoves returns moves ordered for best alpha-beta pruning:
//  1. TT best move (if capture or promotion)
//  2. Promotions
//  3. Winning/even captures (SEE >= 0), MVV-LVA sorted
//  4. TT best move (if quiet non-promotion)
//  5. Killer moves
//  6. Countermove (best response to opponent's last move)
//  7. Remaining quiet moves (history heuristic score, descending)
//  8. Losing captures (SEE < 0), MVV-LVA sorted
func orderMoves(moves []location.Move, ttMove location.Move, killers [2]location.Move, ab *ABDADA, b *board.Board, prevMove *board.LastMove) []location.Move {
	ordered := make([]location.Move, 0, len(moves))
	var promotions, goodCaptures, badCaptures, killerMoves, counterMoves, quiets []location.Move

	// Validate the TT move is actually legal on this board before placing it first.
	// A stale or hash-colliding TT entry can carry over a move that is no longer legal
	// (e.g. wrong piece type at the start square), and evaluating it produces a bogus
	// score that can become the "best" move.
	hasTT := !ttMove.Start.Equals(ttMove.End) && isMoveInList(ttMove, &moves)
	ttIsCapture := hasTT && (b.GetPiece(ttMove.End) != nil || isEnPassantMove(b, ttMove))
	ttIsPromotion := false
	if hasTT {
		ttIsPromotion, _ = ttMove.End.GetPawnPromotion()
	}

	// Countermove lookup: best known response to the opponent's last move.
	var counterMove location.Move
	hasCounter := false
	if ab != nil {
		counterMove, hasCounter = ab.counterMove(prevMove, &moves)
	}

	// Determine the side to move from the piece on the first non-empty start square.
	// Used for SEE computation on captures.
	var stm byte
	for _, m := range moves {
		if p := b.GetPiece(m.Start); p != nil {
			stm = p.GetColor()
			break
		}
	}

	for _, m := range moves {
		if hasTT && m.Start.Equals(ttMove.Start) && m.End.Equals(ttMove.End) {
			continue
		}
		isPromotion, _ := m.End.GetPawnPromotion()
		if isPromotion {
			promotions = append(promotions, m)
		} else if b.GetPiece(m.End) != nil || isEnPassantMove(b, m) {
			// Use SEE to split captures into winning/even (≥0) and losing (<0).
			// The TT move was already skipped above so we don't re-score it here.
			seeScore := b.SEE(m, stm)
			if seeScore >= 0 {
				goodCaptures = append(goodCaptures, m)
			} else {
				badCaptures = append(badCaptures, m)
			}
		} else {
			// Quiet move: check killer and countermove tables first.
			isK := (m.Start.Equals(killers[0].Start) && m.End.Equals(killers[0].End)) ||
				(m.Start.Equals(killers[1].Start) && m.End.Equals(killers[1].End))
			isCM := hasCounter && m.Start.Equals(counterMove.Start) && m.End.Equals(counterMove.End)
			if isK {
				killerMoves = append(killerMoves, m)
			} else if isCM {
				counterMoves = append(counterMoves, m)
			} else {
				quiets = append(quiets, m)
			}
		}
	}

	// Sort quiets by history score descending.
	if ab != nil && len(quiets) > 1 {
		scores := make([]int32, len(quiets))
		for i, m := range quiets {
			scores[i] = ab.historyScore(m)
		}
		for i := 1; i < len(quiets); i++ {
			for j := i; j > 0 && scores[j] > scores[j-1]; j-- {
				quiets[j], quiets[j-1] = quiets[j-1], quiets[j]
				scores[j], scores[j-1] = scores[j-1], scores[j]
			}
		}
	}

	// Sort winning captures by MVV-LVA (best first).
	sortCapturesMVVLVA(goodCaptures, b)
	// Sort losing captures by MVV-LVA too (least bad first).
	sortCapturesMVVLVA(badCaptures, b)

	if hasTT && (ttIsCapture || ttIsPromotion) {
		ordered = append(ordered, ttMove)
	}
	ordered = append(ordered, promotions...)
	ordered = append(ordered, goodCaptures...)
	if hasTT && !ttIsCapture && !ttIsPromotion {
		ordered = append(ordered, ttMove)
	}
	ordered = append(ordered, killerMoves...)
	ordered = append(ordered, counterMoves...)
	ordered = append(ordered, quiets...)
	ordered = append(ordered, badCaptures...)
	return ordered
}

type TTAnswer struct {
	alpha, beta, score int
	bestMove           location.Move
}

func (ab *ABDADA) syncTTWrite(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta int, sm *ScoredMove) {
	if ab.player.TranspositionTableEnabled {
		if ab.player.isAborted() || ab.isKilled() {
			return
		}
		// PosInf/NegInf are search bounds, not board evaluations.  Writing them
		// to the TT would poison future lookups with fake mate scores (the
		// normalise/denormalise roundtrip converts PosInf into ~1999999999).
		if sm.Score >= PosInf || sm.Score <= NegInf || sm.Score == OnEvaluation || sm.Score == -OnEvaluation {
			return
		}
		h := root.Hash()
		entryType := transposition_table.TrueScore
		if sm.Score >= beta {
			entryType = transposition_table.LowerBound
		} else if sm.Score <= alpha {
			entryType = transposition_table.UpperBound
		}
		normalizedScore := NormalizeMateScore(sm.Score, int(depth))
		gen := atomic.LoadUint32(&ab.player.ttGeneration)

		// Only store the best move for TrueScore and LowerBound entries.
		// For UpperBound (fail-low), all moves scored below alpha so the "best"
		// is unreliable — storing it overwrites the previous deeper search's move
		// and corrupts move ordering in aspiration re-searches.
		storeBestMove := entryType != transposition_table.UpperBound

		if e, ok := ab.player.transpositionTable.Read(&h, currentPlayer); ok {
			entry := e.(*transposition_table.TranspositionTableEntryABDADA)
			entry.Lock.Lock()
			defer entry.Lock.Unlock()
			if entry.Depth <= depth {
				if entry.Depth == depth {
					if entry.NumProcessors > 0 {
						entry.NumProcessors--
					}
				} else {
					entry.NumProcessors = 0
				}
				entry.EntryType = entryType
				entry.Score = normalizedScore
				if storeBestMove && !sm.Move.Start.Equals(sm.Move.End) {
					entry.BestMove = sm.Move
				}
				entry.Depth = depth
				entry.Generation = gen
			}
		} else {
			entry := transposition_table.TranspositionTableEntryABDADA{
				Depth:         depth,
				EntryType:     entryType,
				Score:         normalizedScore,
				NumProcessors: 0,
				Generation:    gen,
			}
			if storeBestMove && !sm.Move.Start.Equals(sm.Move.End) {
				entry.BestMove = sm.Move
			}
			ab.player.transpositionTable.Store(&h, currentPlayer, &entry)
		}
	}
}

func (ab *ABDADA) ttRead(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta int, exclusiveProbe bool) TTAnswer {
	answer := TTAnswer{
		alpha: alpha,
		beta:  beta,
		score: NegInf,
	}
	if ab.player.TranspositionTableEnabled {
		h := root.Hash()
		if e, ok := ab.player.transpositionTable.Read(&h, currentPlayer); ok {
			entry := e.(*transposition_table.TranspositionTableEntryABDADA)
			entry.Lock.Lock()

			currentGen := atomic.LoadUint32(&ab.player.ttGeneration)
			// Stale entries (written by a ponder run that has since been invalidated)
			// are demoted to move-ordering-only: we keep the best move hint so the
			// search is guided well, but skip score-based cutoffs to avoid acting on
			// results from a subtree that was never actually reached.
			stale := entry.Generation < currentGen

			if entry.Depth == depth && exclusiveProbe && entry.NumProcessors > 0 {
				answer.score = OnEvaluation
			} else if entry.Depth >= depth {
				if !stale {
					if entry.EntryType == transposition_table.TrueScore {
						s := DenormalizeMateScore(entry.Score, int(depth))
						answer.score = s
						hasMove := !entry.BestMove.Start.Equals(entry.BestMove.End)
						if entry.Depth > depth && hasMove {
							// Close the window only when we have a valid move to return.
							// If BestMove is zero (from an aborted search), don't close
							// alpha==beta or the outer ABDADA loop will be skipped and we'd
							// propagate a zero-move to the root.
							answer.alpha = s
							answer.beta = s
						} else if s > answer.alpha {
							answer.alpha = s
						}
					} else if entry.Depth >= depth {
						if entry.EntryType == transposition_table.UpperBound {
							s := DenormalizeMateScore(entry.Score, int(depth))
							if s < beta {
								if s <= alpha {
									answer.score = s
								}
								answer.beta = s
								atomic.AddUint64(&ab.player.Metrics.MovesABImprovedTransposition, 1)
							}
						} else if entry.EntryType == transposition_table.LowerBound {
							s := DenormalizeMateScore(entry.Score, int(depth))
							if s > alpha {
								answer.score = s
								answer.alpha = s
								atomic.AddUint64(&ab.player.Metrics.MovesABImprovedTransposition, 1)
							}
						}
					}
				}
				// Always use the best move for move ordering, even from stale entries.
				answer.bestMove = entry.BestMove
				if !stale && entry.Depth == depth && answer.alpha < answer.beta {
					entry.NumProcessors++
				}
			} else {
				entry.Depth = depth
				entry.EntryType = transposition_table.Unset
				entry.NumProcessors = 1
			}
			entry.Lock.Unlock()
		} else {
			entry := transposition_table.TranspositionTableEntryABDADA{
				Depth:         depth,
				EntryType:     transposition_table.Unset,
				NumProcessors: 1,
			}
			ab.player.transpositionTable.Store(&h, currentPlayer, &entry)
		}
	}
	return answer
}
