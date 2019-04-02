package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"log"
	"math"
	"os"
)

const (
	NegInf = math.MinInt32
	PosInf = math.MaxInt32
)

const (
	AlgorithmMiniMax             = "MiniMax"
	AlgorithmAlphaBetaWithMemory = "AlphaBetaMemory"
	AlgorithmMTDF                = "MTDF"
	AlgorithmRandom              = "Random"
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
	PieceValueWeight      = 1000
	PawnStructureWeight   = 500
	PieceAdvanceWeight    = 100
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

const (
	OpeningNone = -1
)

// color -> list of openings: { list of moves }
var OpeningMoves = map[byte][][]*location.Move{
	color.Black: {{
		&location.Move{
			Start: location.Location{Row: board.StartRow[color.Black]["Pawn"], Col: 4},
			End:   location.Location{Row: board.StartRow[color.Black]["Pawn"] + 2, Col: 4},
		},
		&location.Move{
			Start: location.Location{Row: board.StartRow[color.Black]["Piece"], Col: 1},
			End:   location.Location{Row: board.StartRow[color.Black]["Piece"] + 2, Col: 2},
		},
		&location.Move{
			Start: location.Location{Row: board.StartRow[color.Black]["Piece"], Col: 5},
			End:   location.Location{Row: board.StartRow[color.Black]["Piece"] + 3, Col: 2},
		},
	}},
	color.White: {{
		&location.Move{
			Start: location.Location{Row: board.StartRow[color.White]["Pawn"], Col: 4},
			End:   location.Location{Row: board.StartRow[color.White]["Pawn"] - 2, Col: 4},
		},
		&location.Move{
			Start: location.Location{Row: board.StartRow[color.White]["Piece"], Col: 6},
			End:   location.Location{Row: board.StartRow[color.White]["Piece"] - 2, Col: 5},
		},
		&location.Move{
			Start: location.Location{Row: board.StartRow[color.White]["Piece"], Col: 5},
			End:   location.Location{Row: board.StartRow[color.White]["Piece"] - 3, Col: 2},
		},
	}},
}

type ScoredMove struct {
	Move         location.Move
	MoveSequence []location.Move
	Score        int
}

type Player struct {
	Algorithm                 string
	TranspositionTableEnabled bool
	PlayerColor               byte
	MaxSearchDepth            int
	CurrentSearchDepth        int
	TurnCount                 int
	Opening                   int
	Metrics                   *Metrics

	evaluationMap  *util.ConcurrentBoardMap
	alphaBetaTable *util.TranspositionTable
	Debug          bool
}

func NewAIPlayer(c byte) *Player {
	return &Player{
		Algorithm:                 AlgorithmAlphaBetaWithMemory,
		TranspositionTableEnabled: true,
		PlayerColor:               c,
		MaxSearchDepth:            4,
		CurrentSearchDepth:        4,
		TurnCount:                 0,
		// Opening:        rand.Intn(len(OpeningMoves[c])),
		Opening:        OpeningNone,
		Metrics:        &Metrics{},
		Debug:          true,
		evaluationMap:  util.NewConcurrentBoardMap(),
		alphaBetaTable: util.NewTranspositionTable(),
	}
}

func betterMove(maximizingP bool, currentBest *ScoredMove, candidate *ScoredMove) bool {
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

func (p *Player) GetBestMove(b *board.Board, previousMove *board.LastMove) *location.Move {
	if p.Opening != OpeningNone && p.TurnCount < len(OpeningMoves[p.PlayerColor][p.Opening]) {
		return OpeningMoves[p.PlayerColor][p.Opening][p.TurnCount]
	} else {
		// reset metrics for each move
		p.Metrics = &Metrics{}

		var m = &ScoredMove{
			Score: 0,
		}
		if p.Algorithm == AlgorithmMiniMax {
			m = p.MiniMax(b, p.MaxSearchDepth, p.PlayerColor, previousMove)
		} else if p.Algorithm == AlgorithmAlphaBetaWithMemory {
			m = p.AlphaBetaWithMemory(b, p.MaxSearchDepth, NegInf, PosInf, p.PlayerColor, previousMove)
		} else if p.Algorithm == AlgorithmMTDF {
			m = p.IterativeMTDF(b, m, previousMove)
		} else if p.Algorithm == AlgorithmRandom {
			m = p.Random(b, previousMove)
		} else {
			panic("invalid ai algorithm")
		}
		if p.Debug {
			p.printMoveDebug(b, m)
		}
		return &m.Move
	}
}

func (p *Player) MakeMove(b *board.Board, previousMove *board.LastMove) *board.LastMove {
	move := board.MakeMove(p.GetBestMove(b, previousMove), b)
	p.TurnCount++
	return move
}

func (p *Player) EvaluateBoard(b *board.Board) *board.Evaluation {
	hash := b.Hash()
	if score, ok := p.evaluationMap.Read(&hash); ok {
		s := p.ensureScorePerspective(score.(int))
		return &board.Evaluation{
			TotalScore: s,
		}
	}

	// TODO(Vadim) make more intricate
	eval := board.NewEvaluation()

	for r := int8(0); r < board.Width; r++ {
		for c := int8(0); c < board.Height; c++ {
			if p := b.GetPiece(location.Location{Row: r, Col: c}); p != nil {
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
		for column := int8(0); column < board.Width; column++ {
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

	// technically ignores en passant, but that should be ok
	// TODO(Vadim) figure out if we can optimize, this makes very slow
	if b.IsInCheckmate(p.PlayerColor, nil) {
		eval.TotalScore = NegInf
	}
	/* if b.IsInCheckmate(p.PlayerColor, nil) {
		eval.TotalScore = NegInf
	} else if b.IsInCheckmate(p.PlayerColor ^ 1, nil) {
		eval.TotalScore = PosInf
	} else if b.IsStalemate(p.PlayerColor, nil) || b.IsStalemate(p.PlayerColor ^ 1, nil) {
		eval.TotalScore = 0
	} */

	p.evaluationMap.Store(&hash, int(eval.TotalScore))

	eval.TotalScore = p.ensureScorePerspective(eval.TotalScore)
	return eval
}

func (p *Player) ensureScorePerspective(score int) int {
	// if the search depth is odd, flip score
	if p.CurrentSearchDepth%2 == 1 {
		score = -score
	}
	return score
}

func (p *Player) Repr() string {
	c := "Black"
	if p.PlayerColor == color.White {
		c = "White"
	}
	return fmt.Sprintf("AI (%s,depth:%d - %s)", p.Algorithm, p.MaxSearchDepth, c)
}

func (p *Player) printMoveDebug(b *board.Board, m *ScoredMove) {
	const LogFile = "moveDebug.log"
	file, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot open file", err)
	}
	defer func() { _ = file.Close() }()
	var result string
	debugBoard := b.Copy()
	for i := len(m.MoveSequence) - 1; i >= 0; i-- {
		move := m.MoveSequence[i]
		start := debugBoard.GetPiece(move.Start)
		end := debugBoard.GetPiece(move.End)
		startStr, endStr := board.GetColorTypeRepr(start), board.GetColorTypeRepr(end)
		if end == nil {
			endStr = "_"
		}
		result += fmt.Sprintf("\t%s to %s\n", startStr, endStr)
		result += fmt.Sprintf("\t\t%s\n", move.Print())
		board.MakeMove(&move, debugBoard)
	}
	result += fmt.Sprintf("Board evaluation metrics\n")
	result += p.evaluationMap.PrintMetrics()
	result += fmt.Sprintf("Transposition table metrics\n")
	result += p.alphaBetaTable.PrintMetrics()
	result += fmt.Sprintf("Move cache metrics\n")
	result += b.MoveCache.PrintMetrics()
	result += fmt.Sprintf("Attack Move cache metrics\n")
	result += b.AttackableCache.PrintMetrics()
	result += fmt.Sprintf("\nAI %s best move leads to score %d\n", p.Repr(), m.Score)
	result += fmt.Sprintf("%s\n", p.Metrics.Print())
	result += fmt.Sprintf("%s best move leads to score %d\n", p.Repr(), m.Score)
	result += fmt.Sprintf("\n\n")
	_, _ = fmt.Fprint(file, result)
}
