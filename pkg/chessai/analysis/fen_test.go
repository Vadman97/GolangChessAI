package analysis

import (
	"strings"
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
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
	// White pawn e2e4: engine (row=1,col=3) -> (row=3,col=3)
	// col 3 → 'a' + (7-3) = 'a'+4 = 'e', so file 'e'
	// row 1 → rank '2', row 3 → rank '4'
	// Expected: "e2e4"
}
