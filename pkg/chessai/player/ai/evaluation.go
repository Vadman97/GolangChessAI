package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
)

type Evaluation struct {
	// [color][pieceType] -> overall piece count
	PieceCounts map[byte]map[byte]uint8
	// [color][pieceType] -> count of pieces off starting position
	PieceAdvanced map[byte]map[byte]uint8
	// [color][column] -> num pawns
	PawnColumns map[byte]map[location.CoordinateType]uint8
	// [color][column] -> num pawns
	PawnRows map[byte]map[location.CoordinateType]uint8
	// [color] -> num moves
	NumMoves   map[byte]uint16
	NumAttacks map[byte]uint16
	TotalScore int
}

func NewEvaluation() *Evaluation {
	e := Evaluation{
		PieceCounts: map[byte]map[byte]uint8{
			color.Black: {},
			color.White: {},
		},
		PieceAdvanced: map[byte]map[byte]uint8{
			color.Black: {},
			color.White: {},
		},
		PawnColumns: map[byte]map[location.CoordinateType]uint8{
			color.Black: {},
			color.White: {},
		},
		PawnRows: map[byte]map[location.CoordinateType]uint8{
			color.Black: {},
			color.White: {},
		},
		NumMoves:   map[byte]uint16{},
		NumAttacks: map[byte]uint16{},
	}
	return &e
}

var PieceValue = map[byte]int{
	piece.PawnType:   1,
	piece.BishopType: 3,
	piece.KnightType: 3,
	piece.RookType:   5,
	piece.QueenType:  9,
	piece.KingType:   100000,
}

const (
	PieceValueWeight      = 100
	PawnStructureWeight   = 50
	PieceAdvanceWeight    = 50
	PieceNumMovesWeight   = 10
	PieceNumAttacksWeight = 10
	KingDisplacedWeight   = -2 * PieceValueWeight   // neg 2 pawns
	RookDisplacedWeight   = -1 * PieceValueWeight   // neg 1 pawn
	KingCastledWeight     = 3 * PieceValueWeight    // three pawn
	KingCheckedWeight     = 1 * PieceValueWeight    // one pawn
	Weight50Rule          = -PieceValueWeight / 100 // neg 1 pawn if we do nothing in 50 moves
)

const (
	PawnDuplicateWeight = -1
	PawnAdvancedWeight  = 1
)

type EvaluationPair struct {
	Evaluations [color.NumColors]*Evaluation
}

func (p *AIPlayer) EvaluateBoard(b *board.Board, whoMoves color.Color) *Evaluation {
	eval := NewEvaluation()
	// first see if we have calculations we cannot cache
	if b.MovesSinceNoDraw == 100 {
		// Alex: This value may change, but AI right now prevents draws
		eval.TotalScore = 0
	} else {
		eval = p.evaluateBoardCached(b, whoMoves)
	}
	eval.TotalScore += Weight50Rule * b.MovesSinceNoDraw
	return eval
}

/**
 * Symmetric heuristic evaluation, relative to whoMoves color
 * https://www.chessprogramming.org/Evaluation#Side_to_move_relative
 */
func (p *AIPlayer) evaluateBoardCached(b *board.Board, whoMoves color.Color) *Evaluation {
	hash := b.Hash()
	if p.evaluationMap != nil {
		if value, ok := p.evaluationMap.Read(&hash); ok {
			entry := value.(EvaluationPair)
			if entry.Evaluations[whoMoves] != nil {
				return &Evaluation{
					TotalScore: entry.Evaluations[whoMoves].TotalScore,
				}
			}
		}
	}

	eval := NewEvaluation()
	// technically ignores en passant, but that should be ok
	if b.IsInCheckmate(whoMoves^1, nil) {
		eval.TotalScore = PosInf
	} else if b.IsInCheckmate(whoMoves, nil) {
		eval.TotalScore = NegInf
	} else if b.IsStalemate(whoMoves, nil) || b.IsStalemate(whoMoves^1, nil) {
		eval.TotalScore = 0
	} else {
		for r := location.CoordinateType(0); r < board.Height; r++ {
			for c := location.CoordinateType(0); c < board.Width; c++ {
				if p := b.GetPiece(location.NewLocation(r, c)); p != nil {
					eval.PieceCounts[p.GetColor()][p.GetPieceType()]++
					eval.NumMoves[p.GetColor()] += uint16(len(*p.GetMoves(b, false)))
					aMoves := p.GetAttackableMoves(b)
					if aMoves != nil {
						eval.NumAttacks[p.GetColor()] += uint16(len(*aMoves))
					}

					if p.GetPieceType() == piece.PawnType {
						eval.PawnColumns[p.GetColor()][c]++
						eval.PawnRows[p.GetColor()][r]++
						if r != board.StartRow[p.GetColor()]["Pawn"] {
							eval.PieceAdvanced[p.GetColor()][p.GetPieceType()]++
						}
						// do not give bonus for advancing king
					} else if p.GetPieceType() != piece.KingType {
						if r != board.StartRow[p.GetColor()]["Piece"] {
							eval.PieceAdvanced[p.GetColor()][p.GetPieceType()]++
						}
					}
				}
			}
		}
		for c := byte(0); c < color.NumColors; c++ {
			score := 0
			for pieceType, value := range PieceValue {
				score += PieceValueWeight * value * int(eval.PieceCounts[c][pieceType])
				score += PieceAdvanceWeight * int(eval.PieceAdvanced[c][pieceType])
			}
			if b.GetFlag(board.FlagCastled, c) {
				score += KingCastledWeight
			} else {
				// has not castled but
				if b.GetFlag(board.FlagKingMoved, c) {
					score += KingDisplacedWeight
				}
				if b.GetFlag(board.FlagLeftRookMoved, c) || b.GetFlag(board.FlagRightRookMoved, c) {
					score += RookDisplacedWeight
				}
			}
			if b.IsKingInCheck(c) {
				score += KingCheckedWeight
			}
			for column := location.CoordinateType(0); column < board.Width; column++ {
				// duplicate score grows exponentially for each additional pawn
				score += PawnStructureWeight * PawnDuplicateWeight * ((1 << (eval.PawnColumns[c][column] - 1)) - 1)
			}
			goalRow := board.StartRow[c^1]["Piece"]
			for row := location.CoordinateType(0); row < board.Height; row++ {
				// boost score linearly for pawns that are closer to enemy start row
				dist := int8(goalRow) - int8(row)
				if dist < 0 {
					dist = -dist
				}
				// height - 1 is distance from pawn start
				progress := int(board.Height - 1 - dist)
				// normalize for number of pawns 8
				score += (PawnStructureWeight * PawnAdvancedWeight * progress * int(eval.PawnRows[c][row])) / 8
			}
			// possible moves
			score += PieceNumMovesWeight * int(eval.NumMoves[c])
			// possible attacks
			score += PieceNumAttacksWeight * int(eval.NumAttacks[c])

			if c == whoMoves {
				eval.TotalScore += score
			} else {
				eval.TotalScore -= score
			}
		}
	}

	if p.evaluationMap != nil {
		entry := &EvaluationPair{
			Evaluations: [2]*Evaluation{},
		}
		if v, ok := p.evaluationMap.Read(&hash); ok {
			entry = v.(*EvaluationPair)
		}
		entry.Evaluations[whoMoves] = eval
		p.evaluationMap.Store(&hash, entry)
	}

	return eval
}
