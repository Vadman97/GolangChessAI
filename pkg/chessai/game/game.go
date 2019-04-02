package game

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"time"
)

type Game struct {
	CurrentBoard     *board.Board
	CurrentTurnColor byte
	Players          map[byte]*ai.Player
	LastMoveTime     map[byte]time.Duration
	TotalMoveTime    map[byte]time.Duration
	MovesPlayed      uint
	PreviousMove     *board.LastMove
	GameStatus       byte
}

/**
 * Makes a move.  Returns boolean indicating if game is still active.
 */
func (g *Game) PlayTurn() bool {
	if g.GameStatus != Active {
		panic("Game is not active!")
	}
	start := time.Now()
	g.PreviousMove = g.Players[g.CurrentTurnColor].MakeMove(g.CurrentBoard, g.PreviousMove)
	g.LastMoveTime[g.CurrentTurnColor] = time.Now().Sub(start)
	g.TotalMoveTime[g.CurrentTurnColor] += g.LastMoveTime[g.CurrentTurnColor]
	g.CurrentTurnColor ^= 1
	g.MovesPlayed++
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
	}
	return g.GameStatus == Active
}

func (g *Game) Print() (result string) {
	// we just played white if we are now on black, show info for white
	if g.CurrentTurnColor == color.Black {
		result += fmt.Sprintf("White %s thought for %s\n", g.Players[color.White].Repr(), g.LastMoveTime[color.White])
	} else {
		result += fmt.Sprintf("Black %s thought for %s\n", g.Players[color.Black].Repr(), g.LastMoveTime[color.Black])
	}
	if g.MovesPlayed%2 == 0 {
		whiteAvg := g.TotalMoveTime[color.White].Seconds() / float64(g.MovesPlayed)
		blackAvg := g.TotalMoveTime[color.Black].Seconds() / float64(g.MovesPlayed)
		result += fmt.Sprintf("Average move time:\n")
		result += fmt.Sprintf("\t White: %fs\n", whiteAvg)
		result += fmt.Sprintf("\t Black: %fs\n", blackAvg)
	}
	result += fmt.Sprintf("Game state: %s", StatusStrings[g.GameStatus])
	return
}

func NewGame(whitePlayer, blackPlayer *ai.Player) *Game {
	g := Game{
		CurrentBoard: &board.Board{
			KingLocations: [color.NumColors]location.Location{
				{Row: 7, Col: 4},
				{Row: 0, Col: 4},
			},
		},
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
		MovesPlayed:  0,
		PreviousMove: nil,
		GameStatus:   Active,
	}
	g.CurrentBoard.ResetDefault()
	return &g
}
