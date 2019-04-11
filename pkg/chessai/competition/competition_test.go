package competition

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

func TestCompetition(t *testing.T) {
	// TODO(Vadim) remove this from a test, output this to file and keep history of AI performance
	//t.Skip()
	rand.Seed(config.Get().TestRandSeed)
	comp := NewCompetition()
	comp.players[color.White].Algorithm = &ai.MTDf{}
	comp.players[color.White].MaxSearchDepth = 512
	comp.players[color.White].MaxThinkTime = 1 * time.Second
	comp.players[color.Black].Algorithm = &ai.Random{}
	// default opponent random
	comp.RunCompetition()
}

func TestCompetition_RecordOutcome(t *testing.T) {
	comp := NewCompetition()
	comp.whiteIndex = 0
	comp.blackIndex = 1
	comp.RecordOutcome(game.Outcome{
		Win: [2]bool{true, false},
		Tie: false,
	})
	assert.Equal(t, 1, comp.wins[color.White])
	assert.Equal(t, 0, comp.wins[color.Black])
	assert.Equal(t, 0, comp.ties)

	comp.whiteIndex = 1
	comp.blackIndex = 0
	comp.RecordOutcome(game.Outcome{
		Win: [2]bool{true, false},
		Tie: false,
	})
	assert.Equal(t, 1, comp.wins[color.White])
	assert.Equal(t, 1, comp.wins[color.Black])
	assert.Equal(t, 0, comp.ties)

	comp.whiteIndex = 0
	comp.blackIndex = 1
	comp.RecordOutcome(game.Outcome{
		Win: [2]bool{false, true},
		Tie: false,
	})
	assert.Equal(t, 1, comp.wins[color.White])
	assert.Equal(t, 2, comp.wins[color.Black])
	assert.Equal(t, 0, comp.ties)

	comp.RecordOutcome(game.Outcome{
		Win: [2]bool{false, false},
		Tie: true,
	})
	assert.Equal(t, 1, comp.wins[color.White])
	assert.Equal(t, 2, comp.wins[color.Black])
	assert.Equal(t, 1, comp.ties)
}
