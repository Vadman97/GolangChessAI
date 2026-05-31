package ai

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"sync/atomic"
)

func (p *AIPlayer) Quiesce(root *board.Board, alpha, beta int, currentPlayer byte, previousMove *board.LastMove) int {
	// Generate all moves first so terminal detection uses correct previousMove (en passant included).
	moves := root.GetAllMoves(currentPlayer, previousMove)
	if p.terminalNode(root, moves) {
		return AdjustMateScore(p.EvaluateBoard(root, currentPlayer).TotalScore, 0)
	}

	inCheck := root.IsKingInCheck(currentPlayer)

	// When in check we must search all evasions — returning standpat is unsound
	// because any static eval ignores the forced response and creates a horizon
	// effect where the engine doesn't see checkmates starting at the qsearch boundary.
	var standPat int
	if !inCheck {
		standPat = p.EvaluateBoard(root, currentPlayer).TotalScore
		if standPat >= beta {
			return beta
		} else if alpha < standPat {
			alpha = standPat
		}

		// Global delta prune: if standPat + the best possible single capture (queen) still
		// can't raise alpha, skip all captures — the position is too far behind.
		const deltaMargin = 3 * PawnValueWeight // safety buffer for positional swings
		maxCapture := PawnValueWeight * PieceValue[piece.QueenType]
		if standPat+maxCapture+deltaMargin < alpha {
			return alpha
		}
	}

	// When in check: search all legal moves to find an evasion.
	// When not in check: search only captures and promotions.
	var ordered []location.Move
	if inCheck {
		ordered = *moves
	} else {
		// Collect captures and promotions; sort captures by MVV-LVA so the best trades
		// are tried first — this improves alpha-beta pruning in quiescence significantly.
		var captures, promos []location.Move
		for _, m := range *moves {
			isPromotion, _ := m.End.GetPawnPromotion()
			if isPromotion {
				promos = append(promos, m)
			} else if !root.IsEmpty(m.End) || isEnPassantMove(root, m) {
				captures = append(captures, m)
			}
		}
		sortCapturesMVVLVA(captures, root)
		ordered = append(captures, promos...)
	}

	// Examine captures (MVV-LVA sorted) then promotions.
	// Promotions must be included even when the destination is empty: a pawn advancing
	// to the back rank and becoming a queen is an 8-pawn swing that standPat cannot see.
	for _, m := range ordered {
		if p.isAborted() {
			break
		}
		isPromotion, _ := m.End.GetPawnPromotion()
		// SEE-based pruning: skip captures that lose material (SEE < 0) when not in check.
		// This is more accurate than delta pruning: SEE accounts for the full exchange
		// sequence rather than just the immediate captured piece value.
		// Promotions are always searched regardless of SEE (they can't be meaningfully rated).
		if !inCheck && !isPromotion && root.GetPiece(m.End) != nil {
			capturer := root.GetPiece(m.Start)
			var stm byte
			if capturer != nil {
				stm = capturer.GetColor()
			}
			if root.SEE(m, stm) < 0 {
				continue
			}
		}

		child := root.Copy()
		lastMove := board.MakeMove(&m, child)
		atomic.AddUint64(&p.Metrics.MovesConsidered, 1)
		score := -p.Quiesce(child, -beta, -alpha, currentPlayer^1, lastMove)

		if score >= beta {
			atomic.AddUint64(&p.Metrics.MovesPrunedAB, 1)
			return beta
		} else if score > alpha {
			alpha = score
		}
	}
	return alpha
}
