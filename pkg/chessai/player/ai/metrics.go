package ai

import "fmt"

type Metrics struct {
	MovesConsidered int64
	MovesPruned     int64
}

func (metrics *Metrics) Print() string {
	return fmt.Sprintf("Pruned %d Considered %d Percent pruned", metrics.MovesPruned, metrics.MovesConsidered)
}
