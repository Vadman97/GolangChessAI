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
	PlayTime         map[byte]time.Duration
	MovesPlayed      uint
	PreviousMove     *board.LastMove
	GameStatus       byte
}

func (g *Game) PlayTurn() {
	start := time.Now()
	g.PreviousMove = g.Players[g.CurrentTurnColor].MakeMove(g.CurrentBoard, g.PreviousMove)
	g.PlayTime[g.CurrentTurnColor] += time.Now().Sub(start)
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
	}
}

func (g *Game) Print() (result string) {
	result += fmt.Sprintf("White %s has thought for %s\n", g.Players[color.White].Repr(), g.PlayTime[color.White])
	result += fmt.Sprintf("Black %s has thought for %s", g.Players[color.Black].Repr(), g.PlayTime[color.Black])
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
		PlayTime: map[byte]time.Duration{
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
