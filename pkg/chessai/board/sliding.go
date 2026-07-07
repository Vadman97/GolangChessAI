package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
)

// Sliding-piece ray directions as raw row/col deltas. The generators below
// walk plain ints with pieceDataRC instead of Location.AddRelative — the
// encode/decode round-trip per ray square dominated move/attack generation
// in search profiles.
// Direction order matches the original per-piece generators (rook: Up, Right,
// Down, Left; bishop: RightUp, RightDown, LeftUp, LeftDown) so generated move
// lists keep their historical order and equal-score tie-breaks are unchanged.
var (
	orthoDirs = [4][2]int{{-1, 0}, {0, 1}, {1, 0}, {0, -1}}
	diagDirs  = [4][2]int{{-1, 1}, {1, 1}, {-1, -1}, {1, -1}}
)

// slidingAttackBits returns the attack bitboard along dirs from (row, col):
// each ray includes its first occupied square and then stops.
func (b *Board) slidingAttackBits(row, col int, dirs *[4][2]int) BitBoard {
	attackable := BitBoard(0)
	for _, d := range dirs {
		r, c := row+d[0], col+d[1]
		for r >= 0 && r < Height && c >= 0 && c < Width {
			attackable |= BitBoard(1) << uint(r*Width+c)
			if b.pieceDataRC(r, c) != 0 {
				break
			}
			r += d[0]
			c += d[1]
		}
	}
	return attackable
}

// appendSlidingMoves appends legal moves along dirs from start to *moves.
// Returns true when onlyFirstMove is set and a move was found.
func (b *Board) appendSlidingMoves(pieceColor byte, start location.Location, dirs *[4][2]int, moves *[]location.Move, onlyFirstMove bool) bool {
	rowC, colC := start.Get()
	row, col := int(rowC), int(colC)
	for _, d := range dirs {
		r, c := row+d[0], col+d[1]
		for r >= 0 && r < Height && c >= 0 && c < Width {
			data := b.pieceDataRC(r, c)
			if data != 0 && data&0x1 == pieceColor {
				break // friendly piece blocks the ray
			}
			possibleMove := location.Move{
				Start: start,
				End:   location.NewLocation(location.CoordinateType(r), location.CoordinateType(c)),
			}
			if !b.willMoveLeaveKingInCheck(pieceColor, possibleMove) {
				*moves = append(*moves, possibleMove)
				if onlyFirstMove {
					return true
				}
			}
			if data != 0 {
				break // enemy piece: capture ends the ray
			}
			r += d[0]
			c += d[1]
		}
	}
	return false
}
