package board

import (
	"ChessAI3/chessai/board/piece"
	"log"
)

const (
	Height = 8
	Width  = 8
)

const (
	// 3 bits for piece type
	// 1 bit for piece color
	PiecesPerRow = Width
	BitsPerPiece = 4
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

type Board struct {
	// TODO(Vadim) more efficient to use ints - faster to copy int than set of bytes
	board [Height]uint32

	// max 4 flags if we use byte
	flags byte
}

func (b *Board) Copy() *Board {
	newBoard := Board{}
	for i := 0; i < Height; i++ {
		newBoard.board[i] = b.board[i]
	}
	newBoard.flags = b.flags
	return &newBoard
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
	return (b.flags & ^((1 << flag) << (color * NumFlagBits))) != 0
}

func (b *Board) move(m *Move) {
	// copy Start piece to End
	pos := getBitOffset(m.Start)
	data := (b.board[m.Start.Row] & (PieceMask << pos)) >> pos
	b.board[m.End.Row] &= PieceMask << pos
	b.board[m.End.Row] |= data

	// note: encode is fast, decode is slower TODO(Vadim) verify this
	// clear piece at Start
	b.SetPiece(m.Start, nil)
}

func getBitOffset(l Location) byte {
	return (l.Col % PiecesPerRow) * BitsPerPiece
}

func decodeData(l Location, data byte) Piece {
	// constants: 3 upper bits contain piece type, bottom 1 bit contains color
	pieceTypeData := (data & 0xE) >> 1
	colorData := data & 0x1

	var p Piece
	if pieceTypeData == piece.RookType {
		p = Rook{}
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
	// piece type in upper bits, color in bottom bit
	if p != nil {
		data |= 0xE & (byte(p.GetPieceType()) << 1)
		data |= 0x1 & p.GetColor()
	}
	return
}
