package ai

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"math/rand"
)

func (p *Player) Random(b *board.Board) *ScoredMove {
	moves := *b.GetAllMoves(p.PlayerColor)
	idx := rand.Intn(len(moves))
	return &ScoredMove{
		Move: moves[idx],
	}
}
