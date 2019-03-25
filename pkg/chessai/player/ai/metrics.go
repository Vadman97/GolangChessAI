package ai

import "fmt"

type Metrics struct {
	MovesConsidered          int64
	MovesPrunedAB            int64
	MovesPrunedTransposition int64
}

func (metrics *Metrics) Print() (res string) {
	res += fmt.Sprintf("Considered %d\n", metrics.MovesConsidered)
	res += fmt.Sprintf("\tPruned     %d\n", metrics.MovesPrunedAB+metrics.MovesPrunedTransposition)
	if metrics.MovesPrunedAB > 0 || metrics.MovesPrunedTransposition > 0 {
		res += fmt.Sprintf("\t\tPrunedAB:    %d\n", metrics.MovesPrunedAB)
		res += fmt.Sprintf("\t\tPrunedTrans: %d\n", metrics.MovesPrunedTransposition)
	}
	return
}
