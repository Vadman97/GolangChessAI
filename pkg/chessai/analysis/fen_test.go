package analysis

import (
	"strings"
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
)

// suppress unused import warning
var _ = board.Height

func TestStartingPositionFEN(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	fen := BoardToFEN(b, color.White, nil, 1)
	// Standard starting FEN
	want := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

	// Compare piece placement (first field) since castling/ep may differ in encoding order
	gotParts := strings.Fields(fen)
	wantParts := strings.Fields(want)

	if gotParts[0] != wantParts[0] {
		t.Errorf("piece placement mismatch:\n  got:  %s\n  want: %s", gotParts[0], wantParts[0])
	}
	if gotParts[1] != wantParts[1] {
		t.Errorf("active color mismatch: got %s, want %s", gotParts[1], wantParts[1])
	}
	// Check castling contains all four rights
	for _, ch := range []string{"K", "Q", "k", "q"} {
		if !strings.Contains(gotParts[2], ch) {
			t.Errorf("castling missing %q in %q", ch, gotParts[2])
		}
	}
	t.Logf("FEN: %s", fen)
}

func TestMoveToUCI(t *testing.T) {
	// Coordinate mapping: engine col 0 = h-file, col 7 = a-file.
	// file = 'a' + (7 - col),  rank = '1' + row
	cases := []struct {
		name     string
		startRow int
		startCol int
		endRow   int
		endCol   int
		want     string
	}{
		// e2e4: engine (row=1,col=3) → (row=3,col=3); 'a'+(7-3)='e', rank='1'+1='2' / '1'+3='4'
		{"e2e4", 1, 3, 3, 3, "e2e4"},
		// d7d5: engine (row=6,col=4) → (row=4,col=4); 'a'+(7-4)='d', rank='1'+6='7' / '1'+4='5'
		{"d7d5", 6, 4, 4, 4, "d7d5"},
		// g1f3: engine (row=0,col=1) → (row=2,col=2); 'a'+(7-1)='g' / 'a'+(7-2)='f'
		{"g1f3", 0, 1, 2, 2, "g1f3"},
		// a1a8: engine (row=0,col=7) → (row=7,col=7)
		{"a1a8", 0, 7, 7, 7, "a1a8"},
		// h1h8: engine (row=0,col=0) → (row=7,col=0)
		{"h1h8", 0, 0, 7, 0, "h1h8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := location.Move{
				Start: location.NewLocation(location.CoordinateType(tc.startRow), location.CoordinateType(tc.startCol)),
				End:   location.NewLocation(location.CoordinateType(tc.endRow), location.CoordinateType(tc.endCol)),
			}
			got := MoveToUCI(m)
			if got != tc.want {
				t.Errorf("MoveToUCI(%d,%d→%d,%d) = %q, want %q", tc.startRow, tc.startCol, tc.endRow, tc.endCol, got, tc.want)
			}
		})
	}
}

func TestParseFENStartingPosition(t *testing.T) {
	parsed, err := ParseFEN("rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if err != nil {
		t.Fatal(err)
	}
	got := BoardToFEN(parsed.Board, parsed.Active, parsed.Previous, parsed.FullMove)
	gotParts := strings.Fields(got)
	if gotParts[0] != "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR" {
		t.Fatalf("piece placement mismatch: %s", gotParts[0])
	}
	if parsed.Active != color.White {
		t.Fatalf("expected white to move, got %d", parsed.Active)
	}
	if parsed.Previous != nil {
		t.Fatal("did not expect previous move without en-passant target")
	}
}

func TestParseFENEnPassantPreviousMove(t *testing.T) {
	parsed, err := ParseFEN("2b2rk1/r1qn2bp/p3p2p/1pppP3/8/1PNB1N2/P1P1QPPP/2K1R1R1 w - b6 0 16")
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Previous == nil {
		t.Fatal("expected previous move for en-passant target")
	}
	if got := MoveToUCI(*parsed.Previous.Move); got != "b7b5" {
		t.Fatalf("expected inferred previous move b7b5, got %s", got)
	}
}
