package location

import (
	"fmt"
	"math"
)

var UpMove = Location{-1, 0}
var RightUpMove = Location{-1, 1}
var RightMove = Location{0, 1}
var RightDownMove = Location{1, 1}
var DownMove = Location{1, 0}
var LeftDownMove = Location{1, -1}
var LeftMove = Location{0, -1}
var LeftUpMove = Location{-1, -1}

type Location struct {
	Row, Col int8
}

func (l *Location) Set(v Location) {
	l.Row = v.Row
	l.Col = v.Col
}

func (l *Location) Add(v Location) Location {
	newLoc := Location{l.Row, l.Col}
	newLoc.Row += v.Row
	newLoc.Col += v.Col
	return newLoc
}

func (l *Location) Sub(v Location) byte {
	return byte(math.Abs(float64(v.Row-l.Row)) + math.Abs(float64(v.Col-l.Col)))
}

func (l *Location) Equals(v Location) bool {
	return v.Row == l.Row && v.Col == l.Col
}

func (l *Location) InBounds() bool {
	return l.Row >= 0 && l.Col >= 0 && l.Row < 8 && l.Col < 8
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
