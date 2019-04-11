package competition

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"math/rand"
	"testing"
)

func TestCompetition(t *testing.T) {
	// TODO(Vadim) remove this from a test
	t.Skip()
	rand.Seed(config.Get().TestRandSeed)
	comp := NewCompetition()
	comp.players[color.White].Algorithm = ai.AlgorithmMTDF
	comp.players[color.White].MaxSearchDepth = 4
	comp.players[color.Black].Algorithm = ai.AlgorithmRandom
	comp.RunCompetition()
}
