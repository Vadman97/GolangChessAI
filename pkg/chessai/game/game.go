package game

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
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
}

func (g *Game) PlayTurn() {
	start := time.Now()
	g.PreviousMove = g.Players[g.CurrentTurnColor].MakeMove(g.CurrentBoard, g.PreviousMove)
	g.PlayTime[g.CurrentTurnColor] += time.Now().Sub(start)
	g.CurrentTurnColor ^= 1
	g.MovesPlayed++
}

func (g *Game) Print() (result string) {
	result += fmt.Sprintf("White %s has thought for %s\n", g.Players[color.White].Repr(), g.PlayTime[color.White])
	result += fmt.Sprintf("Black %s has thought for %s", g.Players[color.Black].Repr(), g.PlayTime[color.Black])
	return
}

func NewGame(whitePlayer, blackPlayer *ai.Player) *Game {
	g := Game{
		CurrentBoard:     &board.Board{},
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
	}
	g.CurrentBoard.ResetDefault()
	return &g
}
