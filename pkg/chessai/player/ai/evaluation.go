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
	KingDisplacedWeight   = -2 * PieceValueWeight // neg 2 pawns
	RookDisplacedWeight   = -1 * PieceValueWeight // neg 1 pawn
	KingCastledWeight     = 3 * PieceValueWeight  // three pawn
	KingCheckedWeight     = 1 * PieceValueWeight  // one pawn
)

const (
	PawnDuplicateWeight = -1
)

func (p *Player) EvaluateBoard(b *board.Board) *Evaluation {
	hash := b.Hash()
	if p.evaluationMap != nil {
		if score, ok := p.evaluationMap.Read(&hash); ok {
			return &Evaluation{
				TotalScore: score.(int),
			}
		}
	}

	eval := NewEvaluation()
	// technically ignores en passant, but that should be ok
	// TODO(Vadim) figure out if we can optimize, this makes very slow #47
	/*else if b.IsInCheckmate(p.PlayerColor, nil) {
		eval.TotalScore = NegInf
	} else if b.IsStalemate(p.PlayerColor, nil) || b.IsStalemate(p.PlayerColor ^ 1, nil) {
		eval.TotalScore = 0
	}*/
	if b.IsInCheckmate(p.PlayerColor^1, nil) {
		eval.TotalScore = PosInf
	} else if b.IsInCheckmate(p.PlayerColor, nil) {
		eval.TotalScore = NegInf
	} else if b.MovesSinceNoDraw == 100 {
		// TODO(Alex) This value may change, but AI right now prevents draws
		eval.TotalScore = 0
	} else {
		for r := location.CoordinateType(0); r < board.Width; r++ {
			for c := location.CoordinateType(0); c < board.Height; c++ {
				if p := b.GetPiece(location.NewLocation(r, c)); p != nil {
					eval.PieceCounts[p.GetColor()][p.GetPieceType()]++
					eval.NumMoves[p.GetColor()] += uint16(len(*p.GetMoves(b)))
					aMoves := p.GetAttackableMoves(b)
					if aMoves != nil {
						eval.NumAttacks[p.GetColor()] += uint16(len(*aMoves))
					}

					if p.GetPieceType() == piece.PawnType {
						eval.PawnColumns[p.GetColor()][c]++
						if r != board.StartRow[p.GetColor()]["Pawn"] {
							eval.PieceAdvanced[p.GetColor()][p.GetPieceType()]++
						}
					} else {
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
			// possible moves
			score += PieceNumMovesWeight * int(eval.NumMoves[c])
			// possible attacks
			score += PieceNumAttacksWeight * int(eval.NumAttacks[c])

			if c == p.PlayerColor {
				eval.TotalScore += score
			} else {
				eval.TotalScore -= score
			}
		}
	}

	if p.evaluationMap != nil {
		p.evaluationMap.Store(&hash, int(eval.TotalScore))
	}

	return eval
}
