package ai

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/piece"
	"ChessAI3/chessai/util"
	"fmt"
	"math"
)

const (
	NegInf = math.MinInt32
	PosInf = math.MaxInt32
)

const (
	AlgorithmMiniMax             = iota
	AlgorithmAlphaBetaWithMemory = iota
)

var PieceValue = map[byte]int{
	piece.PawnType:   1,
	piece.BishopType: 3,
	piece.KnightType: 3,
	piece.RookType:   5,
	piece.QueenType:  9,
	piece.KingType:   100,
}

type ScoredMove struct {
	Move  *board.Move
	Score int
}

type Player struct {
	TurnCount      int
	PlayerColor    byte
	Algorithm      int
	evaluationMap  *util.ConcurrentScoreMap
	alphaBetaTable *util.TranspositionTable
}

func NewAIPlayer(c byte) *Player {
	return &Player{
		Algorithm:      AlgorithmAlphaBetaWithMemory,
		TurnCount:      0,
		PlayerColor:    c,
		evaluationMap:  util.NewConcurrentScoreMap(),
		alphaBetaTable: util.NewTranspositionTable(),
	}
}

func compare(maximizingP bool, currentBest *ScoredMove, candidate *ScoredMove) *ScoredMove {
	if maximizingP {
		if candidate.Score > currentBest.Score {
			return candidate
		} else {
			return currentBest
		}
	} else {
		if candidate.Score < currentBest.Score {
			return candidate
		} else {
			return currentBest
		}
	}
}

func (p *Player) GetBestMove(b *board.Board) *board.Move {
	var m *ScoredMove
	if p.Algorithm == AlgorithmMiniMax {
		m = p.MiniMax(b, 4, p.PlayerColor)
	} else if p.Algorithm == AlgorithmAlphaBetaWithMemory {
		m = p.AlphaBetaWithMemory(b, 8, NegInf, PosInf, p.PlayerColor)
	} else {
		panic("invalid ai algorithm")
	}
	fmt.Printf("AI Player best move leads to score %d\n", m.Score)
	p.evaluationMap.PrintMetrics()
	p.alphaBetaTable.PrintMetrics()
	return m.Move
}

func (p *Player) MakeMove(b *board.Board) {
	board.MakeMove(p.GetBestMove(b), b)
	p.TurnCount++
}

func (p *Player) EvaluateBoard(b *board.Board) *board.Evaluation {
	hash := b.Hash()
	if score, ok := p.evaluationMap.Read(&hash); ok {
		return &board.Evaluation{
			TotalScore: int(score),
		}
	}

	// TODO(Vadim) make more intricate
	eval := board.Evaluation{
		PieceCounts: map[byte]map[byte]uint8{
			color.Black: {},
			color.White: {},
		},
	}

	for r := int8(0); r < board.Width; r++ {
		for c := int8(0); c < board.Height; c++ {
			if p := b.GetPiece(board.Location{Row: r, Col: c}); p != nil {
				_, ok := eval.PieceCounts[p.GetColor()][p.GetPieceType()]
				if !ok {
					eval.PieceCounts[p.GetColor()][p.GetPieceType()] = 1
				} else {
					eval.PieceCounts[p.GetColor()][p.GetPieceType()]++
				}
			}
		}
	}

	for c := byte(0); c < color.NumColors; c++ {
		for pieceType, value := range PieceValue {
			cMult := 1
			if c != p.PlayerColor {
				cMult = -1
			}
			eval.TotalScore += value * int(eval.PieceCounts[c][pieceType]) * cMult
		}
	}

	p.evaluationMap.Store(&hash, int32(eval.TotalScore))

	return &eval
}
