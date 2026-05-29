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
	KingDisplacedWeight = -1 * PawnValueWeight
	// Small penalty for moving a rook before castling — not large enough to force premature castling.
	RookDisplacedWeight = -PawnValueWeight / 5
	KingCheckedWeight   = -PawnValueWeight / 4
	// KingCastledWeight reduced from 100 to 40: the old +100 cp bonus (plus ~70 PST swing = +170 total)
	// was overriding concrete tactical continuations — the engine would castle even in clearly winning
	// positions where an active rook move or attack was better.
	KingCastledWeight = PawnValueWeight * 2 / 5
	// neg 1 pawn if we do nothing in 50 moves
	Weight50Rule = -PawnValueWeight / PawnValueWeight
	// King safety: penalize exposed king files and enemy sliders on them
	KingOpenFilePenalty    = -20
	KingEnemySliderPenalty = -30
)

const (
	PawnDuplicateWeight = -1
	PawnAdvancedWeight  = 1
	// IsolatedPawnPenalty: a pawn with no friendly pawns on adjacent files is weak.
	IsolatedPawnPenalty = -15
	// Rook file activity bonuses.
	RookOpenFileBonus     = 15 // no pawns of either color on the file
	RookSemiOpenFileBonus = 8  // no friendly pawns but enemy pawns present
	// KnightOutpostBonus: knight on ranks 3-5 (from own back rank) that no enemy
	// pawn can attack. Outpost knights dominate the middlegame.
	KnightOutpostBonus = 20
	// KnightPasserBlockadeBonus: per passed pawn whose next-advance square is
	// attacked by a friendly knight. Scaled by pawn rank (rank/6 × base value).
	// Prevents the engine from undervaluing defensive knight maneuvers that
	// simultaneously blockade multiple advanced passed pawns (K+PP vs K+N endings).
	KnightPasserBlockadeBonus = 150
)

// knightMoveDeltas lists all 8 relative (row, col) offsets for knight moves.
var knightMoveDeltas = [8][2]int{
	{-2, -1}, {-2, 1}, {-1, -2}, {-1, 2},
	{2, -1}, {2, 1}, {1, -2}, {1, 2},
}

// isKnightOutpost returns true if no enemy pawn can attack the given square,
// meaning the knight sitting there cannot be chased away by a pawn push.
func isKnightOutpost(b *board.Board, row, col location.CoordinateType, knightColor color.Color) bool {
	enemy := knightColor ^ 1
	// Enemy pawns attack diagonally forward. For White (attacking toward Black's back rank
	// = higher rows), White pawns at (row-1, col±1) attack (row, col).
	// For Black (attacking toward White's back rank = lower rows), Black pawns at
	// (row+1, col±1) attack (row, col).
	var attackRow location.CoordinateType
	if enemy == color.White {
		// White pawns attack upward: they'd be at row-1 to attack row.
		if row == 0 {
			return true
		}
		attackRow = row - 1
	} else {
		// Black pawns attack downward: they'd be at row+1 to attack row.
		if row == board.Height-1 {
			return true
		}
		attackRow = row + 1
	}
	for _, dc := range []int{-1, 1} {
		ac := int(col) + dc
		if ac < 0 || ac >= board.Width {
			continue
		}
		p := b.GetPiece(location.NewLocation(attackRow, location.CoordinateType(ac)))
		if p != nil && p.GetColor() == enemy && p.GetPieceType() == piece.PawnType {
			return false
		}
	}
	return true
}

// passedPawnBonus is added to the score for a passed pawn at the given rank
// (distance from own back rank, 0 = back rank, 7 = promotion). Additive on top
// of pawnPST — passed pawns are far more dangerous than normal advanced pawns.
// Conservative values prevent the shallow search (depth 6-8) from projecting
// pawn advances that a deeper search (SF depth 15) knows White can stop.
// Old values {0,0,0,10,35,75,110} caused 1000+ cp eval inflation at search horizon.
var passedPawnBonus = [8]int{0, 0, 0, 5, 20, 50, 75, 0}

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
func NormalizeMateScore(score, depth int) int {
	if score >= WinScore {
		return score - depth
	} else if score <= LossScore {
		return score + depth
	}
	return score
}

// DenormalizeMateScore re-applies the depth component after reading a score
// from the transposition table at a (possibly different) depth.
func DenormalizeMateScore(score, depth int) int {
	if score >= WinScore {
		return score + depth
	} else if score <= LossScore {
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
			return &Evaluation{
				TotalScore: score,
			}
		}
	}
	eval = EvaluateBoardNoCache(b, whoMoves)

	if p.evaluationMap != nil {
		p.evaluationMap.Store(&hash, 0, &evaluationPair{
			score:    eval.TotalScore,
			whoMoves: whoMoves,
		})
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
	} else if b.IsStalemate(whoMoves, nil) || b.IsStalemate(whoMoves^1, nil) || b.IsInsufficientMaterial() {
		// want to discourage us from stalemating other player or getting stalemated
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
		for row := location.CoordinateType(0); row < board.Height; row++ {
			for col := location.CoordinateType(0); col < board.Width; col++ {
				if gamePiece := b.GetPiece(location.NewLocation(row, col)); gamePiece != nil {
					c := gamePiece.GetColor()
					pt := gamePiece.GetPieceType()

					// Use pseudo-legal (attackable) squares for mobility — avoids willMoveLeaveKingInCheck
					// per candidate, which would copy the board for every move of every piece.
					// Pseudo-legal mobility is a standard eval heuristic and allows searching much deeper.
					numPseudoLegal := int(hamming.CountBitsUint64(uint64(gamePiece.GetAttackableMoves(b))))
					eval.NumMoves[c] += uint16(numPseudoLegal)

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
						}
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
				// has not castled but
				if b.GetFlag(board.FlagKingMoved, pColor) {
					score += KingDisplacedWeight
				}
				if b.GetFlag(board.FlagLeftRookMoved, pColor) || b.GetFlag(board.FlagRightRookMoved, pColor) {
					score += RookDisplacedWeight
				}
			}
			if b.IsKingInCheck(pColor) {
				score += KingCheckedWeight
			}
			score += kingSafety(b, pColor)
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
