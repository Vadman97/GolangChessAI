package location

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
)

type CoordinateType = uint8

type RelativeLocation struct {
	Row, Col int8
}

var UpMove = RelativeLocation{-1, 0}
var RightUpMove = RelativeLocation{-1, 1}
var RightMove = RelativeLocation{0, 1}
var RightDownMove = RelativeLocation{1, 1}
var DownMove = RelativeLocation{1, 0}
var LeftDownMove = RelativeLocation{1, -1}
var LeftMove = RelativeLocation{0, -1}
var LeftUpMove = RelativeLocation{-1, -1}

var Cols = []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}

type Location struct {
	// row stored in 3 bits, col stored in 3 bits
	// 2 bits store pawn promotion piece
	data byte
}

func NewLocation(row, col CoordinateType) (l Location) {
	l.data |= (byte(row) & 0x7) << 5
	l.data |= (byte(col) & 0x7) << 2
	return
}

func (l *Location) Set(v Location) {
	l.data = v.data
}

func (l Location) CreatePawnPromotion(promotedType byte) Location {
	found := false
	for _, option := range piece.PawnPromotionOptions {
		if promotedType == option {
			found = true
			break
		}
	}
	if !found {
		panic("trying to promote pawn to invalid piece type")
	}
	// 0 corresponds to no promotion, thus the +1
	l.data |= promotedType - piece.PawnPromotionOptions[0] + 1
	return l
}

func (l *Location) GetPawnPromotion() (isPromotion bool, promotedType byte) {
	data := l.data & 0x3
	if data != 0 {
		isPromotion = true
		promotedType = data + piece.PawnPromotionOptions[0] - 1
	}
	return
}

func (l Location) Get() (row, col CoordinateType) {
	row = CoordinateType((l.data & (byte(0x7) << 5)) >> 5)
	col = CoordinateType((l.data & (byte(0x7) << 2)) >> 2)
	return
}

func (l Location) GetRow() (row CoordinateType) {
	row, _ = l.Get()
	return
}

func (l Location) GetColLetter() rune {
	_, col := l.Get()
	return Cols[col]
}

func (l Location) GetCol() (col CoordinateType) {
	_, col = l.Get()
	return
}

func (l Location) GetPromotionPiece() byte {
	return l.data & (byte(0x3))
}

func (l Location) Add(v Location) Location {
	row, col := l.Get()
	row2, col2 := v.Get()
	return NewLocation(row+row2, col+col2)
}

func (l Location) AddRelative(v RelativeLocation) (res Location, sumInBounds bool) {
	row, col := l.Get()
	r, c := int8(row)+v.Row, int8(col)+v.Col
	sumInBounds = inBounds(r, c)
	if sumInBounds {
		res = NewLocation(CoordinateType(r), CoordinateType(c))
	}
	return
}

func (l Location) Equals(v Location) bool {
	return v.data == l.data
}

func inBounds(row, col int8) bool {
	return row >= 0 && col >= 0 && row < 8 && col < 8
}

func (l Location) String() string {
	r, c := l.Get()
	return fmt.Sprintf("(%+v, %+v)", r, c)
}

type Move struct {
	Start, End Location
}

func (m *Move) GetStart() Location {
	return m.Start
}

func (m *Move) GetEnd() Location {
	return m.End
}

func (m *Move) Set(v *Move) {
	m.Start.Set(v.Start)
	m.End.Set(v.End)
}

func (m *Move) Equals(v *Move) bool {
	return m.Start.Equals(v.Start) && m.End.Equals(v.End)
}

func (m Move) String() string {
	return fmt.Sprintf("move from %s to %s", m.Start.String(), m.End.String())
}

func (m Move) UCIString() string {
	return fmt.Sprintf("%c%d%c%d", m.Start.GetColLetter(), m.Start.GetRow()+1, m.End.GetColLetter(), m.End.GetCol()+1)
}
