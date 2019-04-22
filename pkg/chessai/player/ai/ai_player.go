package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"log"
	"math/rand"
	"os"
	"runtime"
	"time"
)

const (
	PosInf       = int(1000000000)
	NegInf       = int(-PosInf)
	OnEvaluation = int(1111111111)
)

const (
	AlgorithmMiniMax             = "MiniMax"
	AlgorithmAlphaBetaWithMemory = "α/β Memory"
	AlgorithmMTDf                = "MTDf"
	AlgorithmABDADA              = "ABDADA (α/β Parallel)"
	AlgorithmNegaScout           = "NegaScout"
	AlgorithmRandom              = "Random"
	AlgorithmJamboree            = "Jamboree (α/β Parallel)"
)

const (
	OpeningNone = -1
)

// color -> list of openings: { list of moves }
var OpeningMoves = map[color.Color][][]*location.Move{
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
	Move           location.Move
	MoveSequence   []location.Move
	Score          int
	ReturnThisMove bool
}

func (s ScoredMove) NegScore() ScoredMove {
	s.Score = -s.Score
	return s
}

type AIPlayer struct {
	Algorithm                 Algorithm
	TranspositionTableEnabled bool
	PlayerColor               color.Color
	MaxSearchDepth            int
	MaxThinkTime              time.Duration
	LastSearchDepth           int
	LastSearchTime            time.Duration
	TurnCount                 int
	Opening                   int
	Metrics                   *Metrics

	Debug              bool
	PrintInfo          bool
	evaluationMap      *util.ConcurrentBoardMap
	transpositionTable *util.ConcurrentBoardMap
	printer            chan string
	abort              bool
}

func NewAIPlayer(c color.Color, algorithm Algorithm) *AIPlayer {
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
		transpositionTable:        util.NewConcurrentBoardMap(),
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
		p.abort = false
		// reset metrics for each move
		p.Metrics = &Metrics{}

		if p.Algorithm != nil {
			scoredMove := p.Algorithm.GetBestMove(p, b, previousMove)
			if p.Debug {
				p.printMoveDebug(b, scoredMove)
			}
			if logger != nil {
				logger.MarkPerformance(b, scoredMove, p)
			}
			if scoredMove.Move.Start.Equals(scoredMove.Move.End) {
				log.Printf("%s resigns, no best move available. Picking random.\n", p)
				return &(&Random{
					Rand: rand.New(rand.NewSource(time.Now().UnixNano())),
				}).RandomMove(b, p.PlayerColor, previousMove).Move
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

func (p AIPlayer) String() string {
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
		result += fmt.Sprintf("\t\t%s\n", move)
		board.MakeMove(&move, debugBoard)
	}
	result += fmt.Sprintf("%s\n", p.Metrics)
	result += fmt.Sprintf("%s best move leads to score %d\n", p, m.Score)
	p.printer <- fmt.Sprint(result)
	result += fmt.Sprintf("Board evaluation metrics\n")
	result += p.evaluationMap.String()
	result += fmt.Sprintf("Transposition table metrics\n")
	result += p.transpositionTable.String()
	if b.MoveCache != nil {
		result += fmt.Sprintf("Move cache metrics\n")
		result += b.MoveCache.String()
	}
	if b.AttackableCache != nil {
		result += fmt.Sprintf("Attack Move cache metrics\n")
		result += b.AttackableCache.String()
	}
	result += fmt.Sprintf("\n\n")
	_, _ = fmt.Fprint(file, result)
}

func (p *AIPlayer) ClearCaches(force bool) {
	cleared := false
	if force {
		log.Println("WARNING: Force clearing player caches (negative affects if during game)")
		p.evaluationMap = util.NewConcurrentBoardMap()
		p.transpositionTable = util.NewConcurrentBoardMap()
		cleared = true
	} else {
		if p.evaluationMap.GetTotalWrites() > 1000000 {
			log.Println("WARNING: Clearing player evaluation cache due to size")
			p.evaluationMap = util.NewConcurrentBoardMap()
			cleared = true
		}
		if p.transpositionTable.GetTotalWrites() > 1000000 {
			log.Println("WARNING: Clearing player transposition table due to size")
			p.transpositionTable = util.NewConcurrentBoardMap()
			cleared = true
		}
	}
	if cleared {
		runtime.GC()
		log.Println("Forcing garbage collection")
	}
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

func (p *AIPlayer) trackThinkTime(stop, done chan bool, start time.Time) {
	if p.MaxThinkTime != 0 {
		for {
			select {
			case <-stop:
				done <- true
				return
			default:
				thinkTime := time.Now().Sub(start)
				if thinkTime > p.MaxThinkTime {
					p.abort = true
					p.printer <- fmt.Sprintf("requesting AI hard abort, out of time!\n")
				}
			}
			time.Sleep(1 * time.Millisecond)
		}
	}
	done <- true
}

func (p *AIPlayer) terminalNode(b *board.Board, moves *[]location.Move) bool {
	return len(*moves) == 0 || b.PreviousPositionsSeen >= 3 || b.MovesSinceNoDraw >= 100
}
