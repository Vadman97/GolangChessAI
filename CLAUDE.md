# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build
go build -o main ./cmd/main.go

# Run web server (serves frontend on :8080)
./main

# Run AI competition mode (10-game series, MTDf vs ABDADA)
./main competition

# Run analysis mode
./main analysis

# Run all tests with coverage
go test -coverprofile=coverage.txt -covermode=atomic ./...

# Run tests in a specific package
go test ./pkg/chessai/board/...
go test ./pkg/chessai/player/ai/...

# Run benchmarks
go test -bench=. ./bench/...

# Frontend (in web/)
npm install
npm start
```

## Architecture

### Module path
`github.com/Vadman97/ChessAI3` (despite the repo being named GolangChessAI).

### Package layout

- **`cmd/main.go`** — entry point; starts the HTTP server (gorilla/mux) or dispatches to competition/analysis mode. Serves the React frontend as static files from `./web/`.

- **`pkg/api/api_handlers/`** — HTTP + WebSocket handlers. `socket_handlers.go` drives the real-time game loop; `game_handlers.go` handles REST endpoints for game state and commands.

- **`pkg/chessai/board/`** — core board representation. The `Board` struct stores pieces in a compact 4-bits-per-piece format (3 bits type + 1 bit color). `BitBoard` (uint64) is used for attackable-square computation. `game_board.go` owns move execution, Zobrist hashing, en passant, castling flags, and caches (`MoveCache`, `AttackableCache`).

- **`pkg/chessai/player/ai/`** — all search algorithms implement the `Algorithm` interface (`GetBestMove`). Available algorithms:
  - `MiniMax` — baseline
  - `AlphaBetaWithMemory` — α/β with transposition table
  - `MTDf` — Memory-Enhanced Test Driver (serial)
  - `NegaScout` — principal variation search
  - `ABDADA` — parallel α/β (uses goroutines + shared transposition table)
  - `Jamboree` — hybrid parallel search
  - `Random` — random move picker

  `AIPlayer` wraps the chosen algorithm, manages think-time via a goroutine timer, handles opening book moves, and owns per-player `evaluationMap` and `transpositionTable` (`ConcurrentBoardMap`).

- **`pkg/chessai/player/ai/evaluation.go`** — static board evaluation: piece values, mobility (num moves/attacks), pawn structure (doubled/isolated detection via column/row maps), piece advancement bonuses.

- **`pkg/chessai/util/`** — `ConcurrentBoardMap` (thread-safe hash map keyed by board hash, used for transposition table and evaluation cache), memory helpers, thread utilities.

- **`pkg/chessai/competition/`** — `Competition` runs head-to-head series between two AI players, tracks Elo ratings, and optionally logs performance to Excel via `excelize`.

- **`pkg/chessai/transposition_table/`** — standalone transposition table type (wraps `ConcurrentBoardMap`).

### Configuration

- **`conf.json`** — engine config loaded by `pkg/chessai/config/config.go`. Controls caches, logging, transposition table toggle, Elo starting value, competition game count, random seed.
- **`game_conf.json`** — active game config: which algorithm to use, move/time limits, search depth.

### Frontend

React app in `web/` (webpack + babel). Communicates with the backend over WebSocket (`/ws`) for real-time move updates and REST (`/api/game`) for game state and commands.
