package util

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"math/rand"
)

func MaxScore(a, b int) (result int) {
	if b > a {
		result = b
	} else {
		result = a
	}
	return
}

func MinScore(a, b int) (result int) {
	if b < a {
		result = b
	} else {
		result = a
	}
	return
}

func RandShuffleMoves(arr []location.Move) (result []location.Move) {
	result = make([]location.Move, len(arr))
	copy(result, arr)
	rand.Shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return
}
