package api

import (
	"encoding/json"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"log"
	"time"
)

const (
	PlayerMove            = "playerMove"
	AIMove                = "aiMove"
	AvailablePlayerMoves  = "availablePlayerMoves"
	GameState             = "gameState"
	GameFull              = "gameFull"
	GameNotAvailable      = "gameNotAvailable"
)

type ChessMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}


type GameStateJSON struct {
	CurrentBoard     [board.Height][board.Width]*PieceJSON  `json:"currentBoard"`
	CurrentTurnColor string                                 `json:"currentTurn"`
	MovesPlayed      uint	                                `json:"movesPlayed"`
	PreviousMove     *MoveJSON                              `json:"previousMove"`
	GameStatus       string                                 `json:"gameStatus"`
	MoveLimit        int32                                  `json:"moveLimit"`
	TimeLimit        time.Duration                          `json:"timeLimit"`
}

type PieceJSON struct {
	PieceType  string  `json:"type"`
	Color      string  `json:"color"`
}

type MoveJSON struct {
	Start     [2]uint8   `json:"start"`
	End       [2]uint8   `json:"end"`
	IsCapture bool       `json:"isCapture"`
	Piece     PieceJSON  `json:"piece"`
}

type AvailableMovesJSON struct {
	Moves []*MoveJSON
}

func CreateChessMessage(msgType string, data interface{}) ChessMessage {
	dataBytes, err := json.Marshal(data)

	if err != nil {
		log.Printf("Unable to create Chess Message - %v", err)
		return ChessMessage{}
	}

	chessMessage := ChessMessage{
		Type: msgType,
		Data: string(dataBytes),
	}

	return chessMessage
}
