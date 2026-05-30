package server

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/board"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/color"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/game"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/game_config"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/location"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/piece"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type EventType string

const (
	EventTypePing              = "ping"
	EventTypeGameStart         = "gameStart"
	EventTypeGameFinish        = "gameFinish"
	EventTypeChallenge         = "challenge"
	EventTypeChallengeCanceled = "challengeCanceled"
	EventTypeChallengeDeclined = "challengeDeclined"
)

type StateType string

const (
	StateTypeGame     = "gameState"
	StateTypeGameFull = "gameFull"
)

type Game struct {
	ID              string  `json:"id"`
	GameID          string  `json:"gameId"`
	FullID          string  `json:"fullId"`
	BoardSerialized string  `json:"fen"`
	Color           string  `json:"color"`
	LastMove        string  `json:"lastMove"`
	HasMoved        bool    `json:"hasMoved"`
	IsMyTurn        bool    `json:"isMyTurn"`
	SecondsLeft     float64 `json:"secondsLeft"`
	Source          string  `json:"source"`
}

type Challenge struct {
	ID string `json:"id"`
}

type Event struct {
	Type               EventType  `json:"type"`
	Game               *Game      `json:"game"`
	Challenge          *Challenge `json:"challenge"`
	ChallengeDirection string     `json:"direction"`
}
type GameEvent struct {
	Type        StateType  `json:"type"`
	Moves       string     `json:"moves"`
	WhiteTimeMS int        `json:"wtime"`
	BlackTimeMS int        `json:"btime"`
	Status      string     `json:"status"`
	State       *GameEvent `json:"state"`
}

type ChallengeConfig struct {
	Username      string
	ClockLimitSec int
	ClockIncSec   int
	Rated         bool
}

type Lichess struct {
	Client *Client
	// TODO(vkorolik) per game mutex
	Mutex sync.Mutex
	// TODO(vkorolik)
	// store a map of gameID -> game for concurrent games?
	GameID           string
	Player           *ai.AIPlayer
	Game             *game.Game
	Events           chan Event
	GameEvents       chan GameEvent
	ChallengeOnStart *ChallengeConfig
	// exitAfterGame signals Run() to stop after the first game finishes.
	exitAfterGame chan struct{}
	// movesApplied tracks how many total moves from lichess events we've applied
	// to our local board. Used to skip duplicate events (e.g. after stream reconnect).
	movesApplied int

	// Pondering: search during the opponent's turn to warm the TT.
	ponderStop chan struct{}
	ponderDone chan struct{}
}

func ConnectLichess() Server {
	return ConnectLichessWithChallenge(nil)
}

func ConnectLichessWithChallenge(challenge *ChallengeConfig) Server {
	base, _ := url.Parse("https://lichess.org/api")
	s := &Lichess{
		Mutex: sync.Mutex{},
		Client: &Client{
			BaseURL:    base,
			UserAgent:  "GolangChessAI",
			APIKey:     os.Getenv("LICHESS_TOKEN"),
			HttpClient: new(http.Client),
		},
		Events:           make(chan Event),
		GameEvents:       make(chan GameEvent),
		ChallengeOnStart: challenge,
	}
	if challenge != nil {
		s.exitAfterGame = make(chan struct{})
	}
	return s
}

// startPonder begins a background search on the current board position so the
// transposition table is warm when it's our turn again. Must be called with
// the Lichess mutex held (it snapshots the board and then releases into a goroutine).
func (l *Lichess) startPonder() {
	if l.Player == nil || l.Game == nil {
		return
	}
	boardSnap := l.Game.CurrentBoard.Copy()
	prevMove := l.Game.PreviousMove
	player := l.Player

	stop := make(chan struct{})
	done := make(chan struct{})
	l.ponderStop = stop
	l.ponderDone = done

	go func() {
		defer close(done)
		// When stop is closed, set the abort flag so the search returns promptly.
		go func() {
			select {
			case <-stop:
				player.Abort()
			case <-done:
			}
		}()
		player.MaxThinkTime = 60 * time.Second
		player.GetBestMove(boardSnap, prevMove, nil)
		log.Debugf("ponder finished naturally")
	}()
	log.Debugf("pondering started")
}

// stopPonder aborts any in-progress ponder and waits for it to finish.
// Safe to call when no ponder is running. Must be called before starting
// the main search so the two searches don't race on the abort flag.
func (l *Lichess) stopPonder() {
	stop, done := l.ponderStop, l.ponderDone
	if stop == nil {
		return
	}
	select {
	case <-stop:
		// already stopped (ponder finished naturally before opponent moved)
	default:
		close(stop)
	}
	if done != nil {
		<-done
	}
	l.ponderStop = nil
	l.ponderDone = nil
	if l.Player != nil {
		l.Player.ResetAbort()
	}
	log.Debugf("ponder stopped")
}

// resetGame tears down any active game and clears all game-level state.
// Safe to call when no game is running. Must be called with the mutex held.
func (l *Lichess) resetGame() {
	l.stopPonder()
	l.Player = nil
	l.Game = nil
	l.movesApplied = 0
}

// thinkTimeForClock allocates think time from the remaining clock.
// Adapts to game phase: more time per move in the middlegame and endgame
// where precision is critical, less in the opening where fewer decisions matter.
// Capped at 10s so we never burn the clock on one move.
func thinkTimeForClock(timeLeft time.Duration, turnCount int) time.Duration {
	// Estimated remaining moves for THIS side based on how many moves we've made.
	// Early: expect ~30 more; middlegame: ~20; endgame: ~12.
	var divisor time.Duration
	switch {
	case turnCount < 10:
		divisor = 30
	case turnCount < 25:
		divisor = 20
	default:
		divisor = 12
	}
	think := timeLeft / divisor
	if think < 50*time.Millisecond {
		think = 50 * time.Millisecond
	}
	if think > 10*time.Second {
		think = 10 * time.Second
	}
	return think
}

func (l *Lichess) handleEvent(event *Event) error {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	switch event.Type {
	case EventTypeGameStart:
		if l.Game != nil {
			log.Warnf("gameStart received while game %s active — resetting stale game state", l.GameID)
			l.resetGame()
		}
		l.GameID = event.Game.GameID
		// human is the other player
		playerColor := color.White
		enemyColor := color.Black
		if event.Game.Color == "black" {
			playerColor = color.Black
			enemyColor = color.White
		}
		enemyPlayer := player.NewHumanPlayer(enemyColor)
		l.Player = ai.NewAIPlayer(playerColor, ai.NameToAlgorithm[ai.AlgorithmABDADA])
		l.Player.MaxSearchDepth = game_config.Get().AIMaxSearchDepth
		l.Player.MaxThinkTime = thinkTimeForClock(time.Duration(event.Game.SecondsLeft*float64(time.Second)), l.Player.TurnCount)

		// Create game and start game loop
		if playerColor == color.White {
			l.Game = game.NewGame(l.Player, enemyPlayer)
		} else {
			l.Game = game.NewGame(enemyPlayer, l.Player)
		}

		l.Game.MoveLimit = game_config.Get().MovesToPlay
		l.Game.TimeLimit = game_config.Get().SecondsToPlay * time.Second
		l.movesApplied = 0

		go func() {
			err := l.StreamBoardUpdate(event.Game.GameID, l.GameEvents)
			if err != nil {
				log.Errorf("failed to stream board update %s", err)
			}
		}()

		if l.Game.CurrentTurnColor == playerColor {
			l.Game.PlayTurn()
			if err := l.MakeMove(event.Game.GameID, l.Game.PreviousMove); err != nil {
				log.Errorf("first move rejected by lichess, resetting game state: %s", err)
				l.resetGame()
				return nil
			}
			l.movesApplied++ // our move is now on the board; keep movesApplied in sync
		}
		// Ponder while waiting for the opponent's first move.
		l.startPonder()
		// otherwise we wait for board updates and react there..
	case EventTypeGameFinish:
		if l.Game == nil {
			log.Warnf("gameFinish received but no active game — ignoring")
			break
		}
		l.resetGame()
		if l.exitAfterGame != nil {
			select {
			case <-l.exitAfterGame:
			default:
				close(l.exitAfterGame)
			}
		}
	case EventTypeChallenge:
		if event.Challenge == nil {
			return errors.New("challenge event missing challenge data")
		}
		// Only accept incoming challenges; outgoing ones (direction=="out") cannot be accepted.
		if event.ChallengeDirection == "out" {
			log.Debugf("ignoring our own outgoing challenge %s", event.Challenge.ID)
			break
		}
		if l.Game != nil {
			log.Infof("ignoring challenge %s: game %s already active", event.Challenge.ID, l.GameID)
			break
		}
		if err := l.AcceptChallenge(event.Challenge.ID); err != nil {
			log.Errorf("failed to accept challenge %s: %s", event.Challenge.ID, err)
		}
	case EventTypePing:
		log.Debugf("ping...")
	default:
		log.Warnf("unhandled event %+v", *event)
	}
	return nil
}

// parseUCIMove converts a UCI move string (e.g. "e2e4", "a7a8q") to a Move.
func parseUCIMove(uci string) *location.Move {
	sCol := 7 - (uci[0] - 'a')
	sRow := uci[1] - '0' - 1
	fCol := 7 - (uci[2] - 'a')
	fRow := uci[3] - '0' - 1
	endLoc := location.NewLocation(fRow, fCol)
	if len(uci) == 5 {
		var promoType byte
		switch uci[4] {
		case 'q':
			promoType = piece.QueenType
		case 'r':
			promoType = piece.RookType
		case 'b':
			promoType = piece.BishopType
		case 'n':
			promoType = piece.KnightType
		}
		endLoc = endLoc.CreatePawnPromotion(promoType)
	}
	return &location.Move{
		Start: location.NewLocation(sRow, sCol),
		End:   endLoc,
	}
}

func (l *Lichess) handleBoardUpdate(event *GameEvent) error {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	switch event.Type {
	case StateTypeGameFull:
		if event.State == nil {
			log.Warnf("gameFull event missing state %+v", *event)
			return nil
		}
		// Reconnect path: replay the full move history to sync the board, then
		// act if it's our turn. handleBoardUpdateLocked is not used here because
		// its "not our turn" early-return would skip the board sync entirely.
		return l.handleGameFullLocked(event.State)
	case StateTypeGame:
		return l.handleBoardUpdateLocked(event)
	default:
		log.Warnf("unhandled game event %+v", *event)
	}
	return nil
}

// handleGameFullLocked syncs the board from a gameFull event by replaying all
// historical moves, then responds if it is our turn. Must be called with the
// mutex held.
func (l *Lichess) handleGameFullLocked(state *GameEvent) error {
	if l.Player == nil || l.Game == nil {
		log.Errorf("gameFull received but no active game")
		return nil
	}
	if state.Moves == "" {
		// No moves yet: game just started. If we're White, gameStart already
		// handled our first move; if Black, just wait for the opponent.
		l.startPonder()
		return nil
	}
	moves := strings.Split(state.Moves, " ")
	// Replay any moves we haven't applied yet (all of them on a fresh reconnect).
	log.Infof("gameFull: replaying moves %d..%d to sync board", l.movesApplied, len(moves)-1)
	l.stopPonder()
	l.Player.IncrementTTGeneration()
	for l.movesApplied < len(moves) {
		m := parseUCIMove(moves[l.movesApplied])
		l.Game.PlayTurnMove(m)
		l.movesApplied++
	}
	// After replay, check whose turn it is.
	playerTimeMS := state.WhiteTimeMS
	if l.Player.PlayerColor == color.Black {
		playerTimeMS = state.BlackTimeMS
	}
	playerTimeLeft := time.Duration(playerTimeMS) * time.Millisecond
	l.Player.MaxThinkTime = thinkTimeForClock(playerTimeLeft, l.Player.TurnCount)
	if len(moves)%2 == int(l.Player.PlayerColor) {
		// It's our turn.
		log.Infof("gameFull: our turn after replay, thinking... have time %s, set max to %s", playerTimeLeft, l.Player.MaxThinkTime)
		if l.Game.GameStatus == game.Active {
			l.Game.PlayTurn()
		}
		if err := l.MakeMove(l.GameID, l.Game.PreviousMove); err != nil {
			log.Errorf("move rejected after gameFull replay: %s", err)
			l.resetGame()
			return nil
		}
		l.movesApplied++ // our reply is now on the board; keep movesApplied in sync
	}
	l.startPonder()
	return nil
}

func (l *Lichess) handleBoardUpdateLocked(event *GameEvent) error {
	switch event.Type {
	case StateTypeGame:
		if l.Player == nil || l.Game == nil {
			log.Errorf("received board event after game over %+v", event)
			return nil
		}
		if event.Moves == "" {
			// No moves yet — game just started and it is white's turn.
			return nil
		}
		moves := strings.Split(event.Moves, " ")
		if len(moves)%2 != int(l.Player.PlayerColor) {
			return nil
		}
		// Skip events we've already applied — lichess can resend after a stream reconnect.
		if len(moves) <= l.movesApplied {
			log.Debugf("skipping already-applied event (event has %d moves, applied %d)", len(moves), l.movesApplied)
			return nil
		}
		// Catch up on any missed intermediate moves (e.g. after a brief stream gap).
		for l.movesApplied < len(moves)-1 {
			l.Game.PlayTurnMove(parseUCIMove(moves[l.movesApplied]))
			l.movesApplied++
		}
		m := parseUCIMove(moves[len(moves)-1])
		log.Infof("saw opponent move %s (%s)", m.String(), m.UCIString())
		// Stop any in-progress ponder before touching the board or the player.
		l.stopPonder()
		// Invalidate ponder TT entries: opponent deviated from our predicted move,
		// so entries written during the ponder are from the wrong subtree.
		l.Player.IncrementTTGeneration()
		l.Game.PlayTurnMove(m)
		l.movesApplied = len(moves)
		playerTimeMS := event.WhiteTimeMS
		if l.Player.PlayerColor == color.Black {
			playerTimeMS = event.BlackTimeMS
		}
		playerTimeLeft := time.Duration(playerTimeMS) * time.Millisecond
		l.Player.MaxThinkTime = thinkTimeForClock(playerTimeLeft, l.Player.TurnCount)
		log.Infof("player thinking... have time %s, set max to %s", playerTimeLeft, l.Player.MaxThinkTime)
		if l.Game.GameStatus == game.Active {
			l.Game.PlayTurn()
		}
		if err := l.MakeMove(l.GameID, l.Game.PreviousMove); err != nil {
			log.Errorf("move rejected by lichess, resetting game state: %s", err)
			l.resetGame()
			return nil
		}
		l.movesApplied++ // our reply is now on the board; keep movesApplied in sync
		// Begin pondering on the resulting position while the opponent thinks.
		l.startPonder()
	default:
		log.Warnf("unhandled game event %+v", *event)
	}
	return nil
}

func (l *Lichess) ChallengeUser(cfg *ChallengeConfig) error {
	u := fmt.Sprintf("/api/challenge/%s", cfg.Username)
	params := url.Values{}
	params.Set("rated", fmt.Sprintf("%t", cfg.Rated))
	params.Set("clock.limit", fmt.Sprintf("%d", cfg.ClockLimitSec))
	params.Set("clock.increment", fmt.Sprintf("%d", cfg.ClockIncSec))
	params.Set("color", "random")
	params.Set("variant", "standard")

	rel := &url.URL{Path: u}
	fullURL := l.Client.BaseURL.ResolveReference(rel)
	req, err := http.NewRequest("POST", fullURL.String(), strings.NewReader(params.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", l.Client.APIKey))
	req.Header.Set("User-Agent", l.Client.UserAgent)

	resp, err := l.Client.HttpClient.Do(req)
	if err != nil {
		return err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Infof("challenge %s status %s %s", cfg.Username, resp.Status, string(bodyBytes))
	return nil
}

func (l *Lichess) Run() {
	if l.ChallengeOnStart != nil {
		if err := l.ChallengeUser(l.ChallengeOnStart); err != nil {
			log.Errorf("failed to send challenge: %s", err)
		}
	}

	var g errgroup.Group
	g.Go(func() error {
		backoff := 3 * time.Second
		for {
			err := l.Stream(l.Events)
			if err != nil {
				log.Errorf("failed to stream event %s — reconnecting in %s", err, backoff)
				time.Sleep(backoff)
				// Exponential backoff capped at 30s for EOF/context errors
				// to avoid hammering Lichess when rate-limited.
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			backoff = 3 * time.Second // reset on success
			return nil
		}
	})
	g.Go(func() error {
		// exitAfterGame is nil when not in challenge mode; a nil channel in
		// select blocks forever, so this case only fires in challenge mode.
		var exitCh <-chan struct{} = l.exitAfterGame
		for {
			select {
			case e := <-l.Events:
				if err := l.handleEvent(&e); err != nil {
					log.Errorf("failed to handle event %s", err)
					return err
				}
			case <-exitCh:
				log.Infof("game finished, exiting")
				return nil
			}
		}
	})
	g.Go(func() error {
		for {
			ge := <-l.GameEvents
			if err := l.handleBoardUpdate(&ge); err != nil {
				log.Errorf("failed to handle board update %s", err)
				return err
			}
		}
	})
	err := g.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func (l *Lichess) AcceptChallenge(challengeID string) error {
	u := fmt.Sprintf("/api/challenge/%s/accept", challengeID)
	r, err := l.Client.newRequest("POST", u, nil)
	if err != nil {
		return err
	}
	resp, err := l.Client.HttpClient.Do(r)
	if err != nil {
		return err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Infof("accept challenge %s status %s %s", challengeID, resp.Status, string(bodyBytes))
	return nil
}

func (l *Lichess) MakeMove(gameID string, move *board.LastMove) error {
	moveStr := move.Move.UCIString()
	if move.PromotionPiece != nil {
		moveStr += strings.ToLower(string((*move.PromotionPiece).GetChar()))
	}
	oferringDraw := "false"
	if l.Game.GameStatus == game.RepeatedActionThreeTimeDraw {
		oferringDraw = "true"
	}
	u := fmt.Sprintf("/api/bot/game/%s/move/%s?offeringDraw=%s", gameID, moveStr, oferringDraw)
	r, err := l.Client.newRequest("POST", u, nil)
	if err != nil {
		return err
	}
	resp, err := l.Client.HttpClient.Do(r)
	if err != nil {
		return err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	log.Infof("make move %s status %+v %d %s", u, resp.Status, resp.StatusCode, string(bodyBytes))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("make move rejected: %s", string(bodyBytes))
	}
	return nil
}

func (l *Lichess) Stream(s chan<- Event) error {
	// Inactivity watchdog: if no data arrives for 2 minutes, cancel the request.
	// Lichess sends keepalive newlines every ~10s during normal operation; 2 minutes
	// only fires when rate-limited (connection held open but no data sent).
	// The watchdog is reset on every successful line read so normal streams stay open.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const inactivityTimeout = 2 * time.Minute
	watchdog := time.AfterFunc(inactivityTimeout, cancel)
	defer watchdog.Stop()

	r, err := l.Client.newRequest("GET", "/api/stream/event", nil)
	if err != nil {
		return err
	}
	r = r.WithContext(ctx)

	response, err := l.Client.HttpClient.Do(r)
	if err != nil {
		return err
	}
	if response.StatusCode == 429 {
		// Rate-limited: back off for 5 minutes before the caller retries.
		_ = response.Body.Close()
		log.Warnf("event stream rate limited (HTTP 429) — backing off 5 minutes")
		time.Sleep(5 * time.Minute)
		return fmt.Errorf("rate limited (429)")
	}

	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		watchdog.Reset(inactivityTimeout) // data received — reset the inactivity timer
		if len(line) < 3 {
			s <- Event{Type: EventTypePing}
			continue
		}
		// Detect rate-limit responses delivered as body text (CDN redirect pattern).
		if bytes.Contains(line, []byte("Too many requests")) || bytes.Contains(line, []byte("/429")) {
			_ = response.Body.Close()
			log.Warnf("rate limited in event stream body — backing off 5 minutes")
			time.Sleep(5 * time.Minute)
			return fmt.Errorf("rate limited (body 429)")
		}
		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			log.Errorf("failed to unmarshal line to event %s", line)
		}
		s <- event
	}
}

func (l *Lichess) StreamBoardUpdate(gameID string, s chan<- GameEvent) error {
	r, err := l.Client.newRequest("GET", fmt.Sprintf("/api/bot/game/stream/%s", gameID), nil)
	if err != nil {
		return err
	}

	response, err := l.Client.HttpClient.Do(r)
	if err != nil {
		return err
	}
	if response.StatusCode == 429 {
		_ = response.Body.Close()
		time.Sleep(60 * time.Second)
		return fmt.Errorf("rate limited (429)")
	}

	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		if len(line) < 3 {
			continue
		}
		log.Infof("game event line %s", string(line))
		var event GameEvent
		if err := json.Unmarshal(line, &event); err != nil {
			log.Errorf("failed to unmarshal line to game event %s", line)
		}
		s <- event
	}
}

// HTTPClient interface
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	BaseURL    *url.URL
	UserAgent  string
	APIKey     string
	HttpClient HTTPClient
}

func (c *Client) newRequest(method, path string, body interface{}) (*http.Request, error) {
	if c.BaseURL == nil {
		return nil, errors.New("BaseURL is undefined")
	}
	if c.APIKey == "" {
		return nil, errors.New("APIKey is undefined")
	}

	rel := &url.URL{Path: path}
	u := c.BaseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	// Default request is json
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	return req, nil
}

func (c *Client) do(req *http.Request,
	v interface{}) (*http.Response, error) {
	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if v != nil {
		err = json.NewDecoder(resp.Body).Decode(v)
	}

	return resp, err
}
