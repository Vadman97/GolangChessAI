package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"runtime/debug"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"

	"github.com/Vadman97/GolangChessAI/pkg/api/api_handlers"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/analysis"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/competition"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/server"
	"github.com/gorilla/mux"
)

func main() {
	// Search is heavily allocation-bound (board/piece decoding, move slices) and
	// runs in short, high-throughput bursts under a wall-clock budget. The default
	// GOGC=100 triggers frequent concurrent GC cycles that contend with search
	// goroutines and disproportionately hurt multi-threaded ABDADA scaling (measured
	// only ~2.5x speedup on 8 threads vs 1, with mallocgc/scanObject ~30% of CPU
	// profile time). Raising GOGC trades memory for fewer GC cycles and measured
	// ~10-15% more search nodes/sec at threads=8. SetMemoryLimit caps worst-case
	// heap growth for long-running server/lichess modes.
	debug.SetGCPercent(400)
	debug.SetMemoryLimit(2 << 30) // 2 GiB soft cap
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
			// Restart the server if it exits unexpectedly (rate limits, crashes, etc.)
			for {
				server.ConnectLichess().Run()
				log.Println("lichess server exited — restarting in 30s")
				time.Sleep(30 * time.Second)
			}
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
			// Usage: ./main log-replay [flags] [logPath] [sfDepth] [stockfishPath]
			fs := flag.NewFlagSet("log-replay", flag.ExitOnError)
			appendFENs := fs.String("append-fens", "", "optional ABDADA benchmark FEN file to append mistakes/blunders to")
			appendMinLoss := fs.Int("append-min-loss", 50, "minimum centipawn loss to append with --append-fens")
			if err := fs.Parse(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			logPath := "/tmp/chess.lichess.log"
			sfDepth := 15
			sfPath := "./stockfish"
			args := fs.Args()
			if len(args) > 0 {
				logPath = args[0]
			}
			if len(args) > 1 {
				if d, err := strconv.Atoi(args[1]); err == nil && d > 0 {
					sfDepth = d
				}
			}
			if len(args) > 2 {
				sfPath = args[2]
			}
			analysis.RunLogReplayWithConfig(analysis.LogReplayConfig{
				LogPath:        logPath,
				StockfishPath:  sfPath,
				StockfishDepth: sfDepth,
				AppendFENsPath: *appendFENs,
				AppendMinLoss:  *appendMinLoss,
			})
			return
		} else if os.Args[1] == "san-fens" {
			fs := flag.NewFlagSet("san-fens", flag.ExitOnError)
			pliesArg := fs.String("plies", "", "optional comma-separated ply numbers to print")
			if err := fs.Parse(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			moveText := strings.Join(fs.Args(), " ")
			if strings.TrimSpace(moveText) == "" {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					log.Fatal(err)
				}
				moveText = string(data)
			}
			replayed, err := analysis.ReplaySANMoves(moveText)
			if err != nil {
				log.Fatal(err)
			}
			plies, err := parseOptionalPlySet(*pliesArg)
			if err != nil {
				log.Fatal(err)
			}
			for _, r := range replayed {
				if len(plies) > 0 && !plies[r.Ply] {
					continue
				}
				fmt.Printf("%d | %s | %s | %s\n", r.Ply, r.SAN, r.UCI, r.FENBefore)
			}
			return
		} else if os.Args[1] == "fen-apply" {
			fs := flag.NewFlagSet("fen-apply", flag.ExitOnError)
			if err := fs.Parse(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			if fs.NArg() != 2 {
				log.Fatal("usage: fen-apply '<fen>' <uci-move>")
			}
			parsed, err := analysis.ParseFEN(fs.Arg(0))
			if err != nil {
				log.Fatal(err)
			}
			move, err := analysis.MatchUCIMove(parsed.Board, parsed.Active, parsed.Previous, fs.Arg(1))
			if err != nil {
				log.Fatal(err)
			}
			last := board.MakeMove(&move, parsed.Board)
			next := parsed.Active ^ 1
			fullMove := parsed.FullMove
			if parsed.Active == color.Black {
				fullMove++
			}
			fmt.Println(analysis.BoardToFEN(parsed.Board, next, last, fullMove))
			return
		} else if os.Args[1] == "abdada-bench" {
			fs := flag.NewFlagSet("abdada-bench", flag.ExitOnError)
			fenPath := fs.String("fens", "testdata/abdada_fens.txt", "path to ABDADA benchmark FEN file")
			threadList := fs.String("threads", "1,2,4,8", "comma-separated ABDADA thread counts")
			depth := fs.Int("depth", 0, "fixed search depth; omit when using --think-ms")
			thinkMS := fs.Int("think-ms", 0, "fixed think time per move in milliseconds")
			runs := fs.Int("runs", 1, "runs per FEN and thread count")
			stockfishPath := fs.String("stockfish", "", "optional Stockfish binary path")
			sfDepth := fs.Int("sf-depth", 0, "Stockfish depth for best-move and loss comparison")
			showRoot := fs.Int("show-root", 0, "print top N ABDADA root move scores before thread benchmark")
			jsonPath := fs.String("json", "", "optional path to write a JSON benchmark report")
			if err := fs.Parse(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			threads, err := analysis.ParseThreadList(*threadList)
			if err != nil {
				log.Fatal(err)
			}
			if *depth > 0 && *thinkMS > 0 {
				log.Fatal("use either --depth or --think-ms, not both")
			}
			if err := analysis.RunABDADABench(analysis.ABDADABenchConfig{
				FENPath:        *fenPath,
				Threads:        threads,
				Depth:          *depth,
				ThinkTime:      time.Duration(*thinkMS) * time.Millisecond,
				Runs:           *runs,
				StockfishPath:  *stockfishPath,
				StockfishDepth: *sfDepth,
				ShowRoot:       *showRoot,
				JSONPath:       *jsonPath,
			}); err != nil {
				log.Fatal(err)
			}
			return
		} else if os.Args[1] == "abdada-matrix" {
			fs := flag.NewFlagSet("abdada-matrix", flag.ExitOnError)
			fenPath := fs.String("fens", "testdata/abdada_fens.txt", "path to ABDADA benchmark FEN file")
			fen := fs.String("fen", "", "single FEN to diagnose instead of --fens")
			depth := fs.Int("depth", 0, "fixed search depth; omit when using --think-ms")
			thinkMS := fs.Int("think-ms", 0, "fixed think time per move in milliseconds")
			runs := fs.Int("runs", 1, "runs per FEN and mode")
			stockfishPath := fs.String("stockfish", "", "optional Stockfish binary path")
			sfDepth := fs.Int("sf-depth", 0, "Stockfish depth for best-move and loss comparison")
			modes := fs.String("modes", "abdada1tt,abdada8tt,abdada1nott,abdada8nott,abdada1safe,abdada8safe,negascouttt", "comma-separated modes; includes abdada1tt/8tt, abdada1nott/8nott, abdada1safe/8safe, abdada1nolmr/nonull/nofutility/norazor, negascouttt/nott")
			forceMove := fs.String("force-move", "", "optional legal UCI root move to score instead of selecting the best move; requires --depth")
			if err := fs.Parse(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			if *depth > 0 && *thinkMS > 0 {
				log.Fatal("use either --depth or --think-ms, not both")
			}
			if err := analysis.RunABDADAMatrix(analysis.ABDADAMatrixConfig{
				FENPath:        *fenPath,
				FEN:            *fen,
				ForceMove:      *forceMove,
				Depth:          *depth,
				ThinkTime:      time.Duration(*thinkMS) * time.Millisecond,
				Runs:           *runs,
				StockfishPath:  *stockfishPath,
				StockfishDepth: *sfDepth,
				Modes:          *modes,
			}); err != nil {
				log.Fatal(err)
			}
			return
		} else if os.Args[1] == "abdada-bench-diff" {
			fs := flag.NewFlagSet("abdada-bench-diff", flag.ExitOnError)
			if err := fs.Parse(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			if fs.NArg() != 2 {
				log.Fatal("usage: abdada-bench-diff before.json after.json")
			}
			if err := analysis.RunABDADABenchDiff(fs.Arg(0), fs.Arg(1)); err != nil {
				log.Fatal(err)
			}
			return
		} else if os.Args[1] == "uci-replay" {
			fs := flag.NewFlagSet("uci-replay", flag.ExitOnError)
			verbose := fs.Bool("verbose", false, "print FEN after every ply")
			jsonOut := fs.Bool("json", false, "print JSON states instead of text")
			if err := fs.Parse(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			moveText := strings.Join(fs.Args(), " ")
			if strings.TrimSpace(moveText) == "" {
				data, err := io.ReadAll(os.Stdin)
				if err != nil {
					log.Fatal(err)
				}
				moveText = string(data)
			}
			if *jsonOut {
				states, err := analysis.ReplayUCIMoves(moveText)
				if err != nil {
					log.Fatal(err)
				}
				if err := analysis.PrintUCIReplayJSON(states); err != nil {
					log.Fatal(err)
				}
			} else if err := analysis.RunUCIReplay(moveText, *verbose); err != nil {
				log.Fatal(err)
			}
			return
		} else if os.Args[1] == "lichess-state-replay" {
			fs := flag.NewFlagSet("lichess-state-replay", flag.ExitOnError)
			logPath := fs.String("log", "/tmp/chess.lichess.log", "path to Lichess bot log")
			gameID := fs.String("game", "", "optional Lichess game ID; default uses latest game in log")
			verbose := fs.Bool("verbose", false, "print FEN after every ply")
			if err := fs.Parse(os.Args[2:]); err != nil {
				log.Fatal(err)
			}
			if err := analysis.RunLichessStateReplay(*logPath, *gameID, *verbose); err != nil {
				log.Fatal(err)
			}
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
		} else if os.Args[1] == "negascout-matchup" {
			games := 4
			thinkTime := 1 * time.Second
			if len(os.Args) > 2 {
				if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
					games = n
				}
			}
			if len(os.Args) > 3 {
				if ms, err := strconv.Atoi(os.Args[3]); err == nil && ms > 0 {
					thinkTime = time.Duration(ms) * time.Millisecond
				}
			}
			competition.RunMatchup("NegaScout", "ABDADA", &ai.NegaScout{}, &ai.ABDADA{}, games, thinkTime, nil)
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

func parseOptionalPlySet(s string) (map[int]bool, error) {
	out := map[int]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		ply, err := strconv.Atoi(part)
		if err != nil || ply <= 0 {
			return nil, fmt.Errorf("invalid ply %q", part)
		}
		out[ply] = true
	}
	return out, nil
}
