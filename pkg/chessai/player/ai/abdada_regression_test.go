package ai

import (
	"strings"
	"testing"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

func boardFromTestFEN(t *testing.T, fen string) (*board.Board, color.Color) {
	t.Helper()
	fields := strings.Fields(fen)
	if len(fields) < 4 {
		t.Fatalf("invalid FEN %q", fen)
	}

	b := &board.Board{}
	b.ResetDefault()
	for row := location.CoordinateType(0); row < board.Height; row++ {
		for col := location.CoordinateType(0); col < board.Width; col++ {
			b.SetPiece(location.NewLocation(row, col), nil)
		}
	}

	ranks := strings.Split(fields[0], "/")
	if len(ranks) != board.Height {
		t.Fatalf("invalid FEN placement %q", fields[0])
	}
	for fenRankIdx, rank := range ranks {
		engineRow := location.CoordinateType(board.Height - 1 - fenRankIdx)
		file := 0
		for _, ch := range rank {
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
				continue
			}
			if file >= board.Width {
				t.Fatalf("too many files in FEN rank %q", rank)
			}
			engineCol := location.CoordinateType(board.Width - 1 - file)
			loc := location.NewLocation(engineRow, engineCol)
			pieceChar := ch
			if pieceChar >= 'a' && pieceChar <= 'z' {
				pieceChar -= 'a' - 'A'
			}
			pt := piece.NameToType[pieceChar]
			if pt == piece.NilType {
				t.Fatalf("invalid piece %q in FEN", ch)
			}
			p := board.PieceFromType(pt)
			if ch >= 'a' && ch <= 'z' {
				p.SetColor(color.Black)
			} else {
				p.SetColor(color.White)
			}
			p.SetPosition(loc)
			b.SetPiece(loc, p)
			if pt == piece.KingType {
				b.KingLocations[p.GetColor()] = loc
			}
			file++
		}
		if file != board.Width {
			t.Fatalf("rank %q has %d files", rank, file)
		}
	}

	for _, c := range []color.Color{color.White, color.Black} {
		b.SetFlag(board.FlagKingMoved, c, true)
		b.SetFlag(board.FlagLeftRookMoved, c, true)
		b.SetFlag(board.FlagRightRookMoved, c, true)
	}
	if fields[2] != "-" {
		for _, castle := range fields[2] {
			switch castle {
			case 'K':
				b.SetFlag(board.FlagKingMoved, color.White, false)
				b.SetFlag(board.FlagLeftRookMoved, color.White, false)
			case 'Q':
				b.SetFlag(board.FlagKingMoved, color.White, false)
				b.SetFlag(board.FlagRightRookMoved, color.White, false)
			case 'k':
				b.SetFlag(board.FlagKingMoved, color.Black, false)
				b.SetFlag(board.FlagLeftRookMoved, color.Black, false)
			case 'q':
				b.SetFlag(board.FlagKingMoved, color.Black, false)
				b.SetFlag(board.FlagRightRookMoved, color.Black, false)
			}
		}
	}

	side := color.White
	if fields[1] == "b" {
		side = color.Black
	}
	return b, side
}

func testABDADAFixedDepthMove(t *testing.T, fen string, depth int) string {
	t.Helper()
	b, side := boardFromTestFEN(t, fen)
	p := NewAIPlayer(side, &ABDADA{NumThreads: 4})
	p.TranspositionTableEnabled = true
	p.MaxSearchDepth = depth
	p.MaxThinkTime = 0
	move := p.GetBestMove(b, nil, nil)
	return testMoveToUCI(*move)
}

func testMoveToUCI(m location.Move) string {
	from := string([]byte{'a' + byte(7-m.Start.GetCol()), '1' + byte(m.Start.GetRow())})
	to := string([]byte{'a' + byte(7-m.End.GetCol()), '1' + byte(m.End.GetRow())})
	return from + to
}

func TestABDADACdTsyKE6AvoidsQuietBlunderWhenPawnBreakAvailable(t *testing.T) {
	got := testABDADAFixedDepthMove(t, "r1bq1rk1/1p1n2bp/p3p2p/2ppP3/8/2NBQN2/PPP2PPP/2KR2R1 w - - 0 13", 5)
	if got == "b2b3" {
		t.Fatalf("ABDADA repeated cdTsyKE6 blunder 13.b3; expected an active kingside/central move")
	}
}

func TestABDADACdTsyKE6DefendsMatingNetWithRook(t *testing.T) {
	got := testABDADAFixedDepthMove(t, "2q5/P6p/3Qpk1p/4n3/7P/1b4R1/1KP5/5r2 w - - 0 43", 5)
	if got == "b2b3" {
		t.Fatalf("ABDADA repeated cdTsyKE6 blunder 43.Kxb3; expected to keep active rook defense")
	}
}

func TestABDADACdTsyKE6AvoidsForcedMateWalk(t *testing.T) {
	got := testABDADAFixedDepthMove(t, "2q5/P6p/3Qpk1p/1r2n3/7P/K5R1/2P5/8 w - - 3 45", 5)
	if got == "a3a4" {
		t.Fatalf("ABDADA repeated cdTsyKE6 blunder 45.Ka4; expected to avoid the forced mate walk")
	}
}
