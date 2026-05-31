package analysis

import "testing"

func TestReplaySANMovesHandlesCastleAndPromotion(t *testing.T) {
	moves := "e4 d5 exd5 Qxd5 Nf3 Qe4+ Be2 Qg6 Nc3 Qxg2 Rg1 Qh3 d4 c5 dxc5 Nf6 Be3 Nbd7 Nb5 Kd8 Rg3 Qf5 Ng5 Ne4 Rf3 Qxg5 Bxg5 Nxg5 Ra3 a5 Qd5 h6 c6 e6 cxb7 Rb8 bxc8=R+ Rxc8"

	replayed, err := ReplaySANMoves(moves)
	if err != nil {
		t.Fatalf("ReplaySANMoves() error = %v", err)
	}

	want := map[int]string{
		1:  "e2e4",
		37: "b7c8r",
		38: "b8c8",
	}
	for ply, uci := range want {
		if got := replayed[ply-1].UCI; got != uci {
			t.Fatalf("ply %d UCI = %s, want %s", ply, got, uci)
		}
	}
}

func TestReplaySANMovesHandlesKingsideCastle(t *testing.T) {
	replayed, err := ReplaySANMoves("Nf3 Nf6 g3 g6 Bg2 Bg7 O-O O-O")
	if err != nil {
		t.Fatalf("ReplaySANMoves() error = %v", err)
	}

	if got := replayed[6].UCI; got != "e1g1" {
		t.Fatalf("white castle UCI = %s, want e1g1", got)
	}
	if got := replayed[7].UCI; got != "e8g8" {
		t.Fatalf("black castle UCI = %s, want e8g8", got)
	}
}
