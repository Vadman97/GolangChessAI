package board

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/Vadman97/ChessAI3/pkg/chessai/util"
	"log"
	"math/rand"
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

var StartRow = map[byte]map[string]int8{
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
}

func (b *Board) Hash() (result [33]byte) {
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
	return &newBoard
}

func (b *Board) ResetDefault() {
	b.board[0] = StartingRowHex[0]
	b.board[1] = StartingRowHex[1]
	b.board[6] = StartingRowHex[6]
	b.board[7] = StartingRowHex[7]
	b.MoveCache = util.NewConcurrentBoardMap()
	b.AttackableCache = util.NewConcurrentBoardMap()
}

func (b *Board) ResetDefaultSlow() {
	for c := int8(0); c < Width; c++ {
		StartingRow[c].SetPosition(location.Location{0, c})
		StartingRow[c].SetColor(color.Black)
		b.SetPiece(location.Location{0, c}, StartingRow[c])
		b.SetPiece(location.Location{1, c}, &Pawn{location.Location{Row: 1, Col: c}, color.Black})

		b.SetPiece(location.Location{6, c}, &Pawn{location.Location{Row: 6, Col: c}, color.White})
		StartingRow[c].SetPosition(location.Location{7, c})
		StartingRow[c].SetColor(color.White)
		b.SetPiece(location.Location{7, c}, StartingRow[c])
	}
}

func (b *Board) SetPiece(l location.Location, p Piece) {
	// set the bytes associated with this piece (only 1 if we store piece in 4 bytes)
	data := uint32(encodeData(p)) << getBitOffset(l)
	b.board[l.Row] &^= PieceMask << getBitOffset(l)
	b.board[l.Row] |= data
}

func (b *Board) GetPiece(l location.Location) Piece {
	data := b.getPieceData(l)
	return decodeData(l, data)
}

func (b *Board) getPieceData(l location.Location) byte {
	pos := getBitOffset(l)
	return byte((b.board[l.Row] & (PieceMask << pos)) >> pos)
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
			result += fmt.Sprintf("%+v", GetColorTypeRepr(b.GetPiece(location.Location{int8(r), int8(c)})))
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
	moves := *b.GetAllMoves(byte(rand.Int() % color.NumColors))
	if len(moves) > 0 {
		MakeMove(&moves[rand.Int()%len(moves)], b)
	}
}

func (b *Board) RandomizeIllegal() {
	// random board with random pieces (not fully random cuz i'm lazy)
	if b.TestRandGen == nil {
		b.TestRandGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	for r := int8(0); r < Height; r++ {
		for c := int8(0); c < Width; c++ {
			p := StartingRow[b.TestRandGen.Int()%len(StartingRow)]
			p.SetPosition(location.Location{r, c})
			p.SetColor(byte(b.TestRandGen.Int() % 2))
			b.SetPiece(location.Location{r, c}, p)
		}
	}
	b.flags = byte(b.TestRandGen.Uint32())
}

/*
 *	Only this is cached and not GetAllAttackableMoves for now because this calls GetAllAttackableMoves
 *	May need to cache that one too when we use it for CheckMate / Tie evaluation
 */
func (b *Board) GetAllMoves(color byte) *[]location.Move {
	// TODO(Vadim) when king under attack, moves that block check are the only possible ones
	h := b.Hash()
	entry := &MoveCacheEntry{
		moves: make(map[byte]interface{}),
	}
	if cacheEntry, cacheExists := b.MoveCache.Read(&h); cacheExists {
		entry = cacheEntry.(*MoveCacheEntry)
		// we've gotten the other color but not the one we want
		if entry.moves[color] == nil {
			entry.moves[color] = b.getAllMoves(color)
			b.MoveCache.Store(&h, entry)
		}
	} else {
		entry.moves[color] = b.getAllMoves(color)
		b.MoveCache.Store(&h, entry)
	}
	return entry.moves[color].(*[]location.Move)
}

func (b *Board) getAllMoves(color byte) *[]location.Move {
	var moves []location.Move
	for r := 0; r < Height; r++ {
		// this is just a speedup - if the whole row is empty don't look at pieces
		if b.board[r] == 0 {
			continue
		}
		for c := 0; c < Width; c++ {
			l := location.Location{int8(r), int8(c)}
			if !b.IsEmpty(l) {
				p := b.GetPiece(l)
				if p.GetColor() == color {
					moves = append(moves, *p.GetMoves(b)...)
				}
			}
		}
	}
	return &moves
}

/*
 * Caches getAllAttackableMoves
 */
func (b *Board) GetAllAttackableMoves(color byte) AttackableBoard {
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
			loc := location.Location{Row: int8(r), Col: int8(c)}
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

func (b *Board) move(m *location.Move) {
	// more efficient function than using SetPiece(end, GetPiece(start)) - tested with benchmark

	// copy Start piece to End
	startOff := getBitOffset(m.Start)
	endOff := getBitOffset(m.End)
	data := (b.board[m.Start.Row] & (PieceMask << startOff)) >> startOff
	b.board[m.End.Row] &^= PieceMask << endOff
	b.board[m.End.Row] |= data << endOff

	// clear piece at Start
	b.SetPiece(m.Start, nil)
}

func getBitOffset(l location.Location) byte {
	// 28 = ((Width - 1) * BitsPerPiece)
	return 28 - byte(l.Col*BitsPerPiece)
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
