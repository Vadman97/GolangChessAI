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
	moves := b.GetAllMoves(currentPlayer, previousMove)
	// max recursion or terminal node
	if depth == 0 || len(*moves) == 0 {
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
		miniMax.player.printer <- fmt.Sprintf("Start MM %s\n", miniMax.player)
		newBest := miniMax.MiniMax(b, miniMax.currentSearchDepth, miniMax.player.PlayerColor, previousMove)
		close(thinking)
		<-done
		// did not abort search, good value
		if !miniMax.player.abort {
			best = newBest
			miniMax.player.LastSearchDepth = miniMax.currentSearchDepth
			miniMax.player.printer <- fmt.Sprintf("Best D:%d M:%s\n", miniMax.player.LastSearchDepth, best.Move)
		} else {
			// -1 due to discard of current level due to hard abort
			miniMax.player.LastSearchDepth = miniMax.currentSearchDepth - 1
			miniMax.player.printer <- fmt.Sprintf("MiniMax hard abort! evaluated to depth %d\n", miniMax.player.LastSearchDepth)
			break
		}
	}
	return best
}

type MiniMax struct {
	player             *AIPlayer
	currentSearchDepth int
}

func (miniMax *MiniMax) GetName() string {
	return AlgorithmMiniMax
}

func (miniMax *MiniMax) GetBestMove(p *AIPlayer, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	miniMax.player = p
	if miniMax.player.MaxThinkTime != 0 {
		// time limited mode
		return miniMax.IterativeMiniMax(b, previousMove)
	} else {
		// strict depth mode
		miniMax.player.LastSearchDepth = p.MaxSearchDepth
		return miniMax.MiniMax(b, p.MaxSearchDepth, p.PlayerColor, previousMove)
	}
}
