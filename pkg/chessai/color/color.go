package color

const (
	White     = byte(iota)
	Black     = byte(iota)
	WhiteChar = 'W'
	BlackChar = 'B'
	NumColors = 2
)

var Names = [...]string{
	"White", "Black",
}
