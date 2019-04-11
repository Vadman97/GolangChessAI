package util

import (
	"fmt"
	"time"
)

func PrintPrinter(printer chan string, printInfo bool) {
	for len(printer) > 0 {
		select {
		case str := <-printer:
			if printInfo {
				fmt.Print(str)
			}
		}
	}
	time.Sleep(10 * time.Millisecond)
}
