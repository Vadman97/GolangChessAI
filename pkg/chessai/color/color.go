package color

type Color = byte

const (
	White     = Color(iota)
	Black     = Color(iota)
	WhiteChar = 'W'
	BlackChar = 'B'
	NumColors = 2
)

var Names = [...]string{
	"White", "Black",
}
