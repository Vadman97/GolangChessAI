package board

import (
	"fmt"
	"math"
)

const (
	Height = 8
	Width  = 8
)

type Location struct {
	row, col byte
}

func (l *Location) Set(v *Location) {
	l.row = v.row
	l.col = v.col
}

func (l *Location) Add(v *Location) {
	l.row += v.row
	l.col += v.col
}

func (l *Location) Sub(v *Location) byte {
	return byte(math.Abs(float64(v.row-l.row)) + math.Abs(float64(v.col-l.col)))
}

func (l *Location) Equals(v *Location) bool {
	return v.row == l.row && v.col == l.col
}

func (l *Location) Print() string {
	return fmt.Sprintf("(R: %d, C: %d)", l.row, l.col)
}

func (l *Location) InBounds() bool {
	// row, col cannot be < 0 because byte is unsigned
	return l.row < Height && l.col < Width
}

type Move struct {
	start, end Location
}

func (m *Move) Set(v *Move) {
	m.start.Set(&v.start)
	m.end.Set(&v.end)
}

func (m *Move) Equals(v *Move) bool {
	return m.start.Equals(&v.start) && m.end.Equals(&v.end)
}

func (m *Move) Print() string {
	return fmt.Sprintf("Move from %s to %s", m.start.Print(), m.end.Print())
}

func (m *Move) GetDistance() byte {
	return m.end.Sub(&m.start)
}
