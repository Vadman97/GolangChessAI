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
						// do not give bonus for advancing king
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
			// possible moves
			score += PieceNumMovesWeight * int(eval.NumMoves[pColor])
			// possible attacks
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
