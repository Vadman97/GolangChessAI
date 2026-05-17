# Running GolangChessAI as a Lichess Bot

This guide walks through registering your account as a bot, generating an API token, and running the engine against live opponents on [lichess.org](https://lichess.org).

## Prerequisites

- A **separate** lichess account to use as the bot (a bot account cannot play human games; you cannot convert your main account)
- Go 1.14+ installed
- The repo cloned and building successfully (`go build ./...`)

## 1. Upgrade the Lichess Account to Bot

Log in to the bot account on lichess.org, then run:

```bash
curl -d '' https://lichess.org/api/bot/account/upgrade \
  -H "Authorization: Bearer <your-token>"
```

Or use the lichess API console at <https://lichess.org/api#tag/Bot/operation/botAccountUpgrade>.

> **Warning:** This is irreversible. The account can only play as a bot afterward.

## 2. Generate an API Token

1. Go to <https://lichess.org/account/oauth/token/create>
2. Select the following scopes:
   - `bot:play` — required to make moves and stream games
3. Click **Create** and copy the token — it is shown only once.

## 3. Configure the Engine

The bot uses `game_conf.json` at the repo root to control AI behavior:

```json
{
  "Algorithm": "ABDADA (α/β Parallel)",
  "MovesToPlay": 1000,
  "SecondsToPlay": 7200,
  "AIMaxSearchDepth": 255,
  "AIMaxThinkTimeMs": 3000,
  "AIScaleThinkTimeWithHuman": false
}
```

Key fields:

| Field | Description |
|---|---|
| `Algorithm` | Search algorithm. `"ABDADA (α/β Parallel)"` is recommended for best play. |
| `AIMaxSearchDepth` | Maximum ply depth. `255` lets think-time be the effective limit. |
| `AIMaxThinkTimeMs` | Default think time per move in milliseconds. Overridden dynamically based on remaining clock time. |
| `MovesToPlay` | Maximum moves before the game is auto-aborted. `1000` is effectively unlimited. |
| `SecondsToPlay` | Total time budget in seconds before the engine aborts. `7200` = 2 hours. |

## 4. Build and Run

```bash
# Build
go build -o chess-bot ./cmd/main.go

# Run the lichess bot mode
LICHESS_TOKEN=<your-token> ./chess-bot lichess
```

The engine will connect to the lichess event stream and automatically accept incoming challenges, play moves, and handle game-over events.

## 5. Accepting Challenges

Challenges must be issued to the bot account by other users or by a script. The engine currently accepts all incoming challenges automatically. To send a challenge via the API:

```bash
curl -X POST https://lichess.org/api/challenge/<bot-username> \
  -H "Authorization: Bearer <challenger-token>" \
  -d 'clock.limit=300&clock.increment=0&color=random'
```

## 6. Logging

The engine uses [logrus](https://github.com/sirupsen/logrus) for structured logging. By default it logs to stdout. Log level can be changed at runtime; set `LOGRUS_LEVEL=debug` for verbose output including every move streamed from lichess.

## Known Limitations

- **Single concurrent game** — the engine handles one game at a time. A second `gameStart` event while a game is active returns an error. Concurrent game support is tracked as a TODO in `lichess.go`.
- **No challenge filtering** — all challenges are accepted regardless of time control, variant, or rating. Add filtering in `handleEvent` if needed.
- **No pawn promotion selection** — the UCI move encoder does not yet append a promotion piece character. Pawn promotions will default to queen on lichess's side.
- **Time management is approximate** — think time is scaled from the remaining clock but the formula is a heuristic. Very fast time controls (< 1 minute) may cause time forfeits.
