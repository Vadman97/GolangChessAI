package main

import (
	"encoding/json"
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/api"
	"github.com/Vadman97/ChessAI3/pkg/chessai/competition"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"path"
	"sync"
	"time"
)

var client *websocket.Conn
var clientMutex = &sync.Mutex{}
var broadcast = make(chan api.ChessMessage)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}


func main() {
	if len(os.Args) > 1 && os.Args[1] == "competition" {
		comp := competition.NewCompetition()
		comp.RunAICompetition()
		return
	}

	// Setup HTTP Routes
	r := mux.NewRouter()

	r.HandleFunc("/", HomeHandler).Methods("GET")

	// WebSocket Route
	r.HandleFunc("/ws", handleConnections)

	// API Routes
	gameApiRouter := r.PathPrefix("/api/game").Subrouter()
	gameApiRouter.
		Path("/").
		Methods("GET").
		HandlerFunc(api.GetGameStateHandler)

	gameApiRouter.
		Path("/").
		Methods("POST").
		Queries("command", "{command}").
		HandlerFunc(api.PostGameCommandHandler)

	// Start WebSocket Handlers
	go handleMessages()

	// Set Static Files (MUST be below routes otherwise it'll conflict)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))

	// Start HTTP Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost:%s in the browser", port)

	server := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf(":%s", port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	p := path.Dir("./web/index.html")
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, p)
}


func handleConnections(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		return
	}

	defer ws.Close()

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

	// Wait for Messages (Loop Forever)
	for {
		var msg api.ChessMessage
		err := client.ReadJSON(&msg)

		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Print("Client disconnected unexpectedly")

				clientMutex.Lock()
				client = nil
				clientMutex.Unlock()
			} else {
				log.Printf("WebSocket Error - %v", err)
			}
			return
		}

		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		switch msg.Type {
		case api.PlayerMove:
			var moveJSON api.MoveJSON
			err := json.Unmarshal([]byte(msg.Data), &moveJSON)
			if err != nil {
				log.Printf("Invalid Player Move - %v", err)
				continue
			}
			api.HandlePlayerMove(moveJSON, client)
		}
	}
}
