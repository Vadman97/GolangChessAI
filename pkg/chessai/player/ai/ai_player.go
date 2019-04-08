package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"log"
	"math"
	"math/rand"
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

const (
	OpeningNone = -1
)

// color -> list of openings: { list of moves }
var OpeningMoves = map[byte][][]*location.Move{
	color.Black: {{
		&location.Move{
			Start: location.NewLocation(board.StartRow[color.Black]["Pawn"], 4),
			End:   location.NewLocation(board.StartRow[color.Black]["Pawn"]+2, 4),
		},
		&location.Move{
			Start: location.NewLocation(board.StartRow[color.Black]["Piece"], 1),
			End:   location.NewLocation(board.StartRow[color.Black]["Piece"]+2, 2),
		},
		&location.Move{
			Start: location.NewLocation(board.StartRow[color.Black]["Piece"], 5),
			End:   location.NewLocation(board.StartRow[color.Black]["Piece"]+3, 2),
		},
	}},
	color.White: {{
		&location.Move{
			Start: location.NewLocation(board.StartRow[color.White]["Pawn"], 4),
			End:   location.NewLocation(board.StartRow[color.White]["Pawn"]-2, 4),
		},
		&location.Move{
			Start: location.NewLocation(board.StartRow[color.White]["Piece"], 6),
			End:   location.NewLocation(board.StartRow[color.White]["Piece"]-2, 5),
		},
		&location.Move{
			Start: location.NewLocation(board.StartRow[color.White]["Piece"], 5),
			End:   location.NewLocation(board.StartRow[color.White]["Piece"]-3, 2),
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
	p := &Player{
		Algorithm:                 AlgorithmAlphaBetaWithMemory,
		TranspositionTableEnabled: true,
		PlayerColor:               c,
		TurnCount:                 0,
		Opening:                   OpeningNone,
		Metrics:                   &Metrics{},
		Debug:                     true,
		evaluationMap:             util.NewConcurrentBoardMap(),
		alphaBetaTable:            util.NewTranspositionTable(),
	}
	if config.Get().UseOpenings {
		p.Opening = rand.Intn(len(OpeningMoves[c]))
	}
	return p
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
			Score: NegInf,
		}
		if p.Algorithm == AlgorithmMiniMax {
			m = p.MiniMax(b, p.MaxSearchDepth, p.PlayerColor, previousMove)
		} else if p.Algorithm == AlgorithmAlphaBetaWithMemory {
			m = p.AlphaBetaWithMemory(b, p.MaxSearchDepth, NegInf, PosInf, p.PlayerColor, previousMove)
		} else if p.Algorithm == AlgorithmMTDF {
			m = p.IterativeMTDF(b, m, previousMove)
		} else if p.Algorithm == AlgorithmRandom {
			m = p.RandomMove(b, previousMove)
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
	fmt.Print(result)
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

func (p *Player) ClearCaches() {
	p.evaluationMap = util.NewConcurrentBoardMap()
	p.alphaBetaTable = util.NewTranspositionTable()
}
