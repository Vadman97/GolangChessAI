package game

const (
	Active                           = byte(iota)
	WhiteWin                         = byte(iota)
	BlackWin                         = byte(iota)
	RegularStalemate                 = byte(iota)
	FiftyMoveStalemate               = byte(iota)
	RepeatedActionThreeTimeStalemate = byte(iota)
	Aborted                          = byte(iota)
)

var StatusStrings = [...]string{
	"Active",
	"White Win",
	"Black Win",
	"Generic Stalemate",
	"Fifty Move Stalemate",
	"Repeated Action Three Times Stalemate",
	"Aborted",
}
