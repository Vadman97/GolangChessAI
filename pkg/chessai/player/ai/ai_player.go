package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/transposition_table"
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
	AlgorithmAlphaBetaWithMemory = "α/β Memory"
	AlgorithmMTDf                = "MTDf"
	AlgorithmABDADA              = "ABDADA (α/β Parallel)"
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
	GetBestMove(*AIPlayer, *board.Board, *board.LastMove) *ScoredMove
}

type AIPlayer struct {
	Algorithm                 Algorithm
	TranspositionTableEnabled bool
	PlayerColor               byte
	MaxSearchDepth            int
	MaxThinkTime              time.Duration
	TurnCount                 int
	Opening                   int
	Metrics                   *Metrics

	Debug          bool
	PrintInfo      bool
	evaluationMap  *util.ConcurrentBoardMap
	alphaBetaTable *transposition_table.TranspositionTable
	printer        chan string
}

func NewAIPlayer(c byte, algorithm Algorithm) *AIPlayer {
	p := &AIPlayer{
		Algorithm:                 algorithm,
		TranspositionTableEnabled: config.Get().TranspositionTableEnabled,
		PlayerColor:               c,
		TurnCount:                 0,
		Opening:                   OpeningNone,
		Metrics:                   &Metrics{},
		Debug:                     config.Get().LogDebug,
		PrintInfo:                 config.Get().PrintPlayerInfo,
		evaluationMap:             util.NewConcurrentBoardMap(),
		alphaBetaTable:            transposition_table.NewTranspositionTable(),
		printer:                   make(chan string, 1000000),
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

func (p *AIPlayer) GetBestMove(b *board.Board, previousMove *board.LastMove, logger *PerformanceLogger) *location.Move {
	if p.Opening != OpeningNone && p.TurnCount < len(OpeningMoves[p.PlayerColor][p.Opening]) {
		return OpeningMoves[p.PlayerColor][p.Opening][p.TurnCount]
	} else {
		thinking := make(chan bool)
		go p.printThread(thinking)
		defer close(thinking)
		// reset metrics for each move
		p.Metrics = &Metrics{}

		if p.Algorithm != nil {
			scoredMove := p.Algorithm.GetBestMove(p, b, previousMove)
			if p.Debug {
				p.printMoveDebug(b, scoredMove)
			}
			logger.MarkPerformance(b, scoredMove, p)
			if scoredMove.Move.Start.Equals(scoredMove.Move.End) {
				p.printer <- fmt.Sprintf("%s resigns, no best move available. Picking random.\n", p)
				return &p.RandomMove(b, previousMove).Move
			}
			return &scoredMove.Move
		} else {
			panic("invalid ai algorithm")
		}
	}
}

func (p *AIPlayer) MakeMove(b *board.Board, move *location.Move) *board.LastMove {
	lastMove := board.MakeMove(move, b)
	p.TurnCount++
	return lastMove
}

func (p *AIPlayer) String() string {
	return fmt.Sprintf("AI (%s - %s)",
		p.Algorithm.GetName(), color.Names[p.PlayerColor])
}

func (p *AIPlayer) printMoveDebug(b *board.Board, m *ScoredMove) {
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
	result += fmt.Sprintf("%s\n", p.Metrics.Print())
	result += fmt.Sprintf("%s best move leads to score %d\n", p, m.Score)
	p.printer <- fmt.Sprint(result)
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

func (p *AIPlayer) ClearCaches() {
	// TODO(Vadim) find better way to pick when to clear, based on size #49
	p.evaluationMap = util.NewConcurrentBoardMap()
	p.alphaBetaTable = transposition_table.NewTranspositionTable()
}

func (p *AIPlayer) printThread(stop chan bool) {
	for {
		select {
		case <-stop:
			return
		default:
			util.PrintPrinter(p.printer, p.PrintInfo)
		}
	}
}
