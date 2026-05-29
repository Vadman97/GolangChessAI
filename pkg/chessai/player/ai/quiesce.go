package ai

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

// deltaMargin is the centipawn safety buffer for delta pruning — roughly a minor piece.
// Keeps the pruning conservative so we don't incorrectly prune positional swings.
const deltaMargin = 3 * PawnValueWeight

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

		// Futility delta prune: if even capturing the queen can't raise alpha, skip all captures.
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
		if p.abort {
			break
		}
		isPromotion, _ := m.End.GetPawnPromotion()
		// Per-capture delta prune: skip if even winning this piece won't raise alpha.
		// Not applied when in check — we must consider all evasions.
		if !inCheck && !isPromotion {
			capturedPiece := root.GetPiece(m.End)
			if capturedPiece != nil {
				gain := PawnValueWeight * PieceValue[capturedPiece.GetPieceType()]
				if standPat+gain+deltaMargin < alpha {
					continue
				}
			}
		}

		child := root.Copy()
		lastMove := board.MakeMove(&m, child)
		p.Metrics.MovesConsidered++
		score := -p.Quiesce(child, -beta, -alpha, currentPlayer^1, lastMove)

		if score >= beta {
			p.Metrics.MovesPrunedAB++
			return beta
		} else if score > alpha {
			alpha = score
		}
	}
	return alpha
}
