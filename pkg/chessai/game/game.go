package game

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/api"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/gorilla/websocket"
	"log"
	"math"
	"runtime"
	"time"
)

type Game struct {
	CurrentBoard      *board.Board
	CurrentTurnColor  byte
	Players           map[byte]player.Player
	CurrentMoveTime   map[byte]time.Duration
	LastMoveTime      map[byte]time.Duration
	TotalMoveTime     map[byte]time.Duration
	MovesPlayed       uint
	PreviousMove      *board.LastMove
	GameStatus        byte
	CacheMemoryLimit  uint64
	MoveLimit         int32
	TimeLimit         time.Duration
	PerformanceLogger *ai.PerformanceLogger
	PrintInfo         bool
	SocketBroadcast   chan api.ChessMessage
	printer           chan string
	quit			  chan bool
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
	} else if g.GameStatus == Stalemate || g.GameStatus == FiftyMoveDraw || g.GameStatus == RepeatedActionThreeTimeDraw {
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
		g.printer <- fmt.Sprintf("\nPlayer %s thinking...\n", g.Players[g.CurrentTurnColor])
		start := time.Now()
		quitTimeUpdates := make(chan bool)
		// print think time for slow players, regardless of what's going on
		go g.periodicUpdates(quitTimeUpdates, start)

		var move *location.Move
		switch p := g.Players[g.CurrentTurnColor].(type) {
		case *player.HumanPlayer:
			move = p.WaitForMove()
		case *ai.AIPlayer:
			move = p.GetBestMove(g.CurrentBoard, g.PreviousMove, g.PerformanceLogger)
		}

		g.PreviousMove = g.Players[g.CurrentTurnColor].MakeMove(g.CurrentBoard, move)

		// quit time updates (never prints if quick player)
		close(quitTimeUpdates)
		g.UpdateTime(start)
		g.CurrentTurnColor ^= 1
		g.MovesPlayed++

		// check that the next player is not in checkmate
		// priority goes to win, then stalemate, then fifty move draw
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
		} else if g.CurrentBoard.MovesSinceNoDraw >= 100 {
			// 50 Move Rule (50 moves per color)
			g.GameStatus = FiftyMoveDraw
		}

		if g.GameStatus == Active {
			g.printer <- fmt.Sprintf("Move #%d by %s\n", g.MovesPlayed, color.Names[g.CurrentTurnColor^1])
		} else {
			g.printer <- fmt.Sprintf("Game Over! Result is: %s\n", StatusStrings[g.GameStatus])
		}
	}
	g.printer <- fmt.Sprintln(g)
	if g.GameStatus != Active {
		var aiPlayers []*ai.AIPlayer
		for c := color.White; c < color.NumColors; c++ {
			if aiPlayer, isAI := g.Players[c].(*ai.AIPlayer); isAI {
				aiPlayers = append(aiPlayers, aiPlayer)
			}
		}

		g.PerformanceLogger.CompletePerformanceLog(aiPlayers)
		g.printThread()
	}
	return g.GameStatus == Active
}

func (g *Game) Loop(client *websocket.Conn) {
	g.SocketBroadcast <- api.CreateChessMessage(api.GameState, g.GetJSON())

	var humanColor color.Color
	for c := color.White; c < color.NumColors; c++ {
		if _, isHuman := g.Players[c].(*player.HumanPlayer); isHuman {
			humanColor = c
		}
	}

	for i := 0; i < int(g.MoveLimit); i++ {
		select {
		case <-g.quit:
			break
		default:
			// TODO DEBUG (Remove below)
			log.Printf("Turn %d", i)
			CurrentTurnColor := g.CurrentTurnColor

			// Send Pre-Move Information
			if CurrentTurnColor == humanColor {
				availableMovesJSON := api.CreateAvailableMovesJSON(g.CurrentBoard.GetAllAvailableMoves(humanColor))
				g.SocketBroadcast <- api.CreateChessMessage(api.AvailablePlayerMoves, availableMovesJSON)
			}

			active := g.PlayTurn()

			// Send Post-Move Information
			if CurrentTurnColor != humanColor {
				lastMoveJSON := api.CreateMoveJSON(g.PreviousMove)
				g.SocketBroadcast <- api.CreateChessMessage(api.AIMove, lastMoveJSON)
			}

			if !active {
				break
			}
		}
	}
}

func (g Game) String() (result string) {
	// we just played white if we are now on black, show info for white
	result += fmt.Sprintln(g.CurrentBoard)
	result += g.PrintThinkTime(g.CurrentTurnColor^1, g.LastMoveTime)
	if g.MovesPlayed%2 == 0 || g.GameStatus != Active {
		whiteAvg := g.TotalMoveTime[color.White].Seconds() / float64(g.MovesPlayed/2)
		blackAvg := g.TotalMoveTime[color.Black].Seconds() / float64(g.MovesPlayed/2)
		result += fmt.Sprintf("Average move time:\n")
		result += fmt.Sprintf("\t White: %fs\n", whiteAvg)
		result += fmt.Sprintf("\t Black: %fs\n", blackAvg)
	}
	result += fmt.Sprintf("Total game duration: %s\n", g.GetTotalPlayTime())
	result += fmt.Sprintf("Total game turns: %d\n", (g.MovesPlayed-1)/2+1)
	result += fmt.Sprintf("Game state: %s", StatusStrings[g.GameStatus])
	return
}

func (g *Game) PrintThinkTime(c byte, moveTime map[byte]time.Duration) (result string) {
	if c == color.White {
		result += fmt.Sprintf("White %s thought for %s\n", g.Players[color.White], moveTime[color.White])
	} else {
		result += fmt.Sprintf("Black %s thought for %s\n", g.Players[color.Black], moveTime[color.Black])
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
			g.CurrentMoveTime[g.CurrentTurnColor] = time.Now().Sub(start)
			g.printer <- fmt.Sprintf("%s", g.PrintThinkTime(g.CurrentTurnColor, g.CurrentMoveTime))
			if aiPlayer, isAI := g.Players[g.CurrentTurnColor].(*ai.AIPlayer); isAI {
				g.printer <- fmt.Sprintf("\t%s\n\t", aiPlayer.Metrics)
			}
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

func (g *Game) ClearCaches(clearPlayers bool) {
	g.CurrentBoard.AttackableCache = util.NewConcurrentBoardMap()
	g.CurrentBoard.MoveCache = util.NewConcurrentBoardMap()
	if clearPlayers {
		for c := color.White; c < color.NumColors; c++ {
			if aiPlayer, isAI := g.Players[c].(*ai.AIPlayer); isAI {
				aiPlayer.ClearCaches()
			}
		}
	}
}

func (g *Game) GetTotalPlayTime() time.Duration {
	return g.TotalMoveTime[color.White] + g.TotalMoveTime[color.Black]
}

func (g *Game) GetJSON() *api.GameStateJSON {
	gameJSON := &api.GameStateJSON{
		CurrentBoard:     [board.Height][board.Width]*api.PieceJSON{},
		CurrentTurnColor: color.Names[g.CurrentTurnColor],
		MovesPlayed:      g.MovesPlayed,
		GameStatus:       StatusStrings[g.GameStatus],
		MoveLimit:        g.MoveLimit,
		TimeLimit:        g.TimeLimit,
	}

	// Set Human Color (if there is one)
	for c := color.White; c < color.NumColors; c++ {
		if _, isHuman := g.Players[c].(*player.HumanPlayer); isHuman {
			gameJSON.HumanColor = color.Names[c]
		}
	}

	// Set Board JSON
	for r := uint8(0); r < board.Height; r++ {
		for c := uint8(0); c < board.Width; c++ {
			pieceFromLoc := g.CurrentBoard.GetPiece(location.NewLocation(r, c))
			if pieceFromLoc == nil {
				continue
			}

			gameJSON.CurrentBoard[r][c] = &api.PieceJSON{
				Color: color.Names[pieceFromLoc.GetColor()],
				PieceType: piece.TypeToName[pieceFromLoc.GetPieceType()],
			}
		}
	}

	// Set PreviousMove
	if g.PreviousMove != nil {
		gameJSON.PreviousMove = api.CreateMoveJSON(g.PreviousMove)
	}

	return gameJSON
}

func (g *Game) memoryThread() {
	for g.GameStatus == Active {
		if util.GetMemoryUsed() > g.CacheMemoryLimit {
			g.printer <- fmt.Sprintf("Clearing caches\n")
			g.ClearCaches(false)
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
	util.PrintPrinter(g.printer, g.PrintInfo)
}

func NewGame(whitePlayer, blackPlayer player.Player) *Game {
	performanceLogger := ai.CreatePerformanceLogger(config.Get().LogPerformanceToExcel,
		config.Get().LogPerformance,
		config.Get().ExcelPerformanceFileName,
		config.Get().PerformanceLogFileName)
	g := Game{
		CurrentBoard:     &board.Board{},
		CurrentTurnColor: color.White,
		Players: map[byte]player.Player{
			color.White: whitePlayer,
			color.Black: blackPlayer,
		},
		TotalMoveTime: map[byte]time.Duration{
			color.White: 0,
			color.Black: 0,
		},
		CurrentMoveTime: map[byte]time.Duration{
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
		SocketBroadcast:   make(chan api.ChessMessage, 10),
		printer:           make(chan string, 100000),
		quit:              make(chan bool),
	}
	g.CurrentBoard.ResetDefault()
	go g.memoryThread()
	go g.printThread()
	return &g
}
