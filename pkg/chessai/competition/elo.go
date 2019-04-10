package competition

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"math"
)

type Elo = int

type GameOutcome struct {
	Win [color.NumColors]bool
	Tie bool
}

func CalculateRatings(elos [color.NumColors]Elo, outcome GameOutcome) (newElos [color.NumColors]Elo) {
	var transformed [color.NumColors]float64
	var expScores [color.NumColors]float64
	var outcomeScores [color.NumColors]float64

	transformed[color.White] = math.Pow(10, float64(elos[color.White])/400)
	transformed[color.Black] = math.Pow(10, float64(elos[color.Black])/400)

	expScores[color.White] = transformed[color.White] / (transformed[color.White] + transformed[color.Black])
	expScores[color.Black] = transformed[color.Black] / (transformed[color.White] + transformed[color.Black])

	if outcome.Win[color.White] {
		outcomeScores[color.White] = 1
		outcomeScores[color.Black] = 0
	} else if outcome.Win[color.Black] {
		outcomeScores[color.White] = 0
		outcomeScores[color.Black] = 1
	} else if outcome.Tie {
		outcomeScores[color.White] = 0.5
		outcomeScores[color.Black] = 0.5
	}

	// according to https://en.wikipedia.org/wiki/Elo_rating_system#Most_accurate_K-factor, ICC uses non-staggered K=32
	const K = 32
	newElos[color.White] = elos[color.White] + Elo(math.Round(K*(outcomeScores[color.White]-expScores[color.White])))
	newElos[color.Black] = elos[color.Black] + Elo(math.Round(K*(outcomeScores[color.Black]-expScores[color.Black])))

	return
}
