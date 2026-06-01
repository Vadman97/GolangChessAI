package analysis

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
)

type ABDADABenchConfig struct {
	FENPath        string
	Threads        []int
	Depth          int
	ThinkTime      time.Duration
	Runs           int
	StockfishPath  string
	StockfishDepth int
	ShowRoot       int
	JSONPath       string
}

type ABDADAMatrixConfig struct {
	FENPath        string
	FEN            string
	Depth          int
	ThinkTime      time.Duration
	Runs           int
	StockfishPath  string
	StockfishDepth int
	Modes          string
}

type matrixMode struct {
	Name            string
	Algorithm       string
	Threads         int
	TT              bool
	DisableNullMove bool
	DisableLMR      bool
	DisableFutility bool
	DisableRazoring bool
}

type benchPosition struct {
	FEN      string
	Tag      string
	Expected []string
	Bad      []string
	Notes    string
}

type benchRun struct {
	Move          string
	Score         int
	Depth         int
	Elapsed       time.Duration
	Considered    uint64
	PrunedAB      uint64
	PrunedTT      uint64
	TTImproved    uint64
	TimedOut      bool
	StockfishLoss *int
}

type benchTotals struct {
	positions           int
	runs                int
	missedExpected      int
	knownBad            int
	parallelRegressions int
	totalLoss           int
	lossCount           int
	byThread            map[int]*threadTotals
}

type threadTotals struct {
	positions      int
	missedExpected int
	knownBad       int
	totalLoss      int
	lossCount      int
	totalElapsed   time.Duration
}

type benchReport struct {
	FENPath         string                `json:"fenPath"`
	Depth           int                   `json:"depth,omitempty"`
	ThinkTimeMS     int64                 `json:"thinkTimeMs,omitempty"`
	Runs            int                   `json:"runs"`
	StockfishPath   string                `json:"stockfishPath,omitempty"`
	StockfishDepth  int                   `json:"stockfishDepth,omitempty"`
	Positions       []benchReportPosition `json:"positions"`
	Summary         benchReportSummary    `json:"summary"`
	ThreadSummaries []benchThreadSummary  `json:"threadSummaries"`
}

type benchReportPosition struct {
	FEN       string                `json:"fen"`
	Tag       string                `json:"tag,omitempty"`
	Expected  []string              `json:"expected,omitempty"`
	Bad       []string              `json:"bad,omitempty"`
	Notes     string                `json:"notes,omitempty"`
	Stockfish *benchStockfishReport `json:"stockfish,omitempty"`
	Threads   []benchThreadReport   `json:"threads"`
}

type benchStockfishReport struct {
	Depth int    `json:"depth"`
	Best  string `json:"best"`
	Score int    `json:"score"`
}

type benchThreadReport struct {
	Threads int              `json:"threads"`
	Summary benchRunSummary  `json:"summary"`
	Runs    []benchRunReport `json:"runs"`
	Flags   []string         `json:"flags,omitempty"`
}

type benchRunReport struct {
	Move          string `json:"move"`
	Score         int    `json:"score"`
	Depth         int    `json:"depth"`
	ElapsedMS     int64  `json:"elapsedMs"`
	Considered    uint64 `json:"considered"`
	PrunedAB      uint64 `json:"prunedAb"`
	PrunedTT      uint64 `json:"prunedTt"`
	TTImproved    uint64 `json:"ttImproved"`
	TimedOut      bool   `json:"timedOut"`
	StockfishLoss *int   `json:"stockfishLoss,omitempty"`
}

type benchRunSummary struct {
	Move           string `json:"move"`
	Score          int    `json:"score"`
	Count          int    `json:"count"`
	Depth          int    `json:"depth"`
	AvgElapsedMS   int64  `json:"avgElapsedMs"`
	NodesPerSecond int64  `json:"nodesPerSecond"`
	PrunedAB       uint64 `json:"prunedAb"`
	PrunedTT       uint64 `json:"prunedTt"`
	TimedOut       bool   `json:"timedOut"`
	AvgLoss        *int   `json:"avgLoss,omitempty"`
}

type benchReportSummary struct {
	Positions           int  `json:"positions"`
	Runs                int  `json:"runs"`
	MissedExpected      int  `json:"missedExpected"`
	KnownBad            int  `json:"knownBad"`
	ParallelRegressions int  `json:"parallelRegressions"`
	AvgLoss             *int `json:"avgLoss,omitempty"`
}

type benchThreadSummary struct {
	Threads        int   `json:"threads"`
	Positions      int   `json:"positions"`
	MissedExpected int   `json:"missedExpected"`
	KnownBad       int   `json:"knownBad"`
	AvgElapsedMS   int64 `json:"avgElapsedMs"`
	AvgLoss        *int  `json:"avgLoss,omitempty"`
}

// RunABDADABench executes the repeatable ABDADA optimization benchmark described
// in docs/abdada-optimization-plan.md.
func RunABDADABench(cfg ABDADABenchConfig) error {
	if cfg.FENPath == "" {
		cfg.FENPath = "testdata/abdada_fens.txt"
	}
	if len(cfg.Threads) == 0 {
		cfg.Threads = []int{1, 2, 4, 8}
	}
	if cfg.Runs <= 0 {
		cfg.Runs = 1
	}
	if cfg.Depth <= 0 && cfg.ThinkTime <= 0 {
		cfg.Depth = 5
	}

	positions, err := loadBenchPositions(cfg.FENPath)
	if err != nil {
		return err
	}
	if len(positions) == 0 {
		return fmt.Errorf("no benchmark FENs found in %s", cfg.FENPath)
	}

	var sf *StockfishEngine
	if cfg.StockfishPath != "" && cfg.StockfishDepth > 0 {
		sf, err = NewStockfishEngine(cfg.StockfishPath)
		if err != nil {
			return err
		}
		defer sf.Close()
	}

	totals := benchTotals{byThread: map[int]*threadTotals{}}
	report := benchReport{
		FENPath:        cfg.FENPath,
		Depth:          cfg.Depth,
		ThinkTimeMS:    cfg.ThinkTime.Milliseconds(),
		Runs:           cfg.Runs,
		StockfishPath:  cfg.StockfishPath,
		StockfishDepth: cfg.StockfishDepth,
	}
	for posIdx, pos := range positions {
		totals.positions++
		positionReport := benchReportPosition{
			FEN:      pos.FEN,
			Tag:      pos.Tag,
			Expected: pos.Expected,
			Bad:      pos.Bad,
			Notes:    pos.Notes,
		}
		fmt.Printf("FEN %d/%d: %s\n", posIdx+1, len(positions), pos.FEN)
		if pos.Tag != "" {
			fmt.Printf("Tag: %s\n", pos.Tag)
		}
		if pos.Notes != "" {
			fmt.Printf("Notes: %s\n", pos.Notes)
		}
		var sfBest EvalResult
		if sf != nil {
			sfBest = sf.Analyze(pos.FEN, cfg.StockfishDepth)
			fmt.Printf("Stockfish depth=%d best=%s score=%+d\n", cfg.StockfishDepth, sfBest.BestMove, sfBest.CentipawnsSTM)
			positionReport.Stockfish = &benchStockfishReport{
				Depth: cfg.StockfishDepth,
				Best:  sfBest.BestMove,
				Score: sfBest.CentipawnsSTM,
			}
		}
		if cfg.ShowRoot > 0 {
			if err := printRootMoveScores(pos.FEN, cfg.Depth, cfg.ShowRoot, sf, cfg.StockfishDepth, sfBest); err != nil {
				return fmt.Errorf("%s root scores: %w", pos.Tag, err)
			}
		}

		singleThreadMove := ""
		var singleThreadLoss *int
		for _, threads := range cfg.Threads {
			results := make([]benchRun, 0, cfg.Runs)
			for run := 0; run < cfg.Runs; run++ {
				result, err := runABDADABenchSearch(pos.FEN, threads, cfg.Depth, cfg.ThinkTime)
				if err != nil {
					return fmt.Errorf("%s threads=%d run=%d: %w", pos.Tag, threads, run+1, err)
				}
				if sf != nil && result.Move != "" {
					candidate := sf.AnalyzeMove(pos.FEN, result.Move, cfg.StockfishDepth)
					loss := stockfishLoss(sfBest, candidate)
					result.StockfishLoss = &loss
				}
				results = append(results, result)
			}
			summary := summarizeBenchRuns(results)
			totals.runs += len(results)
			threadTotal := totals.byThread[threads]
			if threadTotal == nil {
				threadTotal = &threadTotals{}
				totals.byThread[threads] = threadTotal
			}
			threadTotal.positions++
			threadTotal.totalElapsed += summary.avgElapsed
			if threads == 1 {
				singleThreadMove = summary.move
				singleThreadLoss = summary.avgLoss
			}
			flags := []string{}
			if threads > 1 {
				if singleThreadLoss != nil && summary.avgLoss != nil {
					if *summary.avgLoss > *singleThreadLoss+50 {
						flags = append(flags, "parallel regression")
						totals.parallelRegressions++
					}
				} else if singleThreadMove != "" && summary.move != singleThreadMove {
					flags = append(flags, "parallel regression")
					totals.parallelRegressions++
				}
			}
			if len(pos.Expected) > 0 && !containsString(pos.Expected, summary.move) {
				flags = append(flags, "missed expected")
				totals.missedExpected++
				threadTotal.missedExpected++
			}
			if len(pos.Bad) > 0 && containsString(pos.Bad, summary.move) {
				flags = append(flags, "known bad move")
				totals.knownBad++
				threadTotal.knownBad++
			}
			fmt.Printf("ABDADA threads=%d: best=%s score=%+d stable=%d/%d avg=%s depth=%d nodes/s=%d pruned=%d tt-pruned=%d timeout=%t",
				threads,
				summary.move,
				summary.score,
				summary.count,
				len(results),
				summary.avgElapsed.Round(time.Millisecond),
				summary.depth,
				summary.nodesPerSecond,
				summary.prunedAB,
				summary.prunedTT,
				summary.timedOut,
			)
			if summary.avgLoss != nil {
				fmt.Printf(" loss=%dcp", *summary.avgLoss)
				totals.totalLoss += *summary.avgLoss
				totals.lossCount++
				threadTotal.totalLoss += *summary.avgLoss
				threadTotal.lossCount++
			}
			fmt.Printf("%s\n", formatBenchFlags(flags))
			positionReport.Threads = append(positionReport.Threads, benchThreadReport{
				Threads: threads,
				Summary: makeBenchRunSummary(summary),
				Runs:    makeBenchRunReports(results),
				Flags:   flags,
			})
		}
		fmt.Println()
		report.Positions = append(report.Positions, positionReport)
	}
	printBenchTotals(totals, cfg.Threads)
	report.Summary = makeBenchReportSummary(totals)
	report.ThreadSummaries = makeBenchThreadSummaries(totals, cfg.Threads)
	if cfg.JSONPath != "" {
		if err := writeBenchJSON(cfg.JSONPath, report); err != nil {
			return err
		}
		fmt.Printf("Wrote JSON benchmark report to %s\n", cfg.JSONPath)
	}
	return nil
}

func formatBenchFlags(flags []string) string {
	var b strings.Builder
	for _, flag := range flags {
		b.WriteString(" Flag: ")
		b.WriteString(flag)
	}
	return b.String()
}

func makeBenchRunSummary(summary benchSummary) benchRunSummary {
	return benchRunSummary{
		Move:           summary.move,
		Score:          summary.score,
		Count:          summary.count,
		Depth:          summary.depth,
		AvgElapsedMS:   summary.avgElapsed.Milliseconds(),
		NodesPerSecond: summary.nodesPerSecond,
		PrunedAB:       summary.prunedAB,
		PrunedTT:       summary.prunedTT,
		TimedOut:       summary.timedOut,
		AvgLoss:        summary.avgLoss,
	}
}

func makeBenchRunReports(results []benchRun) []benchRunReport {
	reports := make([]benchRunReport, 0, len(results))
	for _, result := range results {
		reports = append(reports, benchRunReport{
			Move:          result.Move,
			Score:         result.Score,
			Depth:         result.Depth,
			ElapsedMS:     result.Elapsed.Milliseconds(),
			Considered:    result.Considered,
			PrunedAB:      result.PrunedAB,
			PrunedTT:      result.PrunedTT,
			TTImproved:    result.TTImproved,
			TimedOut:      result.TimedOut,
			StockfishLoss: result.StockfishLoss,
		})
	}
	return reports
}

func makeBenchReportSummary(totals benchTotals) benchReportSummary {
	return benchReportSummary{
		Positions:           totals.positions,
		Runs:                totals.runs,
		MissedExpected:      totals.missedExpected,
		KnownBad:            totals.knownBad,
		ParallelRegressions: totals.parallelRegressions,
		AvgLoss:             averagePtr(totals.totalLoss, totals.lossCount),
	}
}

func makeBenchThreadSummaries(totals benchTotals, threads []int) []benchThreadSummary {
	summaries := make([]benchThreadSummary, 0, len(threads))
	for _, threadCount := range threads {
		t := totals.byThread[threadCount]
		if t == nil || t.positions == 0 {
			continue
		}
		summaries = append(summaries, benchThreadSummary{
			Threads:        threadCount,
			Positions:      t.positions,
			MissedExpected: t.missedExpected,
			KnownBad:       t.knownBad,
			AvgElapsedMS:   (t.totalElapsed / time.Duration(t.positions)).Milliseconds(),
			AvgLoss:        averagePtr(t.totalLoss, t.lossCount),
		})
	}
	return summaries
}

func averagePtr(total, count int) *int {
	if count == 0 {
		return nil
	}
	avg := total / count
	return &avg
}

func writeBenchJSON(path string, report benchReport) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func stockfishLoss(best, candidate EvalResult) int {
	loss := best.CentipawnsSTM - candidate.CentipawnsSTM
	if loss < 0 {
		return 0
	}
	if loss > 2000 {
		return 2000
	}
	return loss
}

// RunABDADAMatrix diagnoses a FEN set across search modes so quality failures
// can be attributed to parallelism, TT use, time aborts, or algorithm choice.
func RunABDADAMatrix(cfg ABDADAMatrixConfig) error {
	if cfg.FENPath == "" {
		cfg.FENPath = "testdata/abdada_fens.txt"
	}
	if cfg.Runs <= 0 {
		cfg.Runs = 1
	}
	if cfg.Depth <= 0 && cfg.ThinkTime <= 0 {
		cfg.Depth = 5
	}
	modes, err := parseMatrixModes(cfg.Modes)
	if err != nil {
		return err
	}
	var positions []benchPosition
	if strings.TrimSpace(cfg.FEN) != "" {
		if _, err := ParseFEN(cfg.FEN); err != nil {
			return err
		}
		positions = []benchPosition{{FEN: cfg.FEN, Tag: "single-fen"}}
	} else {
		positions, err = loadBenchPositions(cfg.FENPath)
		if err != nil {
			return err
		}
	}
	if len(positions) == 0 {
		return fmt.Errorf("no FENs to diagnose")
	}

	var sf *StockfishEngine
	if cfg.StockfishPath != "" && cfg.StockfishDepth > 0 {
		sf, err = NewStockfishEngine(cfg.StockfishPath)
		if err != nil {
			return err
		}
		defer sf.Close()
	}

	totalFlags := 0
	for posIdx, pos := range positions {
		fmt.Printf("FEN %d/%d: %s\n", posIdx+1, len(positions), pos.FEN)
		if pos.Tag != "" {
			fmt.Printf("Tag: %s\n", pos.Tag)
		}
		var sfBest EvalResult
		if sf != nil {
			sfBest = sf.Analyze(pos.FEN, cfg.StockfishDepth)
			fmt.Printf("Stockfish depth=%d best=%s score=%+d\n", cfg.StockfishDepth, sfBest.BestMove, sfBest.CentipawnsSTM)
		}

		baselineMove := ""
		var baselineLoss *int
		modeSummaries := map[string]benchSummary{}
		for _, mode := range modes {
			results := make([]benchRun, 0, cfg.Runs)
			for run := 0; run < cfg.Runs; run++ {
				result, err := runMatrixSearch(pos.FEN, mode, cfg.Depth, cfg.ThinkTime)
				if err != nil {
					return fmt.Errorf("%s mode=%s run=%d: %w", pos.Tag, mode.Name, run+1, err)
				}
				if sf != nil && result.Move != "" {
					candidate := sf.AnalyzeMove(pos.FEN, result.Move, cfg.StockfishDepth)
					loss := stockfishLoss(sfBest, candidate)
					result.StockfishLoss = &loss
				}
				results = append(results, result)
			}
			summary := summarizeBenchRuns(results)
			modeSummaries[mode.Name] = summary
			if baselineMove == "" {
				baselineMove = summary.move
				baselineLoss = summary.avgLoss
			}
			flags := []string{}
			if baselineLoss != nil && summary.avgLoss != nil && *summary.avgLoss > *baselineLoss+50 {
				flags = append(flags, "worse than baseline")
			} else if baselineLoss == nil && summary.avgLoss == nil && baselineMove != "" && summary.move != baselineMove {
				flags = append(flags, "disagrees with baseline")
			}
			if len(pos.Expected) > 0 && !containsString(pos.Expected, summary.move) {
				flags = append(flags, "missed expected")
			}
			if len(pos.Bad) > 0 && containsString(pos.Bad, summary.move) {
				flags = append(flags, "known bad move")
			}
			totalFlags += len(flags)
			fmt.Printf("%-16s move=%s score=%+d stable=%d/%d avg=%s depth=%d nodes/s=%d tt-pruned=%d timeout=%t",
				mode.Name,
				summary.move,
				summary.score,
				summary.count,
				cfg.Runs,
				summary.avgElapsed.Round(time.Millisecond),
				summary.depth,
				summary.nodesPerSecond,
				summary.prunedTT,
				summary.timedOut,
			)
			if summary.avgLoss != nil {
				fmt.Printf(" loss=%dcp", *summary.avgLoss)
			}
			fmt.Printf("%s\n", formatBenchFlags(flags))
		}
		printMatrixDiagnosis(modeSummaries)
		fmt.Println()
	}
	fmt.Printf("Matrix summary: positions=%d modes=%d runs-per-mode=%d flags=%d\n", len(positions), len(modes), cfg.Runs, totalFlags)
	return nil
}

func parseMatrixModes(s string) ([]matrixMode, error) {
	if strings.TrimSpace(s) == "" {
		s = "abdada1tt,abdada8tt,abdada1nott,abdada8nott,abdada1safe,abdada8safe,negascouttt"
	}
	parts := strings.Split(s, ",")
	modes := make([]matrixMode, 0, len(parts))
	for _, part := range parts {
		key := strings.ToLower(strings.TrimSpace(part))
		switch key {
		case "abdada1tt":
			modes = append(modes, matrixMode{Name: "abdada-1-tt", Algorithm: ai.AlgorithmABDADA, Threads: 1, TT: true})
		case "abdada2tt":
			modes = append(modes, matrixMode{Name: "abdada-2-tt", Algorithm: ai.AlgorithmABDADA, Threads: 2, TT: true})
		case "abdada4tt":
			modes = append(modes, matrixMode{Name: "abdada-4-tt", Algorithm: ai.AlgorithmABDADA, Threads: 4, TT: true})
		case "abdada8tt":
			modes = append(modes, matrixMode{Name: "abdada-8-tt", Algorithm: ai.AlgorithmABDADA, Threads: 8, TT: true})
		case "abdada1nott":
			modes = append(modes, matrixMode{Name: "abdada-1-no-tt", Algorithm: ai.AlgorithmABDADA, Threads: 1, TT: false})
		case "abdada2nott":
			modes = append(modes, matrixMode{Name: "abdada-2-no-tt", Algorithm: ai.AlgorithmABDADA, Threads: 2, TT: false})
		case "abdada4nott":
			modes = append(modes, matrixMode{Name: "abdada-4-no-tt", Algorithm: ai.AlgorithmABDADA, Threads: 4, TT: false})
		case "abdada8nott":
			modes = append(modes, matrixMode{Name: "abdada-8-no-tt", Algorithm: ai.AlgorithmABDADA, Threads: 8, TT: false})
		case "abdada1safe":
			modes = append(modes, safeABDADAMode("abdada-1-safe", 1, true))
		case "abdada8safe":
			modes = append(modes, safeABDADAMode("abdada-8-safe", 8, true))
		case "abdada1safenott":
			modes = append(modes, safeABDADAMode("abdada-1-safe-no-tt", 1, false))
		case "abdada8safenott":
			modes = append(modes, safeABDADAMode("abdada-8-safe-no-tt", 8, false))
		case "abdada1nonull":
			modes = append(modes, matrixMode{Name: "abdada-1-no-null", Algorithm: ai.AlgorithmABDADA, Threads: 1, TT: true, DisableNullMove: true})
		case "abdada1nolmr":
			modes = append(modes, matrixMode{Name: "abdada-1-no-lmr", Algorithm: ai.AlgorithmABDADA, Threads: 1, TT: true, DisableLMR: true})
		case "abdada1nofutility":
			modes = append(modes, matrixMode{Name: "abdada-1-no-futility", Algorithm: ai.AlgorithmABDADA, Threads: 1, TT: true, DisableFutility: true})
		case "abdada1norazor":
			modes = append(modes, matrixMode{Name: "abdada-1-no-razor", Algorithm: ai.AlgorithmABDADA, Threads: 1, TT: true, DisableRazoring: true})
		case "abdada8nonull":
			modes = append(modes, matrixMode{Name: "abdada-8-no-null", Algorithm: ai.AlgorithmABDADA, Threads: 8, TT: true, DisableNullMove: true})
		case "abdada8nolmr":
			modes = append(modes, matrixMode{Name: "abdada-8-no-lmr", Algorithm: ai.AlgorithmABDADA, Threads: 8, TT: true, DisableLMR: true})
		case "abdada8nofutility":
			modes = append(modes, matrixMode{Name: "abdada-8-no-futility", Algorithm: ai.AlgorithmABDADA, Threads: 8, TT: true, DisableFutility: true})
		case "abdada8norazor":
			modes = append(modes, matrixMode{Name: "abdada-8-no-razor", Algorithm: ai.AlgorithmABDADA, Threads: 8, TT: true, DisableRazoring: true})
		case "negascouttt":
			modes = append(modes, matrixMode{Name: "negascout-tt", Algorithm: ai.AlgorithmNegaScout, Threads: 1, TT: true})
		case "negascoutnott":
			modes = append(modes, matrixMode{Name: "negascout-no-tt", Algorithm: ai.AlgorithmNegaScout, Threads: 1, TT: false})
		default:
			return nil, fmt.Errorf("unknown matrix mode %q", part)
		}
	}
	return modes, nil
}

func safeABDADAMode(name string, threads int, tt bool) matrixMode {
	return matrixMode{
		Name:            name,
		Algorithm:       ai.AlgorithmABDADA,
		Threads:         threads,
		TT:              tt,
		DisableNullMove: true,
		DisableLMR:      true,
		DisableFutility: true,
		DisableRazoring: true,
	}
}

func runMatrixSearch(fen string, mode matrixMode, depth int, thinkTime time.Duration) (benchRun, error) {
	parsed, err := ParseFEN(fen)
	if err != nil {
		return benchRun{}, err
	}
	parsed.Board.CacheGetAllMoves = false
	parsed.Board.CacheGetAllAttackableMoves = false
	maxDepth := depth
	if maxDepth <= 0 {
		maxDepth = 64
	}
	var algorithm ai.Algorithm
	switch mode.Algorithm {
	case ai.AlgorithmABDADA:
		algorithm = &ai.ABDADA{
			NumThreads:      mode.Threads,
			DisableNullMove: mode.DisableNullMove,
			DisableLMR:      mode.DisableLMR,
			DisableFutility: mode.DisableFutility,
			DisableRazoring: mode.DisableRazoring,
		}
	case ai.AlgorithmNegaScout:
		algorithm = &ai.NegaScout{}
	default:
		return benchRun{}, fmt.Errorf("unsupported matrix algorithm %s", mode.Algorithm)
	}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	player.TranspositionTableEnabled = mode.TT
	player.MaxSearchDepth = maxDepth
	player.MaxThinkTime = thinkTime
	player.PrintInfo = false
	player.Debug = false

	start := time.Now()
	best := algorithm.GetBestMove(player, parsed.Board, parsed.Previous)
	elapsed := time.Since(start)

	return benchRun{
		Move:       MoveToUCI(best.Move),
		Score:      best.Score,
		Depth:      player.LastSearchDepth,
		Elapsed:    elapsed,
		Considered: player.Metrics.MovesConsidered,
		PrunedAB:   player.Metrics.MovesPrunedAB,
		PrunedTT:   player.Metrics.MovesPrunedTransposition,
		TTImproved: player.Metrics.MovesABImprovedTransposition,
		TimedOut:   thinkTime > 0 && elapsed >= thinkTime,
	}, nil
}

func printMatrixDiagnosis(summaries map[string]benchSummary) {
	base, ok := summaries["abdada-1-tt"]
	if !ok {
		return
	}
	check := func(name, label string) {
		s, ok := summaries[name]
		if !ok {
			return
		}
		if s.move != base.move {
			fmt.Printf("  Diagnosis: %s differs from abdada-1-tt (%s vs %s)\n", label, s.move, base.move)
		}
		if base.avgLoss != nil && s.avgLoss != nil && *s.avgLoss > *base.avgLoss+50 {
			fmt.Printf("  Diagnosis: %s loses %dcp more than abdada-1-tt\n", label, *s.avgLoss-*base.avgLoss)
		}
	}
	check("abdada-8-tt", "parallel TT")
	check("abdada-1-no-tt", "single-thread no-TT")
	check("abdada-8-no-tt", "parallel no-TT")
	check("abdada-1-no-null", "single-thread no-null")
	check("abdada-1-no-lmr", "single-thread no-LMR")
	check("abdada-1-no-futility", "single-thread no-futility")
	check("abdada-1-no-razor", "single-thread no-razor")
	check("abdada-8-no-null", "parallel no-null")
	check("abdada-8-no-lmr", "parallel no-LMR")
	check("abdada-8-no-futility", "parallel no-futility")
	check("abdada-8-no-razor", "parallel no-razor")
	check("abdada-1-safe", "single-thread no-pruning")
	check("abdada-8-safe", "parallel no-pruning")
	check("abdada-1-safe-no-tt", "single-thread no-pruning no-TT")
	check("abdada-8-safe-no-tt", "parallel no-pruning no-TT")
	check("negascout-tt", "NegaScout TT")
	check("negascout-no-tt", "NegaScout no-TT")
}

func printBenchTotals(totals benchTotals, threads []int) {
	fmt.Println("Summary:")
	fmt.Printf("  positions=%d runs=%d missed-expected=%d known-bad=%d parallel-regressions=%d",
		totals.positions,
		totals.runs,
		totals.missedExpected,
		totals.knownBad,
		totals.parallelRegressions,
	)
	if totals.lossCount > 0 {
		fmt.Printf(" avg-loss=%dcp", totals.totalLoss/totals.lossCount)
	}
	fmt.Println()
	for _, threadCount := range threads {
		t := totals.byThread[threadCount]
		if t == nil || t.positions == 0 {
			continue
		}
		fmt.Printf("  threads=%d positions=%d missed-expected=%d known-bad=%d avg=%s",
			threadCount,
			t.positions,
			t.missedExpected,
			t.knownBad,
			(t.totalElapsed / time.Duration(t.positions)).Round(time.Millisecond),
		)
		if t.lossCount > 0 {
			fmt.Printf(" avg-loss=%dcp", t.totalLoss/t.lossCount)
		}
		fmt.Println()
	}
}

func printRootMoveScores(fen string, depth, limit int, sf *StockfishEngine, sfDepth int, sfBest EvalResult) error {
	if depth <= 0 {
		depth = 4
	}
	parsed, err := ParseFEN(fen)
	if err != nil {
		return err
	}
	algorithm := &ai.ABDADA{NumThreads: 1}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	player.MaxSearchDepth = depth
	player.PrintInfo = false
	player.Debug = false
	scores := algorithm.ScoreRootMoves(player, parsed.Board, parsed.Previous, depth)
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			return MoveToUCI(scores[i].Move) < MoveToUCI(scores[j].Move)
		}
		return scores[i].Score > scores[j].Score
	})
	if limit > len(scores) {
		limit = len(scores)
	}
	fmt.Printf("Root scores depth=%d top=%d:\n", depth, limit)
	for i := 0; i < limit; i++ {
		uci := MoveToUCI(scores[i].Move)
		fmt.Printf("  %2d. %s score=%+d", i+1, uci, scores[i].Score)
		if sf != nil && sfDepth > 0 {
			candidate := sf.AnalyzeMove(fen, uci, sfDepth)
			loss := stockfishLoss(sfBest, candidate)
			fmt.Printf(" sf=%+d loss=%dcp", candidate.CentipawnsSTM, loss)
		}
		fmt.Println()
	}
	return nil
}

func runABDADABenchSearch(fen string, threads, depth int, thinkTime time.Duration) (benchRun, error) {
	parsed, err := ParseFEN(fen)
	if err != nil {
		return benchRun{}, err
	}
	parsed.Board.CacheGetAllMoves = false
	parsed.Board.CacheGetAllAttackableMoves = false
	maxDepth := depth
	if maxDepth <= 0 {
		maxDepth = 64
	}
	algorithm := &ai.ABDADA{NumThreads: threads}
	player := ai.NewAIPlayer(parsed.Active, algorithm)
	player.MaxSearchDepth = maxDepth
	player.MaxThinkTime = thinkTime
	player.PrintInfo = false
	player.Debug = false

	start := time.Now()
	best := algorithm.GetBestMove(player, parsed.Board, parsed.Previous)
	elapsed := time.Since(start)

	return benchRun{
		Move:       MoveToUCI(best.Move),
		Score:      best.Score,
		Depth:      player.LastSearchDepth,
		Elapsed:    elapsed,
		Considered: player.Metrics.MovesConsidered,
		PrunedAB:   player.Metrics.MovesPrunedAB,
		PrunedTT:   player.Metrics.MovesPrunedTransposition,
		TTImproved: player.Metrics.MovesABImprovedTransposition,
		TimedOut:   thinkTime > 0 && elapsed >= thinkTime,
	}, nil
}

type benchSummary struct {
	move           string
	score          int
	count          int
	depth          int
	avgElapsed     time.Duration
	nodesPerSecond int64
	prunedAB       uint64
	prunedTT       uint64
	timedOut       bool
	avgLoss        *int
}

func summarizeBenchRuns(results []benchRun) benchSummary {
	type vote struct {
		count int
		best  benchRun
	}
	votes := map[string]vote{}
	var totalElapsed time.Duration
	var totalNodes, totalPrunedAB, totalPrunedTT uint64
	totalLoss := 0
	lossCount := 0
	timedOut := false
	for _, result := range results {
		v := votes[result.Move]
		v.count++
		if v.best.Move == "" || result.Score > v.best.Score {
			v.best = result
		}
		votes[result.Move] = v
		totalElapsed += result.Elapsed
		totalNodes += result.Considered
		totalPrunedAB += result.PrunedAB
		totalPrunedTT += result.PrunedTT
		timedOut = timedOut || result.TimedOut
		if result.StockfishLoss != nil {
			totalLoss += *result.StockfishLoss
			lossCount++
		}
	}
	keys := make([]string, 0, len(votes))
	for move := range votes {
		keys = append(keys, move)
	}
	sort.Strings(keys)
	bestVote := vote{best: benchRun{Score: ai.NegInf}}
	for _, move := range keys {
		v := votes[move]
		if v.count > bestVote.count || (v.count == bestVote.count && v.best.Score > bestVote.best.Score) {
			bestVote = v
		}
	}
	avgElapsed := totalElapsed / time.Duration(len(results))
	nodesPerSecond := int64(0)
	if totalElapsed > 0 {
		nodesPerSecond = int64(float64(totalNodes) / totalElapsed.Seconds())
	}
	var avgLoss *int
	if lossCount > 0 {
		v := totalLoss / lossCount
		avgLoss = &v
	}
	return benchSummary{
		move:           bestVote.best.Move,
		score:          bestVote.best.Score,
		count:          bestVote.count,
		depth:          bestVote.best.Depth,
		avgElapsed:     avgElapsed,
		nodesPerSecond: nodesPerSecond,
		prunedAB:       totalPrunedAB / uint64(len(results)),
		prunedTT:       totalPrunedTT / uint64(len(results)),
		timedOut:       timedOut,
		avgLoss:        avgLoss,
	}
}

func loadBenchPositions(path string) ([]benchPosition, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var positions []benchPosition
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		pos := benchPosition{FEN: parts[0]}
		if _, err := ParseFEN(pos.FEN); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, lineNo, err)
		}
		if len(parts) > 1 {
			pos.Tag = parts[1]
		}
		if len(parts) > 2 {
			pos.Expected = splitCSV(parts[2])
		}
		if len(parts) > 3 {
			pos.Bad = splitCSV(parts[3])
		}
		if len(parts) > 4 {
			pos.Notes = parts[4]
		}
		positions = append(positions, pos)
	}
	return positions, scanner.Err()
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func containsString(values []string, needle string) bool {
	for _, v := range values {
		if v == needle {
			return true
		}
	}
	return false
}

func ParseThreadList(s string) ([]int, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	threads := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid thread count %q", part)
		}
		threads = append(threads, n)
	}
	return threads, nil
}
