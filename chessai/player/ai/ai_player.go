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
	AlgorithmMiniMax             = "MiniMax"
	AlgorithmAlphaBetaWithMemory = "AlphaBeta"
)

var PieceValue = map[byte]int{
	piece.PawnType:   1,
	piece.BishopType: 3,
	piece.KnightType: 3,
	piece.RookType:   5,
	piece.QueenType:  9,
	piece.KingType:   100,
}

const (
	PieceValueWeight    = 1000
	PawnStructureWeight = 500
	PieceAdvanceWeight  = 100
	PieceNumMovesWeight = 10
	KingDisplacedWeight = -1000
	KingCastledWeight   = 500
)

const (
	PawnDuplicateWeight = -1
)

// color -> openers
var OpeningMoves = map[byte][]*board.Move{
	color.Black: {
		&board.Move{
			Start: board.Location{Row: board.StartRow[color.Black]["Pawn"], Col: 4},
			End:   board.Location{Row: board.StartRow[color.Black]["Pawn"] + 2, Col: 4},
		},
		&board.Move{
			Start: board.Location{Row: board.StartRow[color.Black]["Piece"], Col: 1},
			End:   board.Location{Row: board.StartRow[color.Black]["Piece"] + 2, Col: 2},
		},
	},
	color.White: {
		&board.Move{
			Start: board.Location{Row: board.StartRow[color.White]["Pawn"], Col: 4},
			End:   board.Location{Row: board.StartRow[color.White]["Pawn"] - 2, Col: 4},
		},
		&board.Move{
			Start: board.Location{Row: board.StartRow[color.White]["Piece"], Col: 6},
			End:   board.Location{Row: board.StartRow[color.White]["Piece"] - 2, Col: 5},
		},
	},
}

type ScoredMove struct {
	Move         board.Move
	MoveSequence []board.Move
	Score        int
}

type Player struct {
	TurnCount      int
	PlayerColor    byte
	Algorithm      string
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

func compare(maximizingP bool, currentBest *ScoredMove, candidate *ScoredMove) bool {
	if maximizingP {
		if candidate.Score > currentBest.Score {
			return true
		} else {
			return false
		}
	} else {
		if candidate.Score < currentBest.Score {
			return true
		} else {
			return false
		}
	}
}

func (p *Player) GetBestMove(b *board.Board) *board.Move {
	if p.TurnCount < len(OpeningMoves) {
		return OpeningMoves[p.PlayerColor][p.TurnCount]
	} else {
		var m *ScoredMove
		if p.Algorithm == AlgorithmMiniMax {
			m = p.MiniMax(b, 4, p.PlayerColor)
		} else if p.Algorithm == AlgorithmAlphaBetaWithMemory {
			m = p.AlphaBetaWithMemory(b, 8, NegInf, PosInf, p.PlayerColor)
		} else {
			panic("invalid ai algorithm")
		}
		c := "Black"
		if p.PlayerColor == color.White {
			c = "White"
		}
		fmt.Printf("AI (%s - %s) best move leads to score %d\n", p.Algorithm, c, m.Score)
		debugBoard := b.Copy()
		//for i := 0; i < len(m.MoveSequence); i++ {
		for i := len(m.MoveSequence) - 1; i >= 0; i-- {
			move := m.MoveSequence[i]
			start := debugBoard.GetPiece(move.Start)
			end := debugBoard.GetPiece(move.End)
			startStr, endStr := board.GetColorTypeRepr(start), board.GetColorTypeRepr(end)
			if end == nil {
				endStr = "_"
			}
			fmt.Printf("\t%s to %s\n", startStr, endStr)
			fmt.Printf("\t\t%s\n", move.Print())
			board.MakeMove(&move, debugBoard)
		}
		p.evaluationMap.PrintMetrics()
		p.alphaBetaTable.PrintMetrics()
		return &m.Move
	}
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
	eval := board.NewEvaluation()

	for r := int8(0); r < board.Width; r++ {
		for c := int8(0); c < board.Height; c++ {
			if p := b.GetPiece(board.Location{Row: r, Col: c}); p != nil {
				eval.PieceCounts[p.GetColor()][p.GetPieceType()]++
				eval.NumMoves[p.GetColor()] += uint16(len(*p.GetMoves(b)))

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
			// TODO(Vadim) piece advance does not work right
			score += PieceAdvanceWeight * int(eval.PieceAdvanced[c][pieceType])
		}
		if b.GetFlag(board.FlagKingMoved, c) && !b.GetFlag(board.FlagCastled, c) {
			score += KingDisplacedWeight
		}
		if b.GetFlag(board.FlagCastled, c) {
			score += KingCastledWeight
		}
		for column := int8(0); column < board.Width; column++ {
			// duplicate score grows exponentially for each additional pawn
			score += PawnStructureWeight * PawnDuplicateWeight * ((1 << (eval.PawnColumns[c][column] - 1)) - 1)
		}
		// count possible moves
		score += PieceNumMovesWeight * int(eval.NumMoves[c])

		if c == p.PlayerColor {
			eval.TotalScore += score
		} else {
			eval.TotalScore -= score
		}
	}

	p.evaluationMap.Store(&hash, int32(eval.TotalScore))
	return eval
}
