package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"time"
)

func (miniMax *MiniMax) MiniMaxRecurse(b *board.Board, m location.Move, depth int, currentPlayer color.Color,
	previousMove *board.LastMove) *ScoredMove {
	newBoard := b.Copy()
	previousMove = board.MakeMove(&m, newBoard)
	miniMax.player.Metrics.MovesConsidered++
	candidate := miniMax.MiniMax(newBoard, depth-1, currentPlayer^1, previousMove)
	candidate.Move = m
	candidate.MoveSequence = append(candidate.MoveSequence, m)
	return candidate
}

func (miniMax *MiniMax) MiniMax(b *board.Board, depth int, currentPlayer color.Color, previousMove *board.LastMove) *ScoredMove {
	if depth == 0 {
		return &ScoredMove{
			Score: miniMax.player.EvaluateBoard(b, miniMax.player.PlayerColor).TotalScore,
		}
	}

	var best ScoredMove
	if currentPlayer == miniMax.player.PlayerColor {
		// maximizing player
		best.Score = NegInf
	} else {
		// minimizing player
		best.Score = PosInf
	}
	moves := b.GetAllMoves(currentPlayer, previousMove)
	for _, m := range *moves {
		if miniMax.player.abort {
			break
		}
		candidate := miniMax.MiniMaxRecurse(b, m, depth, currentPlayer, previousMove)
		if betterMove(currentPlayer == miniMax.player.PlayerColor, &best, candidate) {
			best = *candidate
		}
	}

	return &best
}

func (miniMax *MiniMax) IterativeMiniMax(b *board.Board, previousMove *board.LastMove) *ScoredMove {
	miniMax.player.abort = false
	start := time.Now()
	best := &ScoredMove{}
	for miniMax.currentSearchDepth = 1; miniMax.currentSearchDepth <= miniMax.player.MaxSearchDepth; miniMax.currentSearchDepth += 1 {
		thinking, done := make(chan bool), make(chan bool, 1)
		go miniMax.player.trackThinkTime(thinking, done, start)
		miniMax.player.printer <- fmt.Sprintf("Start MM %s\n", miniMax.player.String())
		newBest := miniMax.MiniMax(b, miniMax.currentSearchDepth, miniMax.player.PlayerColor, previousMove)
		close(thinking)
		<-done
		// did not abort search, good value
		if !miniMax.player.abort {
			best = newBest
			miniMax.lastSearchDepth = miniMax.currentSearchDepth
			miniMax.player.printer <- fmt.Sprintf("Best D:%d M:%s\n", miniMax.lastSearchDepth, best.Move.Print())
		} else {
			// -1 due to discard of current level due to hard abort
			miniMax.lastSearchDepth = miniMax.currentSearchDepth - 1
			miniMax.player.printer <- fmt.Sprintf("MiniMax hard abort! evaluated to depth %d\n", miniMax.lastSearchDepth)
			break
		}
	}
	miniMax.lastSearchTime = time.Now().Sub(start)
	return best
}

type MiniMax struct {
	player             *AIPlayer
	lastSearchDepth    int
	currentSearchDepth int
	lastSearchTime     time.Duration
}

func (miniMax *MiniMax) GetName() string {
	return fmt.Sprintf("%s,[D:%d;T:%s]", AlgorithmMiniMax, miniMax.lastSearchDepth, miniMax.lastSearchTime)
}

func (miniMax *MiniMax) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	miniMax.player = p
	return miniMax.IterativeMiniMax(b, previousMove)
}
