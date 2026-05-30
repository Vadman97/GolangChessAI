package board

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

// seePieceValues maps piece type to centipawn value for SEE exchange calculations.
// Indexed by the 3-bit piece type field stored in raw board data (piece.XxxType constants).
var seePieceValues = [7]int{
	0,     // 0: NilType
	500,   // 1: RookType
	300,   // 2: KnightType
	300,   // 3: BishopType
	900,   // 4: QueenType
	10000, // 5: KingType
	100,   // 6: PawnType
}

// SEEPieceValue returns the SEE centipawn value for a piece type.
func SEEPieceValue(pt byte) int {
	if int(pt) < len(seePieceValues) {
		return seePieceValues[pt]
	}
	return 0
}

// smallestAttackerFor finds the from-square and piece type of the cheapest attacker
// of color stm on square 'to'. Checks piece types in ascending value order so the
// first match is guaranteed to be the cheapest attacker.
// Since it is called after each makeFastMove, X-ray attackers are revealed automatically.
func (b *Board) smallestAttackerFor(to location.Location, stm byte) (from location.Location, pt byte, found bool) {
	// Pawn (value 100): stm pawns attack 'to' from one row diagonally behind.
	// White(0) pawns advance in the +row direction, so they attack from row-1.
	// Black(1) pawns advance in the -row direction, so they attack from row+1.
	pawnRowDelta := int8(-1)
	if stm == 1 {
		pawnRowDelta = 1
	}
	for _, colDelta := range [2]int8{-1, 1} {
		if loc, ok := to.AddRelative(location.RelativeLocation{Row: pawnRowDelta, Col: colDelta}); ok {
			data := b.getPieceData(loc)
			if data != 0 && data&0x1 == stm && (data&0xE)>>1 == piece.PawnType {
				return loc, piece.PawnType, true
			}
		}
	}

	// Knight (value 300): 8 fixed offsets, reuses possibleMoves from knight.go.
	for _, delta := range possibleMoves {
		if loc, ok := to.AddRelative(delta); ok {
			data := b.getPieceData(loc)
			if data != 0 && data&0x1 == stm && (data&0xE)>>1 == piece.KnightType {
				return loc, piece.KnightType, true
			}
		}
	}

	// Bishop (value 300): scan diagonals for the closest bishop of stm's color.
	for _, dir := range [4]location.RelativeLocation{
		location.RightUpMove, location.RightDownMove, location.LeftUpMove, location.LeftDownMove,
	} {
		loc := to
		for {
			var ok bool
			if loc, ok = loc.AddRelative(dir); !ok {
				break
			}
			data := b.getPieceData(loc)
			if data == 0 {
				continue
			}
			if data&0x1 == stm && (data&0xE)>>1 == piece.BishopType {
				return loc, piece.BishopType, true
			}
			break // any non-empty square blocks the diagonal
		}
	}

	// Rook (value 500): scan ranks and files for the closest rook of stm's color.
	for _, dir := range [4]location.RelativeLocation{
		location.UpMove, location.DownMove, location.LeftMove, location.RightMove,
	} {
		loc := to
		for {
			var ok bool
			if loc, ok = loc.AddRelative(dir); !ok {
				break
			}
			data := b.getPieceData(loc)
			if data == 0 {
				continue
			}
			if data&0x1 == stm && (data&0xE)>>1 == piece.RookType {
				return loc, piece.RookType, true
			}
			break
		}
	}

	// Queen (value 900): scan all 8 directions.
	for _, dir := range [8]location.RelativeLocation{
		location.UpMove, location.DownMove, location.LeftMove, location.RightMove,
		location.RightUpMove, location.RightDownMove, location.LeftUpMove, location.LeftDownMove,
	} {
		loc := to
		for {
			var ok bool
			if loc, ok = loc.AddRelative(dir); !ok {
				break
			}
			data := b.getPieceData(loc)
			if data == 0 {
				continue
			}
			if data&0x1 == stm && (data&0xE)>>1 == piece.QueenType {
				return loc, piece.QueenType, true
			}
			break
		}
	}

	// King (value 10000): 8 adjacent squares.
	for _, dir := range [8]location.RelativeLocation{
		location.UpMove, location.RightUpMove, location.RightMove, location.RightDownMove,
		location.DownMove, location.LeftDownMove, location.LeftMove, location.LeftUpMove,
	} {
		if loc, ok := to.AddRelative(dir); ok {
			data := b.getPieceData(loc)
			if data != 0 && data&0x1 == stm && (data&0xE)>>1 == piece.KingType {
				return loc, piece.KingType, true
			}
		}
	}

	return
}

// seeRecurse implements the recursive exchange simulation for SEE.
// stm = the side whose piece (value capturedValue) is currently on 'to' after a capture.
// Returns: the maximum material gain for the OPPONENT (stm^1) from recapturing on 'to',
// assuming optimal play by both sides. Returns 0 if recapturing is unprofitable.
func (b *Board) seeRecurse(to location.Location, stm byte, capturedValue int) int {
	opp := stm ^ 1
	attackerFrom, attackerPt, found := b.smallestAttackerFor(to, opp)
	if !found {
		return 0 // opponent has no attacker; exchange ends
	}
	attackerValue := seePieceValues[attackerPt]

	// Simulate opponent's recapture.
	move := location.Move{Start: attackerFrom, End: to}
	undo := b.makeFastMove(&move)

	// Now stm can recapture the opponent's piece (value attackerValue) on 'to'.
	// stm's best gain from further exchange:
	stmFurtherGain := b.seeRecurse(to, opp, attackerValue)

	b.unmakeFastMove(undo)

	// Opponent's net from recapturing: gains capturedValue, risks losing attackerValue.
	// max(0, ...) because the opponent can choose NOT to recapture if it's losing.
	net := capturedValue - stmFurtherGain
	if net < 0 {
		return 0
	}
	return net
}

// SEE computes the Static Exchange Evaluation for a capture move.
// Returns the net material gain in centipawns for the side making the capture (stm).
// Positive = winning capture, negative = losing, zero = even exchange.
// stm is the color making the initial capture (move.Start must contain stm's piece).
func (b *Board) SEE(move location.Move, stm byte) int {
	// Value of the piece being captured.
	capturedData := b.getPieceData(move.End)
	if capturedData == 0 {
		return 0 // not a capture
	}
	capturedValue := seePieceValues[(capturedData&0xE)>>1]

	// Value of the capturing piece (what the opponent can gain if they recapture).
	attackerData := b.getPieceData(move.Start)
	if attackerData == 0 {
		return 0
	}
	attackerValue := seePieceValues[(attackerData&0xE)>>1]

	// Simulate the initial capture.
	undo := b.makeFastMove(&move)

	// Opponent's best gain from recapturing on 'to'.
	opponentGain := b.seeRecurse(move.End, stm, attackerValue)

	b.unmakeFastMove(undo)

	// Net gain for stm: what was captured minus what opponent gains back.
	// No max(0, ...) at the root — negative SEE means a losing capture.
	return capturedValue - opponentGain
}
