package ai

import (
	"fmt"
	"math"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
	"log"
	"runtime"
	"sync/atomic"
	"time"
)

const (
	maxKillerDepth   = 64
	aspirationDelta    = 50   // initial window half-width in centipawns
	aspirationMaxDelta = 500  // full-window fallback after repeated failures
	aspirationWiden    = 4    // multiply delta by this on each failure
	nullMoveMinDepth = 3   // minimum depth to attempt null move
	nullMoveR        = 2   // null move depth reduction (increases to 3 at depth>=7)
	lmrMinDepth      = 3   // minimum depth before LMR kicks in
	lmrMinMoveIdx    = 3   // LMR applies after this many moves have been searched
	maxExtensions    = 4   // total check-extension budget per branch
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
	kill               bool
	currentSearchDepth int
	NumThreads         int
	// killers[depth%maxKillerDepth][0..1]: last two quiet moves causing beta cutoffs at this depth.
	killers [maxKillerDepth][2]location.Move
	// history[from][to]: accumulated depth^2 bonuses for quiet moves that caused cutoffs.
	// Shared across threads; atomic int32 for lock-free updates.
	history [board.Height * board.Width][board.Height * board.Width]int32
}

func (ab *ABDADA) GetName() string {
	return AlgorithmABDADA
}

func (ab *ABDADA) isKiller(m location.Move, depth int) bool {
	k := &ab.killers[depth%maxKillerDepth]
	return (m.Start.Equals(k[0].Start) && m.End.Equals(k[0].End)) ||
		(m.Start.Equals(k[1].Start) && m.End.Equals(k[1].End))
}

func (ab *ABDADA) storeKiller(depth int, m location.Move) {
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

// ABDADA is the core parallel alpha-beta search function.
// nullMoveOk: false immediately after a null move (prevents consecutive null moves).
// ply: distance from the root (0 at root), used for killer indexing.
// extensions: remaining check-extension budget (starts at maxExtensions at root).
func (ab *ABDADA) ABDADA(root *board.Board, depth, alpha, beta int, exclusiveProbe bool, currentPlayer color.Color, previousMove *board.LastMove, nullMoveOk bool, ply int, extensions int) ScoredMove {
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

	ttAnswer := ab.ttRead(root, currentPlayer, uint16(depth), alpha, beta, exclusiveProbe)
	movesArr := root.GetAllMoves(currentPlayer, previousMove)
	alpha, beta = ttAnswer.alpha, ttAnswer.beta
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
	// Skip when in check, in zugzwang-prone endgames, or after a prior null move.
	if nullMoveOk && !inCheck && depth >= nullMoveMinDepth && !onlyKingAndPawns(root, currentPlayer) {
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

	killerPair := ab.killers[ply%maxKillerDepth]
	orderedMoves := orderMoves(*movesArr, ttAnswer.bestMove, killerPair, ab, root)

	iteration := 0
	allDone := false
	for iteration < 2 && alpha < beta && !allDone {
		// Don't abort before evaluating at least one move: a zero BestMove
		// would propagate to the root and trigger a random fallback.
		if (ab.player.abort || ab.kill) && !best.Move.Start.Equals(best.Move.End) {
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
			if (ab.player.abort || ab.kill) && !firstMove {
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

			// Late Move Reductions: quietly search less-promising moves at reduced depth.
			// Conditions: not a capture, not a promotion, not a killer, not the TT move,
			// not when in check, not a near-promotion pawn advance, only after lmrMinMoveIdx
			// moves already searched.
			doLMR := iteration == 1 &&
				depth >= lmrMinDepth &&
				moveIdx > lmrMinMoveIdx &&
				!isCapture && !isPromo && !isKiller && !isTTMove && !inCheck && !isNearPromo

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
				lmr := ab.ABDADA(child, depth-1-reduction, -(util.MaxScore(alpha, best.Score)+1), -util.MaxScore(alpha, best.Score), false, currentPlayer^1, pm, true, ply+1, extensions)
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
					// Update killer and history for quiet cutoff moves.
					if !isCapture && !isPromo {
						ab.storeKiller(ply, move)
						ab.updateHistory(move, depth)
					}
					ab.syncTTWrite(root, currentPlayer, uint16(depth), alpha, beta, &best)
					return best
				}
				if best.Score > alpha {
					alpha = best.Score
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

func (ab *ABDADA) getBestMove(b *board.Board, depth, alpha, beta int, previousMove *board.LastMove) ScoredMove {
	ab.player.abort = false
	if ab.NumThreads == 0 {
		ab.NumThreads = runtime.NumCPU()
		log.Printf("ABDADA runs in parallel, defaulting to #%d threads (# cpu cores)\n", ab.NumThreads)
	}
	moveChan := make(chan ScoredMove, ab.NumThreads)
	for i := 0; i < ab.NumThreads; i++ {
		rootCopy := b.Copy()
		go func(moveChan chan ScoredMove, root *board.Board) {
			moveChan <- ab.ABDADA(root, depth, alpha, beta, false, ab.player.PlayerColor, previousMove, true, 0, 4)
		}(moveChan, rootCopy)
	}

	var bestMoves []ScoredMove

	const AbortAfterFirst = true
	if AbortAfterFirst {
		bestMoves = append(bestMoves, <-moveChan)
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
	}
	return bestMove
}

func (ab *ABDADA) iterativeABDADA(b *board.Board, previousMove *board.LastMove) ScoredMove {
	start := time.Now()
	best := ScoredMove{Score: NegInf}
	iterativeIncrement := config.Get().IterativeIncrement

	for ab.currentSearchDepth = iterativeIncrement; ab.currentSearchDepth <= ab.player.MaxSearchDepth; ab.currentSearchDepth += iterativeIncrement {
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

			if ab.player.abort {
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

		if !ab.player.abort {
			best = newGuess
			ab.player.LastSearchDepth = ab.currentSearchDepth
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

// orderMoves returns moves ordered for best alpha-beta pruning:
//  1. TT best move (if capture)
//  2. All other captures (MVV-LVA sorted)
//  3. TT best move (if quiet)
//  4. Killer moves
//  5. Remaining quiet moves (sorted by history heuristic score, descending)
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

func orderMoves(moves []location.Move, ttMove location.Move, killers [2]location.Move, ab *ABDADA, b *board.Board) []location.Move {
	ordered := make([]location.Move, 0, len(moves))
	var captures, killerMoves, quiets []location.Move

	// Validate the TT move is actually legal on this board before placing it first.
	// A stale or hash-colliding TT entry can carry over a move that is no longer legal
	// (e.g. wrong piece type at the start square), and evaluating it produces a bogus
	// score that can become the "best" move.
	hasTT := !ttMove.Start.Equals(ttMove.End) && isMoveInList(ttMove, &moves)
	ttIsCapture := hasTT && (b.GetPiece(ttMove.End) != nil || isEnPassantMove(b, ttMove))

	for _, m := range moves {
		if hasTT && m.Start.Equals(ttMove.Start) && m.End.Equals(ttMove.End) {
			continue
		}
		if b.GetPiece(m.End) != nil || isEnPassantMove(b, m) {
			captures = append(captures, m)
		} else {
			// Check killers before adding to quiets.
			isK := (m.Start.Equals(killers[0].Start) && m.End.Equals(killers[0].End)) ||
				(m.Start.Equals(killers[1].Start) && m.End.Equals(killers[1].End))
			if isK {
				killerMoves = append(killerMoves, m)
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
		// Simple insertion sort — quiet lists are usually short.
		for i := 1; i < len(quiets); i++ {
			for j := i; j > 0 && scores[j] > scores[j-1]; j-- {
				quiets[j], quiets[j-1] = quiets[j-1], quiets[j]
				scores[j], scores[j-1] = scores[j-1], scores[j]
			}
		}
	}

	// Sort captures by MVV-LVA (most-valuable-victim × 10 − least-valuable-attacker).
	sortCapturesMVVLVA(captures, b)

	if hasTT && ttIsCapture {
		ordered = append(ordered, ttMove)
	}
	ordered = append(ordered, captures...)
	if hasTT && !ttIsCapture {
		ordered = append(ordered, ttMove)
	}
	ordered = append(ordered, killerMoves...)
	ordered = append(ordered, quiets...)
	return ordered
}

type TTAnswer struct {
	alpha, beta, score int
	bestMove           location.Move
}

func (ab *ABDADA) syncTTWrite(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta int, sm *ScoredMove) {
	if ab.player.TranspositionTableEnabled {
		// PosInf/NegInf are search bounds, not board evaluations.  Writing them
		// to the TT would poison future lookups with fake mate scores (the
		// normalise/denormalise roundtrip converts PosInf into ~1999999999).
		if sm.Score >= PosInf || sm.Score <= NegInf {
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

		if e, ok := ab.player.transpositionTable.Read(&h, currentPlayer); ok {
			entry := e.(*transposition_table.TranspositionTableEntryABDADA)
			if entry.Depth <= depth {
				entry.Lock.Lock()
				if entry.Depth == depth {
					entry.NumProcessors--
				} else {
					entry.NumProcessors = 0
				}
				entry.EntryType = entryType
				entry.Score = normalizedScore
				if !sm.Move.Start.Equals(sm.Move.End) {
					entry.BestMove = sm.Move
				}
				entry.Depth = depth
				entry.Generation = gen
				entry.Lock.Unlock()
			}
		} else {
			entry := transposition_table.TranspositionTableEntryABDADA{
				Depth:         depth,
				EntryType:     entryType,
				Score:         normalizedScore,
				NumProcessors: 0,
				Generation:    gen,
			}
			if !sm.Move.Start.Equals(sm.Move.End) {
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
						if entry.Depth > depth {
							answer.alpha = s
							answer.beta = s
						} else if s > answer.alpha {
							answer.alpha = s
						}
					} else if entry.Depth >= depth {
						if entry.EntryType == transposition_table.UpperBound {
							s := DenormalizeMateScore(entry.Score, int(depth))
							if s < beta {
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
