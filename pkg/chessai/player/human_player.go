package player

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
)

type HumanPlayer struct {
	PlayerColor   byte
	TurnCount     int
	Move          chan *location.Move
}

func NewHumanPlayer(c color.Color) *HumanPlayer {
	p := &HumanPlayer{
		PlayerColor: c,
		TurnCount: 0,
		Move: make(chan *location.Move),
	}

	return p
}

func (p *HumanPlayer) WaitForMove() *location.Move {
	return <-p.Move
}

func (p *HumanPlayer) MakeMove(b *board.Board, move *location.Move) *board.LastMove {
	lastMove := board.MakeMove(move, b)
	p.TurnCount++
	return lastMove
}

func (p *HumanPlayer) String() string {
	c := "Black"
	if p.PlayerColor == color.White {
		c = "White"
	}
	return fmt.Sprintf("Human (%s)", c)
}
