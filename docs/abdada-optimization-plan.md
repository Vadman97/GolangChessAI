# ABDADA Optimization Plan

## Goal

Improve the ABDADA engine's playing strength by optimizing the parallel search itself before making broad evaluation changes.

The primary target is not "win one more Lichess game"; it is to make ABDADA more stable, reproducible, and tactically reliable under fixed search limits.

## Recommended Workflow

Use local, repeatable tests as the main improvement loop. Use Lichess games only as an external smoke test after local ABDADA benchmarks improve.

The loop:

1. Build a fixed set of benchmark FENs.
2. Run ABDADA across thread counts and search limits.
3. Compare against Stockfish and against ABDADA's own 1-thread behavior.
4. Detect ABDADA-specific regressions.
5. Patch one search behavior at a time.
6. Add regression tests for any serious failure.
7. Rerun the same benchmark before playing another live game.

## Benchmark Command

Add a command shaped like:

```bash
./main abdada-bench \
  --fens testdata/abdada_fens.txt \
  --threads 1,2,4,8 \
  --depth 6 \
  --runs 5 \
  --stockfish ./stockfish \
  --sf-depth 12
```

The command should support fixed-depth and fixed-time modes:

```bash
./main abdada-bench --fens testdata/abdada_fens.txt --threads 1,2,4,8 --depth 6 --runs 5
./main abdada-bench --fens testdata/abdada_fens.txt --threads 1,2,4,8 --think-ms 500 --runs 5
```

## FEN Set

Create `testdata/abdada_fens.txt` with positions from:

- recent Lichess blunders
- ABDADA self-play blunders
- tactical positions with hanging pieces, pins, discovered attacks, and mate threats
- quiet middlegames with high branching factor
- endgames with passed pawns
- positions where ABDADA finds mate or must defend mate
- positions where parallel ABDADA disagrees with 1-thread ABDADA

Each line should allow optional metadata:

```text
fen | tag | expected-best-move-list | known-bad-move-list | notes
```

Example:

```text
rnb1kbnr/ppp1pppp/8/3R1qN1/8/2N5/PPPPBP1P/R1BQK3 b Qkq - 9 10 | lichess-cNRjooIj | f5f4 | f5f6 | 10...Qf6 blunder
```

## Metrics To Report

For each FEN, thread count, and run:

- best move
- score
- depth reached
- nodes or moves considered
- elapsed time
- pruning totals
- TT reads, writes, and hit rate
- ABDADA `OnEvaluation` count if exposed
- number of root workers started and completed
- whether timeout occurred
- Stockfish best move
- Stockfish centipawn loss for ABDADA's move

Aggregate:

- best move stability, e.g. `4/5 runs chose e8g8`
- score variance
- average time to depth
- average nodes per second
- blunder count by thread count
- cases where more threads produce a worse move than 1 thread

Example output:

```text
FEN: rnb1kbnr/ppp1pppp/8/3R1qN1/8/2N5/PPPPBP1P/R1BQK3 b Qkq - 9 10
Stockfish: f5f4

ABDADA threads=1: best f5f4 score +42 stable 5/5 avg 182ms loss 0cp
ABDADA threads=8: best f5f6 score +280 stable 2/5 avg 94ms loss 116cp
Flag: parallel regression
```

## ABDADA Failure Classes

Prioritize failures that are specific to parallel search:

- Same FEN gives different best moves across repeated runs.
- More threads produce worse moves than 1 thread.
- Timeout returns a worse partial-depth move instead of the last completed-depth move.
- TT cutoffs use stale, shallow, aborted, or wrong-bound entries.
- `OnEvaluation` exclusive-probe behavior suppresses useful search.
- Root workers duplicate too much work.
- Root worker cancellation kills better searches too early.
- Mate scores become unstable across TT normalization/denormalization.
- Search metrics mutate unsafely or hide real search behavior.

## Optimization Priorities

### 1. Correctness and Stability

Before performance tuning:

- keep ABDADA clean under `go test -race` for targeted tests
- ensure aborted searches do not write TT entries
- ensure timeout selection prefers last completed depth unless no legal move was completed
- verify TT bounds are classified with the original alpha window
- verify all TT best moves are legal before use
- expose deterministic mode for reproducible debugging

### 2. Root Search Coordination

Investigate whether the current root strategy wastes parallelism.

Questions:

- Are all workers searching duplicate trees?
- Does `AbortAfterFirst` throw away better work from slower workers?
- Should workers split root moves instead of racing the same root?
- Can root move scores be merged safely without ABDADA-specific TT contamination?

Likely experiments:

- disable `AbortAfterFirst` and compare strength/time
- wait for all workers at shallow depths, first-completed only at timeout
- assign disjoint root move batches per worker
- keep a shared root result table with atomic best-score updates

### 3. TT Behavior

Measure and tune:

- hit rate by depth
- cutoff rate by entry type
- stale generation use
- entries written after timeout or kill
- entries with zero best move
- rate of legal vs invalid TT best moves

Possible changes:

- store generation per search, not only ponder miss
- avoid overwriting deeper best moves with shallower upper-bound entries
- separate move-ordering TT from cutoff TT
- add counters for exact, lower, upper, stale, and rejected entries

### 4. Move Ordering and Reductions

Only tune once correctness is stable:

- TT move ordering
- capture ordering and SEE cost
- killer/history/countermove value under lock contention
- LMR thresholds
- null move pruning guards
- aspiration window size and fallback behavior

## Regression Tests

Every serious ABDADA failure should become a test.

Test types:

- `TestABDADAStableBestMoveForFEN`
- `TestABDADAThreadsDoNotRegressVsSingleThread`
- `TestABDADATimeoutKeepsCompletedDepthMove`
- `TestABDADATTSkipsAbortedWrites`
- `TestABDADATTMateScoreRoundTrip`

For tactical failures, prefer allowed move sets over a single exact move when several moves are equivalent:

```text
position: ...
bad move: d5d4
allowed: e8g8,c8g4
```

## Live Lichess Use

Use Lichess hard-bot games after local improvement, not as the main loop.

Recommended live loop:

1. Run `abdada-bench`; require no new local regressions.
2. Play one or more Lichess games.
3. Run `log-replay` on `/tmp/chess.lichess.log`.
4. Add new blunder FENs to `testdata/abdada_fens.txt`.
5. Return to the local benchmark loop.

## Success Criteria

ABDADA optimization is improving when:

- repeated runs on the same FEN become more stable
- 2/4/8-thread results are no worse than 1-thread results on benchmark FENs
- Stockfish centipawn loss decreases on the fixed FEN set
- timeout behavior stops selecting unstable partial-depth moves
- ABDADA reaches equal or greater depth in the same time without higher blunder rate
- Lichess blunders become less frequent after local benchmark gains
