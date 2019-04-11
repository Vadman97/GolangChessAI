package game

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"math"
	"runtime"
	"time"
)

type Game struct {
	CurrentBoard     *board.Board
	CurrentTurnColor byte
	Players          map[byte]*ai.Player
	LastMoveTime     map[byte]time.Duration
	TotalMoveTime    map[byte]time.Duration
	MovesPlayed      uint

	PreviousMove      *board.LastMove
	GameStatus        byte
	CacheMemoryLimit  uint64
	MoveLimit         int32
	TimeLimit         time.Duration
	PerformanceLogger *ai.PerformanceLogger
	PrintInfo         bool
	printer           chan string
}

type Outcome struct {
	Win [color.NumColors]bool
	Tie bool
}

func (g *Game) GetGameOutcome() (outcome Outcome) {
	if g.GameStatus == WhiteWin {
		outcome.Win[color.White] = true
	} else if g.GameStatus == BlackWin {
		outcome.Win[color.Black] = true
	} else if g.GameStatus == Stalemate {
		outcome.Tie = true
	}
	return
}

/**
 * Makes a move.  Returns boolean indicating if game is still active.
 */
func (g *Game) PlayTurn() bool {
	if g.GameStatus != Active {
		panic("Game is not active!")
	}

	if g.MovesPlayed%2 == 0 && g.GetTotalPlayTime() > g.TimeLimit {
		g.printer <- fmt.Sprintf("Aborting - out of time\n")
		g.GameStatus = Aborted
	} else {
		g.printer <- fmt.Sprintf("\nPlayer %s thinking...\n", g.Players[g.CurrentTurnColor].Repr())
		start := time.Now()
		quitTimeUpdates := make(chan bool)
		// print think time for slow players, regardless of what's going on
		go g.periodicUpdates(quitTimeUpdates, start)
		g.PreviousMove = g.Players[g.CurrentTurnColor].MakeMove(g.CurrentBoard, g.PreviousMove, g.PerformanceLogger)
		// quit time updates (never prints if quick player)
		close(quitTimeUpdates)
		g.UpdateTime(start)
		g.CurrentTurnColor ^= 1
		g.MovesPlayed++

		g.CurrentBoard.UpdateDrawCounter(g.PreviousMove)

		if g.CurrentBoard.IsInCheckmate(g.CurrentTurnColor, g.PreviousMove) {
			if g.CurrentTurnColor == color.White {
				g.GameStatus = BlackWin
			} else {
				g.GameStatus = WhiteWin
			}
		} else if g.CurrentBoard.IsStalemate(g.CurrentTurnColor, g.PreviousMove) {
			g.GameStatus = Stalemate
		} else if g.CurrentBoard.IsStalemate(g.CurrentTurnColor^1, g.PreviousMove) {
			g.GameStatus = Stalemate
		} else if g.GameStatus == Active && g.CurrentBoard.MovesSinceNoDraw >= 100 {
			// 50 Move Rule (50 moves per color)
			g.GameStatus = Stalemate
		}

		if g.GameStatus == Active {
			g.printer <- fmt.Sprintf("Move #%d by %s\n", g.MovesPlayed, color.Names[g.CurrentTurnColor^1])
		} else {
			g.printer <- fmt.Sprintf("Game Over! Result is: %s\n", StatusStrings[g.GameStatus])
		}
	}
	if g.GameStatus != Active {
		g.PerformanceLogger.CompletePerformanceLog(g.Players[color.White], g.Players[color.Black])
	}
	g.printer <- fmt.Sprintln(g.Print())
	return g.GameStatus == Active
}

func (g *Game) Print() (result string) {
	// we just played white if we are now on black, show info for white
	result += fmt.Sprintln(g.CurrentBoard.Print())
	result += g.PrintThinkTime(g.CurrentTurnColor ^ 1)
	whiteAvg := g.TotalMoveTime[color.White].Seconds() / float64(g.MovesPlayed/2)
	blackAvg := g.TotalMoveTime[color.Black].Seconds() / float64(g.MovesPlayed/2)
	result += fmt.Sprintf("Average move time:\n")
	result += fmt.Sprintf("\t White: %fs\n", whiteAvg)
	result += fmt.Sprintf("\t Black: %fs\n", blackAvg)
	result += fmt.Sprintf("Total game duration: %s\n", g.GetTotalPlayTime())
	result += fmt.Sprintf("Total game turns: %d\n", g.MovesPlayed/2)
	result += fmt.Sprintf("Game state: %s", StatusStrings[g.GameStatus])
	return
}

func (g *Game) PrintThinkTime(c byte) (result string) {
	if c == color.White {
		result += fmt.Sprintf("White %s thought for %s\n", g.Players[color.White].Repr(), g.LastMoveTime[color.White])
	} else {
		result += fmt.Sprintf("Black %s thought for %s\n", g.Players[color.Black].Repr(), g.LastMoveTime[color.Black])
	}
	return
}

func (g *Game) periodicUpdates(stop chan bool, start time.Time) {
	// only start printing if the player is thinking for more than 30 sec
	time.Sleep(30 * time.Second)
	for {
		select {
		case <-stop:
			return
		default:
			g.UpdateTime(start)
			g.printer <- fmt.Sprintf("%s", g.PrintThinkTime(g.CurrentTurnColor))
			g.printer <- fmt.Sprintf("\t%s\n\t", g.Players[g.CurrentTurnColor].Metrics.Print())
			g.printer <- util.GetMemStatString()
			g.printer <- fmt.Sprintln()
			// TODO(Vadim) decide if any other player things to print here
		}
		time.Sleep(30 * time.Second)
	}
}

func (g *Game) UpdateTime(start time.Time) {
	g.LastMoveTime[g.CurrentTurnColor] = time.Now().Sub(start)
	g.TotalMoveTime[g.CurrentTurnColor] += g.LastMoveTime[g.CurrentTurnColor]
}

func (g *Game) ClearCaches() {
	g.CurrentBoard.AttackableCache = util.NewConcurrentBoardMap()
	g.CurrentBoard.MoveCache = util.NewConcurrentBoardMap()
	for c := color.White; c < color.NumColors; c++ {
		g.Players[c].ClearCaches()
	}
}

func (g *Game) GetTotalPlayTime() time.Duration {
	return g.TotalMoveTime[color.White] + g.TotalMoveTime[color.Black]
}

func (g *Game) memoryThread() {
	for g.GameStatus == Active {
		if util.GetMemoryUsed() > g.CacheMemoryLimit {
			g.printer <- fmt.Sprintf("Clearing caches\n")
			g.ClearCaches()
			runtime.GC()
			g.printer <- fmt.Sprintf("Cleared!\n")
			g.printer <- util.GetMemStatString()
		}
		time.Sleep(1 * time.Second)
	}
}

func (g *Game) printThread() {
	for g.GameStatus == Active {
		util.PrintPrinter(g.printer, g.PrintInfo)
	}
}

func NewGame(whitePlayer, blackPlayer *ai.Player) *Game {
	performanceLogger := ai.CreatePerformanceLogger(config.Get().LogPerformanceToExcel,
		config.Get().LogPerformance,
		config.Get().ExcelPerformanceFileName,
		config.Get().PerformanceLogFileName)
	g := Game{
		CurrentBoard:     &board.Board{},
		CurrentTurnColor: color.White,
		Players: map[byte]*ai.Player{
			color.White: whitePlayer,
			color.Black: blackPlayer,
		},
		TotalMoveTime: map[byte]time.Duration{
			color.White: 0,
			color.Black: 0,
		},
		LastMoveTime: map[byte]time.Duration{
			color.White: 0,
			color.Black: 0,
		},
		MovesPlayed:       0,
		PreviousMove:      nil,
		GameStatus:        Active,
		CacheMemoryLimit:  config.Get().MemoryLimit,
		MoveLimit:         math.MaxInt32,
		TimeLimit:         math.MaxInt64,
		PerformanceLogger: performanceLogger,
		PrintInfo:         true,
		printer:           make(chan string, 100000),
	}
	g.CurrentBoard.ResetDefault()
	go g.memoryThread()
	go g.printThread()
	return &g
}
