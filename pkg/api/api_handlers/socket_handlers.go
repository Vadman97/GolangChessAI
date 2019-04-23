package api_handlers

import (
	"encoding/json"
	"github.com/Vadman97/ChessAI3/pkg/api"
	"github.com/Vadman97/ChessAI3/pkg/chessai/color"
	"github.com/Vadman97/ChessAI3/pkg/chessai/game"
	"github.com/Vadman97/ChessAI3/pkg/chessai/location"
	"github.com/Vadman97/ChessAI3/pkg/chessai/piece"
	"github.com/Vadman97/ChessAI3/pkg/chessai/player"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)


var client *websocket.Conn
var clientMutex = &sync.Mutex{}
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	defer ws.Close()

	// If game hasn't started yet, reject the connection
	if getGame() == nil {
		log.Print("Client attempted to connect, no game has begun...")

		msg := api.ChessMessage{
			Type: api.GameNotAvailable,
			Data: "",
		}
		err := ws.WriteJSON(msg)
		if err != nil {
			log.Printf("Client Send Error - %v", err)
		}
		return
	}

	// Allow 1 Client to connect at a time
	// TODO(Alex) Might a client queue, but right now the client will have to reload
	clientMutex.Lock()
	if client != nil {
		clientMutex.Unlock()

		log.Print("Client attempted to connect, but a game is currently in progress...")
		msg := api.ChessMessage{
			Type: api.GameFull,
			Data: "",
		}
		err := ws.WriteJSON(msg)
		if err != nil {
			log.Printf("Client Send Error - %v", err)
		}
		return
	}
	clientMutex.Unlock()

	// Initialize Client
	clientMutex.Lock()
	client = ws
	clientMutex.Unlock()
	log.Print("Client connected")

	// Start Game
	if client != nil && getGame() != nil {
		go getGame().Loop(client)
	}

	// Wait for Messages (Loop Forever)
	for {
		var msg api.ChessMessage
		err := client.ReadJSON(&msg)

		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived,
				websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Print("Client disconnected unexpectedly")

				clientMutex.Lock()
				client = nil
				clientMutex.Unlock()

				getGame().GameStatus = game.Aborted
				setGame(nil)
			} else {
				log.Printf("WebSocket Error - %v", err)
			}
			return
		}

		getGame().SocketBroadcast <- msg
	}
}

func HandleMessages(g *game.Game) {
	for {
		msg := <-g.SocketBroadcast
		switch msg.Type {
		// Client -> Server
		case api.PlayerMove:
			var moveJSON api.MoveJSON

			err := json.Unmarshal([]byte(msg.Data), &moveJSON)
			if err != nil {
				log.Printf("Invalid Player Move - %v", err)
				continue
			}
			HandlePlayerMove(moveJSON)

		// Server -> Client
		case api.GameState:
			fallthrough
		case api.GameStatus:
			fallthrough
		case api.AvailablePlayerMoves:
			fallthrough
		case api.AIMove:
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("Unable to send to client - %v", err)
				continue
			}
		}

	}
}


func HandlePlayerMove(moveJSON api.MoveJSON) {
	for c := color.White; c < color.NumColors; c++ {
		humanPlayer, isHuman := getGame().Players[c].(*player.HumanPlayer)
		if isHuman {
			move := &location.Move{
				Start: location.NewLocation(moveJSON.Start[0], moveJSON.Start[1]),
				End: location.NewLocation(moveJSON.End[0], moveJSON.End[1]),
			}

			// Add Pawn Promotion Information if it exists
			if moveJSON.Piece != (api.PieceJSON{}) {
				pieceType := rune(moveJSON.Piece.PieceType[0])
				move.End = move.End.CreatePawnPromotion(piece.NameToType[pieceType])
			}
			humanPlayer.Move <- move
			return
		}
	}
}
