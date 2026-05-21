package ai

/*
	Loosely based on https://github.com/official-stockfish/Stockfish/blob/9a11a291942a8a7b1ebb36282c666ca8d1be1892/src/evaluate.cpp
*/

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/steakknife/hamming"
)

// Score encodes a middlegame and endgame value in a single integer.
// The lower 16 bits store the middlegame value; the upper 16 bits store the endgame value.
type Score int

func MakeScore(midGame, endGame int) Score {
	return Score((int)(uint(endGame)<<16) + midGame)
}

func (s Score) endGameValue() int {
	return int(int16(uint(s+0x8000) >> 16))
}

func (s Score) midGameValue() int {
	return int(int16(s))
}

const (
	TermMaterial   = 8
	TermImbalance  = iota
	TermMobility   = iota
	TermThreat     = iota
	TermPassed     = iota
	TermSpace      = iota
	TermInitiative = iota
)

const (
	LazyThreshold  = 1500
	SpaceThreshold = 12222
)

var KingAttackWeights = [...]int{
	piece.PawnType:   0,
	piece.KnightType: 77,
	piece.BishopType: 55,
	piece.RookType:   44,
	piece.QueenType:  10,
	piece.KingType:   100,
}

const (
	QueenSafeCheck  = 780
	RookSafeCheck   = 1080
	BishopSaveCheck = 635
	KnightSafeCheck = 790
)

// MobilityBonus[PieceType][attacked] contains bonuses for middle and end game,
// indexed by piece type and number of attacked squares in the mobility area.
var MobilityBonus = [...][32]Score{
	piece.KnightType: {
		MakeScore(-62, -81), MakeScore(-53, -56), MakeScore(-12, -30), MakeScore(-4, -14), MakeScore(3, 8), MakeScore(13, 15), MakeScore(22, 23), MakeScore(28, 27), MakeScore(33, 33),
	},
	piece.BishopType: {
		MakeScore(-48, -59), MakeScore(-20, -23), MakeScore(16, -3), MakeScore(26, 13), MakeScore(38, 24), MakeScore(51, 42), MakeScore(55, 54), MakeScore(63, 57), MakeScore(63, 65), MakeScore(68, 73), MakeScore(81, 78), MakeScore(81, 86), MakeScore(91, 88), MakeScore(98, 97),
	},
	piece.RookType: {
		MakeScore(-58, -76), MakeScore(-27, -18), MakeScore(-15, 28), MakeScore(-10, 55), MakeScore(-5, 69), MakeScore(-2, 82), MakeScore(9, 112), MakeScore(16, 118), MakeScore(30, 132), MakeScore(29, 142), MakeScore(32, 155), MakeScore(38, 165), MakeScore(46, 166), MakeScore(48, 169), MakeScore(58, 171),
	},
	piece.QueenType: {
		MakeScore(-39, -36), MakeScore(-21, -15), MakeScore(3, 8), MakeScore(3, 18), MakeScore(14, 34), MakeScore(22, 54), MakeScore(28, 61), MakeScore(41, 73), MakeScore(43, 79), MakeScore(48, 92), MakeScore(56, 94), MakeScore(60, 104), MakeScore(60, 113), MakeScore(66, 120), MakeScore(67, 123), MakeScore(70, 126), MakeScore(71, 133), MakeScore(73, 136), MakeScore(79, 140), MakeScore(88, 143), MakeScore(88, 148), MakeScore(99, 166), MakeScore(102, 170), MakeScore(102, 175), MakeScore(106, 184), MakeScore(109, 191), MakeScore(113, 206), MakeScore(116, 212),
	},
}

// RookOnFile bonuses when there is no friendly pawn on the rook file.
var RookOnFileSemiOpen = MakeScore(18, 7)
var RookOnFileOpen = MakeScore(44, 20)

// ThreatByMinor/ByRook bonuses according to which piece type attacks which one.
var ThreatByMinor = [...]Score{
	MakeScore(0, 0), MakeScore(0, 31), MakeScore(39, 42), MakeScore(57, 44), MakeScore(68, 112), MakeScore(62, 120),
}
var ThreatByRook = [...]Score{
	MakeScore(0, 0), MakeScore(0, 24), MakeScore(38, 71), MakeScore(38, 61), MakeScore(0, 38), MakeScore(51, 38),
}

// PassedRank/File bonuses for passed pawns.
var PassedRank = [...]Score{
	MakeScore(0, 0), MakeScore(5, 18), MakeScore(12, 23), MakeScore(10, 31), MakeScore(57, 62), MakeScore(163, 167), MakeScore(271, 250),
}
var PassedFile = [...]Score{
	MakeScore(-1, 7), MakeScore(0, 9), MakeScore(-9, -8), MakeScore(-30, -14),
	MakeScore(-30, -14), MakeScore(-9, -8), MakeScore(0, 9), MakeScore(-1, 7),
}

// Assorted bonuses and penalties.
var BishopPawns = MakeScore(3, 7)
var CorneredBishop = MakeScore(50, 50)
var FlankAttacks = MakeScore(8, 0)
var Hanging = MakeScore(69, 36)
var KingProtector = MakeScore(7, 8)
var KnightOnQueen = MakeScore(16, 12)
var LongDiagonalBishop = MakeScore(45, 0)
var MinorBehindPawn = MakeScore(18, 3)
var Outpost = MakeScore(9, 3)
var PawnlessFlank = MakeScore(17, 95)
var RestrictedPiece = MakeScore(7, 7)
var RookOnPawn = MakeScore(10, 32)
var SliderOnQueen = MakeScore(59, 18)
var ThreatByKing = MakeScore(24, 89)
var ThreatByPawnPush = MakeScore(48, 39)
var ThreatByRank = MakeScore(13, 0)
var ThreatBySafePawn = MakeScore(173, 94)
var TrappedRook = MakeScore(47, 4)
var WeakQueen = MakeScore(49, 15)
var WeakUnopposedPawn = MakeScore(12, 23)

type Evaluation struct {
	// [color][pieceType] -> overall piece count
	PieceCounts map[color.Color]map[byte]uint8
	// [color][pieceType] -> count of pieces off starting position
	PieceAdvanced map[color.Color]map[byte]uint8
	// [color][column] -> num pawns
	PawnColumns map[color.Color]map[location.CoordinateType]uint8
	// [color][row] -> num pawns
	PawnRows map[color.Color]map[location.CoordinateType]uint8
	// [color] -> num moves/attacks
	NumMoves   map[color.Color]uint16
	NumAttacks map[color.Color]uint16

	// Stockfish-inspired fields for future use.
	MobilityArea        [color.NumColors]board.BitBoard
	MobilityScore       [color.NumColors]Score
	AttackedBy          [color.NumColors][piece.NumPieces]board.BitBoard
	AttackedBy2         [color.NumColors][piece.NumPieces]board.BitBoard
	KingRing            [color.NumColors]board.BitBoard
	KingAttackersCount  [color.NumColors]int
	KingAttackersWeight [color.NumColors]int
	KingAttacksCount    [color.NumColors]int

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
)

const (
	PawnDuplicateWeight = -1
	PawnAdvancedWeight  = 1
)

const (
	// MopupThreshold is the minimum material advantage (in PieceValue units) before mop-up activates.
	MopupThreshold = 5
	MopupWeight    = PawnValueWeight / 20
)

const (
	WinScore       = PosInf
	LossScore      = NegInf
	StalemateScore = 0
)

// AdjustMateScore encodes depth into win/loss scores so the search prefers
// shorter paths to checkmate.
func AdjustMateScore(score, depth int) int {
	if score >= WinScore {
		return WinScore + depth
	} else if score <= LossScore {
		return LossScore - depth
	}
	return score
}

// NormalizeMateScore removes the depth component before storing a score in the
// transposition table so that distance-to-mate is relative to the stored position.
func NormalizeMateScore(score, depth int) int {
	if score >= WinScore {
		return score - depth
	} else if score <= LossScore {
		return score + depth
	}
	return score
}

// DenormalizeMateScore re-applies the depth component after reading a score
// from the transposition table.
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

func EvaluateBoardNoCache(b *board.Board, whoMoves color.Color) *Evaluation {
	eval := NewEvaluation()
	// technically ignores en passant, but that should be ok
	if b.IsInCheckmate(whoMoves^1, nil) {
		eval.TotalScore = WinScore
	} else if b.IsInCheckmate(whoMoves, nil) {
		eval.TotalScore = LossScore
	} else if b.IsStalemate(whoMoves, nil) || b.IsStalemate(whoMoves^1, nil) || b.IsInsufficientMaterial() {
		eval.TotalScore = StalemateScore
	} else {
		for row := location.CoordinateType(0); row < board.Height; row++ {
			for col := location.CoordinateType(0); col < board.Width; col++ {
				if gamePiece := b.GetPiece(location.NewLocation(row, col)); gamePiece != nil {
					eval.PieceCounts[gamePiece.GetColor()][gamePiece.GetPieceType()]++

					numMoves := len(*gamePiece.GetMoves(b, false))
					eval.NumMoves[gamePiece.GetColor()] += uint16(numMoves)

					numAttacks := hamming.CountBitsUint64(uint64(gamePiece.GetAttackableMoves(b))) - numMoves
					eval.NumAttacks[gamePiece.GetColor()] += uint16(numAttacks)

					if gamePiece.GetPieceType() == piece.PawnType {
						eval.PawnColumns[gamePiece.GetColor()][col]++
						eval.PawnRows[gamePiece.GetColor()][row]++
						if row != board.StartRow[gamePiece.GetColor()]["Pawn"] {
							eval.PieceAdvanced[gamePiece.GetColor()][gamePiece.GetPieceType()]++
						}
					} else if gamePiece.GetPieceType() != piece.KingType {
						if row != board.StartRow[gamePiece.GetColor()]["Piece"] {
							eval.PieceAdvanced[gamePiece.GetColor()][gamePiece.GetPieceType()]++
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
			if b.GetFlag(board.FlagCastled, pColor) {
				score += KingCastledWeight
			} else {
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
			for column := location.CoordinateType(0); column < board.Width; column++ {
				score += PawnStructureWeight * PawnDuplicateWeight * ((1 << (eval.PawnColumns[pColor][column] - 1)) - 1)
			}
			goalRow := board.StartRow[pColor^1]["Piece"]
			for row := location.CoordinateType(0); row < board.Height; row++ {
				dist := int8(goalRow) - int8(row)
				if dist < 0 {
					dist = -dist
				}
				progress := int(board.Height - 1 - dist)
				score += (PawnStructureWeight * PawnAdvancedWeight * progress * int(eval.PawnRows[pColor][row])) / 8
			}
			score += PieceNumMovesWeight * int(eval.NumMoves[pColor])
			score += PieceNumAttacksWeight * int(eval.NumAttacks[pColor])

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
			edgeBonus := abs(loserRow-3) + abs(loserCol-3)
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
