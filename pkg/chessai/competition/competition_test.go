package competition

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"math/rand"
	"testing"
	"time"
)

func TestCompetition(t *testing.T) {
	// TODO(Vadim) remove this from a test, output this to file and keep history of AI performance
	t.Skip()
	rand.Seed(config.Get().TestRandSeed)
	comp := NewCompetition()
	comp.players[color.White].Algorithm = &ai.MTDf{}
	comp.players[color.White].MaxSearchDepth = 512
	comp.players[color.White].MaxThinkTime = 1 * time.Second
	// default opponent random
	comp.RunCompetition()
}
