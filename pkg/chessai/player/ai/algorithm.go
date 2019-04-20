package ai

import "github.com/Vadman97/ChessAI3/pkg/chessai/board"

const (
	AlgorithmMiniMax             = "MiniMax"
	AlgorithmAlphaBetaWithMemory = "α/β Memory"
	AlgorithmMTDf                = "MTDf"
	AlgorithmABDADA              = "ABDADA (α/β Parallel)"
	AlgorithmNegaScout           = "NegaScout"
	AlgorithmRandom              = "Random"
)

type Algorithm interface {
	GetName() string
	GetBestMove(*AIPlayer, *board.Board, *board.LastMove) *ScoredMove
}

var NameToAlgorithm = map[string]Algorithm{
	AlgorithmMiniMax: &MiniMax{},
	AlgorithmAlphaBetaWithMemory: &AlphaBetaWithMemory{},
	AlgorithmMTDf: &MTDf{},
	AlgorithmABDADA: &ABDADA{},
	AlgorithmNegaScout: &NegaScout{},
	AlgorithmRandom: &Random{},
}
