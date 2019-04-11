package competition

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"math/rand"
	"time"
)

const NumberOfGames = 100
const StartingElo = Elo(1200)

type Competition struct {
	wins                   [color.NumColors]int
	ties                   int
	players                [color.NumColors]*ai.Player
	elos                   [color.NumColors]Elo
	gameNumber             int
	whiteIndex, blackIndex int
	competitionRand        *rand.Rand
}

func NewCompetition() (c *Competition) {
	c = &Competition{
		players: [2]*ai.Player{
			ai.NewAIPlayer(color.White, nil),
			ai.NewAIPlayer(color.Black, nil),
		},
		elos: [2]Elo{StartingElo, StartingElo},
	}
	return
}

// TODO(Vadim) add reading AI / outputting results to file

func (c *Competition) RunCompetition() {
	c.competitionRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	for c.gameNumber = 1; c.gameNumber <= NumberOfGames; c.gameNumber++ {
		fmt.Printf(c.Print())
		// randomize color of players each game
		c.randomizePlayers()
		g := game.NewGame(c.players[c.whiteIndex], c.players[c.blackIndex])
		c.disablePrinting(g)
		active := true
		for active {
			active = g.PlayTurn()
		}
		fmt.Println(g.Print())
		g.ClearCaches()
		outcome := c.derandomizeGameOutcome(g.GetGameOutcome())
		c.elos = CalculateRatings(c.elos, outcome)
		c.RecordOutcome(outcome)
	}
}

func (c *Competition) Print() (result string) {
	result += fmt.Sprintf("\n=== Game %d ===\n", c.gameNumber)
	result += fmt.Sprintf("\tWhite Elo: %d\n", c.elos[color.White])
	result += fmt.Sprintf("\tBlack Elo: %d\n", c.elos[color.Black])
	result += fmt.Sprintf("\tWW:%d,BW:%d,T:%d\n", c.wins[color.White], c.wins[color.Black], c.ties)
	return result
}

func (c *Competition) RecordOutcome(outcome game.Outcome) {
	if outcome.Win[color.White] {
		c.wins[color.White]++
	} else if outcome.Win[color.Black] {
		c.wins[color.Black]++
	} else if outcome.Tie {
		c.ties++
	}
}

/**
 * swap players randomly so the competition white players is swapped
 * competition maintains constant perspective of the two players
 */
func (c *Competition) randomizePlayers() {
	c.whiteIndex = c.competitionRand.Int() % color.NumColors
	c.blackIndex = c.whiteIndex ^ 1
	c.players[c.whiteIndex].PlayerColor = color.White
	c.players[c.blackIndex].PlayerColor = color.Black
}

/**
 * randomizePlayers will swap players in game, which affects outcome.
 * swap again to make outcome match our perspective on white/black
 */
func (c *Competition) derandomizeGameOutcome(out game.Outcome) game.Outcome {
	out.Win[color.White], out.Win[color.Black] = out.Win[c.whiteIndex], out.Win[c.blackIndex]
	return out
}

func (c *Competition) disablePrinting(g *game.Game) {
	g.PrintInfo = false
	c.players[color.White].PrintInfo = false
	c.players[color.Black].PrintInfo = false
}
