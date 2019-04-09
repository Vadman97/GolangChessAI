package ai

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
)

func (miniMax *MiniMax) MiniMaxRecurse(b *board.Board, m location.Move, depth int, currentPlayer byte,
	previousMove *board.LastMove) *ScoredMove {
	newBoard := b.Copy()
	previousMove = board.MakeMove(&m, newBoard)
	miniMax.player.Metrics.MovesConsidered++
	candidate := miniMax.MiniMax(newBoard, depth-1, currentPlayer^1, previousMove)
	candidate.Move = m
	candidate.MoveSequence = append(candidate.MoveSequence, m)
	return candidate
}

func (miniMax *MiniMax) MiniMax(b *board.Board, depth int, currentPlayer byte, previousMove *board.LastMove) *ScoredMove {
	if depth == 0 {
		return &ScoredMove{
			Score: miniMax.player.EvaluateBoard(b).TotalScore,
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
		candidate := miniMax.MiniMaxRecurse(b, m, depth, currentPlayer, previousMove)
		if betterMove(currentPlayer == miniMax.player.PlayerColor, &best, candidate) {
			best = *candidate
		}
	}

	return &best
}

type MiniMax struct {
	player *Player
}

func (miniMax *MiniMax) GetName() string {
	return fmt.Sprintf("%s,depth:%d", AlgorithmMiniMax, miniMax.player.MaxSearchDepth)
}

func (miniMax *MiniMax) GetBestMove(p *Player, b *board.Board, previousMove *board.LastMove) *ScoredMove {
	miniMax.player = p
	return miniMax.MiniMax(b, p.MaxSearchDepth, p.PlayerColor, previousMove)
}
