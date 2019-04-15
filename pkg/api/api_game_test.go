package api

import (
	"github.com/Vadman97/ChessAI3/pkg/chessai/board"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetGameState(t *testing.T) {
	board := board.Board{}
	board.ResetDefault()

	testGame := &game.Game{
		CurrentBoard: &board,
		CurrentTurnColor: color.White,
		Players: map[byte]*ai.AIPlayer{
			color.White: nil,
			color.Black: nil,
		},
		TotalMoveTime: map[byte]time.Duration{
			color.White: 0,
			color.Black: 10,
		},
		LastMoveTime: map[byte]time.Duration{
			color.White: 1,
			color.Black: 2,
		},
		MovesPlayed:       8,
		PreviousMove:      nil,
		GameStatus:        game.Active,
	}
	setGame(testGame)

	req, err := http.NewRequest("GET", "/api/game", nil)

	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetGameStateHandler)

	// Perform HTTP Request
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	log.Print(rr.Body.String())

	expectedBody := `{
	  "currentBoard": [
	    [
	      {
	        "type": "R",
	        "color": "Black"
	      },
	      {
	        "type": "N",
	        "color": "Black"
	      },
	      {
	        "type": "B",
	        "color": "Black"
	      },
	      {
	        "type": "Q",
	        "color": "Black"
	      },
	      {
	        "type": "K",
	        "color": "Black"
	      },
	      {
	        "type": "B",
	        "color": "Black"
	      },
	      {
	        "type": "N",
	        "color": "Black"
	      },
	      {
	        "type": "R",
	        "color": "Black"
	      }
	    ],
	    [
	      {
	        "type": "P",
	        "color": "Black"
	      },
	      {
	        "type": "P",
	        "color": "Black"
	      },
	      {
	        "type": "P",
	        "color": "Black"
	      },
	      {
	        "type": "P",
	        "color": "Black"
	      },
	      {
	        "type": "P",
	        "color": "Black"
	      },
	      {
	        "type": "P",
	        "color": "Black"
	      },
	      {
	        "type": "P",
	        "color": "Black"
	      },
	      {
	        "type": "P",
	        "color": "Black"
	      }
	    ],
	    [
	      null, null, null, null, null, null, null, null
	    ],
	    [
	      null, null, null, null, null, null, null, null
	    ],
	    [
	      null, null, null, null, null, null, null, null
	    ],
	    [
	      null, null, null, null, null, null, null, null
	    ],
	    [
	      {
	        "type": "P",
	        "color": "White"
	      },
	      {
	        "type": "P",
	        "color": "White"
	      },
	      {
	        "type": "P",
	        "color": "White"
	      },
	      {
	        "type": "P",
	        "color": "White"
	      },
	      {
	        "type": "P",
	        "color": "White"
	      },
	      {
	        "type": "P",
	        "color": "White"
	      },
	      {
	        "type": "P",
	        "color": "White"
	      },
	      {
	        "type": "P",
	        "color": "White"
	      }
	    ],
	    [
	      {
	        "type": "R",
	        "color": "White"
	      },
	      {
	        "type": "N",
	        "color": "White"
	      },
	      {
	        "type": "B",
	        "color": "White"
	      },
	      {
	        "type": "Q",
	        "color": "White"
	      },
	      {
	        "type": "K",
	        "color": "White"
	      },
	      {
	        "type": "B",
	        "color": "White"
	      },
	      {
	        "type": "N",
	        "color": "White"
	      },
	      {
	        "type": "R",
	        "color": "White"
	      }
	    ]
	  ],
	  "currentTurn": "White",
	  "movesPlayed": 8,
	  "previousMove": null,
	  "gameStatus": "Active",
	  "moveLimit": 0,
	  "timeLimit": 0
	}`

	assert.JSONEq(t, expectedBody, rr.Body.String())
}