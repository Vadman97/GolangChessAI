package game

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
	"time"
)

const MovesToPlay = 1000
const TimeToPlay = 2 * time.Minute

func TestBoardAI(t *testing.T) {
	var algorithmsToTest = [...]ai.Algorithm{
		&ai.MTDf{},
		&ai.ABDADA{},
		&ai.MiniMax{},
	}

	rand.Seed(config.Get().TestRandSeed)
	for _, algorithm := range algorithmsToTest {
		runAITest(t, algorithm)
	}
}

func runAITest(t *testing.T, algorithm ai.Algorithm) {
	aiPlayerSmart := ai.NewAIPlayer(color.Black, algorithm)
	aiPlayerSmart.MaxSearchDepth = 100
	aiPlayerSmart.MaxThinkTime = 1000 * time.Millisecond
	aiPlayerDumb := ai.NewAIPlayer(color.White, &ai.Random{
		Rand: rand.New(rand.NewSource(config.Get().TestRandSeed)),
	})
	g := NewGame(aiPlayerDumb, aiPlayerSmart)
	aiPlayerSmart.PrintInfo = false
	aiPlayerDumb.PrintInfo = false
	g.MoveLimit = MovesToPlay
	g.TimeLimit = TimeToPlay

	for i := 0; i < MovesToPlay; i++ {
		active := g.PlayTurn()
		fmt.Printf(util.GetMemStatString())
		if !active {
			break
		}
	}
	smartScore := int64(aiPlayerSmart.EvaluateBoard(g.CurrentBoard, aiPlayerSmart.PlayerColor).TotalScore)
	dumbScore := int64(aiPlayerDumb.EvaluateBoard(g.CurrentBoard, aiPlayerDumb.PlayerColor).TotalScore)
	fmt.Printf("Good %s E %d\n", aiPlayerSmart, smartScore)
	fmt.Printf("Bad %s E %d\n", aiPlayerDumb, dumbScore)
	if smartScore > dumbScore {
		assert.True(t, smartScore > dumbScore)
	} else {
		assert.True(t, smartScore-dumbScore <= int64(ai.PawnValueWeight*ai.PieceValue[piece.PawnType]))
	}
}

// ABDADA vs. Random, diff depths
// Depth:             1,  2,  3,   4
// Moves (ply):      71, 43, 17,  61
// Total game turns: 36, 22,  9,  31
// Time (sec):        1,  3, 38, 352
