package competition

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCompetition_RecordOutcome(t *testing.T) {
	comp := NewCompetition()
	comp.whiteIndex = 0
	comp.blackIndex = 1
	comp.RecordOutcome(comp.derandomizeGameOutcome(game.Outcome{
		Win: [2]bool{true, false},
		Tie: false,
	}))
	assert.Equal(t, 1, comp.wins[color.White])
	assert.Equal(t, 0, comp.wins[color.Black])
	assert.Equal(t, 0, comp.ties)

	comp.whiteIndex = 1
	comp.blackIndex = 0
	comp.RecordOutcome(comp.derandomizeGameOutcome(game.Outcome{
		Win: [2]bool{true, false},
		Tie: false,
	}))
	assert.Equal(t, 1, comp.wins[color.White])
	assert.Equal(t, 1, comp.wins[color.Black])
	assert.Equal(t, 0, comp.ties)

	comp.whiteIndex = 0
	comp.blackIndex = 1
	comp.RecordOutcome(comp.derandomizeGameOutcome(game.Outcome{
		Win: [2]bool{false, true},
		Tie: false,
	}))
	assert.Equal(t, 1, comp.wins[color.White])
	assert.Equal(t, 2, comp.wins[color.Black])
	assert.Equal(t, 0, comp.ties)

	comp.RecordOutcome(comp.derandomizeGameOutcome(game.Outcome{
		Win: [2]bool{false, false},
		Tie: true,
	}))
	assert.Equal(t, 1, comp.wins[color.White])
	assert.Equal(t, 2, comp.wins[color.Black])
	assert.Equal(t, 1, comp.ties)
}
