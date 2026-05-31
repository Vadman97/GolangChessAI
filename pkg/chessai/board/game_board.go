package board

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/util"
	"math/rand"
	"sort"
	"strings"
	"time"
)

const (
	Height = 8
	Width  = 8
)

const (
	// 3 bits for piece type
	// 1 bit for piece Color
	BitsPerPiece = 4
	BytesPerRow  = Width * BitsPerPiece / 8
	PieceMask    = 0xF // BitsPerPiece 1's
	NumFlagBits  = 4
)

// Board Flags
const (
	FlagKingMoved      = iota
	FlagCastled        = iota
	FlagLeftRookMoved  = iota
	FlagRightRookMoved = iota
)

var StartingRow = [...]Piece{
	&Rook{},
	&Knight{},
	&Bishop{},
	&King{},
	&Queen{},
	&Bishop{},
	&Knight{},
	&Rook{},
}

var StartingRowHex = [...]uint32{
	0x246A8642,
	0xCCCCCCCC,
	0, 0, 0, 0,
	0xDDDDDDDD,
	0x357B9753,
}

var StartRow = map[color.Color]map[string]location.CoordinateType{
	color.White: {
		"Piece": 0,
		"Pawn":  1,
	},
	color.Black: {
		"Pawn":  6,
		"Piece": 7,
	},
}

type Board struct {
	// board stores entire layout of pieces on the Width * Height board
	// more efficient to use ints - faster to copy int than set of bytes
	board [Height]uint32

	// flags store information global to board, eg has white king moved
	// max 4 flags if we use byte
	flags byte

	TestRandGen                *rand.Rand
	MoveCache, AttackableCache *util.ConcurrentBoardMap
	KingLocations              [color.NumColors]location.Location

	CacheGetAllMoves, CacheGetAllAttackableMoves bool

	// MovesSinceNoDraw stores the number of moves since no draw conditions have occurred
	// draw conditions: pawn hasn't moved or piece hasn't been captured for 50 turns each side
	MovesSinceNoDraw int
	// previous boards that we have encountered for 3-move-repetition
	PreviousPositions []util.BoardHash
	// number of previous positions we have seen
	PreviousPositionsSeen int

	// Pin data cached for the duration of a single getAllMoves call.
	// pinMaskValid is true only inside getAllMoves; willMoveLeaveKingInCheck uses it
	// to skip the expensive make/unmake+ray-cast for non-pinned, non-king pieces.
	pinMaskValid bool
	pinnedSquares uint64 // bit row*8+col set → that square holds a pinned friendly piece
	kingInCheck   bool   // true if the side-to-move's king is in check at the start of getAllMoves
}

func (b *Board) Hash() (result util.BoardHash) {
	// TODO(Vadim) evenly distribute output over {1,0}^264 via SHA256?
	// store into map[uint64]map[uint64]map[uint64]map[uint64]map[byte]uint32
	// Want to lookup score for a board using hash value
	// Board stored in (8 * 4 + 1) bytes = 33bytes
	for i := 0; i < Height; i++ {
		for bIdx := 0; bIdx < BytesPerRow; bIdx++ {
			p := b.board[i] & (0xFF << byte(bIdx*8)) >> byte(bIdx*8)
			result[i*BytesPerRow+bIdx] |= byte(p)
		}
	}
	result[32] = b.flags
	// 76 ns/op

	//h := sha1.New()
	//hashed := h.Sum(result[:])
	//
	//length := len(result)
	//if len(hashed) < length {
	//	length = len(hashed)
	//}
	//for i := 0; i < length; i++ {
	//	result[i] = hashed[i]
	//}
	// SHA1     461 ns/op
	// SHA2-256 630 ns/op
	// SHA3-256 1415 ns/op

	return result
}

func (b *Board) Equals(board *Board) bool {
	if board.flags == b.flags {
		for i := 0; i < Height; i++ {
			if board.board[i] != b.board[i] {
				return false
			}
		}
		return true
	}
	return false
}

func (b *Board) Copy() *Board {
	newBoard := Board{}
	for i := 0; i < Height; i++ {
		newBoard.board[i] = b.board[i]
	}
	newBoard.flags = b.flags
	newBoard.MoveCache = b.MoveCache
	newBoard.AttackableCache = b.AttackableCache
	newBoard.KingLocations = b.KingLocations
	newBoard.MovesSinceNoDraw = b.MovesSinceNoDraw
	newBoard.CacheGetAllMoves = b.CacheGetAllMoves
	newBoard.CacheGetAllAttackableMoves = b.CacheGetAllAttackableMoves
	newBoard.PreviousPositions = b.PreviousPositions
	newBoard.PreviousPositionsSeen = b.PreviousPositionsSeen
	return &newBoard
}

func (b *Board) ResetDefault() {
	b.board = StartingRowHex
	b.MoveCache = util.NewConcurrentBoardMap()
	b.AttackableCache = util.NewConcurrentBoardMap()
	b.KingLocations = [color.NumColors]location.Location{
		location.NewLocation(0, 3),
		location.NewLocation(7, 3),
	}
	b.MovesSinceNoDraw = 0
	b.CacheGetAllMoves = config.Get().CacheGetAllMoves
	b.CacheGetAllAttackableMoves = config.Get().CacheGetAllAttackableMoves
	b.PreviousPositions = nil
	b.PreviousPositionsSeen = 0
}

func (b *Board) ResetDefaultSlow() {
	b.ResetDefault()
	for c := location.CoordinateType(0); c < Width; c++ {
		StartingRow[c].SetPosition(location.NewLocation(0, c))
		StartingRow[c].SetColor(color.White)
		b.SetPiece(location.NewLocation(0, c), StartingRow[c])
		b.SetPiece(location.NewLocation(1, c), &Pawn{location.NewLocation(1, c), color.White})

		b.SetPiece(location.NewLocation(6, c), &Pawn{location.NewLocation(6, c), color.Black})
		StartingRow[c].SetPosition(location.NewLocation(7, c))
		StartingRow[c].SetColor(color.Black)
		b.SetPiece(location.NewLocation(7, c), StartingRow[c])
	}
}

func (b *Board) SetPiece(l location.Location, p Piece) {
	// set the bytes associated with this piece (only 1 if we store piece in 4 bytes)
	data := uint32(encodeData(p)) << getBitOffset(l)
	row, _ := l.Get()
	b.board[row] &^= PieceMask << getBitOffset(l)
	b.board[row] |= data
}

func (b *Board) GetPiece(l location.Location) Piece {
	data := b.getPieceData(l)
	return decodeData(l, data)
}

func (b *Board) getPieceData(l location.Location) byte {
	pos := getBitOffset(l)
	row, _ := l.Get()
	return byte((b.board[row] & (PieceMask << pos)) >> pos)
}

func (b *Board) SetFlag(flag byte, color color.Color, value bool) {
	if value {
		b.flags |= (1 << flag) << (color * NumFlagBits)
	} else {
		b.flags &^= (1 << flag) << (color * NumFlagBits)
	}
}

func (b *Board) GetFlag(flag byte, color color.Color) bool {
	return (b.flags & ((1 << flag) << (color * NumFlagBits))) != 0
}

func (b *Board) IsEmpty(l location.Location) bool {
	return b.getPieceData(l) == 0
}

func (b Board) String() (result string) {
	/*
		B_R|B_K|B_B|B_Q|B_&|B_B|B_K|B_R
		B_P|B_P|000|B_P|B_P|B_P|B_P|B_P
		000|000|B_P|000|000|000|000|000
		000|000|000|000|000|000|000|000
		000|000|000|000|000|000|000|000
		000|000|000|000|000|000|000|000
		W_P|W_P|W_P|W_P|W_P|W_P|W_P|W_P
		W_R|W_K|W_B|W_Q|W_&|W_B|W_K|W_R
	*/
	for c := 0; c < Height; c++ {
		if c != 0 {
			result += " "
		} else {
			result += "   "
		}
		result += fmt.Sprintf("%d  ", c)
	}
	result += "\n"
	for r := 0; r < Height; r++ {
		result += fmt.Sprintf("%d ", r)
		for c := 0; c < Height; c++ {
			result += fmt.Sprintf("%+v", GetColorTypeRepr(b.GetPiece(location.NewLocation(location.CoordinateType(r), location.CoordinateType(c)))))
			if c < Height-1 {
				result += "|"
			}
		}

		result += fmt.Sprintf(" %d\n", r)
	}
	for c := 0; c < Height; c++ {
		if c != 0 {
			result += " "
		} else {
			result += "   "
		}
		result += fmt.Sprintf("%d  ", c)
	}
	result += "\n"
	return
}

func (b *Board) MakeRandomMove() {
	moves := *b.GetAllMoves(color.Color(rand.Int()%color.NumColors), nil)
	if len(moves) > 0 {
		MakeMove(&moves[rand.Int()%len(moves)], b)
	}
}

func (b *Board) RandomizeIllegal() {
	// random board with random pieces (not fully random cuz i'm lazy)
	if b.TestRandGen == nil {
		b.TestRandGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	for r := location.CoordinateType(0); r < Height; r++ {
		for c := location.CoordinateType(0); c < Width; c++ {
			p := StartingRow[b.TestRandGen.Int()%len(StartingRow)]
			p.SetPosition(location.NewLocation(r, c))
			p.SetColor(color.Color(b.TestRandGen.Int() % 2))
			b.SetPiece(location.NewLocation(r, c), p)
		}
	}
	b.flags = byte(b.TestRandGen.Uint32())
}

/**
 * Check if color has at least one legal move, optimized
 */
func (b *Board) HasLegalMove(color color.Color, previousMove *LastMove) bool {
	return len(*b.getAllMovesCached(color, previousMove, true)) > 0
}

// mvvLvaValue maps piece type to its relative value for MVV-LVA move ordering.
var mvvLvaValue = [7]int{
	0,   // NilType
	5,   // RookType
	3,   // KnightType
	3,   // BishopType
	9,   // QueenType
	100, // KingType
	1,   // PawnType
}

// mvvLvaScore returns a score for a move used to order moves for alpha-beta search.
// Captures of high-value pieces by low-value attackers score highest.
// Non-captures return 0.
func (b *Board) mvvLvaScore(m location.Move) int {
	victimData := b.getPieceData(m.End)
	if victimData == 0 {
		return 0
	}
	attackerData := b.getPieceData(m.Start)
	if attackerData == 0 {
		return 0
	}
	return mvvLvaValue[(victimData&0xE)>>1]*10 - mvvLvaValue[(attackerData&0xE)>>1]
}

func (b *Board) GetAllMoves(color color.Color, previousMove *LastMove) *[]location.Move {
	movesPtr := b.getAllMovesCached(color, previousMove, false)
	if config.Get().RandomMoveOrder {
		moves := *movesPtr
		rand.Shuffle(len(moves), func(i, j int) {
			moves[i], moves[j] = moves[j], moves[i]
		})
		movesPtr = &moves
	} else {
		// MVV-LVA ordering: captures first (high-value victim, low-value attacker), non-captures after.
		// Pre-compute scores to avoid calling mvvLvaScore twice per sort comparison.
		moves := make([]location.Move, len(*movesPtr))
		copy(moves, *movesPtr)
		scores := make([]int, len(moves))
		for i, m := range moves {
			scores[i] = b.mvvLvaScore(m)
		}
		sort.Slice(moves, func(i, j int) bool {
			return scores[i] > scores[j]
		})
		movesPtr = &moves
	}
	return movesPtr
}

func (b *Board) GetAllMovesUnShuffled(color byte, previousMove *LastMove) *[]location.Move {
	return b.getAllMovesCached(color, previousMove, false)
}

/*
 *	Only this is cached and not GetAllAttackableMoves for now because this calls GetAllAttackableMoves
 *	May need to cache that one too when we use it for CheckMate / Tie evaluation
 */
func (b *Board) getAllMovesCached(c color.Color, previousMove *LastMove, onlyFirstMove bool) *[]location.Move {
	var moves *[]location.Move
	if b.CacheGetAllMoves && !onlyFirstMove {
		h := b.Hash()
		if cacheEntry, cacheExists := b.MoveCache.Read(&h, c); cacheExists {
			moves = cacheEntry.(*[]location.Move)
		} else {
			moves = b.getAllMoves(c, onlyFirstMove)
			b.MoveCache.Store(&h, c, moves)
		}

	} else {
		moves = b.getAllMoves(c, onlyFirstMove)
	}
	if previousMove != nil {
		if !onlyFirstMove || (onlyFirstMove && len(*moves) == 0) {
			enPassantMoves := b.getEnPassantMoves(c, previousMove)
			allMoves := append(*moves, *enPassantMoves...)
			return &allMoves
		}
	}
	return moves
}

/**
 * Get moves for all pieces of color c.
 * If onlyFirstMove is set, will only return first move
 */
func (b *Board) getAllMoves(c color.Color, onlyFirstMove bool) *[]location.Move {
	// Compute pin data once for this call so willMoveLeaveKingInCheck can fast-path
	// non-pinned, non-king pieces when the king is not in check.
	b.pinnedSquares, b.kingInCheck = b.computePinData(c)
	b.pinMaskValid = true
	defer func() { b.pinMaskValid = false }()

	moves := make([]location.Move, 0, 40)
	for row := 0; row < Height; row++ {
		rowData := b.board[row]
		if rowData == 0 {
			continue
		}
		// Extract each piece's 4-bit nibble directly from rowData to avoid calling
		// getPieceData twice (once for IsEmpty, once inside GetPiece) and skip the
		// color check before allocating a Piece object.
		// col 0 is stored in the most-significant nibble (bits 31-28), col 7 in bits 3-0.
		tmp := rowData
		for col := uint8(0); col < Width; col++ {
			data := byte(tmp >> 28)
			tmp <<= 4
			if data == 0 || data&0x1 != c {
				continue
			}
			l := location.NewLocation(location.CoordinateType(row), location.CoordinateType(col))
			p := decodeData(l, data)
			additionalMoves := *p.GetMoves(b, onlyFirstMove)
			for _, nextMove := range additionalMoves {
				moves = append(moves, nextMove)
				if onlyFirstMove {
					return &moves
				}
			}
		}
	}
	return &moves
}

/**
 * Determines possible en passant moves based on color, board, and last move.
 */
func (b *Board) getEnPassantMoves(c color.Color, previousMove *LastMove) *[]location.Move {
	if previousMove == nil {
		return nil
	}
	var enPassantMoves []location.Move
	lastPieceMoved := *previousMove.Piece
	if pawn, isPawn := lastPieceMoved.(*Pawn); isPawn && (pawn.GetColor() != c) {
		move := previousMove.Move
		var captureLocation *location.Location = nil
		startRow, _ := move.Start.Get()
		endRow, _ := move.End.Get()
		expectedEnd := StartRow[pawn.Color]["Pawn"] + location.CoordinateType(pawn.forward(2).Row)
		if (startRow == StartRow[pawn.Color]["Pawn"]) && (endRow == expectedEnd) {
			// The capturing pawn lands on the square the double-pushed pawn skipped
			// over: one step ahead of where that pawn started (e.g. black f7-f5 is
			// captured on f6). Using move.End + forward(1) here pointed the wrong way
			// (toward f4) and produced an illegal backwards capture.
			l, _ := move.Start.AddRelative(pawn.forward(1))
			captureLocation = &l
		}
		if captureLocation != nil {
			for i := int8(-1); i <= 1; i += 2 {
				adjacentLoc, inBounds := move.End.AddRelative(location.RelativeLocation{Col: i})
				if inBounds {
					adjacentPiece := b.GetPiece(adjacentLoc)
					if adjacentPiece != nil && adjacentPiece.GetColor() == c && adjacentPiece.GetPieceType() == piece.PawnType {
						potentialMove := location.Move{Start: adjacentLoc, End: *captureLocation}
						if !b.willMoveLeaveKingInCheck(c, potentialMove) {
							enPassantMoves = append(enPassantMoves, potentialMove)
						}
					}
				}
			}
		}
	}
	return &enPassantMoves
}

/*
 * Caches getAllAttackableMoves
 */
func (b *Board) GetAllAttackableMoves(c color.Color) BitBoard {
	if b.CacheGetAllAttackableMoves {
		h := b.Hash()
		var attackable BitBoard
		if cacheEntry, cacheExists := b.AttackableCache.Read(&h, c); cacheExists {
			attackable = cacheEntry.(BitBoard)
		} else {
			attackable = b.getAllAttackableMoves(c)
		}
		b.AttackableCache.Store(&h, c, attackable)
		return attackable
	} else {
		return b.getAllAttackableMoves(c)
	}
}

/**
 * Returns all attack moves for a specific color.
 */
func (b *Board) getAllAttackableMoves(color color.Color) BitBoard {
	attackable := BitBoard(0)
	for r := 0; r < Height; r++ {
		// this is just a speedup - if the whole row is empty don't look at pieces
		if b.board[r] == 0 {
			continue
		}
		for c := 0; c < Width; c++ {
			loc := location.NewLocation(location.CoordinateType(r), location.CoordinateType(c))
			if !b.IsEmpty(loc) {
				pieceOnLocation := b.GetPiece(loc)
				if pieceOnLocation.GetColor() == color {
					attackableMoves := pieceOnLocation.GetAttackableMoves(b)
					attackable = attackable.CombineBitBoards(attackableMoves)
				}
			}
		}
	}
	return attackable
}

/**
 * Returns all attack moves for a specific color.
 */
func (b *Board) GetPieceAttackableMoves(color color.Color) (boards [piece.NumPieces]BitBoard) {
	for pieceType := piece.PawnType; pieceType < piece.NumPieces; pieceType++ {
		boards[pieceType] = BitBoard(0)
	}
	for r := 0; r < Height; r++ {
		// this is just a speedup - if the whole row is empty don't look at pieces
		if b.board[r] == 0 {
			continue
		}
		for c := 0; c < Width; c++ {
			loc := location.NewLocation(location.CoordinateType(r), location.CoordinateType(c))
			if !b.IsEmpty(loc) {
				pieceOnLocation := b.GetPiece(loc)
				if pieceOnLocation.GetColor() == color {
					pType := pieceOnLocation.GetPieceType()
					attackableMoves := pieceOnLocation.GetAttackableMoves(b)
					boards[pType] = boards[pType].CombineBitBoards(attackableMoves)
				}
			}
		}
	}
	return
}

/**
 * Return all available moves for a specific color
 * Differs from GetAllAttackableMoves() since no cache is involved and this returns an map of piece to moves
 * The map key will be a stringified coordinate `(r,c)`
 */
func (b *Board) GetAllAvailableMoves(color color.Color) map[string]*[]location.Move {
	var moveMap = make(map[string]*[]location.Move)

	for r := 0; r < Height; r++ {
		if b.board[r] == 0 {
			continue
		}
		for c := 0; c < Width; c++ {
			loc := location.NewLocation(location.CoordinateType(r), location.CoordinateType(c))
			if !b.IsEmpty(loc) {
				pieceOnLocation := b.GetPiece(loc)
				if pieceOnLocation.GetColor() == color {
					moves := pieceOnLocation.GetMoves(b, false)
					moveMap[loc.String()] = moves
				}
			}
		}
	}

	return moveMap
}

/**
 * Determines if a king of color c is under attack by the opposite color.
 */
func (b *Board) IsKingInCheck(c color.Color) bool {
	oppositeColor := c ^ 1
	bitBoard := b.GetAllAttackableMoves(oppositeColor)
	return bitBoard.IsLocationSet(b.KingLocations[c])
}

// willMoveLeaveKingInCheck returns true if applying m would leave color c's king in check.
// Uses make/unmake with ray-cast detection instead of board copy + full attack map.
//
// Fast path: if pin data is cached for this getAllMoves call AND the king is not currently
// in check AND the moving piece is not pinned AND it is not the king, the move is always
// legal (it cannot expose the king). This eliminates ~85-90% of expensive ray-cast calls.
func (b *Board) willMoveLeaveKingInCheck(c color.Color, m location.Move) bool {
	if b.pinMaskValid && !b.kingInCheck {
		data := b.getPieceData(m.Start)
		if (data&0xE)>>1 != piece.KingType {
			row, col := m.Start.Get()
			if b.pinnedSquares&(uint64(1)<<(uint64(row)*8+uint64(col))) == 0 {
				return false
			}
		}
	}
	undo := b.makeFastMove(&m)
	inCheck := b.isKingInCheckFast(b.KingLocations[c], c)
	b.unmakeFastMove(undo)
	return inCheck
}

/**
 * Checks if the king of color c is in checkmate.
 */
func (b *Board) IsInCheckmate(c color.Color, previousMove *LastMove) bool {
	return !b.HasLegalMove(c, previousMove) && b.IsKingInCheck(c)
}

/**
 * Checks if the board is in a stalemate based on color c not having any moves and its king is also not in check.
 */
func (b *Board) IsStalemate(c byte, previousMove *LastMove) bool {
	return !b.HasLegalMove(c, previousMove) && !b.IsKingInCheck(c)
}

// IsInsufficientMaterial returns true when neither side has enough material to
// force checkmate. Covers K vs K, K+minor vs K, and K+B vs K+B same square color.
func (b *Board) IsInsufficientMaterial() bool {
	type pieceInfo struct {
		pieceType   byte
		squareColor byte // (row+col)%2: 0=light, 1=dark
	}
	var whitePieces, blackPieces []pieceInfo
	for row := location.CoordinateType(0); row < Height; row++ {
		for col := location.CoordinateType(0); col < Width; col++ {
			p := b.GetPiece(location.NewLocation(row, col))
			if p == nil || p.GetPieceType() == piece.KingType {
				continue
			}
			info := pieceInfo{p.GetPieceType(), (row + col) % 2}
			if p.GetColor() == color.White {
				whitePieces = append(whitePieces, info)
			} else {
				blackPieces = append(blackPieces, info)
			}
		}
	}
	wn, bn := len(whitePieces), len(blackPieces)
	// K vs K
	if wn == 0 && bn == 0 {
		return true
	}
	// K+N vs K or K+B vs K
	if (wn == 1 && bn == 0) || (wn == 0 && bn == 1) {
		var solo pieceInfo
		if wn == 1 {
			solo = whitePieces[0]
		} else {
			solo = blackPieces[0]
		}
		return solo.pieceType == piece.KnightType || solo.pieceType == piece.BishopType
	}
	// K+B vs K+B, same square color
	if wn == 1 && bn == 1 {
		if whitePieces[0].pieceType == piece.BishopType && blackPieces[0].pieceType == piece.BishopType {
			return whitePieces[0].squareColor == blackPieces[0].squareColor
		}
	}
	return false
}

/**
 * Checks if the board is reaching a draw based on the previous move (pawn movement, piece capture)
 */
func (b *Board) UpdateDrawCounter(previousMove *LastMove) {
	lastMovedPiece := *previousMove.Piece
	if lastMovedPiece.GetPieceType() == piece.PawnType || previousMove.IsCapture {
		b.MovesSinceNoDraw = 0
	} else {
		b.MovesSinceNoDraw++
	}
}

/**
 * Load board from text for tests
 */
func (b *Board) LoadBoardFromText(boardRows []string) {
	for _, r := range boardRows {
		fmt.Println(r)
	}
	for r := location.CoordinateType(0); r < Height; r++ {
		pieces := strings.Split(boardRows[r], "|")
		for c, pStr := range pieces {
			l := location.NewLocation(r, location.CoordinateType(c))
			var p Piece
			if pStr != "   " && len(pStr) == 3 {
				d := strings.Split(pStr, "_")
				cChar, pChar := rune(d[0][0]), rune(d[1][0])
				p = PieceFromType(piece.NameToType[pChar])
				if p.GetPieceType() == piece.KingType {
					b.KingLocations[ColorFromChar(cChar)] = l
				}
				p.SetColor(ColorFromChar(cChar))
				p.SetPosition(l)
			}
			b.SetPiece(l, p)
		}
	}
}

func (b *Board) move(m *location.Move) {
	// more efficient function than using SetPiece(end, GetPiece(start)) - tested with benchmark

	// copy Start piece to End
	startOff := getBitOffset(m.Start)
	endOff := getBitOffset(m.End)
	startRow := m.Start.GetRow()
	endRow := m.End.GetRow()
	data := (b.board[startRow] & (PieceMask << startOff)) >> startOff
	b.board[endRow] &^= PieceMask << endOff
	b.board[endRow] |= data << endOff

	// clear piece at Start
	b.SetPiece(m.Start, nil)
}

func getBitOffset(l location.Location) byte {
	// 28 = ((Width - 1) * BitsPerPiece)
	return 28 - byte(l.GetCol()*BitsPerPiece)
}

func ColorFromChar(cChar rune) byte {
	if cChar == color.BlackChar {
		return color.Black
	} else if cChar == color.WhiteChar {
		return color.White
	}
	return 0xFF
}

func PieceFromType(pieceTypeData byte) Piece {
	if pieceTypeData == piece.NilType {
		return nil
	} else if pieceTypeData == piece.RookType {
		return &Rook{}
	} else if pieceTypeData == piece.KnightType {
		return &Knight{}
	} else if pieceTypeData == piece.BishopType {
		return &Bishop{}
	} else if pieceTypeData == piece.QueenType {
		return &Queen{}
	} else if pieceTypeData == piece.KingType {
		return &King{}
	} else if pieceTypeData == piece.PawnType {
		return &Pawn{}
	} else {
		panic(fmt.Sprintf("Unknown piece type - error during decode: %b", pieceTypeData))
	}
}

func decodeData(l location.Location, data byte) Piece {
	// constants: 3 upper bits contain piece type, bottom 1 bit contains Color
	pieceTypeData := (data & 0xE) >> 1

	if pieceTypeData == piece.NilType {
		return nil
	}
	colorData := data & 0x1

	p := PieceFromType(pieceTypeData)
	p.SetPosition(l)
	p.SetColor(colorData)
	return p
}

// Returns piece data in lower bits
func encodeData(p Piece) (data byte) {
	// piece type in upper bits, Color in bottom bit
	if p != nil {
		data |= 0xE & (byte(p.GetPieceType()) << 1)
		data |= 0x1 & p.GetColor()
	}
	return
}
