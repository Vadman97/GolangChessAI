package game

const (
	Active                      = byte(iota)
	WhiteWin                    = byte(iota)
	BlackWin                    = byte(iota)
	RegularDraw                 = byte(iota)
	FiftyMoveDraw               = byte(iota)
	RepeatedActionThreeTimeDraw = byte(iota)
	Aborted                     = byte(iota)
)

var StatusStrings = [...]string{
	"Active",
	"White Win",
	"Black Win",
	"Generic Draw",
	"Fifty Move Draw",
	"Repeated Action Three Times Draw",
	"Aborted",
}
