package board

import (
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/piece"
	"fmt"
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
	PiecesPerRow = Width
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

type Board struct {
	// board stores entire layout of pieces on the Width * Height board
	// more efficient to use ints - faster to copy int than set of bytes
	board [Height]uint32

	// flags store information global to board, eg has white king moved
	// max 4 flags if we use byte
	flags byte

	TestRandGen *rand.Rand
}

func (b *Board) Hash() (result [33]byte) {
	// TODO(Vadim) evenly distribute output over {1,0}^264 via SHA256?
	// store into map[uint64]map[uint64]map[uint64]map[uint64]map[byte]uint32
	// Want to lookup score for a board using hash value
	// Board stored in (8 * 4 + 1) bytes = 33bytes
	for i := 0; i < Height; i++ {
		for bIdx := 0; bIdx < BytesPerRow; bIdx++ {
			result[i*BytesPerRow+bIdx] |= byte(b.board[i] & (PieceMask << byte(bIdx*BytesPerRow)))
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
	return &newBoard
}

func (b *Board) ResetDefault() {
	b.board[0] = StartingRowHex[0]
	b.board[1] = StartingRowHex[1]
	b.board[6] = StartingRowHex[6]
	b.board[7] = StartingRowHex[7]
}

func (b *Board) ResetDefaultSlow() {
	for c := int8(0); c < Width; c++ {
		StartingRow[c].SetPosition(Location{0, c})
		StartingRow[c].SetColor(color.Black)
		b.SetPiece(Location{0, c}, StartingRow[c])
		b.SetPiece(Location{1, c}, &Pawn{Location{Row: 1, Col: c}, color.Black})

		b.SetPiece(Location{6, c}, &Pawn{Location{Row: 6, Col: c}, color.White})
		StartingRow[c].SetPosition(Location{7, c})
		StartingRow[c].SetColor(color.White)
		b.SetPiece(Location{7, c}, StartingRow[c])
	}
}

func (b *Board) SetPiece(l Location, p Piece) {
	// set the bytes associated with this piece (only 1 if we store piece in 4 bytes)
	data := uint32(encodeData(p)) << getBitOffset(l)
	b.board[l.Row] &^= PieceMask << getBitOffset(l)
	b.board[l.Row] |= data
}

func (b *Board) GetPiece(l Location) Piece {
	data := b.getPieceData(l)
	return decodeData(l, data)
}

func (b *Board) getPieceData(l Location) byte {
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

func (b *Board) IsEmpty(l Location) bool {
	return b.getPieceData(l) == 0
}

func (b *Board) Print() (result string) {
	for r := 0; r < Height; r++ {
		result += fmt.Sprintf("%#x\n", b.board[r])
	}
	return
}

func (b *Board) RandomizeIllegal() {
	// random board with random pieces (not fully random cuz i'm lazy)
	if b.TestRandGen == nil {
		b.TestRandGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	for r := int8(0); r < Height; r++ {
		for c := int8(0); c < Width; c++ {
			p := StartingRow[b.TestRandGen.Int()%len(StartingRow)]
			p.SetPosition(Location{r, c})
			p.SetColor(byte(b.TestRandGen.Int() % 2))
			b.SetPiece(Location{r, c}, p)
		}
	}
	b.flags = byte(b.TestRandGen.Uint32())
}

func (b *Board) GetAllMoves(getBlack, getWhite bool) (black, white *[]Move) {
	var blackMoves, whiteMoves []Move
	// TODO(Vadim) think of how to optimize this, profile it and write tests
	for r := 0; r < Height; r++ {
		if b.board[r] == 0 {
			continue
		}
		for c := 0; c < Width; c++ {
			l := Location{int8(r), int8(c)}
			if !b.IsEmpty(l) {
				p := b.GetPiece(l)
				moves := p.GetMoves(b)
				if moves != nil {
					if getBlack && p.GetColor() == color.Black {
						blackMoves = append(blackMoves, *moves...)
					} else if getWhite && p.GetColor() == color.White {
						whiteMoves = append(whiteMoves, *moves...)
					}
				}
			}
		}
	}
	if getBlack {
		black = &blackMoves
	}
	if getWhite {
		white = &whiteMoves
	}
	return
}

func (b *Board) move(m *Move) {
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

func getBitOffset(l Location) byte {
	// 28 = ((Width - 1) * BitsPerPiece)
	return 28 - byte(l.Col*BitsPerPiece)
}

func decodeData(l Location, data byte) Piece {
	// constants: 3 upper bits contain piece type, bottom 1 bit contains Color
	pieceTypeData := (data & 0xE) >> 1

	if pieceTypeData == piece.NilType {
		return nil
	}

	var p Piece
	colorData := data & 0x1
	if pieceTypeData == piece.RookType {
		p = &Rook{}
	} else if pieceTypeData == piece.KnightType {
		p = &Knight{}
	} else if pieceTypeData == piece.BishopType {
		p = &Bishop{}
	} else if pieceTypeData == piece.QueenType {
		p = &Queen{}
	} else if pieceTypeData == piece.KingType {
		p = &King{}
	} else if pieceTypeData == piece.PawnType {
		p = &Pawn{}
	} else {
		log.Fatal("Unknown piece type - error during decode: ", pieceTypeData)
	}
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
