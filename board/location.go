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
	Row, Col byte
}

func (l *Location) Set(v Location) {
	l.Row = v.Row
	l.Col = v.Col
}

func (l *Location) Add(v Location) {
	l.Row += v.Row
	l.Col += v.Col
}

func (l *Location) Sub(v Location) byte {
	return byte(math.Abs(float64(v.Row-l.Row)) + math.Abs(float64(v.Col-l.Col)))
}

func (l *Location) Equals(v Location) bool {
	return v.Row == l.Row && v.Col == l.Col
}

func (l *Location) Print() string {
	return fmt.Sprintf("(R: %d, C: %d)", l.Row, l.Col)
}

func (l *Location) InBounds() bool {
	// Row, Col cannot be < 0 because byte is unsigned
	return l.Row < Height && l.Col < Width
}

type Move struct {
	start, end Location
}

func (m *Move) GetStart() Location {
	return m.start
}

func (m *Move) GetEnd() Location {
	return m.end
}

func (m *Move) Set(v *Move) {
	m.start.Set(v.start)
	m.end.Set(v.end)
}

func (m *Move) Equals(v *Move) bool {
	return m.start.Equals(v.start) && m.end.Equals(v.end)
}

func (m *Move) Print() string {
	return fmt.Sprintf("Move from %s to %s", m.start.Print(), m.end.Print())
}

func (m *Move) GetDistance() byte {
	return m.end.Sub(m.start)
}
