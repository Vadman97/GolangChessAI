package player

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
)

type Player interface {
	MakeMove(b *board.Board, move *location.Move) *board.LastMove
	String() string
}
