package competition

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/game"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"strings"
	"time"
)

type tournamentPlayer struct {
	name      string
	algorithm ai.Algorithm
	elo       Elo
	wins      int
	draws     int
	losses    int
}

// matchupRecord holds W/D/L from player A's perspective vs player B.
type matchupRecord struct {
	wins, draws, losses int
}

// RunTournament runs a round-robin tournament among all algorithms.
// Each ordered pair plays gamesPerMatchup games (colors are alternated within
// each pair so each algorithm plays the same number of games as white and black).
func RunTournament(gamesPerMatchup int, thinkTime time.Duration) {
	startingElo := Elo(1200)

	players := []*tournamentPlayer{
		{name: "MiniMax", algorithm: &ai.MiniMax{}, elo: startingElo},
		{name: "AlphaBeta", algorithm: &ai.AlphaBetaWithMemory{}, elo: startingElo},
		{name: "MTDf", algorithm: &ai.MTDf{}, elo: startingElo},
		{name: "ABDADA", algorithm: &ai.ABDADA{}, elo: startingElo},
		{name: "NegaScout", algorithm: &ai.NegaScout{}, elo: startingElo},
		{name: "Jamboree", algorithm: &ai.Jamboree{}, elo: startingElo},
		{name: "Random", algorithm: &ai.Random{Rand: rand.New(rand.NewSource(time.Now().UnixNano()))}, elo: startingElo},
	}

	n := len(players)
	// results[i][j] = record for player i vs player j (from i's perspective)
	results := make([][]matchupRecord, n)
	for i := range results {
		results[i] = make([]matchupRecord, n)
	}

	totalMatchups := n * (n - 1) / 2
	matchupNum := 0

	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			matchupNum++
			fmt.Printf("\n[%d/%d] %s vs %s (%d games, %s/move)\n",
				matchupNum, totalMatchups,
				players[i].name, players[j].name,
				gamesPerMatchup, thinkTime)

			for g := 0; g < gamesPerMatchup; g++ {
				// Alternate who plays white each game.
				var whiteIdx, blackIdx int
				if g%2 == 0 {
					whiteIdx, blackIdx = i, j
				} else {
					whiteIdx, blackIdx = j, i
				}

				outcome := playGame(players[whiteIdx], players[blackIdx], thinkTime)

				// Record from i's perspective.
				var iWon, jWon bool
				if whiteIdx == i {
					iWon = outcome.Win[color.White]
					jWon = outcome.Win[color.Black]
				} else {
					iWon = outcome.Win[color.Black]
					jWon = outcome.Win[color.White]
				}

				switch {
				case iWon:
					results[i][j].wins++
					results[j][i].losses++
					players[i].wins++
					players[j].losses++
				case jWon:
					results[i][j].losses++
					results[j][i].wins++
					players[j].wins++
					players[i].losses++
				default:
					results[i][j].draws++
					results[j][i].draws++
					players[i].draws++
					players[j].draws++
				}

				// Update Elo after each game.
				eloArr := [color.NumColors]Elo{players[i].elo, players[j].elo}
				eloArr = CalculateRatings(eloArr, game.Outcome{
					Win: [color.NumColors]bool{iWon, jWon},
					Tie: !iWon && !jWon,
				})
				players[i].elo = eloArr[color.White]
				players[j].elo = eloArr[color.Black]

				result := "draw"
				if iWon {
					result = players[i].name + " wins"
				} else if jWon {
					result = players[j].name + " wins"
				}
				fmt.Printf("  Game %d (%s=White, %s=Black): %s\n",
					g+1, players[whiteIdx].name, players[blackIdx].name, result)
			}

			fmt.Printf("  Subtotal: %s %d-%d-%d %s\n",
				players[i].name,
				results[i][j].wins, results[i][j].draws, results[i][j].losses,
				players[j].name)
		}
	}

	printTournamentResults(players, results)
}

func playGame(white, black *tournamentPlayer, thinkTime time.Duration) game.Outcome {
	wp := ai.NewAIPlayer(color.White, white.algorithm)
	bp := ai.NewAIPlayer(color.Black, black.algorithm)
	wp.MaxSearchDepth = math.MaxUint8
	bp.MaxSearchDepth = math.MaxUint8
	wp.MaxThinkTime = thinkTime
	bp.MaxThinkTime = thinkTime
	wp.PrintInfo = false
	bp.PrintInfo = false

	g := game.NewGame(wp, bp)
	g.PrintInfo = false

	for g.PlayTurn() {
	}

	runtime.GC()
	return g.GetGameOutcome()
}

func printTournamentResults(players []*tournamentPlayer, results [][]matchupRecord) {
	// Sort by Elo descending for the leaderboard.
	ranked := make([]*tournamentPlayer, len(players))
	copy(ranked, players)
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].elo > ranked[j].elo
	})

	fmt.Println()
	fmt.Println(strings.Repeat("=", 72))
	fmt.Println("TOURNAMENT COMPLETE")
	fmt.Println(strings.Repeat("=", 72))

	// Results matrix — W/D/L from the row player's perspective.
	nameWidth := 12
	cellWidth := 9
	fmt.Printf("\nResults Matrix (W/D/L from row player's perspective):\n\n")

	// Header row.
	fmt.Printf("%-*s", nameWidth, "")
	for _, p := range players {
		fmt.Printf("| %-*s", cellWidth, truncate(p.name, cellWidth))
	}
	fmt.Println("|")
	fmt.Println(strings.Repeat("-", nameWidth+len(players)*(cellWidth+2)+1))

	// Data rows.
	for i, pi := range players {
		fmt.Printf("%-*s", nameWidth, truncate(pi.name, nameWidth))
		for j, pj := range players {
			if i == j {
				fmt.Printf("| %-*s", cellWidth, "---")
			} else {
				_ = pj
				cell := fmt.Sprintf("%d/%d/%d", results[i][j].wins, results[i][j].draws, results[i][j].losses)
				fmt.Printf("| %-*s", cellWidth, cell)
			}
		}
		fmt.Println("|")
	}

	// Leaderboard.
	fmt.Printf("\nLeaderboard:\n\n")
	fmt.Printf("%-4s %-14s %6s %5s %5s %5s  %s\n", "Rank", "Algorithm", "Elo", "W", "D", "L", "Score%")
	fmt.Println(strings.Repeat("-", 54))
	for rank, p := range ranked {
		total := p.wins + p.draws + p.losses
		var pct float64
		if total > 0 {
			pct = (float64(p.wins) + float64(p.draws)*0.5) / float64(total) * 100
		}
		fmt.Printf("%-4d %-14s %6d %5d %5d %5d  %.1f%%\n",
			rank+1, p.name, p.elo, p.wins, p.draws, p.losses, pct)
	}
	fmt.Println()
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}
