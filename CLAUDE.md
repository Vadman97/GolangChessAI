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

## Debugging AI Quality Issues

### Where to look first

**`/tmp/chess.log`** — tournament stdout, captured when Vadim runs `./main tournament 2>&1 | tee /tmp/chess.log`. Contains:
- Matchup headers: `[N/M] AlgoA vs AlgoB (K games, Xs/move)`
- Per-game results: `Game N (AlgoA=White, AlgoB=Black): AlgoA wins / draw`
- Subtotals per matchup: `Subtotal: AlgoA W-D-L AlgoB`
- Final leaderboard with ELO rankings and Score%

**`./performance.log` / `./performance_N.log`** (project root, numbered per game) — per-turn search metrics written by `pkg/chessai/player/ai/performance_logger.go`:
- `Turn N` — move number
- `Considered N` — positions evaluated that turn
- `Pruned X% (N)` — pruning rate breakdown: `PrunedAB`, `PrunedTrans`, `ABImprovedTrans`
- Cache hit ratios for: Board evaluation map, Transposition table, Move cache, Attack move cache
- `AI (AlgoName - Color) best move leads to score N` — static eval score of chosen move

**`./moveDebug.log`** (project root) — move-by-move game trace with piece type, color, and from/to coordinates. Written by `ai_player.go:printMoveDebug`. Shows resign events.

**`pkg/chessai/competition/performance*.log`** — same format as root performance logs but produced by `competition` mode (not tournament).

**`performance*.xlsx`** — Excel versions of the performance logs with charts (pruning breakdown, cache hit rates over time). Useful for spotting trends visually.

### What signals indicate a problem

| Symptom | Likely cause | Where to look |
|---|---|---|
| Algorithm loses more than expected vs weaker algo | Eval bug or search regression | `performance.log` eval scores, `evaluation.go` |
| Pruning % near 0% for AB/MTDf/ABDADA | Move ordering broken or transposition table disabled | `performance.log` Pruned lines, `conf.json` |
| Transposition table hit ratio 0% or NaN | TT disabled or Zobrist hash collision/bug | `performance.log` TT metrics, `conf.json` |
| Eval score swings wildly turn to turn | Horizon effect or eval asymmetry | `moveDebug.log` + `performance.log` score column |
| Algorithm resigns with pieces on board | No legal moves generated (move-gen bug) | `moveDebug.log` resign line, `board/` move generators |
| ABDADA/Jamboree underperforms serial algos | Thread contention or cache thrashing | `performance.log` lock usage counts |

### Useful grep patterns

```bash
# ELO standings from tournament log
grep -E "Rank|Elo|Score%" /tmp/chess.log | tail -30

# Win/loss summary per matchup
grep -E "Subtotal|wins|draw" /tmp/chess.log

# Eval scores over time for a game (spot large swings)
grep "best move leads to score" performance.log

# Pruning efficiency per turn
grep -E "Turn|Pruned" performance.log | paste - -

# Cache health
grep -A3 "Transposition table metrics" performance.log | grep "Hit ratio"

# Resign / no-move events
grep -i "resign\|no best move" moveDebug.log
```

### Key source files for eval/search quality

- `pkg/chessai/player/ai/evaluation.go` — piece values, mobility, pawn structure scoring
- `pkg/chessai/player/ai/abdada.go` — parallel search; thread count logged to stderr at startup
- `pkg/chessai/player/ai/ai_player.go` — think-time cutoff, opening book, cache eviction warnings
- `pkg/chessai/competition/elo.go` — Elo update formula (K-factor etc.)
- `conf.json` — toggle transposition table, cache sizes, Elo start value
- `game_conf.json` — active algorithm, depth limit, time limit

## Current Engine Debugging Notes

### Lichess and post-game analysis workflow

- Lichess bot games are logged to `/tmp/chess.lichess.log`.
- Use `go run ./cmd log-replay /tmp/chess.lichess.log 12 ./stockfish` for a quick Stockfish blunder report.
- Prefer the fixed forward replay path in `pkg/chessai/analysis/log_replay.go`; do not reconstruct positions by unapplying only the AI move from post-move boards. That older approach mis-restores captures when the opponent just moved the captured piece, producing false blunders such as normal `Nxf6+` captures.
- Validate suspicious replay findings directly with Stockfish on the reported FEN before changing search logic. Replay labels are useful triage, not proof.
- The local `stockfish` binary may be untracked in the repo; do not delete or commit it unless asked.

### ABDADA / TT best practices

- Treat ABDADA bugs as likely state-contamination bugs first: shared TT, abort flag, `NumProcessors`, ponder, killer/history/countermove tables, and iterative-deepening partial results.
- Never write TT entries from aborted or killed ABDADA searches. Timed-out searches can return partial bounds from inside the tree; storing them under the current generation can poison later searches.
- Never store sentinel values (`PosInf`, `NegInf`, `OnEvaluation`, `-OnEvaluation`) in TT entries.
- Classify TT bounds using the original search window alpha, not the alpha after move search has raised it.
- Keep ABDADA `NumProcessors` bookkeeping saturating; underflow from `0` to `65535` makes nodes look permanently under evaluation.
- Pondering must search the actual side to move after our move, not always `AIPlayer.PlayerColor`. It should use an isolated search player/algorithm instance while sharing only the expensive caches/TT. Do not let ponder race the live player's abort flag or ABDADA fields.
- On a ponder miss, stale TT scores must be invalidated/demoted before the real search. Best-move hints may still be useful for ordering, but stale scores must not cut off.

### Known quality weaknesses to focus on next

- The engine still misses endgame tactics under blitz timing, especially quiet defensive moves and king/rook/minor-piece endgames. Recent examples include late-game rook/knight moves where ABDADA chose a move live that a clean local search did not reproduce, pointing to timed-search state contamination or pruning risk.
- Move quality improved after the ponder-color fix, but remaining blunders often come from search instability around time aborts. When a bad live move is not reproduced by a clean local search from the same FEN, inspect TT writes, abort handling, and per-search state before tuning evaluation.
- ABDADA's aggressive pruning stack (null move, razoring, futility pruning, LMR, SEE pruning in qsearch) should be treated as suspect in tactical/endgame misses. Disable one heuristic at a time on the exact FEN to isolate regressions.
- Opening play is weak without a book: early moves like `Nc3`/side pawn moves are often playable but suboptimal. Re-enable or improve openings separately from search correctness.
- Evaluation still undervalues some tactical liabilities: loose pieces, trapped rooks/bishops, king safety in simplified positions, passed-pawn races, and quiet mate nets.
- Time management in 3+0 reaches short per-move budgets late. Search correctness under abort is more important than deeper-but-contaminated searches.

### Regression testing habits

- Add focused tests for every TT/search invariant fixed. Existing examples live in `pkg/chessai/player/ai/abdada_tt_test.go`.
- For a suspicious Lichess move, capture the FEN from `log-replay`, run Stockfish on it, then run ABDADA locally with `NumThreads: 1`, TT on/off, and controlled depth/time. A move that only appears live usually indicates shared-state or abort contamination.
- Run targeted tests first:
  - `go test -timeout 60s ./pkg/chessai/player/ai ./pkg/chessai/server ./pkg/chessai/analysis`
  - Add `./pkg/chessai/board ./pkg/chessai/game ./pkg/chessai/util ./pkg/chessai/transposition_table` when move legality, game state, or board replay changes.
- `go test ./...` can take a long time in game/search packages; prefer explicit package lists with timeouts during debugging.
