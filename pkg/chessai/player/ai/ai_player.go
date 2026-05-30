package ai

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

const (
	// PosInf/NegInf are the initial alpha/beta bounds. Must be strictly larger
	// than any achievable score including mate scores (WinScore + search depth).
	// WinScore = 1_000_000_000; depth ≤ ~100 → max mate score ≈ 1_000_000_100.
	// OnEvaluation (1_111_111_111) sits safely between max mate score and PosInf.
	PosInf       = int(2000000000)
	NegInf       = int(-PosInf)
	OnEvaluation = int(1111111111)
)

const (
	OpeningNone = -1
)

// color -> list of openings: { list of moves }
// Coordinates: col 0=h, col 7=a; row 0=White back rank, row 7=Black back rank.
// White moves upward (increasing row), Black moves downward (decreasing row).
var OpeningMoves = map[color.Color][][]*location.Move{
	color.White: {
		// Giuoco Piano / Italian Game: 1.e4 2.Nf3 3.Bc4 4.c3 5.d3 6.0-0
		// c3 prepares d4; d3 keeps solid pawn center; 0-0 for king safety.
		// These 6 moves are always legally playable as the pieces are on their
		// expected squares (sanity-checked before playing each book move).
		{
			{Start: location.NewLocation(1, 3), End: location.NewLocation(3, 3)}, // e2-e4
			{Start: location.NewLocation(0, 1), End: location.NewLocation(2, 2)}, // Ng1-f3
			{Start: location.NewLocation(0, 2), End: location.NewLocation(3, 5)}, // Bf1-c4
			{Start: location.NewLocation(1, 5), End: location.NewLocation(2, 5)}, // c2-c3
			{Start: location.NewLocation(1, 4), End: location.NewLocation(2, 4)}, // d2-d3
			{Start: location.NewLocation(0, 3), End: location.NewLocation(0, 1)}, // 0-0 (e1-g1)
		},
		// London System: 1.d4 2.Nf3 3.Bf4 4.e3 5.Bd3 6.0-0
		// e3 supports the pawn chain; Bd3 develops the bishop; 0-0 for king safety.
		{
			{Start: location.NewLocation(1, 4), End: location.NewLocation(3, 4)}, // d2-d4
			{Start: location.NewLocation(0, 1), End: location.NewLocation(2, 2)}, // Ng1-f3
			{Start: location.NewLocation(0, 5), End: location.NewLocation(3, 2)}, // Bc1-f4
			{Start: location.NewLocation(1, 3), End: location.NewLocation(2, 3)}, // e2-e3
			{Start: location.NewLocation(0, 2), End: location.NewLocation(2, 4)}, // Bf1-d3
			{Start: location.NewLocation(0, 3), End: location.NewLocation(0, 1)}, // 0-0 (e1-g1)
		},
	},
	color.Black: {
		// Classical defence: 1...e5 2...Nf6 3...Bc5 4...c6 5...d6 6...0-0
		// c6 supports d5; d6 solidifies center; 0-0 for king safety.
		{
			{Start: location.NewLocation(6, 3), End: location.NewLocation(4, 3)}, // e7-e5
			{Start: location.NewLocation(7, 1), End: location.NewLocation(5, 2)}, // Ng8-f6
			{Start: location.NewLocation(7, 2), End: location.NewLocation(4, 5)}, // Bf8-c5
			{Start: location.NewLocation(6, 5), End: location.NewLocation(5, 5)}, // c7-c6
			{Start: location.NewLocation(6, 4), End: location.NewLocation(5, 4)}, // d7-d6
			{Start: location.NewLocation(7, 3), End: location.NewLocation(7, 1)}, // 0-0 (e8-g8)
		},
		// Sicilian-style: 1...c5 2...Nc6 3...d6 4...Nf6 5...g6 6...Bg7
		// Heading toward the Dragon variation with fianchettoed bishop.
		{
			{Start: location.NewLocation(6, 5), End: location.NewLocation(4, 5)}, // c7-c5
			{Start: location.NewLocation(7, 6), End: location.NewLocation(5, 5)}, // Nb8-c6
			{Start: location.NewLocation(6, 4), End: location.NewLocation(5, 4)}, // d7-d6
			{Start: location.NewLocation(7, 1), End: location.NewLocation(5, 2)}, // Ng8-f6
			{Start: location.NewLocation(6, 1), End: location.NewLocation(5, 1)}, // g7-g6
			{Start: location.NewLocation(7, 2), End: location.NewLocation(6, 1)}, // Bf8-g7
		},
	},
}

type ScoredMove struct {
	Move           location.Move
	MoveSequence   []location.Move
	Score          int
	ReturnThisMove bool
}

func (s ScoredMove) NegScore() ScoredMove {
	s.Score = -s.Score
	return s
}

type AIPlayer struct {
	Algorithm                 Algorithm
	TranspositionTableEnabled bool
	PlayerColor               color.Color
	MaxSearchDepth            int
	MaxThinkTime              time.Duration
	LastSearchDepth           int
	TurnCount                 int
	Opening                   int
	Metrics                   *Metrics

	Debug              bool
	PrintInfo          bool
	evaluationMap      *util.ConcurrentBoardMap
	transpositionTable *util.ConcurrentBoardMap
	printer            chan string
	abort              bool
	// ttGeneration is incremented on ponder miss so stale ponder entries
	// are demoted to move-ordering-only and cannot cause alpha/beta cutoffs.
	ttGeneration uint32
}

func NewAIPlayer(c color.Color, algorithm Algorithm) *AIPlayer {
	p := &AIPlayer{
		Algorithm:                 algorithm,
		TranspositionTableEnabled: config.Get().TranspositionTableEnabled,
		PlayerColor:               c,
		TurnCount:                 0,
		Opening:                   OpeningNone,
		Metrics:                   &Metrics{},
		Debug:                     config.Get().LogDebug,
		PrintInfo:                 config.Get().PrintPlayerInfo,
		evaluationMap:             util.NewConcurrentBoardMap(),
		transpositionTable:        util.NewConcurrentBoardMap(),
		printer:                   make(chan string, 1000000),
	}
	if config.Get().UseOpenings {
		// Use a separate local source so opening selection doesn't perturb
		// the global rand state used by tests and random-move generation.
		localRand := rand.New(rand.NewSource(time.Now().UnixNano()))
		p.Opening = localRand.Intn(len(OpeningMoves[c]))
	}
	return p
}

func newAlgorithmLike(algorithm Algorithm) Algorithm {
	switch a := algorithm.(type) {
	case *ABDADA:
		return &ABDADA{NumThreads: a.NumThreads}
	case *LazySMP:
		return &LazySMP{}
	case *MiniMax:
		return &MiniMax{}
	case *AlphaBetaWithMemory:
		return &AlphaBetaWithMemory{}
	case *MTDf:
		return &MTDf{}
	case *NegaScout:
		return &NegaScout{}
	case *Jamboree:
		return &Jamboree{}
	case *Random:
		return &Random{}
	default:
		return NameToAlgorithm[algorithm.GetName()]
	}
}

// NewPonderPlayer creates an isolated search player for background pondering.
// It shares the expensive caches with the real player so the ponder warms the
// TT, but keeps abort/search state and algorithm fields separate.
func (p *AIPlayer) NewPonderPlayer(c color.Color) *AIPlayer {
	ponder := &AIPlayer{
		Algorithm:                 newAlgorithmLike(p.Algorithm),
		TranspositionTableEnabled: p.TranspositionTableEnabled,
		PlayerColor:               c,
		MaxSearchDepth:            p.MaxSearchDepth,
		MaxThinkTime:              p.MaxThinkTime,
		Metrics:                   &Metrics{},
		Debug:                     false,
		PrintInfo:                 false,
		evaluationMap:             p.evaluationMap,
		transpositionTable:        p.transpositionTable,
		printer:                   make(chan string, 1000000),
		ttGeneration:              atomic.LoadUint32(&p.ttGeneration),
	}
	return ponder
}

func betterMove(maximizingP bool, currentBest *ScoredMove, candidate *ScoredMove) bool {
	// Always prefer a move with a valid Move over a zero-move, regardless of score.
	if currentBest.Move.Start.Equals(currentBest.Move.End) && !candidate.Move.Start.Equals(candidate.Move.End) {
		return true
	}
	if maximizingP {
		return candidate.Score > currentBest.Score
	}
	return candidate.Score < currentBest.Score
}

func (p *AIPlayer) GetBestMove(b *board.Board, previousMove *board.LastMove, logger *PerformanceLogger) *location.Move {
	if p.Opening != OpeningNone && p.TurnCount < len(OpeningMoves[p.PlayerColor][p.Opening]) {
		bookMove := OpeningMoves[p.PlayerColor][p.Opening][p.TurnCount]
		// Only play the book move if:
		// 1. Our piece is still on the expected starting square.
		// 2. The destination square is empty (not blocked by a friendly piece).
		// 3. The destination square is not immediately attacked by an opponent piece.
		//    (Prevents playing into immediate captures like Bc4 when d5 can take.)
		bookOK := b.GetPiece(bookMove.Start) != nil && b.IsEmpty(bookMove.End)
		if bookOK {
			// Check if the destination square is safe (not under enemy attack).
			enemy := p.PlayerColor ^ 1
			enemyAttacks := b.GetAllAttackableMoves(enemy)
			if enemyAttacks.IsLocationSet(bookMove.End) {
				bookOK = false // destination is attacked; skip this book move
			}
		}
		if bookOK {
			return bookMove
		}
		p.Opening = OpeningNone // book broken for this position; switch to search
	}
	{
		thinking := make(chan bool)
		go p.printThread(thinking)
		defer close(thinking)
		p.abort = false
		// reset metrics for each move
		p.Metrics = &Metrics{}

		if p.Algorithm != nil {
			scoredMove := p.Algorithm.GetBestMove(p, b, previousMove)
			if p.Debug {
				p.printMoveDebug(b, scoredMove)
			}
			if logger != nil {
				logger.MarkPerformance(b, scoredMove, p)
			}
			if scoredMove.Move.Start.Equals(scoredMove.Move.End) {
				log.Printf("%s resigns, no best move available. Picking random.\n", p)
				return &(&Random{
					Rand: rand.New(rand.NewSource(time.Now().UnixNano())),
				}).RandomMove(b, p.PlayerColor, previousMove).Move
			}
			return &scoredMove.Move
		} else {
			panic("invalid ai algorithm")
		}
	}
}

func (p *AIPlayer) MakeMove(b *board.Board, move *location.Move) *board.LastMove {
	lastMove := board.MakeMove(move, b)
	p.TurnCount++
	return lastMove
}

func (p AIPlayer) String() string {
	return fmt.Sprintf("AI (%s - %s)",
		p.Algorithm.GetName(), color.Names[p.PlayerColor])
}

func (p *AIPlayer) printMoveDebug(b *board.Board, m *ScoredMove) {
	LogFile := config.Get().DebugLogFileName
	file, err := os.OpenFile(LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot open file", err)
	}
	defer func() { _ = file.Close() }()
	var result string
	debugBoard := b.Copy()
	for i := len(m.MoveSequence) - 1; i >= 0; i-- {
		move := m.MoveSequence[i]
		start := debugBoard.GetPiece(move.Start)
		end := debugBoard.GetPiece(move.End)
		startStr, endStr := board.GetColorTypeRepr(start), board.GetColorTypeRepr(end)
		if end == nil {
			endStr = "_"
		}
		result += fmt.Sprintf("\t%s to %s\n", startStr, endStr)
		result += fmt.Sprintf("\t\t%s\n", move)
		board.MakeMove(&move, debugBoard)
	}
	result += fmt.Sprintf("%s\n", p.Metrics)
	result += fmt.Sprintf("%s best move leads to score %d\n", p, m.Score)
	p.printer <- fmt.Sprint(result)
	result += fmt.Sprintf("Board evaluation metrics\n")
	result += p.evaluationMap.String()
	result += fmt.Sprintf("Transposition table metrics\n")
	result += p.transpositionTable.String()
	if b.MoveCache != nil {
		result += fmt.Sprintf("Move cache metrics\n")
		result += b.MoveCache.String()
	}
	if b.AttackableCache != nil {
		result += fmt.Sprintf("Attack Move cache metrics\n")
		result += b.AttackableCache.String()
	}
	result += fmt.Sprintf("\n\n")
	_, _ = fmt.Fprint(file, result)
}

func (p *AIPlayer) ClearCaches(force bool) {
	cleared := false
	if force {
		log.Println("WARNING: Force clearing player caches (negative affects if during game)")
		p.evaluationMap = util.NewConcurrentBoardMap()
		p.transpositionTable = util.NewConcurrentBoardMap()
		cleared = true
	} else {
		if p.evaluationMap.GetTotalWrites() > config.Get().CacheMaxPlayerElements {
			log.Println("WARNING: Clearing player evaluation cache due to size")
			p.evaluationMap = util.NewConcurrentBoardMap()
			cleared = true
		}
		if p.transpositionTable.GetTotalWrites() > config.Get().CacheMaxPlayerElements {
			log.Println("WARNING: Clearing player transposition table due to size")
			p.transpositionTable = util.NewConcurrentBoardMap()
			cleared = true
		}
	}
	if cleared {
		runtime.GC()
		log.Println("Forcing garbage collection")
	}
}

func (p *AIPlayer) printThread(stop chan bool) {
	for {
		select {
		case <-stop:
			return
		default:
			util.PrintPrinter(p.printer, p.PrintInfo)
		}
	}
}

func (p *AIPlayer) trackThinkTime(stop, done chan bool, start time.Time) {
	if p.MaxThinkTime != 0 {
		for {
			select {
			case <-stop:
				done <- true
				return
			default:
				thinkTime := time.Now().Sub(start)
				if thinkTime > p.MaxThinkTime {
					p.abort = true
					p.printer <- fmt.Sprintf("requesting AI hard abort, out of time!\n")
				}
			}
			time.Sleep(1 * time.Millisecond)
		}
	}
	done <- true
}

// Abort requests the in-progress search to stop as soon as possible.
func (p *AIPlayer) Abort() { p.abort = true }

// ResetAbort clears the abort flag so a new search can start cleanly.
func (p *AIPlayer) ResetAbort() { p.abort = false }

// IncrementTTGeneration advances the TT generation counter.
// Call this after a ponder miss (opponent played a different move than predicted)
// so stale ponder entries are demoted to move-ordering-only and cannot produce
// incorrect alpha/beta cutoffs in the real search.
func (p *AIPlayer) IncrementTTGeneration() {
	atomic.AddUint32(&p.ttGeneration, 1)
}

func (p *AIPlayer) terminalNode(b *board.Board, moves *[]location.Move) bool {
	return len(*moves) == 0 || b.PreviousPositionsSeen >= 3 || b.MovesSinceNoDraw >= 100 || b.IsInsufficientMaterial()
}
