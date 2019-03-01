package board

import (
	"ChessAI3/chessai/board/color"
	"ChessAI3/chessai/board/piece"
	"fmt"
	"log"
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
	0x357b9753,
	0xdddddddd,
	0, 0, 0, 0,
	0xcccccccc,
	0x246a8642,
}

type Board struct {
	// board stores entire layout of pieces on the Width * Height board
	// more efficient to use ints - faster to copy int than set of bytes
	board [Height]uint32

	// flags store information global to board, eg has white king moved
	// max 4 flags if we use byte
	flags byte
}

func (b *Board) Hash() (result [33]byte) {
	// var scoreMap map[uint64]map[uint64]map[uint64]map[uint64]map[uint64]uint32
	// Want to lookup score for a board using hash value
	// Board stored in (8 * 4 + 1) bytes = 33bytes
	for i := 0; i < Height; i++ {
		for bIdx := 0; bIdx < BytesPerRow; bIdx++ {
			result[i*BytesPerRow+bIdx] |= byte(b.board[i] & (PieceMask << byte(bIdx*BytesPerRow)))
		}
	}
	result[32] = b.flags
	return
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
	// return reflect.DeepEqual(board.Hash(), b.Hash())
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
	for c := byte(0); c < Width; c++ {
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
	pos := getBitOffset(l)
	data := byte((b.board[l.Row] & (PieceMask << pos)) >> pos)
	return decodeData(l, data)
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

func (b *Board) Print() (result string) {
	for r := 0; r < Height; r++ {
		result += fmt.Sprintf("%#x\n", b.board[r])
	}
	return
}

func (b *Board) move(m *Move) {
	// more efficient function than using SetPiece(end, GetPiece(start)) - tested with benchmark

	// copy Start piece to End
	pos := getBitOffset(m.Start)
	data := (b.board[m.Start.Row] & (PieceMask << pos)) >> pos
	b.board[m.End.Row] &= PieceMask << pos
	b.board[m.End.Row] |= data

	// clear piece at Start
	b.SetPiece(m.Start, nil)
}

func getBitOffset(l Location) byte {
	return (l.Col % PiecesPerRow) * BitsPerPiece
}

func decodeData(l Location, data byte) Piece {
	// constants: 3 upper bits contain piece type, bottom 1 bit contains Color
	pieceTypeData := (data & 0xE) >> 1
	colorData := data & 0x1

	var p Piece
	if pieceTypeData == piece.RookType {
		p = &Rook{}
	} else if pieceTypeData == piece.NilType {
		return p
	} else {
		log.Fatal("Unknown piece type - error during decode: ", pieceTypeData)
	}
	// TODO(Vadim) else if for all types
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
