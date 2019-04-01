package ai

import "fmt"

type Metrics struct {
	MovesConsidered              int64
	MovesPrunedAB                int64
	MovesPrunedTransposition     int64
	MovesABImprovedTransposition int64
}

func (metrics *Metrics) Print() (res string) {
	res += fmt.Sprintf("Considered %d\n", metrics.MovesConsidered)
	pruned := metrics.MovesPrunedAB + metrics.MovesPrunedTransposition
	prunedPercent := 100 * float64(pruned) / float64(pruned+metrics.MovesConsidered)
	res += fmt.Sprintf("\tPruned     %f%% (%d)\n", prunedPercent, pruned)
	if pruned > 0 || metrics.MovesABImprovedTransposition > 0 {
		res += fmt.Sprintf("\t\tPrunedAB:    %d\n", metrics.MovesPrunedAB)
		res += fmt.Sprintf("\t\tPrunedTrans: %d\n", metrics.MovesPrunedTransposition)
		res += fmt.Sprintf("\t\tABImprovedTrans: %d\n", metrics.MovesABImprovedTransposition)
	}
	return
}
