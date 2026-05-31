package ai

import (
	"fmt"
	"log"
	"math"
	"sync/atomic"
	"time"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/transposition_table"
)

type NegaScout struct {
	player             *AIPlayer
	currentSearchDepth int
	killers            [maxKillerDepth][2]location.Move
	history            [board.Height * board.Width][board.Height * board.Width]int32
	countermove        [board.Height * board.Width][board.Height * board.Width]location.Move
}

const negaScoutForwardPruning = false

func (n *NegaScout) GetName() string {
	return AlgorithmNegaScout
}

func (n *NegaScout) resetRootSearchHeuristics() {
	n.killers = [maxKillerDepth][2]location.Move{}
	n.history = [board.Height * board.Width][board.Height * board.Width]int32{}
	n.countermove = [board.Height * board.Width][board.Height * board.Width]location.Move{}
}

func (n *NegaScout) resetVolatileSearchHeuristics() {
	n.killers = [maxKillerDepth][2]location.Move{}
}

func (n *NegaScout) isKiller(m location.Move, ply int) bool {
	k := &n.killers[ply%maxKillerDepth]
	return m.Equals(&k[0]) || m.Equals(&k[1])
}

func (n *NegaScout) storeKiller(ply int, m location.Move) {
	k := &n.killers[ply%maxKillerDepth]
	if m.Equals(&k[0]) {
		return
	}
	k[1] = k[0]
	k[0] = m
}

func (n *NegaScout) updateHistory(m location.Move, depth int) {
	n.history[squareIdx(m.Start)][squareIdx(m.End)] += int32(depth * depth)
}

func (n *NegaScout) historyScore(m location.Move) int32 {
	return n.history[squareIdx(m.Start)][squareIdx(m.End)]
}

func (n *NegaScout) counterMove(prev *board.LastMove, moves *[]location.Move) (location.Move, bool) {
	if prev == nil {
		return location.Move{}, false
	}
	cm := n.countermove[squareIdx(prev.Move.Start)][squareIdx(prev.Move.End)]
	if !cm.Start.Equals(cm.End) && isMoveInList(cm, moves) {
		return cm, true
	}
	return location.Move{}, false
}

func (n *NegaScout) storeCounterMove(prev *board.LastMove, m location.Move) {
	if prev == nil {
		return
	}
	n.countermove[squareIdx(prev.Move.Start)][squareIdx(prev.Move.End)] = m
}

func (n *NegaScout) NegaScout(root *board.Board, depth int, alpha, beta ScoredMove, currentPlayer color.Color, previousMove *board.LastMove) ScoredMove {
	return n.search(root, depth, alpha.Score, beta.Score, currentPlayer, previousMove, true, 0, maxExtensions)
}

func (n *NegaScout) search(root *board.Board, depth, alpha, beta int, currentPlayer color.Color, previousMove *board.LastMove, nullMoveOk bool, ply, extensions int) ScoredMove {
	if ply > 0 && root.CurrentPositionRepeats >= 1 {
		return ScoredMove{Score: StalemateScore}
	}

	inCheck := root.IsKingInCheck(currentPlayer)
	if inCheck && extensions > 0 {
		depth++
		extensions--
	}
	if depth <= 0 {
		return ScoredMove{Score: n.player.Quiesce(root, alpha, beta, currentPlayer, previousMove)}
	}

	movesArr := root.GetAllMoves(currentPlayer, previousMove)
	if n.player.terminalNode(root, movesArr) {
		return ScoredMove{Score: AdjustMateScore(n.player.EvaluateBoard(root, currentPlayer).TotalScore, depth)}
	}
	if len(*movesArr) == 1 && extensions > 0 {
		depth++
		extensions--
	}

	originalAlpha := alpha
	ttAnswer := n.ttRead(root, currentPlayer, uint16(depth), alpha, beta)
	alpha, beta = ttAnswer.alpha, ttAnswer.beta
	if alpha >= beta && !ttAnswer.bestMove.Start.Equals(ttAnswer.bestMove.End) {
		atomic.AddUint64(&n.player.Metrics.MovesPrunedTransposition, uint64(len(*movesArr)))
		return ScoredMove{Move: ttAnswer.bestMove, Score: ttAnswer.score}
	}

	if nullMoveOk && !inCheck && depth >= nullMoveMinDepth && !onlyKingAndPawns(root, currentPlayer) && ply > 0 {
		R := nullMoveR
		if depth >= 7 {
			R = 3
		}
		nullVal := n.search(root, depth-1-R, -beta, -beta+1, currentPlayer^1, nil, false, ply+1, extensions)
		nullScore := -nullVal.Score
		if nullScore >= beta {
			atomic.AddUint64(&n.player.Metrics.MovesPrunedAB, 1)
			return ScoredMove{Score: beta}
		}
	}

	var standPat int
	canFutilityPrune := false
	if negaScoutForwardPruning && !inCheck && depth <= futilityMaxDepth && alpha < WinScore && beta > LossScore {
		standPat = n.player.EvaluateBoard(root, currentPlayer).TotalScore
		canFutilityPrune = true
		if depth == razorDepth && standPat+razorMargin < alpha && ply > 0 {
			qScore := n.player.Quiesce(root, alpha-1, alpha, currentPlayer, previousMove)
			if qScore < alpha {
				atomic.AddUint64(&n.player.Metrics.MovesPrunedAB, uint64(len(*movesArr)))
				return ScoredMove{Score: qScore}
			}
		}
	}

	orderedMoves := n.orderMoves(*movesArr, ttAnswer.bestMove, root, previousMove, ply)
	best := ScoredMove{Score: NegInf}

	for moveIdx, move := range orderedMoves {
		if n.player.isAborted() && moveIdx > 0 && !best.Move.Start.Equals(best.Move.End) {
			return best
		}

		isCapture := root.GetPiece(move.End) != nil || isEnPassantMove(root, move)
		isPromo, _ := move.End.GetPawnPromotion()
		isKiller := n.isKiller(move, ply)
		isTTMove := move.Equals(&ttAnswer.bestMove)
		isNearPromo := false
		if mp := root.GetPiece(move.Start); mp != nil && mp.GetPieceType() == piece.PawnType {
			endRow := int(move.End.GetRow())
			isNearPromo = (currentPlayer == color.White && endRow >= 5) || (currentPlayer == color.Black && endRow <= 2)
		}

		if canFutilityPrune && moveIdx > 0 && !isCapture && !isPromo && !isKiller && !isTTMove && !isNearPromo {
			margin := futilityMargin2
			if depth == 1 {
				margin = futilityMargin1
			}
			if standPat+margin <= alpha {
				atomic.AddUint64(&n.player.Metrics.MovesPrunedAB, 1)
				continue
			}
		}

		child, pm := n.player.applyMove(root, &move)
		doLMR := negaScoutForwardPruning &&
			depth >= lmrMinDepth &&
			moveIdx >= lmrMinMoveIdx &&
			!isCapture && !isPromo && !isKiller && !isTTMove && !inCheck && !isNearPromo &&
			best.Score > NegInf

		reduction := 0
		if doLMR {
			reduction = int(math.Log(float64(depth)) * math.Log(float64(moveIdx+1)) / 2.0)
			if reduction < 1 {
				reduction = 1
			}
			if reduction > depth-2 {
				reduction = depth - 2
			}
		}

		var value ScoredMove
		if moveIdx == 0 {
			value = n.search(child, depth-1, -beta, -alpha, currentPlayer^1, pm, true, ply+1, extensions)
			value.Score = -value.Score
		} else {
			searchDepth := depth - 1 - reduction
			value = n.search(child, searchDepth, -alpha-1, -alpha, currentPlayer^1, pm, true, ply+1, extensions)
			value.Score = -value.Score
			if reduction > 0 && value.Score > alpha {
				value = n.search(child, depth-1, -alpha-1, -alpha, currentPlayer^1, pm, true, ply+1, extensions)
				value.Score = -value.Score
			}
			if value.Score > alpha && value.Score < beta {
				value = n.search(child, depth-1, -beta, -alpha, currentPlayer^1, pm, true, ply+1, extensions)
				value.Score = -value.Score
			}
		}
		value.Move = move

		if value.Score > best.Score || best.Move.Start.Equals(best.Move.End) {
			best = value
		}
		if best.Score > alpha {
			alpha = best.Score
		}
		if alpha >= beta {
			atomic.AddUint64(&n.player.Metrics.MovesPrunedAB, uint64(len(orderedMoves)-moveIdx-1))
			if !isCapture && !isPromo {
				n.storeKiller(ply, move)
				n.updateHistory(move, depth)
				n.storeCounterMove(previousMove, move)
			}
			n.ttWrite(root, currentPlayer, uint16(depth), originalAlpha, beta, &best)
			return best
		}
	}

	n.ttWrite(root, currentPlayer, uint16(depth), originalAlpha, beta, &best)
	return best
}

func (n *NegaScout) getBestMove(b *board.Board, depth, alpha, beta int, previousMove *board.LastMove) ScoredMove {
	originalAlpha := alpha
	movesArr := b.GetAllMoves(n.player.PlayerColor, previousMove)
	if len(*movesArr) == 0 {
		return ScoredMove{Score: n.player.EvaluateBoard(b, n.player.PlayerColor).TotalScore}
	}
	ttAnswer := n.ttRead(b, n.player.PlayerColor, uint16(depth), alpha, beta)
	orderedMoves := n.orderMoves(*movesArr, ttAnswer.bestMove, b, previousMove, 0)

	best := ScoredMove{Score: NegInf}
	for moveIdx, move := range orderedMoves {
		if n.player.isAborted() && moveIdx > 0 && !best.Move.Start.Equals(best.Move.End) {
			break
		}
		child, pm := n.player.applyMove(b, &move)

		var value ScoredMove
		if moveIdx == 0 {
			value = n.search(child, depth-1, -beta, -alpha, n.player.PlayerColor^1, pm, true, 1, maxExtensions)
			value.Score = -value.Score
		} else {
			value = n.search(child, depth-1, -alpha-1, -alpha, n.player.PlayerColor^1, pm, true, 1, maxExtensions)
			value.Score = -value.Score
			if value.Score > alpha && value.Score < beta {
				value = n.search(child, depth-1, -beta, -alpha, n.player.PlayerColor^1, pm, true, 1, maxExtensions)
				value.Score = -value.Score
			}
		}
		value.Move = move

		if value.Score > best.Score || best.Move.Start.Equals(best.Move.End) {
			best = value
		}
		if best.Score > alpha {
			alpha = best.Score
		}
		if alpha >= beta {
			break
		}
	}
	if !best.Move.Start.Equals(best.Move.End) {
		n.ttWrite(b, n.player.PlayerColor, uint16(depth), originalAlpha, beta, &best)
	}
	return best
}

func (n *NegaScout) IterativeNegaScout(b *board.Board, previousMove *board.LastMove) ScoredMove {
	start := time.Now()
	best := ScoredMove{Score: NegInf}
	iterativeIncrement := config.Get().IterativeIncrement

	for n.currentSearchDepth = iterativeIncrement; n.currentSearchDepth <= n.player.MaxSearchDepth; n.currentSearchDepth += iterativeIncrement {
		alpha, beta := NegInf, PosInf
		delta := aspirationDelta
		isMate := best.Score >= WinScore || best.Score <= LossScore
		if n.currentSearchDepth > iterativeIncrement && best.Score != NegInf && !isMate {
			alpha = best.Score - delta
			beta = best.Score + delta
		}

		var newGuess ScoredMove
		for {
			thinking, done := make(chan bool), make(chan bool, 1)
			go n.player.trackThinkTime(thinking, done, start)
			newGuess = n.getBestMove(b, n.currentSearchDepth, alpha, beta, previousMove)
			close(thinking)
			<-done

			if n.player.isAborted() {
				break
			}
			if newGuess.Score <= alpha {
				delta *= aspirationWiden
				alpha = newGuess.Score - delta
				if alpha < NegInf {
					alpha = NegInf
				}
			} else if newGuess.Score >= beta {
				delta *= aspirationWiden
				beta = newGuess.Score + delta
				if beta > PosInf {
					beta = PosInf
				}
			} else {
				break
			}
		}

		if !n.player.isAborted() {
			best = stableDepthMove(best, newGuess)
			n.player.LastSearchDepth = n.currentSearchDepth
			n.player.printer <- fmt.Sprintf("Best D:%d M:%s score:%d\n", n.player.LastSearchDepth, best.Move, best.Score)
		} else {
			if best.Move.Start.Equals(best.Move.End) && !newGuess.Move.Start.Equals(newGuess.Move.End) {
				best = newGuess
			}
			n.player.LastSearchDepth = n.currentSearchDepth - iterativeIncrement
			n.player.printer <- fmt.Sprintf("NegaScout hard abort! evaluated to depth %d\n", n.player.LastSearchDepth)
			break
		}
	}
	if best.Move.Start.Equals(best.Move.End) {
		log.Printf("%s has no best move: %s", n.GetName(), best.Move)
	}
	return best
}

func (n *NegaScout) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	previousPlayer := n.player
	n.player = p
	n.player.setAbort(false)
	if previousPlayer != p {
		n.resetRootSearchHeuristics()
	} else {
		n.resetVolatileSearchHeuristics()
	}

	best := n.IterativeNegaScout(b, previousMove)
	return &best
}

func (n *NegaScout) orderMoves(moves []location.Move, ttMove location.Move, b *board.Board, prevMove *board.LastMove, ply int) []location.Move {
	ordered := make([]location.Move, 0, len(moves))
	var promotions, goodCaptures, badCaptures, killerMoves, counterMoves, quiets []location.Move

	hasTT := !ttMove.Start.Equals(ttMove.End) && isMoveInList(ttMove, &moves)
	ttIsCapture := hasTT && (b.GetPiece(ttMove.End) != nil || isEnPassantMove(b, ttMove))
	ttIsPromotion := false
	if hasTT {
		ttIsPromotion, _ = ttMove.End.GetPawnPromotion()
	}

	counterMove, hasCounter := n.counterMove(prevMove, &moves)

	var stm byte
	for _, m := range moves {
		if p := b.GetPiece(m.Start); p != nil {
			stm = p.GetColor()
			break
		}
	}

	killers := n.killers[ply%maxKillerDepth]
	for _, m := range moves {
		if hasTT && m.Equals(&ttMove) {
			continue
		}
		isPromotion, _ := m.End.GetPawnPromotion()
		if isPromotion {
			promotions = append(promotions, m)
		} else if b.GetPiece(m.End) != nil || isEnPassantMove(b, m) {
			if b.SEE(m, stm) >= 0 {
				goodCaptures = append(goodCaptures, m)
			} else {
				badCaptures = append(badCaptures, m)
			}
		} else if m.Equals(&killers[0]) || m.Equals(&killers[1]) {
			killerMoves = append(killerMoves, m)
		} else if hasCounter && m.Equals(&counterMove) {
			counterMoves = append(counterMoves, m)
		} else {
			quiets = append(quiets, m)
		}
	}

	if len(quiets) > 1 {
		scores := make([]int32, len(quiets))
		for i, m := range quiets {
			scores[i] = n.historyScore(m)
		}
		for i := 1; i < len(quiets); i++ {
			for j := i; j > 0 && scores[j] > scores[j-1]; j-- {
				quiets[j], quiets[j-1] = quiets[j-1], quiets[j]
				scores[j], scores[j-1] = scores[j-1], scores[j]
			}
		}
	}

	sortCapturesMVVLVA(goodCaptures, b)
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

func (n *NegaScout) ttRead(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta int) TTAnswer {
	answer := TTAnswer{alpha: alpha, beta: beta, score: NegInf}
	if !n.player.TranspositionTableEnabled {
		return answer
	}
	h := root.Hash()
	e, ok := n.player.transpositionTable.Read(&h, currentPlayer)
	if !ok {
		return answer
	}
	entry, ok := e.(*transposition_table.TranspositionTableEntryNegaScout)
	if !ok {
		return answer
	}

	currentGen := atomic.LoadUint32(&n.player.ttGeneration)
	stale := entry.Generation < currentGen
	answer.bestMove = entry.BestMove
	if stale || entry.Depth < depth {
		return answer
	}

	score := DenormalizeMateScore(entry.Score, int(depth))
	switch entry.EntryType {
	case transposition_table.TrueScore:
		answer.score = score
		answer.alpha = score
		answer.beta = score
	case transposition_table.UpperBound:
		if score < answer.beta {
			answer.beta = score
			atomic.AddUint64(&n.player.Metrics.MovesABImprovedTransposition, 1)
		}
	case transposition_table.LowerBound:
		if score > answer.alpha {
			answer.score = score
			answer.alpha = score
			atomic.AddUint64(&n.player.Metrics.MovesABImprovedTransposition, 1)
		}
	}
	return answer
}

func (n *NegaScout) ttWrite(root *board.Board, currentPlayer color.Color, depth uint16, alpha, beta int, sm *ScoredMove) {
	if !n.player.TranspositionTableEnabled || n.player.isAborted() {
		return
	}
	if sm.Move.Start.Equals(sm.Move.End) || sm.Score >= PosInf || sm.Score <= NegInf || sm.Score == OnEvaluation || sm.Score == -OnEvaluation {
		return
	}

	entryType := transposition_table.TrueScore
	if sm.Score >= beta {
		entryType = transposition_table.LowerBound
	} else if sm.Score <= alpha {
		entryType = transposition_table.UpperBound
	}

	storeBestMove := entryType != transposition_table.UpperBound
	h := root.Hash()
	if e, ok := n.player.transpositionTable.Read(&h, currentPlayer); ok {
		if entry, ok := e.(*transposition_table.TranspositionTableEntryNegaScout); ok && entry.Depth > depth {
			return
		}
	}

	entry := transposition_table.TranspositionTableEntryNegaScout{
		Score:      NormalizeMateScore(sm.Score, int(depth)),
		Depth:      depth,
		EntryType:  entryType,
		Generation: atomic.LoadUint32(&n.player.ttGeneration),
	}
	if storeBestMove {
		entry.BestMove = sm.Move
	}
	n.player.transpositionTable.Store(&h, currentPlayer, &entry)
}
