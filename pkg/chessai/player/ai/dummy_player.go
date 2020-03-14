package ai

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"math/rand"
	"time"
)

func (m *Random) RandomMove(b *board.Board, c color.Color, previousMove *board.LastMove) *ScoredMove {
	moves := *b.GetAllMovesUnShuffled(c, previousMove)
	idx := m.Rand.Intn(len(moves))
	return &ScoredMove{
		Move: moves[idx],
	}
}

type Random struct {
	Rand *rand.Rand
}

func (m *Random) GetName() string {
	return AlgorithmRandom
}

func (m *Random) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	if m.Rand == nil {
		m.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return m.RandomMove(b, p.PlayerColor, previousMove)
}
