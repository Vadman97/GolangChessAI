package board

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"log"
	"math/rand"
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
	&Queen{},
	&King{},
	&Bishop{},
	&Knight{},
	&Rook{},
}

var StartingRowHex = [...]uint32{
	0x3579B753,
	0xDDDDDDDD,
	0, 0, 0, 0,
	0xCCCCCCCC,
	0x2468A642,
}

var StartRow = map[byte]map[string]location.CoordinateType{
	color.Black: {
		"Piece": 0,
		"Pawn":  1,
	},
	color.White: {
		"Pawn":  6,
		"Piece": 7,
	},
}

type MoveCacheEntry struct {
	// color -> move
	moves map[byte]interface{}
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

	// MovesSinceNoDraw stores the number of moves since no draw conditions have occurred
	// draw conditions: pawn hasn't moved or piece hasn't been captured for 50 turns each side
	MovesSinceNoDraw int

	CacheGetAllMoves, CacheGetAllAttackableMoves bool
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
	return &newBoard
}

func (b *Board) ResetDefault() {
	b.board[0] = StartingRowHex[0]
	b.board[1] = StartingRowHex[1]
	b.board[6] = StartingRowHex[6]
	b.board[7] = StartingRowHex[7]
	b.MoveCache = util.NewConcurrentBoardMap()
	b.AttackableCache = util.NewConcurrentBoardMap()
	b.KingLocations = [color.NumColors]location.Location{
		location.NewLocation(7, 4),
		location.NewLocation(0, 4),
	}
	b.MovesSinceNoDraw = 0
	b.CacheGetAllMoves = config.Get().CacheGetAllMoves
	b.CacheGetAllAttackableMoves = config.Get().CacheGetAllAttackableMoves
	b.KingLocations = [color.NumColors]location.Location{
		location.NewLocation(7, 4),
		location.NewLocation(0, 4),
	}
}

func (b *Board) ResetDefaultSlow() {
	b.ResetDefault()
	for c := location.CoordinateType(0); c < Width; c++ {
		StartingRow[c].SetPosition(location.NewLocation(0, c))
		StartingRow[c].SetColor(color.Black)
		b.SetPiece(location.NewLocation(0, c), StartingRow[c])
		b.SetPiece(location.NewLocation(1, c), &Pawn{location.NewLocation(1, c), color.Black})

		b.SetPiece(location.NewLocation(6, c), &Pawn{location.NewLocation(6, c), color.White})
		StartingRow[c].SetPosition(location.NewLocation(7, c))
		StartingRow[c].SetColor(color.White)
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

func (b *Board) SetFlag(flag byte, color byte, value bool) {
	if value {
		b.flags |= (1 << flag) << (color * NumFlagBits)
	} else {
		b.flags &^= (1 << flag) << (color * NumFlagBits)
	}
}

func (b *Board) GetFlag(flag byte, color byte) bool {
	return (b.flags & ((1 << flag) << (color * NumFlagBits))) != 0
}

func (b *Board) IsEmpty(l location.Location) bool {
	return b.getPieceData(l) == 0
}

func (b *Board) Print() (result string) {
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
	moves := *b.GetAllMoves(byte(rand.Int()%color.NumColors), nil)
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
			p.SetColor(byte(b.TestRandGen.Int() % 2))
			b.SetPiece(location.NewLocation(r, c), p)
		}
	}
	b.flags = byte(b.TestRandGen.Uint32())
}

/**
 * Check if color has at least one legal move, optimized
 */
func (b *Board) HasLegalMove(color byte, previousMove *LastMove) bool {
	return len(*b.getAllMovesCached(color, previousMove, true)) > 0
}

func (b *Board) GetAllMoves(color byte, previousMove *LastMove) *[]location.Move {
	return b.getAllMovesCached(color, previousMove, false)
}

/*
 *	Only this is cached and not GetAllAttackableMoves for now because this calls GetAllAttackableMoves
 *	May need to cache that one too when we use it for CheckMate / Tie evaluation
 */
func (b *Board) getAllMovesCached(color byte, previousMove *LastMove, onlyFirstMove bool) *[]location.Move {
	entry := &MoveCacheEntry{
		moves: make(map[byte]interface{}),
	}
	if b.CacheGetAllMoves {
		h := b.Hash()
		if cacheEntry, cacheExists := b.MoveCache.Read(&h); cacheExists {
			entry = cacheEntry.(*MoveCacheEntry)
			// we've gotten the other color but not the one we want
			if entry.moves[color] == nil {
				entry.moves[color] = b.getAllMoves(color, onlyFirstMove)
				// store only if we grabbing all moves
				if !onlyFirstMove {
					b.MoveCache.Store(&h, entry)
				}
			}
		} else {
			entry.moves[color] = b.getAllMoves(color, onlyFirstMove)
			// store only if we grabbing all moves
			if !onlyFirstMove {
				b.MoveCache.Store(&h, entry)
			}
		}
	} else {
		entry.moves[color] = b.getAllMoves(color, onlyFirstMove)
	}
	if previousMove != nil {
		enPassantMoves := b.GetEnPassantMoves(color, previousMove)
		allMoves := append(*(entry.moves[color].(*[]location.Move)), *enPassantMoves...)
		return &allMoves
	}
	return entry.moves[color].(*[]location.Move)
}

/**
 * Get moves for all pieces of color c.
 * If onlyFirstMove is set, will only return first move
 */
func (b *Board) getAllMoves(c byte, onlyFirstMove bool) *[]location.Move {
	var moves []location.Move
	for row := 0; row < Height; row++ {
		// this is just a speedup - if the whole row is empty don't look at pieces
		if b.board[row] == 0 {
			continue
		}
		for col := 0; col < Width; col++ {
			l := location.NewLocation(location.CoordinateType(row), location.CoordinateType(col))
			if !b.IsEmpty(l) {
				p := b.GetPiece(l)
				if p.GetColor() == c {
					additionalMoves := *p.GetMoves(b, onlyFirstMove)
					for _, nextMove := range additionalMoves {
						if !b.willMoveLeaveKingInCheck(c, nextMove) {
							moves = append(moves, nextMove)
						}
					}
					// the first move left the king in check, must try more moves (extra optimization not possible)
					if onlyFirstMove && len(moves) == 0 {
						// we get all the moves from the king to try but still break on first legal
						additionalMoves := *p.GetMoves(b, false)
						for _, nextMove := range additionalMoves {
							if !b.willMoveLeaveKingInCheck(c, nextMove) {
								// still just grab the first move
								moves = append(moves, nextMove)
								return &moves
							}
						}
					}
				}
			}
		}
	}
	return &moves
}

/**
 * Determines possible en passant moves based on color, board, and last move.
 */
func (b *Board) GetEnPassantMoves(c byte, previousMove *LastMove) *[]location.Move {
	if previousMove == nil {
		return nil
	}
	var enPassantMoves []location.Move
	lastPieceMoved := *previousMove.Piece
	if (lastPieceMoved.GetColor() != c) && (lastPieceMoved.GetPieceType() == piece.PawnType) {
		move := previousMove.Move
		var captureLocation *location.Location = nil
		startRow, _ := move.Start.Get()
		endRow, _ := move.End.Get()
		if lastPieceMoved.GetColor() == color.Black {
			if (startRow == 1) && (endRow == 3) {
				l, _ := move.End.AddRelative(location.UpMove)
				captureLocation = &l
			}
		} else {
			if (startRow == 6) && (endRow == 4) {
				l, _ := move.End.AddRelative(location.DownMove)
				captureLocation = &l
			}
		}
		if captureLocation != nil {
			for i := int8(-1); i <= 1; i += 2 {
				adjacentLoc, inBounds := move.End.AddRelative(location.RelativeLocation{0, i})
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
func (b *Board) GetAllAttackableMoves(color byte) AttackableBoard {
	if b.CacheGetAllAttackableMoves {
		h := b.Hash()
		entry := &MoveCacheEntry{
			moves: make(map[byte]interface{}),
		}
		if cacheEntry, cacheExists := b.AttackableCache.Read(&h); cacheExists {
			entry = cacheEntry.(*MoveCacheEntry)
			// we've gotten the other color but not the one we want
			if entry.moves[color] == nil {
				entry.moves[color] = b.getAllAttackableMoves(color)
				b.AttackableCache.Store(&h, entry)
			}
		} else {
			entry.moves[color] = b.getAllAttackableMoves(color)
			b.AttackableCache.Store(&h, entry)
		}
		return entry.moves[color].(AttackableBoard)
	} else {
		return b.getAllAttackableMoves(color)
	}
}

/**
 * Returns all attack moves for a specific color.
 */
func (b *Board) getAllAttackableMoves(color byte) AttackableBoard {
	attackable := CreateEmptyAttackableBoard()
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
					attackable = CombineAttackableBoards(attackable, attackableMoves)
				}
			}
		}
	}
	return attackable
}

/**
 * Determines if a king of color c is under attack by the opposite color.
 */
func (b *Board) IsKingInCheck(c byte) bool {
	oppositeColor := c ^ 1
	attackableBoard := b.GetAllAttackableMoves(oppositeColor)
	return IsLocationUnderAttack(attackableBoard, b.KingLocations[c])
}

/**
 * Applies a move to the board and then checks to see if it will result in king of color c being in check.
 * This could mean that the king was in check and will still be in check, or that king has been put into check as a
 * result of the move.
 */
func (b *Board) willMoveLeaveKingInCheck(c byte, m location.Move) bool {
	boardCopy := b.Copy()
	MakeMove(&m, boardCopy)
	return boardCopy.IsKingInCheck(c)
}

/**
 * Checks if the king of color c is in checkmate.
 */
func (b *Board) IsInCheckmate(c byte, previousMove *LastMove) bool {
	return !b.HasLegalMove(c, previousMove) && b.IsKingInCheck(c)
}

/**
 * Checks if the board is in a stalemate based on color c not having any moves and its king is also not in check.
 */
func (b *Board) IsStalemate(c byte, previousMove *LastMove) bool {
	return !b.HasLegalMove(c, previousMove) && !b.IsKingInCheck(c)
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
		log.Fatal("Unknown piece type - error during decode: ", pieceTypeData)
	}
	return nil
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
