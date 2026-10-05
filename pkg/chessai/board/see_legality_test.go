package board_test

import (
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
)

func TestSEEUsesLegalRecaptures(t *testing.T) {
	tests := []struct {
		name, fen, move string
		want            int
	}{
		{"pinned pawn", "4k3/8/4p3/3p4/2B5/8/8/4R1K1 w - - 0 1", "c4d5", 100},
		{"black pinned pawn", "4r1k1/8/8/2b5/3P4/4P3/8/4K3 b - - 0 1", "c5d4", 100},
		{"pinned knight", "4k3/8/4n3/8/3p4/2B5/8/4R1K1 w - - 0 1", "c3d4", 100},
		{"pinned bishop", "4n1k1/5b2/8/8/2B5/8/8/4R1K1 w - - 0 1", "e1e8", 300},
		{"pinned rook", "4k3/4rp2/8/8/2B5/5Q2/8/4R1K1 w - - 0 1", "f3f7", 100},
		{"pinned queen", "4n1k1/5q2/8/8/2B5/8/8/4R1K1 w - - 0 1", "e1e8", 300},
		// After Bxf7 Nxf7 Qxf7, Re7 has become pinned because Ne5 moved.
		{"pin revealed during exchange", "4k1K1/4rp2/6B1/4n3/8/5Q2/8/4R3 w - - 0 1", "g6f7", 100},
		// Rejecting the first attacker must not hide the second legal pawn.
		{"alternative pawn", "4k3/8/2p1p3/3p4/2B5/8/8/4R1K1 w - - 0 1", "c4d5", -200},
		// A pinned rook may move along the pin ray and continue shielding its king.
		{"recapture along pin", "4k3/8/4r3/3Qp3/8/8/8/4R1K1 w - - 0 1", "d5e5", -300},
		{"legal king recapture", "8/8/4k3/3p4/2B5/8/8/6K1 w - - 0 1", "c4d5", -200},
		{"defended king destination", "4k3/4p3/8/8/1B6/8/4Q3/6K1 w - - 0 1", "e2e7", 100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			position, err := analysis.ParseFEN(tc.fen)
			if err != nil {
				t.Fatal(err)
			}
			move, err := analysis.MatchUCIMove(position.Board, position.Active, position.Previous, tc.move)
			if err != nil {
				t.Fatal(err)
			}
			b := position.Board
			before := analysis.BoardToFEN(b, position.Active, position.Previous, position.FullMove)
			hash, kings := b.Hash(), b.KingLocations
			for i := 0; i < 2; i++ {
				if got := b.SEE(move, position.Active); got != tc.want {
					t.Fatalf("SEE(%s) = %d, want %d", tc.move, got, tc.want)
				}
				if after := analysis.BoardToFEN(b, position.Active, position.Previous, position.FullMove); after != before || b.Hash() != hash || b.KingLocations != kings {
					t.Fatal("exchange evaluation changed the live board")
				}
			}
		})
	}
}
