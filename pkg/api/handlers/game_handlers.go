package handlers

import (
	"encoding/json"
	"github.com/Vadman97/ChessAI3/pkg/api"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game_config"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player/ai"
	"github.com/gorilla/mux"
	"math/rand"
	"net/http"
	"strings"
)

var g *game.Game

func getGame() *game.Game {
	return g
}

func setGame(gameToSet *game.Game) {
	g = gameToSet
}

func GetGameStateHandler(w http.ResponseWriter, r *http.Request) {
	gameJSON := g.GetJSON()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(gameJSON); err != nil {
		panic(err)
	}
}

func PostGameCommandHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	command := strings.ToLower(vars["command"])

	if command == api.Start {
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
		g.TimeLimit = game_config.Get().TimeToPlay

		go g.Loop(client)

		// Initialize WebSocket Handler
		go HandleMessages(g)

	} else if command == api.Restart {
		// TODO (Alex) Implement
	}
}
