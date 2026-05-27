package main

import (
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/api/api_handlers"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/competition"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/server"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"runtime/pprof"
	"strconv"
	"time"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "profile" {
		f, err := os.Create("cpu.pprof")
		if err != nil {
			log.Fatalf("could not create cpu.pprof: %v", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("could not start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
		// shift args so the rest of main sees the real subcommand
		os.Args = append(os.Args[:1], os.Args[2:]...)
		log.Println("CPU profiling enabled — writing cpu.pprof on exit")
	}

	if len(os.Args) > 1 {
		if os.Args[1] == "lichess" {
			server.ConnectLichess().Run()
			return
		} else if os.Args[1] == "lichess-challenge" {
			// Usage: ./main lichess-challenge <username> [limitSecs] [incrementSecs] [rated]
			if len(os.Args) < 3 {
				log.Fatal("usage: lichess-challenge <username> [limitSecs] [incrementSecs] [rated]")
			}
			cfg := &server.ChallengeConfig{
				Username:      os.Args[2],
				ClockLimitSec: 180,
				ClockIncSec:   0,
				Rated:         true,
			}
			if len(os.Args) > 3 {
				if n, err := strconv.Atoi(os.Args[3]); err == nil {
					cfg.ClockLimitSec = n
				}
			}
			if len(os.Args) > 4 {
				if n, err := strconv.Atoi(os.Args[4]); err == nil {
					cfg.ClockIncSec = n
				}
			}
			if len(os.Args) > 5 {
				cfg.Rated = os.Args[5] != "false"
			}
			server.ConnectLichessWithChallenge(cfg).Run()
			return
		} else if os.Args[1] == "stockfish-analysis" {
			// Usage: ./main stockfish-analysis [games] [thinkMs] [sfDepth] [stockfishPath]
			numGames := 2
			thinkMs := 1000
			sfDepth := 15
			sfPath := "./stockfish"
			if len(os.Args) > 2 {
				if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
					numGames = n
				}
			}
			if len(os.Args) > 3 {
				if ms, err := strconv.Atoi(os.Args[3]); err == nil && ms > 0 {
					thinkMs = ms
				}
			}
			if len(os.Args) > 4 {
				if d, err := strconv.Atoi(os.Args[4]); err == nil && d > 0 {
					sfDepth = d
				}
			}
			if len(os.Args) > 5 {
				sfPath = os.Args[5]
			}
			analysis.RunSelfPlayAnalysis(sfPath, numGames, time.Duration(thinkMs)*time.Millisecond, sfDepth)
			return
		} else if os.Args[1] == "log-replay" {
			// Usage: ./main log-replay [logPath] [sfDepth] [stockfishPath]
			logPath := "/tmp/chess.lichess.log"
			sfDepth := 15
			sfPath := "./stockfish"
			if len(os.Args) > 2 {
				logPath = os.Args[2]
			}
			if len(os.Args) > 3 {
				if d, err := strconv.Atoi(os.Args[3]); err == nil && d > 0 {
					sfDepth = d
				}
			}
			if len(os.Args) > 4 {
				sfPath = os.Args[4]
			}
			analysis.RunLogReplay(logPath, sfPath, sfDepth)
			return
		} else if os.Args[1] == "competition" {
			comp := competition.NewCompetition()
			comp.RunAICompetition()
			return
		} else if os.Args[1] == "analysis" {
			comp := competition.NewCompetition()
			comp.RunAIAnalysis()
			return
		} else if os.Args[1] == "abdada-tournament" {
			gamesPerMatchup := 2
			thinkTime := 3 * time.Second
			if len(os.Args) > 2 {
				if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
					gamesPerMatchup = n
				}
			}
			if len(os.Args) > 3 {
				if ms, err := strconv.Atoi(os.Args[3]); err == nil && ms > 0 {
					thinkTime = time.Duration(ms) * time.Millisecond
				}
			}

			hub := api_handlers.NewSpectatorHub()
			go hub.Run()

			tourneyRouter := mux.NewRouter()
			tourneyRouter.HandleFunc("/ws-spectate", hub.HandleSpectatorConnection)
			tourneyRouter.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))

			port := os.Getenv("PORT")
			if port == "" {
				port = "8080"
			}
			tourneyServer := &http.Server{
				Handler: tourneyRouter,
				Addr:    fmt.Sprintf("0.0.0.0:%s", port),
			}
			go func() {
				log.Printf("Spectator view: http://localhost:%s/?spectate=true", port)
				if err := tourneyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("spectator server error: %v", err)
				}
			}()

			competition.RunABDADATournament(gamesPerMatchup, thinkTime, nil, hub.BroadcastCh())
			return
		} else if os.Args[1] == "tournament" {
			gamesPerMatchup := 2
			thinkTime := 3 * time.Second
			if len(os.Args) > 2 {
				if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
					gamesPerMatchup = n
				}
			}
			if len(os.Args) > 3 {
				if ms, err := strconv.Atoi(os.Args[3]); err == nil && ms > 0 {
					thinkTime = time.Duration(ms) * time.Millisecond
				}
			}

			hub := api_handlers.NewSpectatorHub()
			go hub.Run()

			tourneyRouter := mux.NewRouter()
			tourneyRouter.HandleFunc("/ws-spectate", hub.HandleSpectatorConnection)
			tourneyRouter.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))

			port := os.Getenv("PORT")
			if port == "" {
				port = "8080"
			}
			tourneyServer := &http.Server{
				Handler: tourneyRouter,
				Addr:    fmt.Sprintf("0.0.0.0:%s", port),
			}
			go func() {
				log.Printf("Spectator view: http://localhost:%s/?spectate=true", port)
				if err := tourneyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("spectator server error: %v", err)
				}
			}()

			competition.RunTournament(gamesPerMatchup, thinkTime, hub.BroadcastCh())
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
