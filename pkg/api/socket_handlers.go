package api

import "github.com/gorilla/websocket"

const (
	PlayerMove            = "playerMove"
	AIMove                = "aiMove"
	AvailablePlayerMoves  = "availablePlayerMoves"
	GameStatus            = "gameStatus"
	GameFull              = "gameFull"
)

type ChessMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

func HandlePlayerMove(moveJSON MoveJSON, client *websocket.Conn) {
	// TODO(Alex) Parse the Move and play it in the game
}

func SendGameStatus() {

}


