package competition

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/game"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdateElo(t *testing.T) {
	newRatings := CalculateRatings([color.NumColors]Elo{2400, 2000}, game.Outcome{
		Win: [2]bool{true, false},
		Tie: false,
	})
	assert.Equal(t, 2403, newRatings[color.White])
	assert.Equal(t, 1997, newRatings[color.Black])

	newRatings = CalculateRatings([color.NumColors]Elo{2400, 2000}, game.Outcome{
		Win: [2]bool{false, true},
		Tie: false,
	})
	assert.Equal(t, 2371, newRatings[color.White])
	assert.Equal(t, 2029, newRatings[color.Black])

	newRatings = CalculateRatings([color.NumColors]Elo{2400, 2000}, game.Outcome{
		Win: [2]bool{false, false},
		Tie: true,
	})
	assert.Equal(t, 2387, newRatings[color.White])
	assert.Equal(t, 2013, newRatings[color.Black])
}
