package ai

import (
	"strings"
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

// The Stockfish-classical eval is symmetric on the starting position, so White-to-move
// should evaluate to ~0 (only tiny PSQT asymmetry, none here since the start is mirrored).
func TestStockfishClassicStartPositionBalanced(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	score := evaluateStockfishClassicScore(b, color.White)
	if score < -10 || score > 10 {
		t.Fatalf("start position should be ~balanced, got %d cp", score)
	}
}

// Evaluation is side-to-move relative: for the same board, evaluating as White must be
// the negation of evaluating as Black.
func TestStockfishClassicSideToMoveSymmetry(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	w := evaluateStockfishClassicScore(b, color.White)
	bl := evaluateStockfishClassicScore(b, color.Black)
	if w != -bl {
		t.Fatalf("side-to-move asymmetry: white=%d black=%d (expected white == -black)", w, bl)
	}
}

// Removing Black's queen must make the position strongly winning for White (a queen is
// ~9 pawns; even at the conservative SF normalization that is several hundred cp).
func TestStockfishClassicMaterialAdvantage(t *testing.T) {
	b := &board.Board{}
	b.ResetDefault()

	// Find and remove Black's queen.
	removed := false
	for row := location.CoordinateType(0); row < board.Height && !removed; row++ {
		for col := location.CoordinateType(0); col < board.Width; col++ {
			p := b.GetPiece(location.NewLocation(row, col))
			if p != nil && p.GetColor() == color.Black && p.GetPieceType() == piece.QueenType {
				b.SetPiece(location.NewLocation(row, col), nil)
				removed = true
				break
			}
		}
	}
	if !removed {
		t.Fatal("could not find Black queen to remove")
	}

	score := evaluateStockfishClassicScore(b, color.White)
	if score < 400 {
		t.Fatalf("White up a queen should be strongly winning, got %d cp", score)
	}
}

func TestStockfishClassicQueenMinorPressureRecognizesBxg6(t *testing.T) {
	start := boardFromFENPlacement(t, "4r1k1/p2nrp2/b1pp2p1/3p2Q1/N1Pp4/qP2P3/P1B3PP/R4RK1")

	pressure := start.Copy()
	board.MakeMove(&location.Move{
		Start: location.NewLocation(1, 5), // c2
		End:   location.NewLocation(5, 1), // g6
	}, pressure)

	material := start.Copy()
	board.MakeMove(&location.Move{
		Start: location.NewLocation(2, 3), // e3
		End:   location.NewLocation(3, 4), // d4
	}, material)

	pressureScore := evaluateStockfishClassicScore(pressure, color.White)
	materialScore := evaluateStockfishClassicScore(material, color.White)
	// Calibration note: Stockfish scores the position after Bxg6 at roughly
	// +220..+290, and this eval now reads ~220. The old contact-pressure
	// terms called it 300+ ("clearly winning") — the same overtuning that
	// read +2669 in a +380 position and lost game o6lAdkjC. Require a clear
	// pressure signal, not a hallucinated one.
	if pressureScore < 150 {
		t.Fatalf("expected Bxg6 queen/minor pressure to register clearly, got %d", pressureScore)
	}
	if pressureScore <= materialScore {
		t.Fatalf("expected Bxg6 queen/minor pressure to beat exd4, got Bxg6=%d exd4=%d", pressureScore, materialScore)
	}
	if blackScore := evaluateStockfishClassicScore(pressure, color.Black); blackScore != -pressureScore {
		t.Fatalf("expected side-to-move symmetry after Bxg6, white=%d black=%d", pressureScore, blackScore)
	}
}

func boardFromFENPlacement(t *testing.T, placement string) *board.Board {
	t.Helper()
	b := &board.Board{}
	ranks := strings.Split(placement, "/")
	if len(ranks) != 8 {
		t.Fatalf("invalid placement %q", placement)
	}
	for fenRankIdx, rank := range ranks {
		row := location.CoordinateType(7 - fenRankIdx)
		file := 0
		for _, ch := range rank {
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
				continue
			}
			col := location.CoordinateType(7 - file)
			loc := location.NewLocation(row, col)
			p := pieceFromFENChar(t, ch)
			p.SetPosition(loc)
			b.SetPiece(loc, p)
			if p.GetPieceType() == piece.KingType {
				b.KingLocations[p.GetColor()] = loc
			}
			file++
		}
		if file != 8 {
			t.Fatalf("invalid placement rank %q has %d files", rank, file)
		}
	}
	return b
}

func pieceFromFENChar(t *testing.T, ch rune) board.Piece {
	t.Helper()
	c := color.White
	if ch >= 'a' && ch <= 'z' {
		c = color.Black
		ch -= 'a' - 'A'
	}
	var p board.Piece
	switch ch {
	case 'P':
		p = &board.Pawn{}
	case 'N':
		p = &board.Knight{}
	case 'B':
		p = &board.Bishop{}
	case 'R':
		p = &board.Rook{}
	case 'Q':
		p = &board.Queen{}
	case 'K':
		p = &board.King{}
	default:
		t.Fatalf("invalid FEN piece %q", ch)
	}
	p.SetColor(c)
	return p
}
