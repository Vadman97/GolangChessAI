package ai

/*
	Loosely based on https://github.com/official-stockfish/Stockfish/blob/9a11a291942a8a7b1ebb36282c666ca8d1be1892/src/evaluate.cpp
*/

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

type Value int16
type Score int

/// Score enum stores a middlegame and an endgame value in a single integer (enum).
/// The least significant 16 bits are used to store the middlegame value and the
/// upper 16 bits are used to store the endgame value. We have to take care to
/// avoid left-shifting a signed int to avoid undefined behavior.
func MakeScore(midGame, endGame int) Score {
	return Score((int)(uint(endGame)<<16) + midGame)
}

func (s Score) endGameValue() Value {
	return Value(uint(s+0x8000) >> 16)
}

func (s Score) midGameValue() Value {
	return Value(uint(s))
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
	LazyThreshold  = Value(1500)
	SpaceThreshold = Value(12222)
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

// MobilityBonus[PieceType-2][attacked] contains bonuses for middle and end game,
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

// RookOnFile[semiopen/open] contains bonuses for each rook when there is
// no (friendly) pawn on the rook file.
var RookOnFileSemiOpen = MakeScore(18, 7)
var RookOnFileOpen = MakeScore(44, 20)

// ThreatByMinor/ByRook[attacked PieceType] contains bonuses according to
// which piece type attacks which one. Attacks on lesser pieces which are
// pawn-defended are not considered.
var ThreatByMinor = [...]Score{
	MakeScore(0, 0), MakeScore(0, 31), MakeScore(39, 42), MakeScore(57, 44), MakeScore(68, 112), MakeScore(62, 120),
}
var ThreatByRook = [...]Score{
	MakeScore(0, 0), MakeScore(0, 24), MakeScore(38, 71), MakeScore(38, 61), MakeScore(0, 38), MakeScore(51, 38),
}

// PassedRank[Rank] contains a bonus according to the rank of a passed pawn
var PassedRank = [...]Score{
	MakeScore(0, 0), MakeScore(5, 18), MakeScore(12, 23), MakeScore(10, 31), MakeScore(57, 62), MakeScore(163, 167), MakeScore(271, 250),
}

// PassedFile[File] contains a bonus according to the file of a passed pawn
var PassedFile = [...]Score{
	MakeScore(-1, 7), MakeScore(0, 9), MakeScore(-9, -8), MakeScore(-30, -14),
	MakeScore(-30, -14), MakeScore(-9, -8), MakeScore(0, 9), MakeScore(-1, 7),
}

// Assorted bonuses and penalties
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
	Scores        [color.NumColors]Value
	MobilityArea  [color.NumColors]board.BitBoard
	MobilityScore [color.NumColors]Score

	// attackedBy[color][piece type] is a bitboard representing all squares
	// attacked by a given color and piece type. Special "piece types" which
	// is also calculated is ALL_PIECES.
	AttackedBy [color.NumColors][piece.NumPieces]board.BitBoard

	// attackedBy2[color] are the squares attacked by 2 pieces of a given color,
	// possibly via x-ray or by one pawn and one piece. Diagonal x-ray through
	// pawn or squares attacked by 2 pawns are not explicitly added.
	AttackedBy2 [color.NumColors][piece.NumPieces]board.BitBoard

	// kingRing[color] are the squares adjacent to the king, plus (only for a
	// king on its first rank) the squares two ranks in front. For instance,
	// if black's king is on g8, kingRing[BLACK] is f8, h8, f7, g7, h7, f6, g6
	// and h6.
	KingRing [color.NumColors]board.BitBoard

	// kingAttackersCount[color] is the number of pieces of the given color
	// which attack a square in the kingRing of the enemy king.
	KingAttackersCount [color.NumColors]int

	// kingAttackersWeight[color] is the sum of the "weights" of the pieces of
	// the given color which attack a square in the kingRing of the enemy king.
	// The weights of the individual piece types are given by the elements in
	// the KingAttackWeights array.
	KingAttackersWeight [color.NumColors]int

	// kingAttacksCount[color] is the number of attacks by the given color to
	// squares directly adjacent to the enemy king. Pieces which attack more
	// than one square are counted multiple times. For instance, if there is
	// a white knight on g5 and black's king is on g8, this white knight adds 2
	// to kingAttacksCount[WHITE].
	KingAttacksCount [color.NumColors]int

	TotalScore Value // TODO(Vadim) remove
}

func NewEvaluation(ourColor color.Color) *Evaluation {
	e := Evaluation{
		Scores:              [color.NumColors]Value{},
		MobilityArea:        [color.NumColors]board.BitBoard{},
		MobilityScore:       [color.NumColors]Score{},
		AttackedBy:          [color.NumColors][piece.NumPieces]board.BitBoard{},
		AttackedBy2:         [color.NumColors][piece.NumPieces]board.BitBoard{},
		KingRing:            [color.NumColors]board.BitBoard{},
		KingAttackersCount:  [color.NumColors]int{},
		KingAttackersWeight: [color.NumColors]int{},
		KingAttacksCount:    [color.NumColors]int{},
	}

	return &e
}

var PieceValue = [...]int{
	piece.PawnType:   1,
	piece.KnightType: 3,
	piece.BishopType: 3,
	piece.RookType:   5,
	piece.QueenType:  9,
	piece.KingType:   100,
}

const (
	PawnValueWeight       = 100
	PawnStructureWeight   = PawnValueWeight / 2
	PieceAdvanceWeight    = PawnValueWeight / 2
	PieceNumMovesWeight   = PawnValueWeight / 10
	PieceNumAttacksWeight = PawnValueWeight / 10
	KingDisplacedWeight   = -2 * PawnValueWeight
	RookDisplacedWeight   = -1 * PawnValueWeight
	KingCastledWeight     = 3 * PawnValueWeight
	KingCheckedWeight     = 1 * PawnValueWeight
	// neg 1 pawn if we do nothing in 50 moves
	Weight50Rule = -PawnValueWeight / PawnValueWeight
)

const (
	PawnDuplicateWeight = -1
	PawnAdvancedWeight  = 1
)

const (
	WinScore       = Value(PosInf)
	LossScore      = Value(NegInf)
	StalemateScore = Value(NegInf / 2) // should only choose draw to avoid a loss
)

type evaluationPair struct {
	score    Value
	whoMoves color.Color
}

// TODO(Vadim) make this a static function so evaluation cache is global
func (p *AIPlayer) EvaluateBoard(b *board.Board, whoMoves color.Color) *Evaluation {
	eval := NewEvaluation(whoMoves)
	// first see if we have calculations we cannot cache
	if b.MovesSinceNoDraw >= 100 {
		// Vadim: >= instead of == because AI simulation will go beyond 100, it will know no win is possible
		// Alex: This value may change, but AI right now prevents draws
		eval.Scores[color.White] = StalemateScore
		eval.Scores[color.Black] = StalemateScore
	} else if b.PreviousPositionsSeen >= 3 {
		eval.Scores[color.White] = StalemateScore
		eval.Scores[color.Black] = StalemateScore
	} else {
		eval = p.evaluateBoardCached(b, whoMoves)
		eval.Scores[color.White] += Value(Weight50Rule * b.MovesSinceNoDraw)
		eval.Scores[color.Black] += Value(Weight50Rule * b.MovesSinceNoDraw)
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
			e := &Evaluation{Scores: [color.NumColors]Value{}}
			e.Scores[whoMoves] = score
			return e
		}
	}
	eval = EvaluateBoardNoCache(b, whoMoves)

	if p.evaluationMap != nil {
		p.evaluationMap.Store(&hash, 0, &evaluationPair{
			score:    eval.Scores[whoMoves],
			whoMoves: whoMoves,
		})
	}
	return eval
}

func EvaluateBoardNoCache(b *board.Board, whoMoves color.Color) *Evaluation {
	eval := NewEvaluation(whoMoves)
	// TODO(Vadim) make new
	return eval
}
