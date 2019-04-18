package api

import (
	"encoding/json"
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

var g *game.Game

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

func setGame(gameToSet *game.Game) {
	g = gameToSet
}

func GetGameStateHandler(w http.ResponseWriter, r *http.Request) {
	gameJSON := &GameStateJSON{
		CurrentBoard:     [board.Height][board.Width]*PieceJSON{},
		CurrentTurnColor: color.Names[g.CurrentTurnColor],
		MovesPlayed:      g.MovesPlayed,
		GameStatus:       game.StatusStrings[g.GameStatus],
		MoveLimit:        g.MoveLimit,
		TimeLimit:        g.TimeLimit,
	}

	// Set Board JSON
	for r := uint8(0); r < board.Height; r++ {
		for c := uint8(0); c < board.Width; c++ {
			pieceFromLoc := g.CurrentBoard.GetPiece(location.NewLocation(r, c))
			if pieceFromLoc == nil {
				continue
			}

			gameJSON.CurrentBoard[r][c] = &PieceJSON{
				Color: color.Names[pieceFromLoc.GetColor()],
				PieceType: piece.TypeToName[pieceFromLoc.GetPieceType()],
			}
		}
 	}

	// Set PreviousMove
	if g.PreviousMove != nil {
		gameJSON.PreviousMove = &MoveJSON{
			Start: [2]uint8{
				g.PreviousMove.Move.GetStart().GetRow(),
				g.PreviousMove.Move.GetStart().GetCol(),
			},
			End: [2] uint8{
				g.PreviousMove.Move.GetEnd().GetRow(),
				g.PreviousMove.Move.GetEnd().GetCol(),
			},
			IsCapture: g.PreviousMove.IsCapture,
			Piece: PieceJSON{
				PieceType: piece.TypeToName[(*g.PreviousMove.Piece).GetPieceType()],
				Color: color.Names[(*g.PreviousMove.Piece).GetColor()],
			},
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(gameJSON); err != nil {
		panic(err)
	}
}

func PostGameCommandHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	command := vars["command"]

	if command == Start {
		// setGame(game.NewGame())
	} else if command == Restart {
		// TODO (Alex) Implement
	}
}
