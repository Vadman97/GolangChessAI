package api

import (
	"encoding/json"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"log"
	"time"
)

const (
	PlayerMove           = "playerMove"
	AIMove               = "aiMove"
	AvailablePlayerMoves = "availablePlayerMoves"
	GameState            = "gameState"
	GameStatus           = "gameStatus"
	GameFull             = "gameFull"
	GameNotAvailable     = "gameNotAvailable"
)

type ChessMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

type GameStateJSON struct {
	CurrentBoard     [board.Height][board.Width]*PieceJSON `json:"currentBoard"`
	CurrentTurnColor string                                `json:"currentTurn"`
	HumanColor       string                                `json:"humanColor"`
	MovesPlayed      uint                                  `json:"movesPlayed"`
	PreviousMove     *MoveJSON                             `json:"previousMove"`
	GameStatus       string                                `json:"gameStatus"`
	MoveLimit        int32                                 `json:"moveLimit"`
	TimeLimit        time.Duration                         `json:"timeLimit"`
}

type GameStatusJSON struct {
	CurrentTurnColor string `json:"currentTurn"`
	MovesPlayed      uint   `json:"movesPlayed"`
	GameStatus       string `json:"gameStatus"`
	KingInCheck      bool   `json:"kingInCheck"`
}

type PieceJSON struct {
	PieceType string `json:"type"`
	Color     string `json:"color"`
}

type MoveJSON struct {
	Start          [2]uint8  `json:"start"`
	End            [2]uint8  `json:"end"`
	IsCapture      bool      `json:"isCapture"`
	Piece          PieceJSON `json:"piece"`
	PromotionPiece PieceJSON `json:"promotionPiece"`
}

type AvailableMovesJSON struct {
	AvailableMoves map[string][]MoveJSON `json:"availableMoves"`
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

func CreateMoveJSON(m *board.LastMove) *MoveJSON {
	moveJSON := &MoveJSON{
		Start: [2]uint8{
			m.Move.GetStart().GetRow(),
			m.Move.GetStart().GetCol(),
		},
		End: [2]uint8{
			m.Move.GetEnd().GetRow(),
			m.Move.GetEnd().GetCol(),
		},
		IsCapture: m.IsCapture,
		Piece: PieceJSON{
			PieceType: piece.TypeToName[(*m.Piece).GetPieceType()],
			Color:     color.Names[(*m.Piece).GetColor()],
		},
	}

	if m.PromotionPiece != nil {
		moveJSON.PromotionPiece = PieceJSON{
			PieceType: piece.TypeToName[(*m.PromotionPiece).GetPieceType()],
			Color:     color.Names[(*m.PromotionPiece).GetColor()],
		}
	}

	return moveJSON
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
				End: [2]uint8{
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
