package util

import (
	"fmt"
	"runtime"
)

func GetMemoryUsed() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return bToMb(m.Alloc)
}

func PrintMemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB | NumGC = %v \n", bToMb(m.Alloc), m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
