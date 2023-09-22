package main

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/api/api_handlers"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/competition"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/server"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"time"
)

func main() {
	if len(os.Args) > 1 {
		if os.Args[1] == "competition" {
			comp := competition.NewCompetition()
			comp.RunAICompetition()
			return
		} else if os.Args[1] == "analysis" {
			comp := competition.NewCompetition()
			comp.RunAIAnalysis()
			return
		} else if os.Args[1] == "lichess" {
			c := server.ConnectLichess()
			c.Run()
			return
		}
	}
	rand.Seed(time.Now().UnixNano())

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
		Addr:         fmt.Sprintf("0.0.0.0:%s", port),
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
