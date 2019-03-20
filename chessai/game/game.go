package game

import (
	"ChessAI3/chessai/board"
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/player/ai"
	"time"
)

type Game struct {
	CurrentBoard     *board.Board
	CurrentTurnColor byte
	Players          map[byte]*ai.Player
	PlayTime         map[byte]time.Duration
	MovesPlayed      int
}

func (g *Game) PlayTurn() {
	g.Players[g.CurrentTurnColor].MakeMove(g.CurrentBoard)
	g.CurrentTurnColor = (g.CurrentTurnColor + 1) % color.NumColors
	g.MovesPlayed++
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
	}
	g.CurrentBoard.ResetDefault()
	return &g
}
