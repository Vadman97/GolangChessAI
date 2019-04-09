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
	"time"
)

const (
	NegInf = math.MinInt32
	PosInf = math.MaxInt32
)

const (
	AlgorithmMiniMax             = "MiniMax"
	AlgorithmAlphaBetaWithMemory = "AlphaBetaMemory"
	AlgorithmMTDf                = "MTDf"
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

type Algorithm interface {
	GetName() string
	GetBestMove(*Player, *board.Board, *board.LastMove) *ScoredMove
}

type Player struct {
	Algorithm                 Algorithm
	TranspositionTableEnabled bool
	PlayerColor               byte
	MaxSearchDepth            int
	MaxThinkTime              time.Duration
	TurnCount                 int
	Opening                   int
	Metrics                   *Metrics

	evaluationMap  *util.ConcurrentBoardMap
	alphaBetaTable *util.TranspositionTable
	Debug          bool
}

func NewAIPlayer(c byte, algorithm Algorithm) *Player {
	p := &Player{
		Algorithm:                 algorithm,
		TranspositionTableEnabled: config.Get().TranspositionTableEnabled,
		PlayerColor:               c,
		TurnCount:                 0,
		Opening:                   OpeningNone,
		Metrics:                   &Metrics{},
		Debug:                     config.Get().LogDebug,
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

func (p *Player) GetBestMove(b *board.Board, previousMove *board.LastMove, logger *PerformanceLogger) *location.Move {
	if p.Opening != OpeningNone && p.TurnCount < len(OpeningMoves[p.PlayerColor][p.Opening]) {
		return OpeningMoves[p.PlayerColor][p.Opening][p.TurnCount]
	} else {
		// reset metrics for each move
		p.Metrics = &Metrics{}

		if p.Algorithm != nil {
			scoredMove := p.Algorithm.GetBestMove(p, b, previousMove)
			if p.Debug {
				p.printMoveDebug(b, scoredMove)
			}
			if scoredMove.Move.Start.Equals(scoredMove.Move.End) {
				log.Printf("%s resigns, no best move available. Picking random.\n", p.Repr())
				return &p.RandomMove(b, previousMove).Move
			}
			logger.MarkPerformance(b, scoredMove, p)
			if scoredMove.Move.Start.Equals(scoredMove.Move.End) {
				log.Printf("%s resigns, no best move available. Picking random.\n", p.Repr())
				return &p.RandomMove(b, previousMove).Move
			}
			return &scoredMove.Move
		} else {
			panic("invalid ai algorithm")
		}
	}
}

func (p *Player) MakeMove(b *board.Board, previousMove *board.LastMove, logger *PerformanceLogger) *board.LastMove {
	move := board.MakeMove(p.GetBestMove(b, previousMove, logger), b)
	p.TurnCount++
	return move
}

func (p *Player) Repr() string {
	c := "Black"
	if p.PlayerColor == color.White {
		c = "White"
	}
	return fmt.Sprintf("AI (%s,depth:%d - %s)", p.Algorithm.GetName(), p.MaxSearchDepth, c)
}

func (p *Player) printMoveDebug(b *board.Board, m *ScoredMove) {
	LogFile := config.Get().DebugLogFileName
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
	result += fmt.Sprintf("\nAI %s best move leads to score %d\n", p.Repr(), m.Score)
	result += fmt.Sprintf("%s\n", p.Metrics.Print())
	result += fmt.Sprintf("%s best move leads to score %d\n", p.Repr(), m.Score)
	fmt.Print(result)
	result += fmt.Sprintf("Board evaluation metrics\n")
	result += p.evaluationMap.PrintMetrics()
	result += fmt.Sprintf("Transposition table metrics\n")
	result += p.alphaBetaTable.PrintMetrics()
	result += fmt.Sprintf("Move cache metrics\n")
	result += b.MoveCache.PrintMetrics()
	result += fmt.Sprintf("Attack Move cache metrics\n")
	result += b.AttackableCache.PrintMetrics()
	result += fmt.Sprintf("\n\n")
	_, _ = fmt.Fprint(file, result)
}

func (p *Player) ClearCaches() {
	// TODO(Vadim) find better way to pick when to clear, based on size #49
	//p.evaluationMap = util.NewConcurrentBoardMap()
	//p.alphaBetaTable = util.NewTranspositionTable()
}
