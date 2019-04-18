package competition

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"math/rand"
	"time"
)

type Competition struct {
	NumberOfGames          int
	wins                   [color.NumColors]int
	ties                   int
	players                [color.NumColors]*ai.AIPlayer
	elos                   [color.NumColors]Elo
	gameNumber             int
	whiteIndex, blackIndex int
	competitionRand        *rand.Rand
}

func NewCompetition() (c *Competition) {
	var StartingElo = Elo(config.Get().StartingElo)
	c = &Competition{
		NumberOfGames: config.Get().NumberOfCompetitionGames,
		players: [2]*ai.AIPlayer{
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
	for c.gameNumber = 1; c.gameNumber <= c.NumberOfGames; c.gameNumber++ {
		fmt.Println(c)
		// randomize color of players each game
		c.randomizePlayers()
		g := game.NewGame(c.players[c.whiteIndex], c.players[c.blackIndex])
		c.disablePrinting(g)
		active := true
		for active {
			active = g.PlayTurn()
			fmt.Printf("#%d, T: %s, P: %s, memory: %s",
				g.MovesPlayed, g.GetTotalPlayTime(),
				c.players[g.CurrentTurnColor^1].String(), util.GetMemStatString())
		}
		fmt.Println(g)
		g.ClearCaches()
		outcome := c.derandomizeGameOutcome(g.GetGameOutcome())
		c.elos = CalculateRatings(c.elos, outcome)
		c.RecordOutcome(outcome)
	}
}

func (c *Competition) String() (result string) {
	result += fmt.Sprintf("\n=== Game %d ===\n", c.gameNumber)
	result += fmt.Sprintf("[%s] Elo: %d\n", c.players[color.White], c.elos[color.White])
	result += fmt.Sprintf("[%s] Elo: %d\n", c.players[color.Black], c.elos[color.Black])
	result += fmt.Sprintf("White Wins:Black Wins:Ties\t%d:%d:%d\n\n", c.wins[color.White], c.wins[color.Black], c.ties)
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

func (c *Competition) RunAICompetition() {
	// TODO(Vadim) output this to file and keep history of AI performance
	// TODO(Vadim) load ai from file
	rand.Seed(config.Get().TestRandSeed)
	c.players[color.White].Algorithm = &ai.MiniMax{}
	c.players[color.White].MaxSearchDepth = 128
	c.players[color.White].MaxThinkTime = 100 * time.Millisecond
	c.players[color.Black].Algorithm = &ai.ABDADA{}
	c.players[color.Black].MaxSearchDepth = 2
	c.players[color.Black].MaxThinkTime = 100 * time.Millisecond
	c.RunCompetition()
}
