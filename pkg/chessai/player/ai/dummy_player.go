package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"math/rand"
)

func (p *Player) RandomMove(b *board.Board, previousMove *board.LastMove) *ScoredMove {
	moves := *b.GetAllMoves(p.PlayerColor, previousMove)
	idx := rand.Intn(len(moves))
	return &ScoredMove{
		Move: moves[idx],
	}
}
