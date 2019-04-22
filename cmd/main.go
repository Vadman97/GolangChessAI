package main

import (
	"fmt"
	"github.com/Vadman97/ChessAI3/pkg/api/api_handlers"
	"github.com/Vadman97/ChessAI3/pkg/chessai/competition"
	"github.com/gorilla/mux"
	"time"
	"log"
	"net/http"
	"os"
	"path"
)

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
	r.HandleFunc("/ws", api_handlers.HandleConnections)

	// API Routes
	gameApiRouter := r.PathPrefix("/api/game").Subrouter()
	gameApiRouter.
		Path("").
		Methods("GET").
		HandlerFunc(api_handlers.GetGameStateHandler)

	gameApiRouter.
		Path("").
		Methods("POST").
		Queries("command", "{command}").
		HandlerFunc(api_handlers.PostGameCommandHandler)


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
