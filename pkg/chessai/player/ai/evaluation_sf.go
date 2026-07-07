package ai

// Stockfish-classical (hand-crafted) evaluation, ported in spirit to this engine's
// board representation. This is NOT bit-exact with Stockfish: SF uses magic-bitboard
// attack tables, exact tuned constants, and terms (material imbalance table, endgame
// scale factors) that are intentionally omitted here. Constants below are taken from
// the Stockfish 11-era classical evaluator (evaluate.cpp / psqt.cpp); where a term
// needed infrastructure this engine lacks it is approximated and noted.
//
// Implemented terms: tapered MG/EG piece-square tables + material, mobility (with a
// mobility area), pawn structure (isolated/backward/doubled/connected/passed), passed
// pawns with king proximity, a king-danger attack accumulator, threats (hanging
// pieces, threat-by-minor/rook/king, safe-pawn threats), space, and an initiative
// bonus. Material imbalance and endgame scale factors are out of scope by design.
//
// Coordinate conventions in this engine (see evaluation.go): bit index = 8*row+col,
// row 0 = White's back rank, col 0 = h-file / col 7 = a-file. "Relative rank" is the
// rank counted from a side's own back rank (0 = back rank, 7 = promotion). PSQT file
// index uses a=0..h=7 (== 7-col); piece tables mirror files via min(file, 7-file).

import (
	"math/bits"

	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

// sfScore is a packed middlegame/endgame score, mirroring Stockfish's Score type
// (which packs the two halves into one int). Here they are kept as separate ints
// for clarity; the two are interpolated by game phase at the very end.
type sfScore struct{ mg, eg int }

func s2(mg, eg int) sfScore             { return sfScore{mg, eg} }
func (a sfScore) add(b sfScore) sfScore { return sfScore{a.mg + b.mg, a.eg + b.eg} }
func (a sfScore) sub(b sfScore) sfScore { return sfScore{a.mg - b.mg, a.eg - b.eg} }
func (a sfScore) muli(k int) sfScore    { return sfScore{a.mg * k, a.eg * k} }

// sfNormalizeToPawn scales the final tapered score so that one endgame pawn (eg
// value 208 in SF internal units) reads ~100 centipawns, matching the scale the rest
// of this engine (search margins, the other evaluators) is tuned against.
const sfNormalizeToPawn = 206

// Piece values, indexed by this engine's piece-type bytes (Rook=1,Knight=2,Bishop=3,
// Queen=4,King=5,Pawn=6). King has no material value.
var sfPieceValue = [7]sfScore{
	piece.RookType:   {1276, 1380},
	piece.KnightType: {781, 854},
	piece.BishopType: {825, 915},
	piece.QueenType:  {2538, 2682},
	piece.KingType:   {0, 0},
	piece.PawnType:   {126, 208},
}

// Piece-square tables for non-pawn pieces, indexed [relativeRank 0..7][fileQuadrant 0..3]
// where fileQuadrant = min(file, 7-file) with file a=0..h=7. From Stockfish psqt.cpp.
var sfKnightPSQT = [8][4]sfScore{
	{{-175, -96}, {-92, -65}, {-74, -49}, {-73, -21}},
	{{-77, -67}, {-41, -54}, {-27, -18}, {-15, 8}},
	{{-61, -40}, {-17, -27}, {6, -8}, {12, 29}},
	{{-35, -35}, {8, -2}, {40, 13}, {49, 28}},
	{{-34, -45}, {13, -16}, {44, 9}, {51, 39}},
	{{-9, -51}, {22, -44}, {58, -16}, {53, 17}},
	{{-67, -69}, {-27, -50}, {4, -51}, {37, 12}},
	{{-201, -100}, {-83, -88}, {-56, -56}, {-26, -17}},
}

var sfBishopPSQT = [8][4]sfScore{
	{{-37, -40}, {-4, -21}, {-6, -26}, {-16, -8}},
	{{-11, -26}, {6, -9}, {13, -12}, {3, 1}},
	{{-5, -11}, {15, -1}, {-4, -1}, {12, 7}},
	{{-4, -14}, {8, -4}, {18, 0}, {27, 12}},
	{{-8, -12}, {20, -1}, {15, -10}, {22, 11}},
	{{-11, -21}, {4, 4}, {1, 3}, {8, 4}},
	{{-12, -22}, {-10, -14}, {4, -1}, {0, 1}},
	{{-34, -32}, {1, -29}, {-10, -26}, {-16, -17}},
}

var sfRookPSQT = [8][4]sfScore{
	{{-31, -9}, {-20, -13}, {-14, -10}, {-5, -9}},
	{{-21, -12}, {-13, -9}, {-8, -1}, {6, -2}},
	{{-25, 6}, {-11, -8}, {-1, -2}, {3, -6}},
	{{-13, -6}, {-5, 1}, {-4, -9}, {-6, 7}},
	{{-27, -5}, {-15, 8}, {-4, 7}, {3, -6}},
	{{-22, 6}, {-2, 1}, {6, -7}, {12, 10}},
	{{-2, 4}, {12, 5}, {16, 20}, {18, -5}},
	{{-17, 18}, {-19, 0}, {-1, 19}, {9, 13}},
}

var sfQueenPSQT = [8][4]sfScore{
	{{3, -69}, {-5, -57}, {-5, -47}, {4, -26}},
	{{-3, -55}, {5, -31}, {8, -22}, {12, -4}},
	{{-3, -39}, {6, -18}, {13, -9}, {7, 3}},
	{{4, -23}, {5, -3}, {9, 13}, {8, 24}},
	{{0, -29}, {14, -6}, {12, 9}, {5, 21}},
	{{-4, -38}, {10, -18}, {6, -12}, {8, 1}},
	{{-5, -50}, {6, -27}, {10, -24}, {8, -8}},
	{{-2, -75}, {-2, -52}, {1, -43}, {-2, -36}},
}

var sfKingPSQT = [8][4]sfScore{
	{{271, 1}, {327, 45}, {271, 85}, {198, 76}},
	{{278, 53}, {303, 100}, {234, 133}, {179, 135}},
	{{195, 88}, {258, 130}, {169, 169}, {120, 175}},
	{{164, 103}, {190, 156}, {138, 172}, {98, 172}},
	{{154, 96}, {179, 166}, {105, 199}, {70, 199}},
	{{123, 92}, {145, 172}, {81, 184}, {31, 191}},
	{{88, 47}, {120, 121}, {65, 116}, {33, 131}},
	{{59, 11}, {89, 59}, {45, 73}, {-1, 78}},
}

// Pawn PSQT, indexed [relativeRank 0..7][file a=0..h=7]. Relative ranks 0 and 7 are
// never occupied by a pawn. From Stockfish psqt.cpp (full, non-mirrored).
var sfPawnPSQT = [8][8]sfScore{
	{}, // relrank 0 — own back rank
	{{3, -10}, {3, -6}, {10, 10}, {19, 0}, {16, 14}, {19, 7}, {7, -5}, {-5, -19}},
	{{-9, -10}, {-15, -10}, {11, -10}, {15, 4}, {32, 4}, {22, 3}, {5, -6}, {-22, -4}},
	{{-8, 6}, {-23, -2}, {6, -8}, {20, -4}, {40, -13}, {17, -12}, {4, -10}, {-12, -9}},
	{{13, 9}, {0, 4}, {-13, 3}, {1, -12}, {11, -12}, {-2, -7}, {-13, 6}, {5, 9}},
	{{5, 28}, {-12, 20}, {-7, 21}, {22, 28}, {-8, 30}, {-5, 7}, {-15, 6}, {-8, 13}},
	{{-7, 0}, {7, -11}, {3, 12}, {-13, 21}, {5, 25}, {-16, 19}, {10, 4}, {-8, 7}},
	{}, // relrank 7 — promotion
}

// Mobility bonus by number of reachable squares in the mobility area, per piece type.
var sfKnightMobility = [9]sfScore{
	{-62, -81}, {-53, -56}, {-12, -30}, {-4, -14}, {3, 8}, {13, 15}, {22, 23}, {28, 27}, {33, 33},
}
var sfBishopMobility = [14]sfScore{
	{-48, -59}, {-20, -23}, {16, -3}, {26, 13}, {38, 24}, {51, 42}, {55, 54},
	{63, 57}, {63, 65}, {68, 73}, {81, 78}, {81, 86}, {91, 88}, {98, 97},
}
var sfRookMobility = [15]sfScore{
	{-60, -78}, {-20, -17}, {2, 23}, {3, 39}, {3, 70}, {11, 99}, {22, 103}, {31, 121},
	{40, 134}, {40, 139}, {41, 158}, {48, 164}, {57, 168}, {57, 169}, {62, 172},
}
var sfQueenMobility = [28]sfScore{
	{-30, -48}, {-12, -30}, {-8, -7}, {-9, 19}, {20, 40}, {23, 55}, {23, 59}, {35, 75},
	{38, 78}, {53, 96}, {64, 96}, {65, 100}, {65, 121}, {66, 127}, {67, 131}, {67, 133},
	{72, 136}, {72, 141}, {77, 147}, {79, 150}, {93, 151}, {108, 168}, {108, 168}, {108, 171},
	{110, 182}, {114, 182}, {114, 192}, {116, 219},
}

// King-danger attacker weights by piece type (knight/bishop/rook/queen).
var sfKingAttackWeight = [7]int{
	piece.KnightType: 81,
	piece.BishopType: 52,
	piece.RookType:   44,
	piece.QueenType:  10,
}

// Threat bonuses indexed by the attacked piece's type byte.
var sfThreatByMinor = [7]sfScore{
	piece.PawnType:   {5, 32},
	piece.KnightType: {57, 41},
	piece.BishopType: {77, 56},
	piece.RookType:   {88, 119},
	piece.QueenType:  {79, 161},
}
var sfThreatByRook = [7]sfScore{
	piece.PawnType:   {3, 46},
	piece.KnightType: {37, 68},
	piece.BishopType: {42, 60},
	piece.RookType:   {0, 38},
	piece.QueenType:  {58, 41},
}

var (
	sfHanging             = s2(69, 36)
	sfThreatByKing        = s2(24, 89)
	sfThreatBySafePawn    = s2(173, 94)
	sfWeakQueenProtection = s2(14, 0)
	sfRookOnFile          = [2]sfScore{{19, 7}, {48, 29}} // [open]: semi-open, open
	sfKnightOutpost       = s2(56, 34)
	sfBishopPairStandIn   = s2(40, 55) // SF puts the bishop pair in its imbalance table (omitted here)
)

// Passed-pawn base bonus by relative rank (0..7). A passed pawn is never on relrank 0/7.
var sfPassedRank = [8]sfScore{
	{}, {}, {10, 28}, {17, 33}, {15, 41}, {62, 72}, {168, 177}, {},
}

// sfConnectedSeed is Stockfish's per-rank seed for connected (phalanx or defended)
// pawns. The middlegame bonus is the seed; the endgame bonus is scaled down by rank.
var sfConnectedSeed = [8]int{0, 7, 8, 12, 29, 48, 86, 0}

func sfPopcnt(bb board.BitBoard) int { return bits.OnesCount64(uint64(bb)) }

func sfSetBit(bb *board.BitBoard, row, col int) {
	*bb |= board.BitBoard(1) << uint(board.Width*row+col)
}

// sfRelRank returns the rank of `row` from color c's own back rank (0..7).
func sfRelRank(c color.Color, row int) int {
	if c == color.White {
		return row
	}
	return 7 - row
}

// sfChebyshev is the king-move (Chebyshev) distance between two squares.
func sfChebyshev(r1, c1, r2, c2 int) int {
	dr := r1 - r2
	if dr < 0 {
		dr = -dr
	}
	dc := c1 - c2
	if dc < 0 {
		dc = -dc
	}
	if dr > dc {
		return dr
	}
	return dc
}

// sfKingProximity is the Chebyshev distance from a king to a square, capped at 5
// (matching Stockfish's passed-pawn king-proximity term).
func sfKingProximity(b *board.Board, c color.Color, row, col int) int {
	k := b.KingLocations[c]
	d := sfChebyshev(int(k.GetRow()), int(k.GetCol()), row, col)
	if d > 5 {
		return 5
	}
	return d
}

// sfEval holds the per-evaluation attack/occupancy bitboards so terms don't recompute.
type sfEval struct {
	b             *board.Board
	occ           [color.NumColors]board.BitBoard
	occAll        board.BitBoard
	attackedBy    [color.NumColors][7]board.BitBoard
	attackedByAll [color.NumColors]board.BitBoard
	attackedBy2   [color.NumColors]board.BitBoard
	mobilityArea  [color.NumColors]board.BitBoard
	pieceCount    [color.NumColors][7]int
	phase         int // 0..256, 256 = full middlegame (reuses endgamePhase)
	pieces        []pcGeneric
}

// evaluateStockfishClassicScore returns a side-to-move-relative centipawn score using
// the Stockfish-classical-style hand-crafted evaluation. Terminal positions
// (checkmate/stalemate/draws) are handled by the caller before this is reached.
func evaluateStockfishClassicScore(b *board.Board, whoMoves color.Color) int {
	e := &sfEval{b: b}

	// --- Pass 1: occupancy, piece counts, per-piece attack bitboards. ---
	// Preallocated to the max possible piece count (32) to avoid repeated
	// slice growth reallocations on this hot path (runs on every leaf eval).
	pieces := make([]pcGeneric, 0, 32)
	for row := 0; row < board.Height; row++ {
		for col := 0; col < board.Width; col++ {
			p := b.GetPiece(location.NewLocation(location.CoordinateType(row), location.CoordinateType(col)))
			if p == nil {
				continue
			}
			c := p.GetColor()
			pt := p.GetPieceType()
			e.pieceCount[c][pt]++
			sfSetBit(&e.occ[c], row, col)
			attacks := p.GetAttackableMoves(b)
			e.attackedBy2[c] = e.attackedBy2[c].CombineBitBoards(e.attackedByAll[c].IntersectBitBoards(attacks))
			e.attackedByAll[c] = e.attackedByAll[c].CombineBitBoards(attacks)
			e.attackedBy[c][pt] = e.attackedBy[c][pt].CombineBitBoards(attacks)
			pieces = append(pieces, pcGeneric{row, col, c, pt, attacks})
		}
	}
	e.pieces = pieces
	e.occAll = e.occ[color.White].CombineBitBoards(e.occ[color.Black])

	// Reuse the existing phase computation (0..256) by adapting to its array signature.
	var pc pieceTypeCounts
	for _, c := range []color.Color{color.White, color.Black} {
		for pt := byte(1); pt <= piece.PawnType; pt++ {
			pc[c][pt] = uint8(e.pieceCount[c][pt])
		}
	}
	e.phase = endgamePhase(pc)

	// --- Mobility area: squares not occupied by our king/queens, not held by our
	// pawns on low ranks or blocked pawns, and not attacked by enemy pawns. ---
	for _, us := range []color.Color{color.White, color.Black} {
		them := us ^ 1
		var excluded board.BitBoard
		excluded = excluded.CombineBitBoards(e.attackedBy[them][piece.PawnType])
		// king + queens
		for _, pcInf := range pieces {
			if pcInf.c != us {
				continue
			}
			if pcInf.pt == piece.KingType || pcInf.pt == piece.QueenType {
				sfSetBit(&excluded, pcInf.row, pcInf.col)
			}
			if pcInf.pt == piece.PawnType {
				rr := sfRelRank(us, pcInf.row)
				blocked := false
				forward := 1
				if us == color.Black {
					forward = -1
				}
				fr := pcInf.row + forward
				if fr >= 0 && fr < board.Height {
					if _, _, occupied := b.GetPieceTypeColor(location.NewLocation(location.CoordinateType(fr), location.CoordinateType(pcInf.col))); occupied {
						blocked = true
					}
				}
				if rr <= 2 || blocked {
					sfSetBit(&excluded, pcInf.row, pcInf.col)
				}
			}
		}
		e.mobilityArea[us] = ^excluded
	}

	// --- Per-color terms (positive = good for that color). ---
	var total sfScore // from White's perspective
	for _, us := range []color.Color{color.White, color.Black} {
		s := e.colorScore(us)
		if us == color.White {
			total = total.add(s)
		} else {
			total = total.sub(s)
		}
	}

	// --- Initiative: nudges the endgame value by position complexity. ---
	total.eg += e.initiative(total.eg)

	// --- Taper MG/EG by phase, then normalize to ~100cp/pawn. ---
	tapered := (total.mg*e.phase + total.eg*(256-e.phase)) / 256
	cp := tapered * 100 / sfNormalizeToPawn

	if whoMoves == color.Black {
		cp = -cp
	}
	return cp
}

// pcGeneric is one piece on the board plus its precomputed attack set.
type pcGeneric struct {
	row, col int
	c        color.Color
	pt       byte
	attacks  board.BitBoard
}

func sfPSQT(pt byte, relRank, fileAH int) sfScore {
	fq := fileAH
	if 7-fileAH < fq {
		fq = 7 - fileAH
	}
	switch pt {
	case piece.KnightType:
		return sfKnightPSQT[relRank][fq]
	case piece.BishopType:
		return sfBishopPSQT[relRank][fq]
	case piece.RookType:
		return sfRookPSQT[relRank][fq]
	case piece.QueenType:
		return sfQueenPSQT[relRank][fq]
	case piece.KingType:
		return sfKingPSQT[relRank][fq]
	case piece.PawnType:
		return sfPawnPSQT[relRank][fileAH]
	}
	return sfScore{}
}

func sfMobility(pt byte, count int) sfScore {
	if count < 0 {
		count = 0
	}
	switch pt {
	case piece.KnightType:
		if count >= len(sfKnightMobility) {
			count = len(sfKnightMobility) - 1
		}
		return sfKnightMobility[count]
	case piece.BishopType:
		if count >= len(sfBishopMobility) {
			count = len(sfBishopMobility) - 1
		}
		return sfBishopMobility[count]
	case piece.RookType:
		if count >= len(sfRookMobility) {
			count = len(sfRookMobility) - 1
		}
		return sfRookMobility[count]
	case piece.QueenType:
		if count >= len(sfQueenMobility) {
			count = len(sfQueenMobility) - 1
		}
		return sfQueenMobility[count]
	}
	return sfScore{}
}

// colorScore sums all positive-for-`us` terms: material + PSQT, mobility, outposts,
// rook-on-file, bishop pair, pawn structure, threats against the enemy, passed pawns,
// space, and the (negative) king-danger penalty for our own king.
func (e *sfEval) colorScore(us color.Color) sfScore {
	b := e.b
	them := us ^ 1
	var sc sfScore

	forward := 1
	if us == color.Black {
		forward = -1
	}

	for _, p := range e.pieces {
		if p.c != us {
			continue
		}
		relRank := sfRelRank(us, p.row)
		fileAH := 7 - p.col

		// Material + piece-square table.
		sc = sc.add(sfPieceValue[p.pt])
		sc = sc.add(sfPSQT(p.pt, relRank, fileAH))

		switch p.pt {
		case piece.KnightType, piece.BishopType, piece.RookType, piece.QueenType:
			mob := sfPopcnt(p.attacks.IntersectBitBoards(e.mobilityArea[us]))
			sc = sc.add(sfMobility(p.pt, mob))
			if p.pt == piece.KnightType && relRank >= 3 && relRank <= 5 &&
				isKnightOutpost(b, location.CoordinateType(p.row), location.CoordinateType(p.col), us) {
				sc = sc.add(sfKnightOutpost)
			}
			if p.pt == piece.RookType {
				friendlyPawn := e.fileHasPawn(us, p.col)
				enemyPawn := e.fileHasPawn(them, p.col)
				if !friendlyPawn {
					if !enemyPawn {
						sc = sc.add(sfRookOnFile[1]) // open
					} else {
						sc = sc.add(sfRookOnFile[0]) // semi-open
					}
				}
			}
		case piece.PawnType:
			// Pawn structure: doubled/isolated/backward handled with this engine's
			// existing helpers, scaled into MG/EG. Passed pawns get the SF rank bonus.
			if e.isDoubledBehind(us, p.row, p.col) {
				sc = sc.sub(s2(11, 56)) // SF Doubled = S(11,56)
			}
			if e.isIsolated(us, p.col) {
				sc = sc.sub(s2(5, 15)) // SF Isolated = S(5,15)
			} else if backwardPawnPenalty(b, location.CoordinateType(p.row), location.CoordinateType(p.col), us) < 0 {
				sc = sc.sub(s2(9, 24)) // SF Backward = S(9,24)
			}
			if isPassedPawn(b, location.CoordinateType(p.row), location.CoordinateType(p.col), us) {
				sc = sc.add(e.passedPawn(us, p.row, p.col, relRank, forward))
			}
			// Connected pawns: phalanx (friendly pawn on an adjacent file, same rank)
			// or defended (friendly pawn on an adjacent file one rank behind). Bonus is
			// rank-scaled, mirroring Stockfish's Connected term.
			if seed := sfConnectedSeed[relRank]; seed != 0 && e.isConnectedPawn(us, p.row, p.col, forward) {
				sc = sc.add(s2(seed, seed*(relRank-2)/4))
			}
		}
	}

	// Bishop pair (stand-in for SF's imbalance-table term).
	if e.pieceCount[us][piece.BishopType] >= 2 {
		sc = sc.add(sfBishopPairStandIn)
	}

	// Threats we exert on the enemy.
	sc = sc.add(e.threats(us))

	// Space.
	sc = sc.add(e.space(us))

	// Queenless rook/minor endings often need active kings even while the material
	// phase still looks high because several rooks remain.
	sc = sc.add(e.queenlessKingActivity(us))

	// King danger to our own king (negative).
	sc = sc.sub(e.kingDanger(us))

	return sc
}

// fileHasPawn reports whether color c has a pawn on the given column.
func (e *sfEval) fileHasPawn(c color.Color, col int) bool {
	for row := 0; row < board.Height; row++ {
		pt, pc, ok := e.b.GetPieceTypeColor(location.NewLocation(location.CoordinateType(row), location.CoordinateType(col)))
		if ok && pc == c && pt == piece.PawnType {
			return true
		}
	}
	return false
}

// isDoubledBehind reports a pawn of color c that has a friendly pawn directly behind
// it on the same file (the rearmost of a doubled pair is the one penalized in SF).
func (e *sfEval) isDoubledBehind(c color.Color, row, col int) bool {
	behind := 1
	if c == color.White {
		behind = -1 // toward own back rank (lower rows for White)
	}
	r := row + behind
	if r < 0 || r >= board.Height {
		return false
	}
	pt, pc, ok := e.b.GetPieceTypeColor(location.NewLocation(location.CoordinateType(r), location.CoordinateType(col)))
	return ok && pc == c && pt == piece.PawnType
}

// isConnectedPawn reports whether the pawn of color c at (row,col) is part of a
// phalanx (friendly pawn on an adjacent file, same rank) or is defended by a friendly
// pawn (adjacent file, one rank behind). `forward` is +1 for White, -1 for Black.
func (e *sfEval) isConnectedPawn(c color.Color, row, col, forward int) bool {
	for _, dc := range [2]int{-1, 1} {
		ac := col + dc
		if ac < 0 || ac >= board.Width {
			continue
		}
		for _, dr := range [2]int{0, -forward} { // same rank (phalanx) or one rank behind (defender)
			ar := row + dr
			if ar < 0 || ar >= board.Height {
				continue
			}
			pt, pc, ok := e.b.GetPieceTypeColor(location.NewLocation(location.CoordinateType(ar), location.CoordinateType(ac)))
			if ok && pc == c && pt == piece.PawnType {
				return true
			}
		}
	}
	return false
}

// isIsolated reports whether color c has no pawn on either file adjacent to col.
func (e *sfEval) isIsolated(c color.Color, col int) bool {
	for _, dc := range [2]int{-1, 1} {
		ac := col + dc
		if ac < 0 || ac >= board.Width {
			continue
		}
		if e.fileHasPawn(c, ac) {
			return false
		}
	}
	return true
}

// passedPawn returns the SF passed-pawn bonus for a passed pawn of color us:
// a rank-indexed base bonus plus an endgame king-proximity term.
func (e *sfEval) passedPawn(us color.Color, row, col, relRank, forward int) sfScore {
	bonus := sfPassedRank[relRank]
	if relRank > 3 {
		w := 5*relRank - 13
		them := us ^ 1
		blockRow := row + forward
		if blockRow >= 0 && blockRow < board.Height {
			egTerm := (sfKingProximity(e.b, them, blockRow, col)*19/4 -
				sfKingProximity(e.b, us, blockRow, col)*2) * w
			bonus.eg += egTerm
			if relRank != 6 {
				nextRow := blockRow + forward
				if nextRow >= 0 && nextRow < board.Height {
					bonus.eg -= sfKingProximity(e.b, us, nextRow, col) * w
				}
			}
		}
	}
	return bonus
}

// threats scores the pieces of `them` that `us` attacks: hanging pieces, threats by
// our minors and rooks, king threats, weak-queen protection, and safe-pawn threats.
func (e *sfEval) threats(us color.Color) sfScore {
	them := us ^ 1
	var sc sfScore

	// Strongly protected enemy squares: defended by an enemy pawn, or attacked twice
	// by the enemy and not twice by us.
	stronglyProtected := e.attackedBy[them][piece.PawnType].CombineBitBoards(
		e.attackedBy2[them].IntersectBitBoards(^e.attackedBy2[us]))

	// Weak enemy pieces: under our attack and not strongly protected.
	weak := e.occ[them].IntersectBitBoards(e.attackedByAll[us]).IntersectBitBoards(^stronglyProtected)
	// Enemy non-pawn pieces (used for minor/pawn threat targets and defended bonus).
	nonPawn := e.occupiedNonPawn(them)
	defendedNonPawn := nonPawn.IntersectBitBoards(stronglyProtected)

	minorAttacks := e.attackedBy[us][piece.KnightType].CombineBitBoards(e.attackedBy[us][piece.BishopType])
	target := weak.CombineBitBoards(defendedNonPawn).IntersectBitBoards(minorAttacks)
	for x := uint64(target); x != 0; x &= x - 1 {
		sq := bits.TrailingZeros64(x)
		if pt, ok := e.pieceTypeAt(sq); ok {
			sc = sc.add(sfThreatByMinor[pt])
		}
	}

	rookTarget := weak.IntersectBitBoards(e.attackedBy[us][piece.RookType])
	for x := uint64(rookTarget); x != 0; x &= x - 1 {
		sq := bits.TrailingZeros64(x)
		if pt, ok := e.pieceTypeAt(sq); ok {
			sc = sc.add(sfThreatByRook[pt])
		}
	}

	// King threats on weak enemy pieces.
	if weak.IntersectBitBoards(e.attackedBy[us][piece.KingType]) != 0 {
		sc = sc.add(sfThreatByKing)
	}

	// Hanging: weak enemy pieces not defended at all.
	hanging := weak.IntersectBitBoards(^e.attackedByAll[them])
	sc = sc.add(sfHanging.muli(sfPopcnt(hanging)))

	// Weak enemy pieces only defended by their queen.
	sc = sc.add(sfWeakQueenProtection.muli(sfPopcnt(weak.IntersectBitBoards(e.attackedBy[them][piece.QueenType]))))

	// Safe-pawn threats: enemy non-pawn pieces attacked by our pawns sitting on safe
	// squares (not attacked by the enemy, or defended by us).
	safe := (^e.attackedByAll[them]).CombineBitBoards(e.attackedByAll[us])
	safePawns := e.pawnsOn(us, safe)
	pawnThreatTargets := e.pawnAttacksFrom(us, safePawns).IntersectBitBoards(nonPawn)
	sc = sc.add(sfThreatBySafePawn.muli(sfPopcnt(pawnThreatTargets)))

	return sc
}

// occupiedNonPawn returns the bitboard of color c's non-pawn pieces.
func (e *sfEval) occupiedNonPawn(c color.Color) board.BitBoard {
	var out board.BitBoard
	for _, p := range e.pieces {
		if p.c == c && p.pt != piece.PawnType {
			sfSetBit(&out, p.row, p.col)
		}
	}
	return out
}

// pieceTypeAt returns the piece type at a 0..63 bit-square index, and whether
// the square is occupied at all.
func (e *sfEval) pieceTypeAt(sq int) (pieceType byte, ok bool) {
	pt, _, ok := e.b.GetPieceTypeColor(location.NewLocation(location.CoordinateType(sq/board.Width), location.CoordinateType(sq%board.Width)))
	return pt, ok
}

// pawnsOn returns the bitboard of color c's pawns that stand on squares in `mask`.
func (e *sfEval) pawnsOn(c color.Color, mask board.BitBoard) board.BitBoard {
	var out board.BitBoard
	for _, p := range e.pieces {
		if p.c == c && p.pt == piece.PawnType {
			if mask&(board.BitBoard(1)<<uint(board.Width*p.row+p.col)) != 0 {
				sfSetBit(&out, p.row, p.col)
			}
		}
	}
	return out
}

// pawnAttacksFrom returns all squares attacked by the color-c pawns in `pawns`.
func (e *sfEval) pawnAttacksFrom(c color.Color, pawns board.BitBoard) board.BitBoard {
	var out board.BitBoard
	forward := 1
	if c == color.Black {
		forward = -1
	}
	for x := uint64(pawns); x != 0; x &= x - 1 {
		sq := bits.TrailingZeros64(x)
		row, col := sq/board.Width, sq%board.Width
		ar := row + forward
		if ar < 0 || ar >= board.Height {
			continue
		}
		for _, dc := range [2]int{-1, 1} {
			ac := col + dc
			if ac >= 0 && ac < board.Width {
				sfSetBit(&out, ar, ac)
			}
		}
	}
	return out
}

// kingDanger returns a positive penalty score for the king of color `us` based on the
// number and weight of enemy attackers near the king ring. Faithful-in-spirit subset
// of Stockfish's king-safety accumulator (no explicit safe-check enumeration).
func (e *sfEval) kingDanger(us color.Color) sfScore {
	if e.phase <= 64 { // negligible in deep endgames
		return sfScore{}
	}
	them := us ^ 1
	k := e.b.KingLocations[us]
	kr, kc := int(k.GetRow()), int(k.GetCol())
	// Keep a full 3x3 ring on-board by clamping the ring center to [1,6].
	cr, cc := kr, kc
	if cr < 1 {
		cr = 1
	} else if cr > 6 {
		cr = 6
	}
	if cc < 1 {
		cc = 1
	} else if cc > 6 {
		cc = 6
	}
	var kingRing board.BitBoard
	for dr := -1; dr <= 1; dr++ {
		for dc := -1; dc <= 1; dc++ {
			sfSetBit(&kingRing, cr+dr, cc+dc)
		}
	}

	shelterPawns := 0
	for _, p := range e.pieces {
		if p.c == us && p.pt == piece.PawnType {
			sq := board.BitBoard(1) << uint(board.Width*p.row+p.col)
			if kingRing&sq != 0 {
				shelterPawns++
			}
		}
	}

	attackersCount, attackersWeight, attacksOnRing := 0, 0, 0
	// Approximate Stockfish's safe-check/king-attack terms that this port does
	// not enumerate explicitly. This is deliberately gated on an exposed king:
	// queen contact is dangerous when the pawn shelter is gone, but overvalued
	// when f/g/h shelter pawns still blunt the attack.
	closeQueenPressure := 0
	closeQueenMinorPressure := 0
	attackerHasQueen := e.pieceCount[them][piece.QueenType] > 0
	for _, p := range e.pieces {
		if p.c != them {
			continue
		}
		w := sfKingAttackWeight[p.pt]
		if w == 0 {
			continue
		}
		hit := p.attacks.IntersectBitBoards(kingRing)
		if hit != 0 {
			attackersCount++
			attackersWeight += w
			ringHits := sfPopcnt(hit)
			attacksOnRing += ringHits
			if shelterPawns <= 1 &&
				p.pt == piece.QueenType &&
				sfChebyshev(p.row, p.col, kr, kc) <= 2 {
				closeQueenPressure += 700 + 80*ringHits
			}
			if attackerHasQueen &&
				(p.pt == piece.KnightType || p.pt == piece.BishopType) &&
				sfChebyshev(p.row, p.col, kr, kc) <= 2 {
				closeQueenMinorPressure += 1000 + 180*ringHits
			}
		}
	}
	if attackersCount == 0 {
		return sfScore{}
	}
	// The bespoke contact-pressure terms approximate SF's safe-check
	// enumeration but are additive per piece: queen + two minors camping next
	// to an exposed king can add ~3000 raw units, which the quadratic mg term
	// below turns into a >2000cp swing (engine +2669 vs SF +380 in game
	// o6lAdkjC — the eval overconfidence made every attacking move look
	// winning and lost a won game). A lower cap (600-1100) was tried and
	// regressed the bench badly (74→121cp: the terms are load-bearing for
	// finding real attacks), so only clip the pathological multi-piece pileup.
	const maxContactPressure = 1800
	if closeQueenPressure+closeQueenMinorPressure > maxContactPressure {
		total := closeQueenPressure + closeQueenMinorPressure
		closeQueenPressure = closeQueenPressure * maxContactPressure / total
		closeQueenMinorPressure = closeQueenMinorPressure * maxContactPressure / total
	}
	// Undefended attacked ring squares.
	undefended := kingRing.IntersectBitBoards(e.attackedByAll[them]).IntersectBitBoards(^e.attackedByAll[us])

	kingDanger := attackersCount*attackersWeight +
		69*attacksOnRing +
		60*sfPopcnt(undefended) +
		30*attackersCount +
		closeQueenPressure +
		closeQueenMinorPressure
	if e.pieceCount[them][piece.QueenType] == 0 {
		kingDanger -= 400 // attacks are far less dangerous without the enemy queen
	}
	if kingDanger <= 100 {
		return sfScore{}
	}
	return s2(kingDanger*kingDanger/4096, kingDanger/16)
}

// queenlessKingActivity rewards stepping the king into the game once queens are
// off and the remaining material is rook/minor scale. The normal tapered king PSQT
// underweights this because multiple rooks keep the phase high.
func (e *sfEval) queenlessKingActivity(us color.Color) sfScore {
	if e.pieceCount[color.White][piece.QueenType]+e.pieceCount[color.Black][piece.QueenType] != 0 {
		return sfScore{}
	}
	nonPawn := 0
	for _, c := range []color.Color{color.White, color.Black} {
		nonPawn += e.pieceCount[c][piece.RookType]
		nonPawn += e.pieceCount[c][piece.KnightType]
		nonPawn += e.pieceCount[c][piece.BishopType]
	}
	if nonPawn > 6 {
		return sfScore{}
	}

	k := e.b.KingLocations[us]
	row, col := int(k.GetRow()), int(k.GetCol())
	relRank := sfRelRank(us, row)
	if relRank > 3 {
		relRank = 3
	}

	// Doubled Manhattan distance to the board center (3.5, 3.5), so all integer
	// math and no tie between the four center squares.
	centerDist := abs(2*row-7) + abs(2*col-7)
	centerActivity := 14 - centerDist
	if centerActivity < 0 {
		centerActivity = 0
	}

	activity := 110*relRank + 10*centerActivity
	return s2(activity, activity)
}

// space rewards safe squares in the center files behind/near a side's own pawns,
// scaled by the number of own pieces (matters only with material on the board).
func (e *sfEval) space(us color.Color) sfScore {
	// Non-pawn material gate (SF SpaceThreshold ≈ 12222 internal units).
	nonPawnMat := 0
	for _, pt := range []byte{piece.KnightType, piece.BishopType, piece.RookType, piece.QueenType} {
		nonPawnMat += e.pieceCount[us][pt] * sfPieceValue[pt].mg
	}
	if nonPawnMat < 12222 {
		return sfScore{}
	}
	them := us ^ 1
	// Center files c..f (cols 2..5), relative ranks 1..3 (rank 2,3,4 from own side).
	safeCount := 0
	for _, relRank := range []int{1, 2, 3} {
		var row int
		if us == color.White {
			row = relRank
		} else {
			row = 7 - relRank
		}
		for col := 2; col <= 5; col++ {
			sq := board.BitBoard(1) << uint(board.Width*row+col)
			// Safe = not occupied by our pawn, not attacked by an enemy pawn.
			pt, pc, ok := e.b.GetPieceTypeColor(location.NewLocation(location.CoordinateType(row), location.CoordinateType(col)))
			if ok && pc == us && pt == piece.PawnType {
				continue
			}
			if e.attackedBy[them][piece.PawnType]&sq != 0 {
				continue
			}
			safeCount++
		}
	}
	weight := e.pieceCount[us][piece.KnightType] + e.pieceCount[us][piece.BishopType] +
		e.pieceCount[us][piece.RookType] + e.pieceCount[us][piece.QueenType]
	return s2(safeCount*weight*weight/16, 0)
}

// initiative nudges the endgame score by a "complexity" measure of the position
// (pawn count, passers, king outflanking, both-flank pawns, bare-king bonus), in the
// direction of whichever side is currently ahead in the endgame score.
func (e *sfEval) initiative(eg int) int {
	pawnCount := e.pieceCount[color.White][piece.PawnType] + e.pieceCount[color.Black][piece.PawnType]

	passedCount := 0
	queenSide, kingSide := false, false
	for _, p := range e.pieces {
		if p.pt != piece.PawnType {
			continue
		}
		if isPassedPawn(e.b, location.CoordinateType(p.row), location.CoordinateType(p.col), p.c) {
			passedCount++
		}
		if p.col >= 4 {
			queenSide = true
		} else {
			kingSide = true
		}
	}
	bothFlanks := 0
	if queenSide && kingSide {
		bothFlanks = 1
	}

	wk := e.b.KingLocations[color.White]
	bk := e.b.KingLocations[color.Black]
	dCol := int(wk.GetCol()) - int(bk.GetCol())
	if dCol < 0 {
		dCol = -dCol
	}
	dRow := int(wk.GetRow()) - int(bk.GetRow())
	if dRow < 0 {
		dRow = -dRow
	}
	outflanking := dCol - dRow

	nonPawnMat := 0
	for _, c := range []color.Color{color.White, color.Black} {
		for _, pt := range []byte{piece.KnightType, piece.BishopType, piece.RookType, piece.QueenType} {
			nonPawnMat += e.pieceCount[c][pt]
		}
	}
	bareKings := 0
	if nonPawnMat == 0 {
		bareKings = 1
	}

	complexity := 9*passedCount + 11*pawnCount + 9*outflanking + 18*bothFlanks + 49*bareKings - 103

	sign := 0
	if eg > 0 {
		sign = 1
	} else if eg < 0 {
		sign = -1
	}
	// Don't let the initiative term flip the sign of the endgame score.
	adj := complexity
	if adj < -abs(eg) {
		adj = -abs(eg)
	}
	return sign * adj
}
