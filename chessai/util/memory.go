package util

import (
	"fmt"
	"runtime"
)

func MemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB | NumGC = %v \n", bToMb(m.Alloc), m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
