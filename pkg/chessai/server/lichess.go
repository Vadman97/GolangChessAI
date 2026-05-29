package server

import (
	"bufio"
	"bytes"
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
// Uses 1/30 of remaining time, capped at 10s, with a 50ms floor so the
// bot can always make a legal move even in severe time pressure.
func thinkTimeForClock(timeLeft time.Duration) time.Duration {
	think := timeLeft / 30
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
		l.Player.MaxThinkTime = thinkTimeForClock(time.Duration(event.Game.SecondsLeft * float64(time.Second)))

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

func (l *Lichess) handleBoardUpdate(event *GameEvent) error {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	switch event.Type {
	case StateTypeGameFull:
		if event.State == nil {
			log.Warnf("gameFull event missing state %+v", *event)
			return nil
		}
		return l.handleBoardUpdateLocked(event.State)
	case StateTypeGame:
		return l.handleBoardUpdateLocked(event)
	default:
		log.Warnf("unhandled game event %+v", *event)
	}
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
		lastMove := moves[len(moves)-1]
		sCol := 7 - (lastMove[0] - 'a')
		sRow := lastMove[1] - '0' - 1
		fCol := 7 - (lastMove[2] - 'a')
		fRow := lastMove[3] - '0' - 1
		endLoc := location.NewLocation(fRow, fCol)
		if len(lastMove) == 5 {
			var promoType byte
			switch lastMove[4] {
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
		m := &location.Move{
			Start: location.NewLocation(sRow, sCol),
			End:   endLoc,
		}
		log.Infof("saw opponent move %s (%s)", m.String(), m.UCIString())
		// Stop any in-progress ponder before touching the board or the player.
		l.stopPonder()
		// Invalidate ponder TT entries: opponent deviated from our predicted move,
		// so entries written during the ponder are from the wrong subtree.
		l.Player.IncrementTTGeneration()
		// TODO(vkorolik) centralize this with the gameStart
		l.Game.PlayTurnMove(m)
		l.movesApplied = len(moves)
		playerTimeMS := event.WhiteTimeMS
		if l.Player.PlayerColor == color.Black {
			playerTimeMS = event.BlackTimeMS
		}
		playerTimeLeft := time.Duration(playerTimeMS) * time.Millisecond
		l.Player.MaxThinkTime = thinkTimeForClock(playerTimeLeft)
		log.Infof("player thinking... have time %s, set max to %s", playerTimeLeft, l.Player.MaxThinkTime)
		// TODO(vkorolik) partition by gameID to allow concurrent games
		if l.Game.GameStatus == game.Active {
			l.Game.PlayTurn()
		}
		if err := l.MakeMove(l.GameID, l.Game.PreviousMove); err != nil {
			log.Errorf("move rejected by lichess, resetting game state: %s", err)
			l.resetGame()
			return nil
		}
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
		err := l.Stream(l.Events)
		if err != nil {
			log.Errorf("failed to stream event %s", err)
			return err
		}
		return nil
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
	r, err := l.Client.newRequest("GET", "/api/stream/event", nil)
	if err != nil {
		return err
	}

	response, err := l.Client.HttpClient.Do(r)
	if err != nil {
		return err
	}

	reader := bufio.NewReader(response.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		if len(line) < 3 {
			s <- Event{Type: EventTypePing}
			continue
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
