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
					loss := sfBest.CentipawnsSTM - candidate.CentipawnsSTM
					if loss < 0 {
						loss = 0
					}
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
			loss := sfBest.CentipawnsSTM - candidate.CentipawnsSTM
			if loss < 0 {
				loss = 0
			}
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
