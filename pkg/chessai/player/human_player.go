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
