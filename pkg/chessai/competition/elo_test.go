package competition

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdateElo(t *testing.T) {
	newRatings := CalculateRatings([color.NumColors]Elo{2400, 2000}, GameOutcome{
		Win: [2]bool{true, false},
		Tie: false,
	})
	assert.Equal(t, 2403, newRatings[color.White])
	assert.Equal(t, 1997, newRatings[color.Black])

	newRatings = CalculateRatings([color.NumColors]Elo{2400, 2000}, GameOutcome{
		Win: [2]bool{false, true},
		Tie: false,
	})
	assert.Equal(t, 2371, newRatings[color.White])
	assert.Equal(t, 2029, newRatings[color.Black])

	newRatings = CalculateRatings([color.NumColors]Elo{2400, 2000}, GameOutcome{
		Win: [2]bool{false, false},
		Tie: true,
	})
	assert.Equal(t, 2387, newRatings[color.White])
	assert.Equal(t, 2013, newRatings[color.Black])
}
