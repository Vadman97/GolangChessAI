package ai

import (
	"ChessAI3/chessai/board"
	"math/rand"
)

func (p *Player) Random(b *board.Board) *ScoredMove {
	moves := *b.GetAllMoves(p.PlayerColor)
	idx := rand.Intn(len(moves))
	return &ScoredMove{
		Move: moves[idx],
	}
}
