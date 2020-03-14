package api_handlers

import (
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/game"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player"
	"github.com/stretchr/testify/assert"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetGameState(t *testing.T) {
	testBoard := board.Board{}
	testBoard.ResetDefault()

	testGame := &game.Game{
		CurrentBoard:     &testBoard,
		CurrentTurnColor: color.White,
		Players: map[color.Color]player.Player{
			color.White: nil,
			color.Black: nil,
		},
		TotalMoveTime: map[color.Color]time.Duration{
			color.White: 0,
			color.Black: 10,
		},
		LastMoveTime: map[color.Color]time.Duration{
			color.White: 1,
			color.Black: 2,
		},
		MovesPlayed:  8,
		PreviousMove: nil,
		GameStatus:   game.Active,
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
	        "type": "K",
	        "color": "White"
	      },
	      {
	        "type": "Q",
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
	        "type": "K",
	        "color": "Black"
	      },
	      {
	        "type": "Q",
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
	    ]
	  ],
	  "currentTurn": "White",
	  "movesPlayed": 8,
	  "previousMove": null,
	  "gameStatus": "Active",
	  "moveLimit": 0,
	  "timeLimit": 0,
	  "humanColor": ""
	}`

	assert.JSONEq(t, expectedBody, rr.Body.String())
}
