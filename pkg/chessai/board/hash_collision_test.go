package board

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBoardHashLookup(t *testing.T) {
	const N = 10000
	scoreMap := util.NewConcurrentBoardMap()
	bo1 := Board{}
	bo1.ResetDefault()
	hash := bo1.Hash()
	hits := 0
	for i := 0; i < N; i++ {
		bo1.MakeRandomMove()
		hash = bo1.Hash()
		v, ok := scoreMap.Read(&hash, 0)
		if ok {
			hits++
			bo2 := v.(*Board)
			assert.True(t, bo1.Equals(bo2))
			assert.True(t, bo2.Equals(&bo1))
		} else {
			scoreMap.Store(&hash, 0, &bo1)
		}
	}
	fmt.Println(bo1)
	fmt.Printf("Hit ratio %.2f%%\n", float64(hits)/float64(N)*100.0)
	fmt.Printf("Number different boards %d\n", N-hits)
}
