package board

type Evaluation struct {
	// [color][pieceType] -> count
	PieceCounts map[byte]map[byte]uint8
	TotalScore  int
}
