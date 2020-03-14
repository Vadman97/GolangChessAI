package player

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
)

type Player interface {
	MakeMove(b *board.Board, move *location.Move) *board.LastMove
	String() string
}
