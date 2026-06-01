package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type benchMoveDelta struct {
	key       string
	before    benchThreadReport
	after     benchThreadReport
	lossDelta int
	timeDelta int64
}

func RunABDADABenchDiff(beforePath, afterPath string) error {
	before, err := readBenchReport(beforePath)
	if err != nil {
		return fmt.Errorf("read before report: %w", err)
	}
	after, err := readBenchReport(afterPath)
	if err != nil {
		return fmt.Errorf("read after report: %w", err)
	}

	fmt.Printf("ABDADA bench diff: %s -> %s\n", beforePath, afterPath)
	printSummaryDiff(before.Summary, after.Summary)
	printThreadDiffs(before.ThreadSummaries, after.ThreadSummaries)
	printMoveDiffs(before, after)
	return nil
}

func readBenchReport(path string) (benchReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return benchReport{}, err
	}
	defer f.Close()
	var report benchReport
	if err := json.NewDecoder(f).Decode(&report); err != nil {
		return benchReport{}, err
	}
	return report, nil
}

func printSummaryDiff(before, after benchReportSummary) {
	fmt.Println("Summary:")
	fmt.Printf("  missed-expected:      %d -> %d (%+d)\n", before.MissedExpected, after.MissedExpected, after.MissedExpected-before.MissedExpected)
	fmt.Printf("  known-bad:            %d -> %d (%+d)\n", before.KnownBad, after.KnownBad, after.KnownBad-before.KnownBad)
	fmt.Printf("  parallel-regressions: %d -> %d (%+d)\n", before.ParallelRegressions, after.ParallelRegressions, after.ParallelRegressions-before.ParallelRegressions)
	printOptionalIntDiff("  avg-loss:", before.AvgLoss, after.AvgLoss, "cp")
}

func printThreadDiffs(before, after []benchThreadSummary) {
	beforeByThread := map[int]benchThreadSummary{}
	for _, summary := range before {
		beforeByThread[summary.Threads] = summary
	}
	fmt.Println("Threads:")
	for _, afterSummary := range after {
		beforeSummary, ok := beforeByThread[afterSummary.Threads]
		if !ok {
			fmt.Printf("  threads=%d added: missed=%d known-bad=%d avg=%dms\n",
				afterSummary.Threads, afterSummary.MissedExpected, afterSummary.KnownBad, afterSummary.AvgElapsedMS)
			continue
		}
		fmt.Printf("  threads=%d missed %d -> %d (%+d), known-bad %d -> %d (%+d), avg %dms -> %dms (%+dms)",
			afterSummary.Threads,
			beforeSummary.MissedExpected,
			afterSummary.MissedExpected,
			afterSummary.MissedExpected-beforeSummary.MissedExpected,
			beforeSummary.KnownBad,
			afterSummary.KnownBad,
			afterSummary.KnownBad-beforeSummary.KnownBad,
			beforeSummary.AvgElapsedMS,
			afterSummary.AvgElapsedMS,
			afterSummary.AvgElapsedMS-beforeSummary.AvgElapsedMS,
		)
		if beforeSummary.AvgLoss != nil || afterSummary.AvgLoss != nil {
			fmt.Print(", ")
			printOptionalIntDiffInline("avg-loss", beforeSummary.AvgLoss, afterSummary.AvgLoss, "cp")
		}
		fmt.Println()
	}
}

func printMoveDiffs(before, after benchReport) {
	beforeMoves := benchReportThreadMap(before)
	var deltas []benchMoveDelta
	for _, pos := range after.Positions {
		for _, afterThread := range pos.Threads {
			key := benchThreadKey(pos, afterThread.Threads)
			beforeThread, ok := beforeMoves[key]
			if !ok {
				continue
			}
			lossDelta := optionalIntValue(afterThread.Summary.AvgLoss) - optionalIntValue(beforeThread.Summary.AvgLoss)
			timeDelta := afterThread.Summary.AvgElapsedMS - beforeThread.Summary.AvgElapsedMS
			if beforeThread.Summary.Move != afterThread.Summary.Move ||
				lossDelta != 0 ||
				timeDelta != 0 ||
				!sameStringSlice(beforeThread.Flags, afterThread.Flags) {
				deltas = append(deltas, benchMoveDelta{
					key:       key,
					before:    beforeThread,
					after:     afterThread,
					lossDelta: lossDelta,
					timeDelta: timeDelta,
				})
			}
		}
	}
	sort.Slice(deltas, func(i, j int) bool {
		if deltas[i].lossDelta == deltas[j].lossDelta {
			return deltas[i].key < deltas[j].key
		}
		return deltas[i].lossDelta > deltas[j].lossDelta
	})
	if len(deltas) == 0 {
		fmt.Println("Moves: no changed thread/FEN summaries")
		return
	}
	limit := 12
	if len(deltas) < limit {
		limit = len(deltas)
	}
	fmt.Printf("Changed moves/losses (top %d by loss regression):\n", limit)
	for i := 0; i < limit; i++ {
		d := deltas[i]
		fmt.Printf("  %s: move %s -> %s, loss %s -> %s (%+dcp), avg %dms -> %dms (%+dms)",
			d.key,
			d.before.Summary.Move,
			d.after.Summary.Move,
			formatOptionalInt(d.before.Summary.AvgLoss, "cp"),
			formatOptionalInt(d.after.Summary.AvgLoss, "cp"),
			d.lossDelta,
			d.before.Summary.AvgElapsedMS,
			d.after.Summary.AvgElapsedMS,
			d.timeDelta,
		)
		if !sameStringSlice(d.before.Flags, d.after.Flags) {
			fmt.Printf(", flags %v -> %v", d.before.Flags, d.after.Flags)
		}
		fmt.Println()
	}
}

func benchReportThreadMap(report benchReport) map[string]benchThreadReport {
	out := map[string]benchThreadReport{}
	for _, pos := range report.Positions {
		for _, thread := range pos.Threads {
			out[benchThreadKey(pos, thread.Threads)] = thread
		}
	}
	return out
}

func benchThreadKey(pos benchReportPosition, threads int) string {
	label := pos.Tag
	if label == "" {
		label = pos.FEN
	}
	return fmt.Sprintf("%s threads=%d", label, threads)
}

func printOptionalIntDiff(label string, before, after *int, suffix string) {
	fmt.Print(label)
	fmt.Printf("              %s -> %s", formatOptionalInt(before, suffix), formatOptionalInt(after, suffix))
	if before != nil && after != nil {
		fmt.Printf(" (%+d%s)", *after-*before, suffix)
	}
	fmt.Println()
}

func printOptionalIntDiffInline(label string, before, after *int, suffix string) {
	fmt.Printf("%s %s -> %s", label, formatOptionalInt(before, suffix), formatOptionalInt(after, suffix))
	if before != nil && after != nil {
		fmt.Printf(" (%+d%s)", *after-*before, suffix)
	}
}

func formatOptionalInt(v *int, suffix string) string {
	if v == nil {
		return "n/a"
	}
	return fmt.Sprintf("%d%s", *v, suffix)
}

func optionalIntValue(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func sameStringSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
