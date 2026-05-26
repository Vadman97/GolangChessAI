package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

// fastUndo holds the minimal state needed to undo a makeFastMove.
type fastUndo struct {
	startLoc, endLoc   location.Location
	startData, endData byte

	// extra slots for castling (rook src/dst) or en passant (captured pawn)
	extra1Loc, extra2Loc   location.Location
	extra1Data, extra2Data byte

	prevKingLoc location.Location

	isKingMove   bool
	hasCastle    bool
	hasEnPassant bool
}

// setPieceData writes raw 4-bit piece data to the board at l without going through Piece objects.
func (b *Board) setPieceData(l location.Location, data byte) {
	pos := getBitOffset(l)
	row := l.GetRow()
	b.board[row] &^= PieceMask << pos
	b.board[row] |= uint32(data) << pos
}

// makeFastMove applies m to the raw board array with no bookkeeping (no hash, no history, no flags).
// It handles regular moves, captures, castling, and en passant. Returns an undo record.
func (b *Board) makeFastMove(m *location.Move) fastUndo {
	startData := b.getPieceData(m.Start)
	endData := b.getPieceData(m.End)

	undo := fastUndo{
		startLoc:  m.Start,
		endLoc:    m.End,
		startData: startData,
		endData:   endData,
	}

	pieceType := (startData & 0xE) >> 1
	pieceColor := startData & 0x1

	b.setPieceData(m.Start, 0)
	b.setPieceData(m.End, startData)

	switch pieceType {
	case piece.KingType:
		undo.isKingMove = true
		undo.prevKingLoc = b.KingLocations[pieceColor]
		b.KingLocations[pieceColor] = m.End

		startCol := int(m.Start.GetCol())
		endCol := int(m.End.GetCol())
		row := m.Start.GetRow()

		if startCol-endCol == 2 {
			// Left castle: rook moves from col 0 to col 2
			rookSrc := location.NewLocation(row, 0)
			rookDst := location.NewLocation(row, 2)
			undo.hasCastle = true
			undo.extra1Loc = rookSrc
			undo.extra1Data = b.getPieceData(rookSrc)
			undo.extra2Loc = rookDst
			undo.extra2Data = b.getPieceData(rookDst)
			b.setPieceData(rookDst, undo.extra1Data)
			b.setPieceData(rookSrc, 0)
		} else if endCol-startCol == 2 {
			// Right castle: rook moves from col 7 to col 4
			rookSrc := location.NewLocation(row, 7)
			rookDst := location.NewLocation(row, 4)
			undo.hasCastle = true
			undo.extra1Loc = rookSrc
			undo.extra1Data = b.getPieceData(rookSrc)
			undo.extra2Loc = rookDst
			undo.extra2Data = b.getPieceData(rookDst)
			b.setPieceData(rookDst, undo.extra1Data)
			b.setPieceData(rookSrc, 0)
		}

	case piece.PawnType:
		// En passant: pawn moves diagonally to an empty square.
		// The captured pawn sits at (start.Row, end.Col), not at end.
		if m.Start.GetCol() != m.End.GetCol() && endData == 0 {
			captureLoc := location.NewLocation(m.Start.GetRow(), m.End.GetCol())
			undo.hasEnPassant = true
			undo.extra1Loc = captureLoc
			undo.extra1Data = b.getPieceData(captureLoc)
			b.setPieceData(captureLoc, 0)
		}
	}

	return undo
}

// unmakeFastMove restores the board to the state before the corresponding makeFastMove.
func (b *Board) unmakeFastMove(undo fastUndo) {
	b.setPieceData(undo.startLoc, undo.startData)
	b.setPieceData(undo.endLoc, undo.endData)
	if undo.isKingMove {
		b.KingLocations[undo.startData&0x1] = undo.prevKingLoc
	}
	if undo.hasCastle {
		b.setPieceData(undo.extra1Loc, undo.extra1Data)
		b.setPieceData(undo.extra2Loc, undo.extra2Data)
	} else if undo.hasEnPassant {
		b.setPieceData(undo.extra1Loc, undo.extra1Data)
	}
}

// computePinData ray-casts from the king to find:
//  1. Which friendly squares are pinned (a friendly piece is the only blocker between the king
//     and an enemy slider along that ray — if it moves off the ray, the king is exposed).
//  2. Whether the king is currently in check.
//
// Returns (pinnedMask, inCheck). pinnedMask has bit row*8+col set for each pinned square.
// This is called once at the start of getAllMoves so that willMoveLeaveKingInCheck can skip
// the expensive make/unmake+ray-cast for non-pinned, non-king pieces.
func (b *Board) computePinData(c byte) (pinnedMask uint64, inCheck bool) {
	kingLoc := b.KingLocations[c]
	opp := c ^ 1

	// Rook/queen directions (ranks and files)
	for _, dir := range [4]location.RelativeLocation{
		location.UpMove, location.DownMove, location.LeftMove, location.RightMove,
	} {
		var pinnedIdx uint64
		hasPinCandidate := false
		loc := kingLoc
		for {
			var ok bool
			loc, ok = loc.AddRelative(dir)
			if !ok {
				break
			}
			data := b.getPieceData(loc)
			if data == 0 {
				continue
			}
			if data&0x1 == c {
				if !hasPinCandidate {
					row, col := loc.Get()
					pinnedIdx = uint64(row)*8 + uint64(col)
					hasPinCandidate = true
				} else {
					break // second friendly — nothing is pinned on this ray
				}
			} else {
				t := (data & 0xE) >> 1
				if t == piece.RookType || t == piece.QueenType {
					if !hasPinCandidate {
						inCheck = true
					} else {
						pinnedMask |= uint64(1) << pinnedIdx
					}
				}
				break
			}
		}
	}

	// Bishop/queen directions (diagonals)
	for _, dir := range [4]location.RelativeLocation{
		location.RightUpMove, location.RightDownMove, location.LeftUpMove, location.LeftDownMove,
	} {
		var pinnedIdx uint64
		hasPinCandidate := false
		loc := kingLoc
		for {
			var ok bool
			loc, ok = loc.AddRelative(dir)
			if !ok {
				break
			}
			data := b.getPieceData(loc)
			if data == 0 {
				continue
			}
			if data&0x1 == c {
				if !hasPinCandidate {
					row, col := loc.Get()
					pinnedIdx = uint64(row)*8 + uint64(col)
					hasPinCandidate = true
				} else {
					break
				}
			} else {
				t := (data & 0xE) >> 1
				if t == piece.BishopType || t == piece.QueenType {
					if !hasPinCandidate {
						inCheck = true
					} else {
						pinnedMask |= uint64(1) << pinnedIdx
					}
				}
				break
			}
		}
	}

	// Knight attacks on king
	for _, delta := range possibleMoves {
		loc, ok := kingLoc.AddRelative(delta)
		if !ok {
			continue
		}
		data := b.getPieceData(loc)
		if data != 0 && data&0x1 == opp && (data&0xE)>>1 == piece.KnightType {
			inCheck = true
		}
	}

	// Pawn attacks on king: opponent pawns attack diagonally "forward" from their perspective.
	// opp=White(0) attacks toward higher rows → pawn sits one row below king
	// opp=Black(1) attacks toward lower rows → pawn sits one row above king
	pawnRowDelta := int8(2*int(opp) - 1) // opp=0 → -1, opp=1 → +1
	for _, colDelta := range [2]int8{-1, 1} {
		loc, ok := kingLoc.AddRelative(location.RelativeLocation{Row: pawnRowDelta, Col: colDelta})
		if !ok {
			continue
		}
		data := b.getPieceData(loc)
		if data != 0 && data&0x1 == opp && (data&0xE)>>1 == piece.PawnType {
			inCheck = true
		}
	}

	return
}

// isKingInCheckFast checks whether the king of color c at kingLoc is under attack.
// It ray-casts from the king position rather than computing the full opponent attack map.
func (b *Board) isKingInCheckFast(kingLoc location.Location, c byte) bool {
	opp := c ^ 1

	// Rook/queen: check along ranks and files
	for _, dir := range [4]location.RelativeLocation{
		location.UpMove, location.DownMove, location.LeftMove, location.RightMove,
	} {
		loc := kingLoc
		for {
			var inBounds bool
			loc, inBounds = loc.AddRelative(dir)
			if !inBounds {
				break
			}
			data := b.getPieceData(loc)
			if data == 0 {
				continue
			}
			if data&0x1 == opp {
				t := (data & 0xE) >> 1
				if t == piece.RookType || t == piece.QueenType {
					return true
				}
			}
			break // any piece blocks the ray
		}
	}

	// Bishop/queen: check along diagonals
	for _, dir := range [4]location.RelativeLocation{
		location.RightUpMove, location.RightDownMove, location.LeftUpMove, location.LeftDownMove,
	} {
		loc := kingLoc
		for {
			var inBounds bool
			loc, inBounds = loc.AddRelative(dir)
			if !inBounds {
				break
			}
			data := b.getPieceData(loc)
			if data == 0 {
				continue
			}
			if data&0x1 == opp {
				t := (data & 0xE) >> 1
				if t == piece.BishopType || t == piece.QueenType {
					return true
				}
			}
			break
		}
	}

	// Knight: 8 fixed jump offsets (reusing possibleMoves from knight.go)
	for _, delta := range possibleMoves {
		loc, inBounds := kingLoc.AddRelative(delta)
		if !inBounds {
			continue
		}
		data := b.getPieceData(loc)
		if data != 0 && data&0x1 == opp && (data&0xE)>>1 == piece.KnightType {
			return true
		}
	}

	// Pawn: opponent pawns attack diagonally "forward".
	// White (0) pawns attack +1 row; Black (1) pawns attack -1 row.
	// From the king's perspective: check the row where an attacking pawn would sit.
	//   opp=Black(1): pawn attacks (pawnRow-1, …) → pawn sits at kingRow+1
	//   opp=White(0): pawn attacks (pawnRow+1, …) → pawn sits at kingRow-1
	pawnRowDelta := int8(2*int(opp) - 1) // opp=0 → -1, opp=1 → +1
	for _, colDelta := range [2]int8{-1, 1} {
		loc, inBounds := kingLoc.AddRelative(location.RelativeLocation{Row: pawnRowDelta, Col: colDelta})
		if !inBounds {
			continue
		}
		data := b.getPieceData(loc)
		if data != 0 && data&0x1 == opp && (data&0xE)>>1 == piece.PawnType {
			return true
		}
	}

	// Opponent king: adjacent squares (prevents moving into adjacency)
	for _, dir := range [8]location.RelativeLocation{
		location.UpMove, location.RightUpMove, location.RightMove, location.RightDownMove,
		location.DownMove, location.LeftDownMove, location.LeftMove, location.LeftUpMove,
	} {
		loc, inBounds := kingLoc.AddRelative(dir)
		if !inBounds {
			continue
		}
		data := b.getPieceData(loc)
		if data != 0 && data&0x1 == opp && (data&0xE)>>1 == piece.KingType {
			return true
		}
	}

	return false
}
