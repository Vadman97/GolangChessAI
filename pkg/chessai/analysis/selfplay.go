package analysis

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

// BlunderThreshold is the centipawn loss that classifies a move as a blunder.
const BlunderThreshold = 100

// MoveRecord stores a single half-move from a self-play game along with Stockfish analysis.
type MoveRecord struct {
	MoveNum        int
	Color          color.Color
	FENBefore      string
	EngineMove     string // UCI notation
	StockfishBest  string // UCI notation
	CPBefore       int    // Stockfish eval before move (from White's perspective)
	CPAfter        int    // Stockfish eval after move (from White's perspective)
	CPLoss         int    // centipawn loss for the side to move (positive = worse)
	IsBlunder      bool
}

// GameReport holds all move records for one self-play game.
type GameReport struct {
	GameNum  int
	Moves    []MoveRecord
	Blunders []MoveRecord
}

// RunSelfPlayAnalysis plays numGames ABDADA vs ABDADA games, evaluates every move
// with Stockfish at the given depth, and prints a blunder report.
func RunSelfPlayAnalysis(stockfishPath string, numGames int, thinkTime time.Duration, sfDepth int) {
	sf, err := NewStockfishEngine(stockfishPath)
	if err != nil {
		log.Fatalf("Cannot start Stockfish: %v", err)
	}
	defer sf.Close()

	totalMoves, totalBlunders := 0, 0

	for gameNum := 1; gameNum <= numGames; gameNum++ {
		fmt.Printf("\n=== Game %d/%d ===\n", gameNum, numGames)
		report := playAndAnalyze(gameNum, sf, thinkTime, sfDepth)

		for _, rec := range report.Moves {
			side := "White"
			if rec.Color == color.Black {
				side = "Black"
			}
			flag := ""
			if rec.IsBlunder {
				flag = " *** BLUNDER ***"
			}
			fmt.Printf("  Move %2d (%s): %s  [SF best: %s]  cp_loss=%+d%s\n",
				rec.MoveNum, side, rec.EngineMove, rec.StockfishBest, rec.CPLoss, flag)
		}

		fmt.Printf("  Blunders this game: %d / %d moves\n", len(report.Blunders), len(report.Moves))
		totalMoves += len(report.Moves)
		totalBlunders += len(report.Blunders)

		if len(report.Blunders) > 0 {
			fmt.Println("  --- Blunder details ---")
			for _, b := range report.Blunders {
				side := "White"
				if b.Color == color.Black {
					side = "Black"
				}
				fmt.Printf("    Move %2d (%s): played %s, SF wanted %s, loss=%+d cp\n",
					b.MoveNum, side, b.EngineMove, b.StockfishBest, b.CPLoss)
				fmt.Printf("      FEN: %s\n", b.FENBefore)
			}
		}
	}

	fmt.Printf("\n=== Summary: %d blunders in %d moves (%.1f%%) ===\n",
		totalBlunders, totalMoves, 100.0*float64(totalBlunders)/float64(max(1, totalMoves)))
}

func playAndAnalyze(gameNum int, sf *StockfishEngine, thinkTime time.Duration, sfDepth int) GameReport {
	b := &board.Board{}
	b.ResetDefault()

	white := ai.NewAIPlayer(color.White, &ai.ABDADA{})
	black := ai.NewAIPlayer(color.Black, &ai.ABDADA{})
	white.MaxThinkTime = thinkTime
	black.MaxThinkTime = thinkTime
	white.MaxSearchDepth = math.MaxInt8
	black.MaxSearchDepth = math.MaxInt8

	report := GameReport{GameNum: gameNum}
	var previousMove *board.LastMove
	moveNum := 1
	currentColor := color.Color(color.White)

	const maxMoves = 200
	for halfMove := 0; halfMove < maxMoves*2; halfMove++ {
		moves := b.GetAllMoves(currentColor, previousMove)
		if len(*moves) == 0 {
			break
		}
		if b.MovesSinceNoDraw >= 100 || b.PreviousPositionsSeen >= 3 || b.IsInsufficientMaterial() {
			break
		}

		fen := BoardToFEN(b, currentColor, previousMove, moveNum)

		// Get Stockfish's evaluation and best move before the engine plays.
		// sfResult.CentipawnsSTM is from the *mover's* perspective (UCI convention).
		sfResult := sf.Analyze(fen, sfDepth)
		cpSTMBefore := sfResult.CentipawnsSTM

		// Ask ABDADA for its move.
		player := white
		if currentColor == color.Black {
			player = black
		}
		engineMove := player.GetBestMove(b, previousMove, nil)
		if engineMove == nil {
			break
		}
		engineUCI := MoveToUCI(*engineMove)

		// Apply the engine's move.
		moveCopy := location.Move{Start: engineMove.Start, End: engineMove.End}
		previousMove = board.MakeMove(&moveCopy, b)
		if currentColor == color.White {
			moveNum++
		}
		currentColor ^= 1

		// Evaluate the resulting position. sfAfter.CentipawnsSTM is from the
		// *next* player's perspective — i.e., the opponent of the side that just moved.
		fenAfter := BoardToFEN(b, currentColor, previousMove, moveNum)
		sfAfter := sf.Analyze(fenAfter, sfDepth)
		cpSTMAfterOpponent := sfAfter.CentipawnsSTM

		// Centipawn loss for the side that just moved:
		// cpSTMBefore  = mover's eval before their move (their perspective, higher = better)
		// cpSTMAfterOpponent = opponent's eval after the move (opponent's perspective)
		// Convert opponent's eval to mover's: negate it.
		// cpLoss = cpSTMBefore - (-cpSTMAfterOpponent) if negative, mover improved → clamp to 0.
		cpLoss := cpSTMBefore + cpSTMAfterOpponent
		if cpLoss < 0 {
			cpLoss = 0
		}

		rec := MoveRecord{
			MoveNum:       moveNum - 1,
			Color:         currentColor ^ 1, // the color that just moved
			FENBefore:     fen,
			EngineMove:    engineUCI,
			StockfishBest: sfResult.BestMove,
			CPBefore:      cpSTMBefore,
			CPAfter:       -cpSTMAfterOpponent, // converted to mover's perspective
			CPLoss:        cpLoss,
			IsBlunder:     cpLoss >= BlunderThreshold && engineUCI != sfResult.BestMove,
		}
		report.Moves = append(report.Moves, rec)
		if rec.IsBlunder {
			report.Blunders = append(report.Blunders, rec)
		}
	}
	return report
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
