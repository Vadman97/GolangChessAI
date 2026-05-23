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
	RookDisplacedWeight   = -1 * PawnValueWeight
	KingCheckedWeight     = -PawnValueWeight / 4
	KingCastledWeight     = 1 * PawnValueWeight
	// neg 1 pawn if we do nothing in 50 moves
	Weight50Rule = -PawnValueWeight / PawnValueWeight
	// King safety: penalize exposed king files and enemy sliders on them
	KingOpenFilePenalty    = -50
	KingEnemySliderPenalty = -60
)

const (
	PawnDuplicateWeight = -1
	PawnAdvancedWeight  = 1
)

const (
	// MopupThreshold is the minimum material advantage (in PieceValue units, e.g. 5 = one rook)
	// required before the mop-up heuristic activates.
	MopupThreshold = 5
	// MopupWeight scales the mop-up bonus. Small enough to never override real material/checkmate.
	MopupWeight = PawnValueWeight / 20
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
var pawnPST = [8][8]int{
	{0, 0, 0, 0, 0, 0, 0, 0},         // rank 0 — pawns never reach here (promoted)
	{0, 0, 0, 0, 0, 0, 0, 0},         // rank 1 — starting squares
	{0, 0, 0, 0, 0, 0, 0, 0},         // rank 2 — one step advanced
	{2, 2, 4, 20, 20, 4, 2, 2},       // rank 3 — e4/d4: strong center
	{5, 5, 10, 25, 25, 10, 5, 5},     // rank 4 — passed pawn territory
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

// pstBonus returns the PST bonus for a piece at engine coordinates (row, col).
// row 0 = White's back rank; col 0 = h-file, col 7 = a-file.
func pstBonus(pieceType byte, pieceColor color.Color, row, col location.CoordinateType) int {
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
		return kingMiddlegamePST[rank][file] * pstScale
	}
	return 0
}

const (
	WinScore       = PosInf
	LossScore      = NegInf
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
		pstScores := [color.NumColors]int{}
		for row := location.CoordinateType(0); row < board.Height; row++ {
			for col := location.CoordinateType(0); col < board.Width; col++ {
				if gamePiece := b.GetPiece(location.NewLocation(row, col)); gamePiece != nil {
					c := gamePiece.GetColor()
					pt := gamePiece.GetPieceType()
					eval.PieceCounts[c][pt]++

					// Use pseudo-legal (attackable) squares for mobility — avoids willMoveLeaveKingInCheck
					// per candidate, which would copy the board for every move of every piece.
					// Pseudo-legal mobility is a standard eval heuristic and allows searching much deeper.
					numPseudoLegal := int(hamming.CountBitsUint64(uint64(gamePiece.GetAttackableMoves(b))))
					eval.NumMoves[c] += uint16(numPseudoLegal)

					pstScores[c] += pstBonus(pt, c, row, col)

					if pt == piece.PawnType {
						eval.PawnColumns[c][col]++
						eval.PawnRows[c][row]++
						if row != board.StartRow[c]["Pawn"] {
							eval.PieceAdvanced[c][pt]++
						}
					} else if pt != piece.KingType {
						if row != board.StartRow[c]["Piece"] {
							eval.PieceAdvanced[c][pt]++
						}
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
				// duplicate score grows exponentially for each additional pawn
				score += PawnStructureWeight * PawnDuplicateWeight * ((1 << (eval.PawnColumns[pColor][column] - 1)) - 1)
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
	}
	return eval
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
