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
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player"
	"github.com/Vadman97/GolangChessAI/pkg/chessai/player/ai"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"io"
	"math"
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
	StateTypeGame = "gameState"
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

type Event struct {
	Type EventType `json:"type"`
	Game *Game     `json:"game"`
}
type GameEvent struct {
	Type        StateType `json:"type"`
	Moves       string    `json:"moves"`
	WhiteTimeMS int       `json:"wtime"`
	BlackTimeMS int       `json:"btime"`
	Status      string    `json:"status"`
}

type Lichess struct {
	Client *Client
	// TODO(vkorolik) per game mutex
	Mutex sync.Mutex
	// TODO(vkorolik)
	// store a map of gameID -> game for concurrent games?
	GameID     string
	Player     *ai.AIPlayer
	Game       *game.Game
	Events     chan Event
	GameEvents chan GameEvent
}

func ConnectLichess() Server {
	base, _ := url.Parse("https://lichess.org/api")
	return &Lichess{
		Mutex: sync.Mutex{},
		Client: &Client{
			BaseURL:    base,
			UserAgent:  "GolangChessAI",
			APIKey:     os.Getenv("LICHESS_TOKEN"),
			HttpClient: new(http.Client),
		},
		Events:     make(chan Event),
		GameEvents: make(chan GameEvent),
	}
}

func (l *Lichess) handleEvent(event *Event) error {
	l.Mutex.Lock()
	defer l.Mutex.Unlock()
	switch event.Type {
	case EventTypeGameStart:
		if l.Game != nil {
			return errors.New("game already exists")
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
		l.Player.MaxThinkTime = 5 * time.Second

		// Create game and start game loop
		if playerColor == color.White {
			l.Game = game.NewGame(l.Player, enemyPlayer)
		} else {
			l.Game = game.NewGame(enemyPlayer, l.Player)
		}

		l.Game.MoveLimit = game_config.Get().MovesToPlay
		l.Game.TimeLimit = game_config.Get().SecondsToPlay * time.Second

		go func() {
			err := l.StreamBoardUpdate(event.Game.GameID, l.GameEvents)
			if err != nil {
				log.Errorf("failed to stream board update %s", err)
			}
		}()

		if l.Game.CurrentTurnColor == playerColor {
			l.Game.PlayTurn()
			if err := l.MakeMove(event.Game.GameID, l.Game.PreviousMove); err != nil {
				return err
			}
		}
		// otherwise we wait for board updates and react there..
	case EventTypeGameFinish:
		if l.Game == nil {
			return errors.New("game does not exists")
		}
		l.Player = nil
		l.Game = nil
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
	case StateTypeGame:
		if l.Player == nil || l.Game == nil {
			log.Errorf("received board event after game over %+v", event)
			return nil
		}
		moves := strings.Split(event.Moves, " ")
		if len(moves)%2 != int(l.Player.PlayerColor) {
			return nil
		}
		lastMove := moves[len(moves)-1]
		sCol := 7 - (lastMove[0] - 'a')
		sRow := lastMove[1] - '0' - 1
		fCol := 7 - (lastMove[2] - 'a')
		fRow := lastMove[3] - '0' - 1
		m := &location.Move{
			Start: location.NewLocation(sRow, sCol),
			End:   location.NewLocation(fRow, fCol),
		}
		log.Infof("saw opponent move %s (%s)", m.String(), m.UCIString())
		// TODO(vkorolik) centralize this with the gameStart
		l.Game.PlayTurnMove(m)
		playerTimeLeft := time.Duration(event.WhiteTimeMS) * time.Millisecond
		l.Player.MaxThinkTime = time.Duration(math.Max(playerTimeLeft.Seconds()/60.*float64(l.Game.MovesPlayed)/10, time.Millisecond.Seconds())*1000) * time.Millisecond
		log.Infof("player thinking... have time %s, set max to %s", playerTimeLeft, l.Player.MaxThinkTime)
		// TODO(vkorolik) partition by gameID to allow concurrent games
		if l.Game.GameStatus == game.Active {
			l.Game.PlayTurn()
		}
		if err := l.MakeMove(l.GameID, l.Game.PreviousMove); err != nil {
			return err
		}
	default:
		log.Warnf("unhandled game event %+v", *event)
	}
	return nil
}

func (l *Lichess) Run() {
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
		for {
			e := <-l.Events
			if err := l.handleEvent(&e); err != nil {
				log.Errorf("failed to handle event %s", err)
				return err
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

func (l *Lichess) MakeMove(gameID string, move *board.LastMove) error {
	sCol := 7 - move.Move.Start.GetCol()
	fCol := 7 - move.Move.End.GetCol()
	m := &location.Move{
		Start: location.NewLocation(move.Move.Start.GetRow(), sCol),
		End:   location.NewLocation(move.Move.End.GetRow(), fCol),
	}
	moveStr := m.UCIString()
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
