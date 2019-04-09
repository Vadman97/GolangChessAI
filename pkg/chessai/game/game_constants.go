package game

const (
	Active    = byte(iota)
	WhiteWin  = byte(iota)
	BlackWin  = byte(iota)
	Stalemate = byte(iota)
	Aborted   = byte(iota)
)

var StatusStrings = [...]string{
	"Active",
	"White Win",
	"Black Win",
	"Stalemate",
	"Aborted",
}
