package competition

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
)

const NumberOfGames = 100
const StartingElo = Elo(1200)

type Competition struct {
	wins       [color.NumColors]int
	ties       int
	players    [color.NumColors]*ai.Player
	elos       [color.NumColors]Elo
	gameNumber int
}

func NewCompetition() *Competition {
	return &Competition{
		players: [2]*ai.Player{
			ai.NewAIPlayer(color.White),
			ai.NewAIPlayer(color.Black),
		},
		elos: [2]Elo{StartingElo, StartingElo},
	}
}

// TODO(Vadim) add reading AI / outputting results to file

func (c *Competition) RunCompetition() {
	for c.gameNumber = 1; c.gameNumber <= NumberOfGames; c.gameNumber++ {
		fmt.Printf(c.Print())
		g := game.NewGame(c.players[color.White], c.players[color.Black])
		active := true
		for active {
			active = g.PlayTurn()
			util.PrintMemStats()
		}
		g.ClearCaches()
		outcome := g.GetGameOutcome()
		c.elos = CalculateRatings(c.elos, outcome)
		if outcome.Win[color.White] {
			c.wins[color.White]++
		} else if outcome.Win[color.Black] {
			c.wins[color.Black]++
		} else if outcome.Tie {
			c.ties++
		}
	}
}

func (c *Competition) Print() (result string) {
	result += fmt.Sprintf("\n\n\n=== Game %d ===\n", c.gameNumber)
	result += fmt.Sprintf("\tWhite Elo: %d\n", c.elos[color.White])
	result += fmt.Sprintf("\tBlack Elo: %d\n", c.elos[color.Black])
	result += fmt.Sprintf("\tWW:%d,BW:%d,T:%d\n", c.wins[color.White], c.wins[color.Black], c.ties)
	return result
}
