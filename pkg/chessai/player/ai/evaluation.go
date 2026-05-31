package ai

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/steakknife/hamming"
)

type Evaluation struct {
	// [color][pieceType] -> overall piece count
	PieceCounts map[color.Color]map[byte]uint8
	// [color][pieceType] -> count of pieces off starting position
	PieceAdvanced map[color.Color]map[byte]uint8
	// [color][column] -> num pawns
	PawnColumns map[color.Color]map[location.CoordinateType]uint8
	// [color][column] -> num pawns
	PawnRows map[color.Color]map[location.CoordinateType]uint8
	// [color] -> num moves
	NumMoves   map[color.Color]uint16
	NumAttacks map[color.Color]uint16
	TotalScore int
}

func NewEvaluation() *Evaluation {
	e := Evaluation{
		PieceCounts: map[color.Color]map[byte]uint8{
			color.Black: {},
			color.White: {},
		},
		PieceAdvanced: map[color.Color]map[byte]uint8{
			color.Black: {},
			color.White: {},
		},
		PawnColumns: map[color.Color]map[location.CoordinateType]uint8{
			color.Black: {},
			color.White: {},
		},
		PawnRows: map[color.Color]map[location.CoordinateType]uint8{
			color.Black: {},
			color.White: {},
		},
		NumMoves:   map[color.Color]uint16{},
		NumAttacks: map[color.Color]uint16{},
	}
	return &e
}

var PieceValue = map[byte]int{
	piece.PawnType:   1,
	piece.BishopType: 3,
	piece.KnightType: 3,
	piece.RookType:   5,
	piece.QueenType:  9,
	piece.KingType:   100,
}

const (
	PawnValueWeight       = 100
	PawnStructureWeight   = PawnValueWeight / 5
	PieceAdvanceWeight    = PawnValueWeight / 5
	PieceNumMovesWeight   = PawnValueWeight / 20
	PieceNumAttacksWeight = PawnValueWeight / 10
	KingDisplacedWeight   = -1 * PawnValueWeight
	// Small penalty for moving a rook before castling — not large enough to force premature castling.
	RookDisplacedWeight = -PawnValueWeight / 5
	KingCheckedWeight   = -PawnValueWeight / 4
	// KingCastledWeight: +60 cp for castling. Combined with the ~70 cp PST swing (e1→g1), this
	// makes castling worth ~130 cp total — enough to outweigh a simple pawn grab (+100 cp) so the
	// engine doesn't skip castling to capture material in the opening.
	KingCastledWeight = PawnValueWeight * 3 / 5
	// neg 1 pawn if we do nothing in 50 moves
	Weight50Rule = -PawnValueWeight / PawnValueWeight
	// King safety: penalize exposed king files and enemy sliders on them.
	// Prior run doubled these (-20→-40, -30→-60) but that caused 100-150 cp
	// over-penalty for a normally-castled king with any slider on adjacent files,
	// turning winning positions into losing ones in the engine's eval and leading
	// to desperate moves. Restored to validated values.
	KingOpenFilePenalty    = -20
	KingEnemySliderPenalty = -30
)

const (
	PawnDuplicateWeight = -1
	PawnAdvancedWeight  = 1
	// IsolatedPawnPenalty: a pawn with no friendly pawns on adjacent files is weak.
	IsolatedPawnPenalty = -15
	// BackwardPawnPenalty: a pawn that cannot be supported by a friendly pawn from behind
	// (no friendly pawn on adjacent files BEHIND this pawn). Backward pawns on semi-open
	// files are especially weak since the opponent can attack them directly with rooks.
	BackwardPawnPenalty     = -8
	BackwardOnSemiOpenBonus = -8 // extra penalty if the file in front has no friendly pawns
	// TempoBonus: small bonus for the side to move, reflecting the value of initiative.
	// Applied at every leaf node; improves handling of zugzwang and forcing sequences.
	TempoBonus = 10
	// BishopPairBonus: the bishop pair is worth ~half a pawn even in the middlegame
	// and more as the position opens up. A BASE bonus always applies when a side
	// keeps both bishops; BishopPairOpenBonus adds extra in open positions (few pawns).
	// Previously the entire bonus was tapered to ~0 with a full board of pawns
	// (50*(16-14)/16 ≈ 6cp at 14 pawns), so the engine surrendered the pair almost
	// for free in the opening (e.g. an early Bxc6 trading bishop for knight). The
	// base term ensures giving up the pair costs a meaningful amount even early.
	BishopPairBonus     = 30 // base, applies whenever a side has both bishops
	BishopPairOpenBonus = 25 // extra, scaled by how open the position is (fewer pawns)
	// Rook file activity bonuses.
	RookOpenFileBonus     = 15 // no pawns of either color on the file
	RookSemiOpenFileBonus = 8  // no friendly pawns but enemy pawns present
	// Rook on the 7th rank (rank 7 for White = row 6, rank 2 for Black = row 1).
	// A rook penetrating to the 7th attacks the enemy pawn chain and restricts the king.
	// Tapered by endgame phase — most valuable in middlegame with queens still on board.
	RookOnSeventhBonus = 20
	// Rooks are the best blockaders of dangerous passers and belong behind passed
	// pawns in rook endgames. These bonuses are scaled by the pawn's advancement.
	RookPasserBlockadeBonus = 150
	RookBehindPasserBonus   = 30
	// KnightOutpostBonus: knight on ranks 3-5 (from own back rank) that no enemy
	// pawn can attack. Outpost knights dominate the middlegame.
	KnightOutpostBonus = 20
	// KnightPasserBlockadeBonus: per passed pawn whose next-advance square is
	// attacked by a friendly knight. Scaled by pawn rank (rank/6 × base value).
	// Prevents the engine from undervaluing defensive knight maneuvers that
	// simultaneously blockade multiple advanced passed pawns (K+PP vs K+N endings).
	// Reduced from 150 (over-valued, caused tactical misses in exchange calculations).
	KnightPasserBlockadeBonus = 60

	// KingAttackZoneWeight: penalty per enemy-attacked square in the king ring (3×3
	// around the king). Applied quadratically past 2 squares. Reduced from 12 to 6
	// to prevent stacking too much on top of KingEnemySliderPenalty — the combined
	// penalty at 3 attacked squares was 90+ cp, turning winning positions into losing
	// ones in the eval. At 6: 3 squares = 18 cp, 4 squares = 36 cp, still significant
	// for actual mating attacks.
	KingAttackZoneWeight = 6

	// ConnectedPasserBonus: extra bonus when two passed pawns are on adjacent files.
	// Connected passers are drastically harder to stop than isolated passers: one
	// supports the other as they advance together. Applied per pawn in a connected pair.
	ConnectedPasserBonus = 30
)

// knightMoveDeltas lists all 8 relative (row, col) offsets for knight moves.
var knightMoveDeltas = [8][2]int{
	{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
	{2, -1}, {2, 1}, {1, -2}, {1, 2},
}

// isKnightOutpost returns true if the square can never be attacked by an enemy
// pawn — i.e. there is no enemy pawn on either adjacent file that sits "in front"
// of the square and could advance to challenge the knight. This is the standard
// outpost definition. The previous version only checked the single square an enemy
// pawn would attack FROM right now, so it wrongly treated easily-kicked advanced
// knights as outposts (e.g. a knight on e4 with the enemy f-pawn still on f7, which
// simply plays ...f5 to chase it). Such phantom outpost bonuses encouraged the
// engine to make premature, unstable knight sorties.
func isKnightOutpost(b *board.Board, row, col location.CoordinateType, knightColor color.Color) bool {
	enemy := knightColor ^ 1
	// An enemy pawn on an adjacent file can attack the square only if it is "ahead"
	// of the knight from the knight owner's perspective (toward the enemy back rank),
	// since pawns advance toward us. For a White knight, enemy (Black) pawns come from
	// higher rows, so any Black pawn on an adjacent file at a row > this row can
	// eventually reach an attacking square. For a Black knight, the mirror holds.
	for _, dc := range []int{-1, 1} {
		ac := int(col) + dc
		if ac < 0 || ac >= board.Width {
			continue
		}
		var rStart, rEnd, rStep int
		if knightColor == color.White {
			// Black pawns at rows above the knight can advance down to attack it.
			rStart, rEnd, rStep = int(row)+1, board.Height, 1
		} else {
			// White pawns at rows below the knight can advance up to attack it.
			rStart, rEnd, rStep = int(row)-1, -1, -1
		}
		for r := rStart; r != rEnd; r += rStep {
			p := b.GetPiece(location.NewLocation(location.CoordinateType(r), location.CoordinateType(ac)))
			if p != nil && p.GetColor() == enemy && p.GetPieceType() == piece.PawnType {
				return false
			}
		}
	}
	return true
}

// passedPawnBonus is added to the score for a passed pawn at the given rank
// (distance from own back rank, 0 = back rank, 7 = promotion). Additive on top
// of pawnPST — passed pawns are far more dangerous than normal advanced pawns.
// Ranks 5-6 are raised from 50/75 now that LMR correctly skips near-promotion
// pawn advances; the old "inflation" was caused by LMR under-searching the
// opponent's promotion threat, not by the eval values themselves.
var passedPawnBonus = [8]int{0, 0, 0, 5, 20, 80, 130, 0}

// backwardPawnPenalty returns a negative score if the pawn at (row, col) of color c
// is backward — it has advanced from its starting square but has no friendly pawn on
// either adjacent file "behind" it (closer to c's own back rank). Unadvanaced pawns
// (on their starting row) are not penalized. An extra penalty applies when the file
// ahead has no friendly pawns (semi-open file: rook/queen can target it).
func backwardPawnPenalty(b *board.Board, row, col location.CoordinateType, c color.Color) int {
	// Only penalize advanced pawns — starting-position pawns trivially have no
	// "behind" supporters and should not be flagged.
	startRow := board.StartRow[c]["Pawn"]
	if row == startRow {
		return 0
	}

	hasSupport := false
	for _, dc := range [2]int{-1, 1} {
		adjCol := int(col) + dc
		if adjCol < 0 || adjCol >= board.Width {
			continue
		}
		// For White (own back rank = row 0): "behind" = row' < row.
		// For Black (own back rank = row 7): "behind" = row' > row.
		if c == color.White {
			for r := location.CoordinateType(0); r < row; r++ {
				p := b.GetPiece(location.NewLocation(r, location.CoordinateType(adjCol)))
				if p != nil && p.GetColor() == c && p.GetPieceType() == piece.PawnType {
					hasSupport = true
					break
				}
			}
		} else {
			for r := location.CoordinateType(7); r > row; r-- {
				p := b.GetPiece(location.NewLocation(r, location.CoordinateType(adjCol)))
				if p != nil && p.GetColor() == c && p.GetPieceType() == piece.PawnType {
					hasSupport = true
					break
				}
			}
		}
		if hasSupport {
			break
		}
	}
	if hasSupport {
		return 0
	}
	penalty := BackwardPawnPenalty
	// Extra penalty if the file ahead has no friendly pawns (semi-open = rook target).
	hasFriendlyAhead := false
	if c == color.White {
		for r := row + 1; int(r) < board.Height; r++ {
			p := b.GetPiece(location.NewLocation(r, col))
			if p != nil && p.GetColor() == c && p.GetPieceType() == piece.PawnType {
				hasFriendlyAhead = true
				break
			}
		}
	} else {
		if row > 0 {
			for r := row - 1; ; r-- {
				p := b.GetPiece(location.NewLocation(r, col))
				if p != nil && p.GetColor() == c && p.GetPieceType() == piece.PawnType {
					hasFriendlyAhead = true
					break
				}
				if r == 0 {
					break
				}
			}
		}
	}
	if !hasFriendlyAhead {
		penalty += BackwardOnSemiOpenBonus
	}
	return penalty
}

// isPassedPawn returns true if no enemy pawn can block or capture this pawn
// on the way to promotion — i.e. no enemy pawn on the same file or adjacent
// files ahead of this pawn.
func isPassedPawn(b *board.Board, row, col location.CoordinateType, pawnColor color.Color) bool {
	enemy := pawnColor ^ 1
	var rStart, rEnd, rStep int
	if pawnColor == color.White {
		rStart, rEnd, rStep = int(row)+1, board.Height, 1
	} else {
		rStart, rEnd, rStep = int(row)-1, -1, -1
	}
	for dc := -1; dc <= 1; dc++ {
		c := int(col) + dc
		if c < 0 || c >= board.Width {
			continue
		}
		for r := rStart; r != rEnd; r += rStep {
			p := b.GetPiece(location.NewLocation(location.CoordinateType(r), location.CoordinateType(c)))
			if p != nil && p.GetColor() == enemy && p.GetPieceType() == piece.PawnType {
				return false
			}
		}
	}
	return true
}

func rookPasserActivityBonus(b *board.Board, rookRow, rookCol location.CoordinateType, rookColor color.Color) int {
	bonus := 0
	for row := location.CoordinateType(0); row < board.Height; row++ {
		p := b.GetPiece(location.NewLocation(row, rookCol))
		if p == nil || p.GetPieceType() != piece.PawnType || !isPassedPawn(b, row, rookCol, p.GetColor()) {
			continue
		}
		pawnColor := p.GetColor()
		rank := int(row)
		if pawnColor == color.Black {
			rank = 7 - int(row)
		}
		if rank < 3 {
			continue
		}

		if pawnColor == rookColor {
			if (pawnColor == color.White && rookRow < row) || (pawnColor == color.Black && rookRow > row) {
				bonus += RookBehindPasserBonus * rank / 6
			}
			continue
		}

		blocksEnemyPasser := (pawnColor == color.White && rookRow > row) || (pawnColor == color.Black && rookRow < row)
		if blocksEnemyPasser {
			blockade := RookPasserBlockadeBonus * rank / 6
			if abs(int(rookRow)-int(row)) == 1 {
				blockade += RookPasserBlockadeBonus / 4
			}
			bonus += blockade
		}
	}
	return bonus
}

const (
	// MopupThreshold: lowered from 5 (one rook) to 3 so the mop-up activates slightly
	// earlier without distorting evals in materially balanced positions.
	MopupThreshold = 3
	// MopupWeight: 8 is a mild increase from the original 5 — enough to push the winning
	// king towards the enemy without dominating the eval in near-equal positions.
	MopupWeight = PawnValueWeight * 2 / 25

	// KingPassedPawnSupportWeight: in endgames, reward the winning king for being
	// close to its own passed pawns. The escort king is essential for converting
	// passed pawns against a defending king. Scaled by (256-phase)/256 so it
	// fades out completely in the middlegame.
	KingPassedPawnSupportWeight = 4
)

// pstScale converts raw PST centipawn values to the internal score scale.
// PawnValueWeight == 100, so raw cp values from standard tables map 1:1.
const pstScale = 1

// Piece-square tables indexed [rank_from_own_backrank 0..7][file_a_to_h 0..7].
// Values are in centipawns; positive = good for the owning side.
// knightPST: edge files (a/h) are always -50 so any edge move from a starting square
// (-40 at g1/b1) is a strict penalty rather than an apparent improvement.
var knightPST = [8][8]int{
	{-50, -40, -30, -30, -30, -30, -40, -50},
	{-50, -20, 0, 5, 5, 0, -20, -50},
	{-50, 5, 10, 15, 15, 10, 5, -50},
	{-50, 0, 15, 20, 20, 15, 0, -50},
	{-50, 5, 15, 20, 20, 15, 5, -50},
	{-50, 0, 10, 15, 15, 10, 0, -50},
	{-50, -20, 0, 0, 0, 0, -20, -50},
	{-50, -40, -30, -30, -30, -30, -40, -50},
}

var bishopPST = [8][8]int{
	{-20, -10, -10, -10, -10, -10, -10, -20},
	{-10, 0, 0, 0, 0, 0, 0, -10},
	{-10, 0, 5, 10, 10, 5, 0, -10},
	{-10, 5, 5, 10, 10, 5, 5, -10},
	{-10, 0, 10, 10, 10, 10, 0, -10},
	{-10, 10, 10, 10, 10, 10, 10, -10},
	{-10, 5, 0, 0, 0, 0, 5, -10},
	{-20, -10, -10, -10, -10, -10, -10, -20},
}

var rookPST = [8][8]int{
	{0, 0, 0, 0, 0, 0, 0, 0},
	{5, 10, 10, 10, 10, 10, 10, 5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{-5, 0, 0, 0, 0, 0, 0, -5},
	{0, 0, 0, 5, 5, 0, 0, 0},
}

var queenPST = [8][8]int{
	{-20, -10, -10, -5, -5, -10, -10, -20},
	{-10, 0, 0, 0, 0, 0, 0, -10},
	{-10, 0, 5, 5, 5, 5, 0, -10},
	{-5, 0, 5, 5, 5, 5, 0, -5},
	{0, 0, 5, 5, 5, 5, 0, -5},
	{-10, 5, 5, 5, 5, 5, 0, -10},
	{-10, 0, 5, 0, 0, 0, 0, -10},
	{-20, -10, -10, -5, -5, -10, -10, -20},
}

// pawnPST rewards central pawn placement and advanced passers.
// Ranks 0-2 are 0: starting square and one-step advances get no PST bonus so
// existing evaluation baselines are not disturbed. Bonuses start at rank 3
// (d4/e4 for White or d5/e5 for Black) where center occupancy is meaningful.
// d4/e4 at rank 3 are boosted to 50 cp so the engine prefers occupying the
// center over developing knights first — the knight PST jump from the starting
// square to f3/c3 is +50 cp, so the pawn center must match that value.
var pawnPST = [8][8]int{
	{0, 0, 0, 0, 0, 0, 0, 0},         // rank 0 — pawns never reach here (promoted)
	{0, 0, 0, 0, 0, 0, 0, 0},         // rank 1 — starting squares
	{0, 0, 0, 0, 0, 0, 0, 0},         // rank 2 — one step advanced
	{2, 2, 12, 50, 50, 12, 2, 2},     // rank 3 — e4/d4: dominant center control
	{5, 5, 15, 35, 35, 15, 5, 5},     // rank 4 — passed pawn territory
	{10, 10, 20, 30, 30, 20, 10, 10}, // rank 5 — advanced passers
	{50, 50, 50, 50, 50, 50, 50, 50}, // rank 6 — pre-promotion
	{0, 0, 0, 0, 0, 0, 0, 0},         // rank 7 — promoted (not a pawn)
}

var kingMiddlegamePST = [8][8]int{
	{-30, -40, -40, -50, -50, -40, -40, -30},
	{-30, -40, -40, -50, -50, -40, -40, -30},
	{-30, -40, -40, -50, -50, -40, -40, -30},
	{-30, -40, -40, -50, -50, -40, -40, -30},
	{-20, -30, -30, -40, -40, -30, -30, -20},
	{-10, -20, -20, -20, -20, -20, -20, -10},
	{20, 20, 0, 0, 0, 0, 20, 20},
	{20, 30, 10, 0, 0, 10, 30, 20},
}

// kingEndgamePST rewards central king placement in endgames.
var kingEndgamePST = [8][8]int{
	{-50, -40, -30, -20, -20, -30, -40, -50},
	{-30, -20, -10, 0, 0, -10, -20, -30},
	{-30, -10, 20, 30, 30, 20, -10, -30},
	{-30, -10, 30, 40, 40, 30, -10, -30},
	{-30, -10, 30, 40, 40, 30, -10, -30},
	{-30, -10, 20, 30, 30, 20, -10, -30},
	{-30, -30, 0, 0, 0, 0, -30, -30},
	{-50, -30, -30, -30, -30, -30, -30, -50},
}

// endgamePhase returns a value 0–256 where 256 = full middlegame, 0 = full endgame.
// Computed from total non-pawn, non-king material on the board.
func endgamePhase(pieceCounts map[color.Color]map[byte]uint8) int {
	const maxPhase = 2*4 + 2*4 + 2*2 // 2 rooks + 2 minors + 1 queen per side * 2 sides
	phase := 0
	for _, c := range []color.Color{color.White, color.Black} {
		phase += int(pieceCounts[c][piece.QueenType]) * 4
		phase += int(pieceCounts[c][piece.RookType]) * 2
		phase += int(pieceCounts[c][piece.BishopType])
		phase += int(pieceCounts[c][piece.KnightType])
	}
	if phase > maxPhase {
		phase = maxPhase
	}
	return (phase * 256) / maxPhase
}

// pstBonus returns the PST bonus for a piece at engine coordinates (row, col).
// row 0 = White's back rank; col 0 = h-file, col 7 = a-file.
// phase 256 = full middlegame, 0 = full endgame (used only for king).
func pstBonus(pieceType byte, pieceColor color.Color, row, col location.CoordinateType, phase int) int {
	rank := int(row)
	if pieceColor == color.Black {
		rank = 7 - int(row)
	}
	file := 7 - int(col) // convert h=0 → h=7 to a=0 → h=7
	switch pieceType {
	case piece.PawnType:
		return pawnPST[rank][file] * pstScale
	case piece.KnightType:
		return knightPST[rank][file] * pstScale
	case piece.BishopType:
		return bishopPST[rank][file] * pstScale
	case piece.RookType:
		return rookPST[rank][file] * pstScale
	case piece.QueenType:
		return queenPST[rank][file] * pstScale
	case piece.KingType:
		// Taper between middlegame and endgame king tables based on remaining material.
		mg := kingMiddlegamePST[rank][file]
		eg := kingEndgamePST[rank][file]
		return ((mg*phase + eg*(256-phase)) / 256) * pstScale
	}
	return 0
}

const (
	// WinScore/LossScore are the base checkmate values before depth adjustment.
	// Mate scores are WinScore ± depth, so they must stay below PosInf/NegInf.
	WinScore       = 1_000_000_000
	LossScore      = -WinScore
	StalemateScore = 0 // draw is neutral: better than losing, worse than winning
)

// AdjustMateScore encodes depth into win/loss scores so the search prefers
// shorter paths to checkmate. More remaining depth = fewer moves from root.
func AdjustMateScore(score, depth int) int {
	if score >= WinScore {
		return WinScore + depth
	} else if score <= LossScore {
		return LossScore - depth
	}
	return score
}

// NormalizeMateScore removes the depth component before storing a score in the
// transposition table so that the distance-to-mate is relative to the stored
// position rather than the root. Pair with DenormalizeMateScore on retrieval.
// PosInf/NegInf are search sentinels, not board evaluations — skip normalization
// to prevent them from corrupting TT entries as fake mate scores.
func NormalizeMateScore(score, depth int) int {
	if score >= WinScore && score < PosInf {
		return score - depth
	} else if score <= LossScore && score > NegInf {
		return score + depth
	}
	return score
}

// DenormalizeMateScore re-applies the depth component after reading a score
// from the transposition table at a (possibly different) depth.
func DenormalizeMateScore(score, depth int) int {
	if score >= WinScore && score < PosInf {
		return score + depth
	} else if score <= LossScore && score > NegInf {
		return score - depth
	}
	return score
}

type evaluationPair struct {
	score    int
	whoMoves color.Color
}

// TODO(Vadim) make this a static function so evaluation cache is global
func (p *AIPlayer) EvaluateBoard(b *board.Board, whoMoves color.Color) *Evaluation {
	eval := NewEvaluation()
	// first see if we have calculations we cannot cache
	if b.MovesSinceNoDraw >= 100 {
		// Vadim: >= instead of == because AI simulation will go beyond 100, it will know no win is possible
		// Alex: This value may change, but AI right now prevents draws
		eval.TotalScore = StalemateScore
	} else if b.PreviousPositionsSeen >= 3 {
		eval.TotalScore = StalemateScore
	} else {
		eval = p.evaluateBoardCached(b, whoMoves)
		eval.TotalScore += Weight50Rule * b.MovesSinceNoDraw
	}
	return eval
}

/**
 * Symmetric heuristic evaluation, relative to whoMoves color
 * https://www.chessprogramming.org/Evaluation#Side_to_move_relative
 */
func (p *AIPlayer) evaluateBoardCached(b *board.Board, whoMoves color.Color) *Evaluation {
	hash := b.Hash()
	var eval *Evaluation
	if p.evaluationMap != nil {
		if value, ok := p.evaluationMap.Read(&hash, 0); ok {
			entry := value.(*evaluationPair)
			score := entry.score
			// store evaluation only once, flip perspective if needed
			if whoMoves != entry.whoMoves {
				score = -score
			}
			// Tempo bonus applied after cache flip — must NOT be stored in cache
			// since it's asymmetric (always positive for the side to move).
			// Skip for terminal positions: adding to WinScore/LossScore breaks
			// the mate-score threshold comparisons in AdjustMateScore.
			if score > LossScore && score < WinScore {
				score += TempoBonus
			}
			return &Evaluation{
				TotalScore: score,
			}
		}
	}
	eval = EvaluateBoardNoCache(b, whoMoves)

	if p.evaluationMap != nil {
		p.evaluationMap.Store(&hash, 0, &evaluationPair{
			score:    eval.TotalScore, // stored WITHOUT tempo bonus so flipping works
			whoMoves: whoMoves,
		})
	}
	// Tempo bonus applied after storage for same reason: non-terminal positions only.
	if eval.TotalScore > LossScore && eval.TotalScore < WinScore {
		eval.TotalScore += TempoBonus
	}
	return eval
}

// kingSafety penalizes a king with open files in front (no friendly pawn within
// two squares forward) and adds an extra penalty when an enemy rook or queen
// sits on that file. Row 0 is White's back rank; col 0 is the h-file.
func kingSafety(b *board.Board, kingColor color.Color) int {
	kingLoc := b.KingLocations[kingColor]
	kingRow := int(kingLoc.GetRow())
	kingCol := int(kingLoc.GetCol())

	enemy := kingColor ^ 1
	score := 0

	// forward = direction toward the enemy back rank
	forward := 1
	if kingColor == color.Black {
		forward = -1
	}

	for dc := -1; dc <= 1; dc++ {
		col := kingCol + dc
		if col < 0 || col >= board.Width {
			continue
		}
		// look for a friendly pawn within two squares in front of the king
		hasPawn := false
		for dist := 1; dist <= 2; dist++ {
			r := kingRow + forward*dist
			if r < 0 || r >= board.Height {
				break
			}
			p := b.GetPiece(location.NewLocation(location.CoordinateType(r), location.CoordinateType(col)))
			if p != nil && p.GetColor() == kingColor && p.GetPieceType() == piece.PawnType {
				hasPawn = true
				break
			}
		}
		if !hasPawn {
			score += KingOpenFilePenalty
			// additional penalty if an enemy slider threatens down this file
			for row := 0; row < board.Height; row++ {
				p := b.GetPiece(location.NewLocation(location.CoordinateType(row), location.CoordinateType(col)))
				if p != nil && p.GetColor() == enemy {
					pt := p.GetPieceType()
					if pt == piece.RookType || pt == piece.QueenType {
						score += KingEnemySliderPenalty
						break
					}
				}
			}
		}
	}
	return score
}

func EvaluateBoardNoCache(b *board.Board, whoMoves color.Color) *Evaluation {
	eval := NewEvaluation()
	// technically ignores en passant, but that should be ok
	if b.IsInCheckmate(whoMoves^1, nil) {
		eval.TotalScore = WinScore
	} else if b.IsInCheckmate(whoMoves, nil) {
		eval.TotalScore = LossScore
	} else if b.IsStalemate(whoMoves, nil) || b.IsInsufficientMaterial() {
		// Stalemate: the side to move has no legal moves and is not in check.
		// Only check the side to move — checking whoMoves^1 (the side NOT to move)
		// would misidentify positions where the opponent is boxed in (but can still
		// be checkmated on the next move) as draws instead of wins.
		eval.TotalScore = StalemateScore
	} else {
		// First pass: count pieces so endgamePhase can be computed before PST scoring.
		for row := location.CoordinateType(0); row < board.Height; row++ {
			for col := location.CoordinateType(0); col < board.Width; col++ {
				if gamePiece := b.GetPiece(location.NewLocation(row, col)); gamePiece != nil {
					eval.PieceCounts[gamePiece.GetColor()][gamePiece.GetPieceType()]++
				}
			}
		}
		phase := endgamePhase(eval.PieceCounts)

		pstScores := [color.NumColors]int{}
		// combinedAttacks accumulates the pseudo-legal attack BitBoards for each color.
		// Used for the king attack zone penalty without an extra board scan.
		var combinedAttacks [color.NumColors]board.BitBoard
		// passedPawnCols tracks which columns have a passed pawn for each color.
		// Used to detect connected passed pawns (adjacent-file passers).
		var passedPawnCols [color.NumColors][board.Width]bool
		for row := location.CoordinateType(0); row < board.Height; row++ {
			for col := location.CoordinateType(0); col < board.Width; col++ {
				if gamePiece := b.GetPiece(location.NewLocation(row, col)); gamePiece != nil {
					c := gamePiece.GetColor()
					pt := gamePiece.GetPieceType()

					// Use pseudo-legal (attackable) squares for mobility — avoids willMoveLeaveKingInCheck
					// per candidate, which would copy the board for every move of every piece.
					// Pseudo-legal mobility is a standard eval heuristic and allows searching much deeper.
					attackableMoves := gamePiece.GetAttackableMoves(b)
					numPseudoLegal := int(hamming.CountBitsUint64(uint64(attackableMoves)))
					eval.NumMoves[c] += uint16(numPseudoLegal)
					combinedAttacks[c] = combinedAttacks[c].CombineBitBoards(attackableMoves)

					pstScores[c] += pstBonus(pt, c, row, col, phase)

					if pt == piece.PawnType {
						eval.PawnColumns[c][col]++
						eval.PawnRows[c][row]++
						if row != board.StartRow[c]["Pawn"] {
							eval.PieceAdvanced[c][pt]++
						}
						// Passed pawn bonus: awarded per pawn, based on rank from own back rank.
						if isPassedPawn(b, row, col, c) {
							rank := int(row)
							if c == color.Black {
								rank = 7 - int(row)
							}
							pstScores[c] += passedPawnBonus[rank]
							passedPawnCols[c][col] = true
						}
						// Backward pawn: no friendly pawn on adjacent files that is BEHIND this one.
						// "Behind" = closer to own back rank.
						pstScores[c] += backwardPawnPenalty(b, row, col, c)
					} else if pt == piece.KnightType {
						if row != board.StartRow[c]["Piece"] {
							eval.PieceAdvanced[c][pt]++
						}
						rank := int(row)
						if c == color.Black {
							rank = 7 - int(row)
						}
						if rank >= 3 && rank <= 5 && isKnightOutpost(b, row, col, c) {
							pstScores[c] += KnightOutpostBonus
						}
					} else if pt != piece.KingType {
						if row != board.StartRow[c]["Piece"] {
							eval.PieceAdvanced[c][pt]++
						}
					}
				}
			}
		}
		// Rook open/semi-open file bonus: second pass after PawnColumns is complete.
		for row := location.CoordinateType(0); row < board.Height; row++ {
			for col := location.CoordinateType(0); col < board.Width; col++ {
				p := b.GetPiece(location.NewLocation(row, col))
				if p == nil || p.GetPieceType() != piece.RookType {
					continue
				}
				c := p.GetColor()
				friendlyPawns := eval.PawnColumns[c][col] > 0
				enemyPawns := eval.PawnColumns[c^1][col] > 0
				if !friendlyPawns && !enemyPawns {
					pstScores[c] += RookOpenFileBonus
				} else if !friendlyPawns {
					pstScores[c] += RookSemiOpenFileBonus
				}
				// Rook on 7th rank: penetration into the enemy's pawn zone.
				// White's 7th = row 6 (rank 7); Black's 7th = row 1 (rank 2).
				seventhRank := location.CoordinateType(6)
				if c == color.Black {
					seventhRank = 1
				}
				if row == seventhRank {
					// Taper: full bonus in middlegame, half in endgame.
					pstScores[c] += RookOnSeventhBonus * phase / 256
				}
				pstScores[c] += rookPasserActivityBonus(b, row, col, c)
			}
		}

		// Knight passer blockade: bonus for a knight that attacks the square
		// directly in front of an enemy passed pawn. Discourages the engine from
		// ignoring defensive knight maneuvers in K+PP vs K+N-type endings.
		for row := location.CoordinateType(0); row < board.Height; row++ {
			for col := location.CoordinateType(0); col < board.Width; col++ {
				p := b.GetPiece(location.NewLocation(row, col))
				if p == nil || p.GetPieceType() != piece.KnightType {
					continue
				}
				c := p.GetColor()
				enemy := c ^ 1
				// enemyForward: direction enemy pawns advance (+1 for White, -1 for Black).
				enemyForward := 1
				if enemy == color.Black {
					enemyForward = -1
				}
				for _, d := range knightMoveDeltas {
					tr := int(row) + d[0]
					tc := int(col) + d[1]
					if tr < 0 || tr >= board.Height || tc < 0 || tc >= board.Width {
						continue
					}
					// The enemy pawn that would be blocked sits one step behind targetRow.
					pr := tr - enemyForward
					if pr < 0 || pr >= board.Height {
						continue
					}
					ep := b.GetPiece(location.NewLocation(location.CoordinateType(pr), location.CoordinateType(tc)))
					if ep == nil || ep.GetColor() != enemy || ep.GetPieceType() != piece.PawnType {
						continue
					}
					pawnRow := location.CoordinateType(pr)
					pawnCol := location.CoordinateType(tc)
					if !isPassedPawn(b, pawnRow, pawnCol, enemy) {
						continue
					}
					// Rank of the enemy pawn from its own back rank (0=start, 6=one step from promo).
					rank := int(pawnRow)
					if enemy == color.Black {
						rank = 7 - int(pawnRow)
					}
					if rank >= 3 {
						pstScores[c] += KnightPasserBlockadeBonus * rank / 6
					}
				}
			}
		}

		// Connected passed pawn bonus: apply after all pieces have been scanned
		// so passedPawnCols is fully populated.
		for pColor := byte(0); pColor < color.NumColors; pColor++ {
			for col := 1; col < board.Width; col++ {
				if passedPawnCols[pColor][col] && passedPawnCols[pColor][col-1] {
					// Both col and col-1 have a passed pawn — they're connected.
					pstScores[pColor] += ConnectedPasserBonus * 2 // once per pawn
				}
			}
		}

		for pColor := byte(0); pColor < color.NumColors; pColor++ {
			score := 0
			for pieceType, value := range PieceValue {
				score += PawnValueWeight * value * int(eval.PieceCounts[pColor][pieceType])
				score += PieceAdvanceWeight * int(eval.PieceAdvanced[pColor][pieceType])
			}
			score += pstScores[pColor]
			if b.GetFlag(board.FlagCastled, pColor) {
				score += KingCastledWeight
			} else {
				// has not castled
				if b.GetFlag(board.FlagKingMoved, pColor) {
					score += KingDisplacedWeight
				}
				if b.GetFlag(board.FlagLeftRookMoved, pColor) || b.GetFlag(board.FlagRightRookMoved, pColor) {
					score += RookDisplacedWeight
				}
				// King-in-center urgency: as more pieces are developed, the penalty for
				// delaying castling grows. 5 cp per developed minor/major piece (max ~40 cp).
				// Only applies in middlegame (phase > 128) to avoid distorting endgame evals.
				if phase > 128 {
					developed := int(eval.PieceAdvanced[pColor][piece.KnightType]) +
						int(eval.PieceAdvanced[pColor][piece.BishopType]) +
						int(eval.PieceAdvanced[pColor][piece.RookType])
					score -= developed * 5
				}
			}
			if b.IsKingInCheck(pColor) {
				score += KingCheckedWeight
			}
			score += kingSafety(b, pColor)
			// King attack zone: count enemy-attacked squares in the 3×3 ring around our king.
			// Each additional attacked square triggers a penalty; exponential past 2 squares
			// so a concentrated attack is penalized more severely than a diffuse one.
			// Only relevant in the middlegame (phase > 64).
			if phase > 64 {
				enemy := pColor ^ 1
				kingLoc := b.KingLocations[pColor]
				var kingRing board.BitBoard
				for dr := int8(-1); dr <= 1; dr++ {
					for dc := int8(-1); dc <= 1; dc++ {
						if ringLoc, ok := kingLoc.AddRelative(location.RelativeLocation{Row: dr, Col: dc}); ok {
							kingRing.SetLocation(ringLoc)
						}
					}
				}
				attackedInRing := int(hamming.CountBitsUint64(uint64(kingRing.IntersectBitBoards(combinedAttacks[enemy]))))
				if attackedInRing >= 2 {
					// Quadratic scaling: 2 squares = 1×weight, 3 = 3×, 4 = 6×, ...
					score -= KingAttackZoneWeight * attackedInRing * (attackedInRing - 1) / 2
				}
			}
			for column := location.CoordinateType(0); column < board.Width; column++ {
				cnt := eval.PawnColumns[pColor][column]
				if cnt == 0 {
					continue
				}
				// Doubled pawn penalty grows exponentially per extra pawn on the file.
				score += PawnStructureWeight * PawnDuplicateWeight * ((1 << (cnt - 1)) - 1)
				// Isolated pawn: no friendly pawns on either adjacent file.
				leftEmpty := column == 0 || eval.PawnColumns[pColor][column-1] == 0
				rightEmpty := column == board.Width-1 || eval.PawnColumns[pColor][column+1] == 0
				if leftEmpty && rightEmpty {
					score += IsolatedPawnPenalty * int(cnt)
				}
			}
			goalRow := board.StartRow[pColor^1]["Piece"]
			for row := location.CoordinateType(0); row < board.Height; row++ {
				// boost score linearly for pawns that are closer to enemy start row
				dist := int8(goalRow) - int8(row)
				if dist < 0 {
					dist = -dist
				}
				// height - 1 is distance from pawn start
				progress := int(board.Height - 1 - dist)
				// normalize for number of pawns 8
				score += (PawnStructureWeight * PawnAdvancedWeight * progress * int(eval.PawnRows[pColor][row])) / 8
			}
			// pseudo-legal mobility: attackable squares (no board copies, no willMoveLeaveKingInCheck).
			// Weight intentionally lower than the old legal-moves weight because pseudo-legal counts
			// defended friendly squares too, which inflates the count vs strictly legal moves.
			score += PieceNumMovesWeight * int(eval.NumMoves[pColor])

			// Bishop pair bonus: tapered by total pawn count so it's weaker in closed positions.
			if eval.PieceCounts[pColor][piece.BishopType] >= 2 {
				totalPawns := int(eval.PieceCounts[color.White][piece.PawnType]) + int(eval.PieceCounts[color.Black][piece.PawnType])
				// Base bonus always applies; the open-position term adds up to
				// BishopPairOpenBonus more as pawns leave the board.
				// e.g. 30cp at 16 pawns, ~33cp at 14, ~42cp at 8, 55cp at 0.
				score += BishopPairBonus + BishopPairOpenBonus*(16-totalPawns)/16
			}

			if pColor == whoMoves {
				eval.TotalScore += score
			} else {
				eval.TotalScore -= score
			}
		}

		// Mop-up heuristic: when one side has a large material advantage, reward
		// pushing the losing king to the board edge and keeping kings close together.
		whiteMat, blackMat := 0, 0
		for pt, v := range PieceValue {
			if pt == piece.KingType {
				continue
			}
			whiteMat += v * int(eval.PieceCounts[color.White][pt])
			blackMat += v * int(eval.PieceCounts[color.Black][pt])
		}
		advantage := whiteMat - blackMat
		if advantage >= MopupThreshold || advantage <= -MopupThreshold {
			var winner, loser byte
			if advantage > 0 {
				winner, loser = color.White, color.Black
			} else {
				winner, loser = color.Black, color.White
			}
			loserKing := b.KingLocations[loser]
			winnerKing := b.KingLocations[winner]
			loserRow, loserCol := int(loserKing.GetRow()), int(loserKing.GetCol())
			winRow, winCol := int(winnerKing.GetRow()), int(winnerKing.GetCol())
			// distance from center (0–6): higher = more towards edge = better for winner
			edgeBonus := abs(loserRow-3) + abs(loserCol-3)
			// Manhattan distance between kings (0–14): lower = better for winner
			kingDist := abs(winRow-loserRow) + abs(winCol-loserCol)
			mopup := MopupWeight * (edgeBonus + (14 - kingDist))
			if winner == whoMoves {
				eval.TotalScore += mopup
			} else {
				eval.TotalScore -= mopup
			}
		}

		// King-to-passed-pawn support: in endgames, the winning king needs to escort
		// its own passed pawns. Reward proximity (max Manhattan distance 14 → 0 bonus,
		// distance 0 → +14 bonus). Scaled by endgame factor so it's irrelevant with
		// major pieces on the board.
		if phase < 128 {
			endgameFactor := (128 - phase) // 0 at mid-game, 128 at full endgame
			for row := location.CoordinateType(0); row < board.Height; row++ {
				for col := location.CoordinateType(0); col < board.Width; col++ {
					p := b.GetPiece(location.NewLocation(row, col))
					if p == nil || p.GetPieceType() != piece.PawnType {
						continue
					}
					c := p.GetColor()
					if !isPassedPawn(b, row, col, c) {
						continue
					}
					kingLoc := b.KingLocations[c]
					dist := abs(int(kingLoc.GetRow())-int(row)) + abs(int(kingLoc.GetCol())-int(col))
					bonus := KingPassedPawnSupportWeight * (14 - dist) * endgameFactor / 128
					if c == whoMoves {
						eval.TotalScore += bonus
					} else {
						eval.TotalScore -= bonus
					}
				}
			}
		}
	}
	return eval
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
