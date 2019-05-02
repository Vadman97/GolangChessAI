package util

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"math/rand"
)

func RandShuffleMoves(arr []location.Move) (result []location.Move) {
	result = make([]location.Move, len(arr))
	copy(result, arr)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return
}
