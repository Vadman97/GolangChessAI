package game

const (
	Active                      = byte(iota)
	WhiteWin                    = byte(iota)
	BlackWin                    = byte(iota)
	Stalemate                   = byte(iota)
	FiftyMoveDraw               = byte(iota)
	RepeatedActionThreeTimeDraw = byte(iota)
	InsufficientMaterialDraw    = byte(iota)
	Aborted                     = byte(iota)
)

// IsClaimableDraw reports whether a status is a draw the side to move can claim
// while still having legal moves available (threefold repetition / fifty-move
// rule). Lichess does not end the game automatically on these — a bot that simply
// stops moving when it detects one locally will flag and lose. Unlike checkmate /
// stalemate (no legal move) or insufficient material (auto-drawn by the server),
// these require the bot to keep playing and claim the draw via its move.
func IsClaimableDraw(status byte) bool {
	return status == RepeatedActionThreeTimeDraw || status == FiftyMoveDraw
}

var StatusStrings = [...]string{
	"Active",
	"White Win",
	"Black Win",
	"Stalemate",
	"Fifty Move Draw",
	"Repeated Action Three Times Draw",
	"Insufficient Material Draw",
	"Aborted",
}
