package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
)

type LastMove struct {
	Piece          *Piece
	PromotionPiece *Piece
	Move           *location.Move
	IsCapture      bool
}
