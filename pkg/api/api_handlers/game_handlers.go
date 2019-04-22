package api_handlers

import (
	"encoding/json"
	"github.com/Vadman97/ChessAI3/pkg/api"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game_config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/bitly/go-simplejson"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

var g *game.Game

func getGame() *game.Game {
	return g
}

func setGame(gameToSet *game.Game) {
	g = gameToSet
}

func GetGameStateHandler(w http.ResponseWriter, r *http.Request) {
	if g == nil {
		errorResponse := simplejson.New()
		errorResponse.Set("error", "No Game is Available")

		payload, err := errorResponse.MarshalJSON()
		if err != nil {
			log.Println(err)
		}

		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		w.Write(payload)
		return
	}

	gameJSON := g.GetJSON()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(gameJSON); err != nil {
		panic(err)
	}
}

func PostGameCommandHandler(w http.ResponseWriter, r *http.Request) {
	command := strings.ToLower(r.FormValue("command"))

	if command == api.Start {
		if g != nil {
			errorResponse := simplejson.New()
			errorResponse.Set("error", "Game is currently in progress...")

			payload, err := errorResponse.MarshalJSON()
			if err != nil {
				log.Println(err)
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			w.Write(payload)
			return
		}

		// Coin Flip to determine colors
		algorithmName := game_config.Get().Algorithm
		playerIsWhite := rand.Intn(2)

		humanColor := color.White
		if playerIsWhite == 0 {
			humanColor = color.Black
		}

		aiColor := color.Black
		if playerIsWhite == 0 {
			aiColor = color.White
		}

		humanPlayer := player.NewHumanPlayer(humanColor)
		aiPlayer := ai.NewAIPlayer(aiColor, ai.NameToAlgorithm[algorithmName])

		// Create game and start game loop
		if playerIsWhite == 0 {
			g = game.NewGame(aiPlayer, humanPlayer)
		} else {
			g = game.NewGame(humanPlayer, aiPlayer)
		}

		g.MoveLimit = game_config.Get().MovesToPlay
		g.TimeLimit = game_config.Get().SecondsToPlay * time.Second

		// Initialize WebSocket Handler
		go HandleMessages(g)

		// NOTE: The Server WebSocket Listener waits to receive a client before a game is begun
	} else if command == api.Restart {
		// TODO (Alex) Implement
	}

	// Send Success Status
	successResponse := simplejson.New()
	successResponse.Set("success", true)

	payload, err := successResponse.MarshalJSON()
	if err != nil {
		log.Println(err)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}
