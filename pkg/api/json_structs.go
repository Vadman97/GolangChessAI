package api

import (
	"encoding/json"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
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
	HumanColor       string                                 `json:"humanColor"`
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
	AvailableMoves map[string][]MoveJSON  `json:"availableMoves"`
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

func CreateAvailableMovesJSON(moveMap map[string]*[]location.Move) AvailableMovesJSON {
	var jsonMoveMap = make(map[string][]MoveJSON)

	for coord, movesForPiece := range moveMap {
		var movesJSON []MoveJSON
		for _, move := range *movesForPiece {
			movesJSON = append(movesJSON, MoveJSON{
				Start: [2]uint8{
					move.GetStart().GetRow(),
					move.GetStart().GetCol(),
				},
				End: [2] uint8{
					move.GetEnd().GetRow(),
					move.GetEnd().GetCol(),
				},
			})
		}

		jsonMoveMap[coord] = movesJSON
	}

	return AvailableMovesJSON{
		AvailableMoves: jsonMoveMap,
	}
}
