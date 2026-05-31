// Package analysis provides FEN generation, UCI move encoding, and Stockfish
// integration for post-hoc analysis of ABDADA self-play games.
package analysis

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

// The engine's coordinate system is mirrored along the file axis relative to
// standard chess notation:
//   engine col 0  → FEN file h   (engine col 7 → FEN file a)
//   engine row 0  → FEN rank 1   (engine row 7 → FEN rank 8)
//
// So: FEN file = 'a' + (7 - col),  FEN rank = row + 1

var pieceChar = map[byte]byte{
	piece.PawnType:   'P',
	piece.KnightType: 'N',
	piece.BishopType: 'B',
	piece.RookType:   'R',
	piece.QueenType:  'Q',
	piece.KingType:   'K',
}

// BoardToFEN converts the internal board to a standard FEN string.
// lastMove is used to derive the en passant target square (may be nil).
// activeColor is the side to move next. fullMove is the fullmove number (1-based).
func BoardToFEN(b *board.Board, activeColor color.Color, lastMove *board.LastMove, fullMove int) string {
	var sb strings.Builder

	// 1. Piece placement — FEN iterates rank 8..1 (engine row 7..0),
	//    each rank from file a..h (engine col 7..0).
	for row := 7; row >= 0; row-- {
		empty := 0
		for col := 7; col >= 0; col-- {
			l := location.NewLocation(location.CoordinateType(row), location.CoordinateType(col))
			p := b.GetPiece(l)
			if p == nil {
				empty++
			} else {
				if empty > 0 {
					sb.WriteByte(byte('0' + empty))
					empty = 0
				}
				ch := pieceChar[p.GetPieceType()]
				if p.GetColor() == color.Black {
					ch += 'a' - 'A' // lowercase for black
				}
				sb.WriteByte(ch)
			}
		}
		if empty > 0 {
			sb.WriteByte(byte('0' + empty))
		}
		if row > 0 {
			sb.WriteByte('/')
		}
	}

	// 2. Active color
	if activeColor == color.White {
		sb.WriteString(" w ")
	} else {
		sb.WriteString(" b ")
	}

	// 3. Castling rights.
	// Left = h-file direction (kingside), Right = a-file direction (queenside).
	castling := ""
	if !b.GetFlag(board.FlagKingMoved, color.White) && !b.GetFlag(board.FlagCastled, color.White) {
		if !b.GetFlag(board.FlagLeftRookMoved, color.White) {
			castling += "K"
		}
		if !b.GetFlag(board.FlagRightRookMoved, color.White) {
			castling += "Q"
		}
	}
	if !b.GetFlag(board.FlagKingMoved, color.Black) && !b.GetFlag(board.FlagCastled, color.Black) {
		if !b.GetFlag(board.FlagLeftRookMoved, color.Black) {
			castling += "k"
		}
		if !b.GetFlag(board.FlagRightRookMoved, color.Black) {
			castling += "q"
		}
	}
	if castling == "" {
		castling = "-"
	}
	sb.WriteString(castling)

	// 4. En passant target square.
	epSquare := enPassantSquare(lastMove)
	sb.WriteString(" " + epSquare)

	// 5. Halfmove clock and fullmove number.
	sb.WriteString(fmt.Sprintf(" %d %d", b.MovesSinceNoDraw, fullMove))

	return sb.String()
}

// enPassantSquare returns the FEN en passant target square from the last move,
// or "-" if not applicable.
func enPassantSquare(lm *board.LastMove) string {
	if lm == nil {
		return "-"
	}
	p := *lm.Piece
	if p == nil || p.GetPieceType() != piece.PawnType {
		return "-"
	}
	startRow := int(lm.Move.Start.GetRow())
	endRow := int(lm.Move.End.GetRow())
	col := int(lm.Move.End.GetCol())

	// Two-square pawn advance
	diff := endRow - startRow
	if diff != 2 && diff != -2 {
		return "-"
	}
	targetRow := (startRow + endRow) / 2
	// FEN file: 'a' + (7 - col)
	file := byte('a') + byte(7-col)
	rank := byte('1') + byte(targetRow)
	return string([]byte{file, rank})
}

// MoveToUCI converts an internal move to UCI notation (e.g. "e2e4").
func MoveToUCI(m location.Move) string {
	sr, sc := m.Start.GetRow(), m.Start.GetCol()
	er, ec := m.End.GetRow(), m.End.GetCol()
	from := fmt.Sprintf("%c%c", 'a'+byte(7-sc), '1'+sr)
	to := fmt.Sprintf("%c%c", 'a'+byte(7-ec), '1'+er)
	promotion := ""
	if hasPromo, promoType := m.End.GetPawnPromotion(); hasPromo {
		ch := pieceChar[promoType]
		promotion = string([]byte{ch + ('a' - 'A')}) // lowercase
	}
	return from + to + promotion
}

// ParsedFEN is the engine-native representation of a FEN position.
type ParsedFEN struct {
	Board      *board.Board
	Active     color.Color
	Previous   *board.LastMove
	FullMove   int
	HalfMove   int
	Castling   string
	EnPassant  string
	Original   string
	Normalized string
}

// ParseFEN converts a standard FEN string into the engine's mirrored coordinate
// system. The returned Previous move is only populated when the FEN has an
// en-passant target, because that is the only previous-move detail the move
// generator needs.
func ParseFEN(fen string) (*ParsedFEN, error) {
	fields := strings.Fields(fen)
	if len(fields) < 4 {
		return nil, fmt.Errorf("invalid FEN: expected at least 4 fields, got %d", len(fields))
	}
	halfMove := 0
	fullMove := 1
	if len(fields) >= 5 {
		v, err := strconv.Atoi(fields[4])
		if err != nil || v < 0 {
			return nil, fmt.Errorf("invalid FEN halfmove clock %q", fields[4])
		}
		halfMove = v
	}
	if len(fields) >= 6 {
		v, err := strconv.Atoi(fields[5])
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("invalid FEN fullmove number %q", fields[5])
		}
		fullMove = v
	}

	b := &board.Board{}
	b.ResetDefault()
	for r := location.CoordinateType(0); r < board.Height; r++ {
		for c := location.CoordinateType(0); c < board.Width; c++ {
			b.SetPiece(location.NewLocation(r, c), nil)
		}
	}
	b.MovesSinceNoDraw = halfMove
	b.PreviousPositions = nil
	b.PreviousPositionsSeen = 0
	b.CurrentPositionRepeats = 0

	if err := loadFENPieces(b, fields[0]); err != nil {
		return nil, err
	}

	active, err := parseFENColor(fields[1])
	if err != nil {
		return nil, err
	}
	if err := applyFENCastling(b, fields[2]); err != nil {
		return nil, err
	}
	prev, err := previousMoveFromEnPassant(b, active, fields[3])
	if err != nil {
		return nil, err
	}

	return &ParsedFEN{
		Board:      b,
		Active:     active,
		Previous:   prev,
		FullMove:   fullMove,
		HalfMove:   halfMove,
		Castling:   fields[2],
		EnPassant:  fields[3],
		Original:   fen,
		Normalized: strings.Join(fields[:min(len(fields), 6)], " "),
	}, nil
}

func loadFENPieces(b *board.Board, placement string) error {
	ranks := strings.Split(placement, "/")
	if len(ranks) != board.Height {
		return fmt.Errorf("invalid FEN placement: expected 8 ranks, got %d", len(ranks))
	}
	for fenRankIdx, rank := range ranks {
		row := location.CoordinateType(7 - fenRankIdx)
		file := 0
		for _, ch := range rank {
			if ch >= '1' && ch <= '8' {
				file += int(ch - '0')
				continue
			}
			if file >= board.Width {
				return fmt.Errorf("invalid FEN placement: too many files in rank %q", rank)
			}
			pt, ok := piece.NameToType[unicode.ToUpper(ch)]
			if !ok {
				return fmt.Errorf("invalid FEN piece %q", ch)
			}
			col := location.CoordinateType(7 - file)
			loc := location.NewLocation(row, col)
			p := board.PieceFromType(pt)
			if unicode.IsLower(ch) {
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
			return fmt.Errorf("invalid FEN placement: rank %q has %d files", rank, file)
		}
	}
	return nil
}

func parseFENColor(s string) (color.Color, error) {
	switch s {
	case "w":
		return color.White, nil
	case "b":
		return color.Black, nil
	default:
		return 0, fmt.Errorf("invalid FEN active color %q", s)
	}
}

func applyFENCastling(b *board.Board, castling string) error {
	if castling == "-" {
		castling = ""
	}
	for _, ch := range castling {
		if !strings.ContainsRune("KQkq", ch) {
			return fmt.Errorf("invalid FEN castling rights %q", castling)
		}
	}
	for _, c := range []color.Color{color.White, color.Black} {
		b.SetFlag(board.FlagKingMoved, c, true)
		b.SetFlag(board.FlagLeftRookMoved, c, true)
		b.SetFlag(board.FlagRightRookMoved, c, true)
	}
	if strings.Contains(castling, "K") || strings.Contains(castling, "Q") {
		b.SetFlag(board.FlagKingMoved, color.White, false)
	}
	if strings.Contains(castling, "K") {
		b.SetFlag(board.FlagLeftRookMoved, color.White, false)
	}
	if strings.Contains(castling, "Q") {
		b.SetFlag(board.FlagRightRookMoved, color.White, false)
	}
	if strings.Contains(castling, "k") || strings.Contains(castling, "q") {
		b.SetFlag(board.FlagKingMoved, color.Black, false)
	}
	if strings.Contains(castling, "k") {
		b.SetFlag(board.FlagLeftRookMoved, color.Black, false)
	}
	if strings.Contains(castling, "q") {
		b.SetFlag(board.FlagRightRookMoved, color.Black, false)
	}
	return nil
}

func previousMoveFromEnPassant(b *board.Board, active color.Color, ep string) (*board.LastMove, error) {
	if ep == "-" {
		return nil, nil
	}
	epLoc, err := parseSquare(ep)
	if err != nil {
		return nil, err
	}
	mover := active ^ 1
	var startRow, endRow location.CoordinateType
	if mover == color.White {
		startRow = board.StartRow[color.White]["Pawn"]
		endRow = startRow + 2
	} else {
		startRow = board.StartRow[color.Black]["Pawn"]
		endRow = startRow - 2
	}
	start := location.NewLocation(startRow, epLoc.GetCol())
	end := location.NewLocation(endRow, epLoc.GetCol())
	p := b.GetPiece(end)
	if p == nil || p.GetPieceType() != piece.PawnType || p.GetColor() != mover {
		return nil, fmt.Errorf("invalid FEN en-passant target %q: no double-pushed pawn on target file", ep)
	}
	return &board.LastMove{
		Piece: &p,
		Move:  &location.Move{Start: start, End: end},
	}, nil
}

func parseSquare(s string) (location.Location, error) {
	if len(s) != 2 || s[0] < 'a' || s[0] > 'h' || s[1] < '1' || s[1] > '8' {
		return location.Location{}, fmt.Errorf("invalid square %q", s)
	}
	return location.NewLocation(location.CoordinateType(s[1]-'1'), location.CoordinateType(7-(s[0]-'a'))), nil
}
