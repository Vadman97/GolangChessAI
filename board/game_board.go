package board

import (
	"chessAI/board/piece"
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
)

type Board struct {
	board [Height][BytesPerRow]byte
	flags byte // TODO(Vadim) set function, copy properly (copy func)
}

func (b *Board) SetPiece(l Location, p piece.Piece) {
	colStart, colEnd := getColIdx(l)
	b.board[l.Row][colStart:colEnd] = encodeData(p)[:]
}

func (b *Board) GetPiece(l Location) piece.Piece {
	colStart, colEnd := getColIdx(l)
	data := b.board[l.Row][colStart:colEnd]
	return decodeData(l, data)
}

func getColIdx(l Location) (cStart, cEnd byte) {
	cStart = l.Col / 4
	cEnd = cStart
	if l.Col%4 != 0 {
		cEnd = cStart + 1
	}
	return
}

func decodeData(l Location, data []byte) piece.Piece {
	pieceTypeData := data[0] & 0xE
	colorData := data[0] & 0x1

	var p piece.Piece
	if pieceTypeData == piece.RookChar {
		p = piece.Rook{}
	} else {
		log.Fatal("Unknown piece type - error during decode")
	}
	// TODO(Vadim) else if for all types
	p.SetPosition(l)
	p.SetColor(colorData)
	return p
}

func encodeData(p piece.Piece) (data [1]byte) {
	data[0] |= byte(0x7&p.GetChar()) << 1
	data[0] |= 0x1 & p.GetColor()
	return
}
