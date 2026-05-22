// Package analysis provides FEN generation, UCI move encoding, and Stockfish
// integration for post-hoc analysis of ABDADA self-play games.
package analysis

import (
	"fmt"
	"strings"

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
