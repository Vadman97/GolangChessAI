package board

import (
	"fmt"
	"math"
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

func (l *Location) InBounds() bool {
	// Row, Col cannot be < 0 because byte is unsigned
	return l.Row < Height && l.Col < Width
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

func (m *Move) Print() string {
	return fmt.Sprintf("move from %+v to %+v", m.Start, m.End)
}

func (m *Move) GetDistance() byte {
	return m.End.Sub(m.Start)
}
