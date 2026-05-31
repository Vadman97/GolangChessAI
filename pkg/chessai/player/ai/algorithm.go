package ai

import "github.com/Vadman97/GolangChessAI/pkg/chessai/board"

const (
	AlgorithmMiniMax             = "MiniMax"
	AlgorithmAlphaBetaWithMemory = "α/β Memory"
	AlgorithmMTDf                = "MTDf"
	AlgorithmABDADA              = "ABDADA (α/β Parallel)"
	AlgorithmNegaScout           = "NegaScout"
	AlgorithmRandom              = "Random"
	AlgorithmJamboree            = "Jamboree"
	AlgorithmLazySMP             = "LazySMP"
)

type Algorithm interface {
	GetName() string
	GetBestMove(*AIPlayer, *board.Board, *board.LastMove) *ScoredMove
}

var NameToAlgorithm = map[string]Algorithm{
	AlgorithmMiniMax:             &MiniMax{},
	AlgorithmAlphaBetaWithMemory: &AlphaBetaWithMemory{},
	AlgorithmMTDf:                &MTDf{},
	AlgorithmABDADA:              &ABDADA{},
	AlgorithmNegaScout:           &NegaScout{},
	AlgorithmRandom:              &Random{},
	AlgorithmJamboree:            &Jamboree{},
	AlgorithmLazySMP:             &LazySMP{},
}

// NewAlgorithm returns a fresh, unshared instance of the named algorithm. The
// entries in NameToAlgorithm are singletons; reusing one across games would carry
// per-search state (e.g. ABDADA's killer/history/countermove tables, abort flag,
// and thread bookkeeping) from one game into the next. Each game must own its
// algorithm instance so it starts from a clean slate.
func NewAlgorithm(name string) Algorithm {
	return newAlgorithmLike(NameToAlgorithm[name])
}
