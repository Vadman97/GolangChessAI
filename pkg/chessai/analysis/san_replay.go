package analysis

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

type SANReplay struct {
	Ply        int
	SAN        string
	UCI        string
	FENBefore  string
	FENAfter   string
	SideToMove color.Color
}

func ReplaySANMoves(moveText string) ([]SANReplay, error) {
	tokens := strings.Fields(moveText)
	b := &board.Board{}
	b.ResetDefault()
	side := color.White
	fullMove := 1
	var previousMove *board.LastMove
	replayed := make([]SANReplay, 0, len(tokens))

	for _, token := range tokens {
		if strings.HasSuffix(token, ".") || strings.Contains(token, "...") {
			continue
		}
		m, err := matchSANMove(b, side, previousMove, token)
		if err != nil {
			return nil, fmt.Errorf("ply %d %s: %w", len(replayed)+1, token, err)
		}
		fenBefore := BoardToFEN(b, side, previousMove, fullMove)
		after := b.Copy()
		afterMove := board.MakeMove(&m, after)
		nextSide := side ^ 1
		nextFullMove := fullMove
		if side == color.Black {
			nextFullMove++
		}
		replayed = append(replayed, SANReplay{
			Ply:        len(replayed) + 1,
			SAN:        token,
			UCI:        MoveToUCI(m),
			FENBefore:  fenBefore,
			FENAfter:   BoardToFEN(after, nextSide, afterMove, nextFullMove),
			SideToMove: side,
		})
		b = after
		previousMove = afterMove
		side = nextSide
		fullMove = nextFullMove
	}
	return replayed, nil
}

func matchSANMove(b *board.Board, side color.Color, previousMove *board.LastMove, san string) (location.Move, error) {
	clean := cleanSAN(san)
	if clean == "O-O" || clean == "0-0" {
		return matchCastle(b, side, previousMove, true)
	}
	if clean == "O-O-O" || clean == "0-0-0" {
		return matchCastle(b, side, previousMove, false)
	}

	promoteType := byte(0)
	if idx := strings.IndexByte(clean, '='); idx >= 0 {
		if idx+1 >= len(clean) {
			return location.Move{}, fmt.Errorf("invalid promotion SAN")
		}
		promoteType = sanPieceType(rune(clean[idx+1]))
		clean = clean[:idx]
	}
	if len(clean) < 2 {
		return location.Move{}, fmt.Errorf("invalid SAN")
	}

	target := clean[len(clean)-2:]
	targetLoc, err := parseSquare(target)
	if err != nil {
		return location.Move{}, err
	}
	prefix := clean[:len(clean)-2]
	capture := strings.Contains(prefix, "x")
	prefix = strings.ReplaceAll(prefix, "x", "")

	pieceType := piece.PawnType
	if len(prefix) > 0 {
		if pt := sanPieceType(rune(prefix[0])); pt != piece.NilType {
			pieceType = pt
			prefix = prefix[1:]
		}
	}

	disambFile, disambRank := byte(0), byte(0)
	for i := 0; i < len(prefix); i++ {
		ch := prefix[i]
		if ch >= 'a' && ch <= 'h' {
			disambFile = ch
		} else if ch >= '1' && ch <= '8' {
			disambRank = ch
		}
	}

	moves := b.GetAllMoves(side, previousMove)
	var matches []location.Move
	for _, m := range *moves {
		mp := b.GetPiece(m.Start)
		if mp == nil || mp.GetColor() != side || mp.GetPieceType() != pieceType {
			continue
		}
		if m.End.GetRow() != targetLoc.GetRow() || m.End.GetCol() != targetLoc.GetCol() {
			continue
		}
		if promoteType != 0 {
			isPromotion, actual := m.End.GetPawnPromotion()
			if !isPromotion || actual != promoteType {
				continue
			}
		}
		if disambFile != 0 && byte('a'+(7-m.Start.GetCol())) != disambFile {
			continue
		}
		if disambRank != 0 && byte('1'+m.Start.GetRow()) != disambRank {
			continue
		}
		if capture && !isSANMoveCapture(b, m) {
			continue
		}
		matches = append(matches, m)
	}
	if len(matches) != 1 {
		return location.Move{}, fmt.Errorf("expected one legal match, got %d", len(matches))
	}
	return matches[0], nil
}

func cleanSAN(s string) string {
	s = strings.TrimSpace(s)
	for len(s) > 0 {
		r, size := utf8.DecodeLastRuneInString(s)
		if r == utf8.RuneError && size == 0 {
			break
		}
		if r != '+' && r != '#' && r != '!' && r != '?' {
			break
		}
		s = s[:len(s)-size]
	}
	return s
}

func sanPieceType(ch rune) byte {
	switch ch {
	case 'K':
		return piece.KingType
	case 'Q':
		return piece.QueenType
	case 'R':
		return piece.RookType
	case 'B':
		return piece.BishopType
	case 'N':
		return piece.KnightType
	}
	return piece.NilType
}

func matchCastle(b *board.Board, side color.Color, previousMove *board.LastMove, kingside bool) (location.Move, error) {
	moves := b.GetAllMoves(side, previousMove)
	for _, m := range *moves {
		if MoveToUCI(m) == castleUCI(side, kingside) {
			return m, nil
		}
	}
	return location.Move{}, fmt.Errorf("castle move not legal")
}

func castleUCI(side color.Color, kingside bool) string {
	if side == color.White {
		if kingside {
			return "e1g1"
		}
		return "e1c1"
	}
	if kingside {
		return "e8g8"
	}
	return "e8c8"
}

func isSANMoveCapture(b *board.Board, m location.Move) bool {
	return b.GetPiece(m.End) != nil || isEnPassantSANMove(b, m)
}

func isEnPassantSANMove(b *board.Board, m location.Move) bool {
	p := b.GetPiece(m.Start)
	if p == nil || p.GetPieceType() != piece.PawnType {
		return false
	}
	return b.IsEmpty(m.End) && m.Start.GetCol() != m.End.GetCol()
}
