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
	BitsPerPiece = 4
	BytesPerRow  = Width * BitsPerPiece / 8

	NumFlagBits = 4
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
	board [Height][BytesPerRow]byte

	// max 4 flags if we use byte
	flags byte
}

func (b *Board) Copy() *Board {
	newBoard := Board{}
	for i := 0; i < Height; i++ {
		for j := 0; j < BytesPerRow; j++ {
			newBoard.board[i][j] = b.board[i][j]
		}
	}
	newBoard.flags = b.flags
	return &newBoard
}

func (b *Board) SetPiece(l Location, p Piece) {
	colStart, colEnd := getColIdx(l)
	// set the bytes associated with this piece (only 1 if we store piece in 4 bytes)
	for i := colStart; i < colEnd; i++ {
		b.board[l.Row][i] |= encodeData(p)[i-colStart]
	}
}

func (b *Board) GetPiece(l Location) Piece {
	colStart, colEnd := getColIdx(l)
	data := b.board[l.Row][colStart:colEnd]
	return decodeData(l, data)
}

func (b *Board) SetFlag(flag byte, color byte, value bool) {
	if value {
		b.flags |= (1 << flag) << (color * NumFlagBits)
	} else {
		b.flags &= ^((1 << flag) << (color * NumFlagBits))
	}
}

func (b *Board) GetFlag(flag byte, color byte) bool {
	return (b.flags & ^((1 << flag) << (color * NumFlagBits))) != 0
}

func (b *Board) Move(m *Move) {
	// copy start piece to end
	startColStart, startColEnd := getColIdx(m.start)
	endColStart, endColEnd := getColIdx(m.end)
	b.board[m.end.Row][endColStart:endColEnd] = b.board[m.start.Row][startColStart:startColEnd]

	// note: encode is fast, decode is slower TODO(Vadim) verify this
	// clear piece at start
	b.SetPiece(m.start, nil)
}

func getColIdx(l Location) (cStart, cEnd byte) {
	cStart = l.Col / BitsPerPiece
	cEnd = cStart + 1
	if l.Col%4 != 0 {
		cEnd++
	}
	return
}

func decodeData(l Location, data []byte) Piece {
	// constants: 3 upper bits contain piece type, bottom 1 bit contains color
	pieceTypeData := data[0] & 0xE
	colorData := data[0] & 0x1

	var p Piece
	if pieceTypeData == piece.RookType {
		p = Rook{}
	} else if pieceTypeData == piece.NilPiece {
		return p
	} else {
		log.Fatal("Unknown piece type - error during decode")
	}
	// TODO(Vadim) else if for all types
	p.SetPosition(l)
	p.SetColor(colorData)
	return p
}

func encodeData(p Piece) (data [1]byte) {
	if p != nil {
		data[0] |= byte(0x7&p.GetPieceType()) << 1
		data[0] |= 0x1 & p.GetColor()
	} else {
		data[0] |= piece.NilPiece << 1
	}
	return
}
