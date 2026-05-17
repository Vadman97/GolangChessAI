package api_handlers

import (
	"github.com/Vadman97/GolangChessAI/pkg/api"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	"time"
)

type spectatorClient struct {
	ws      *websocket.Conn
	writeCh chan api.ChessMessage
}

// SpectatorHub relays tournament game state to any number of read-only WebSocket clients.
// Each client gets its own write goroutine so a slow/stalled client never blocks the others.
type SpectatorHub struct {
	mu             sync.Mutex
	clients        map[*websocket.Conn]*spectatorClient
	broadcastCh    chan api.ChessMessage
	lastState      *api.ChessMessage
	lastTournament *api.ChessMessage
}

func NewSpectatorHub() *SpectatorHub {
	return &SpectatorHub{
		clients:     make(map[*websocket.Conn]*spectatorClient),
		broadcastCh: make(chan api.ChessMessage, 256),
	}
}

// Run fans out messages from broadcastCh to every registered client's write channel.
// It never touches WebSocket connections directly, so it never blocks on I/O.
func (h *SpectatorHub) Run() {
	for msg := range h.broadcastCh {
		h.mu.Lock()
		switch msg.Type {
		case api.GameState:
			cp := msg
			h.lastState = &cp
		case api.TournamentInfo:
			cp := msg
			h.lastTournament = &cp
		}
		for _, c := range h.clients {
			select {
			case c.writeCh <- msg:
			default:
				log.Printf("spectator write buffer full, dropping %s", msg.Type)
			}
		}
		h.mu.Unlock()
	}
}

// BroadcastCh returns the channel that the tournament writes messages to.
func (h *SpectatorHub) BroadcastCh() chan api.ChessMessage {
	return h.broadcastCh
}

// clientWriteLoop is the sole goroutine that writes to a WebSocket connection.
func clientWriteLoop(c *spectatorClient) {
	for msg := range c.writeCh {
		c.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := c.ws.WriteJSON(msg); err != nil {
			log.Printf("spectator write error: %v", err)
			c.ws.Close()
			return
		}
	}
}

// HandleSpectatorConnection upgrades an HTTP request to a spectator WebSocket.
func (h *SpectatorHub) HandleSpectatorConnection(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("spectator upgrade error: %v", err)
		return
	}

	client := &spectatorClient{
		ws:      ws,
		writeCh: make(chan api.ChessMessage, 64),
	}

	// Enqueue catch-up state before registering (hub.Run hasn't seen this client yet).
	h.mu.Lock()
	lastTournament := h.lastTournament
	lastState := h.lastState
	h.mu.Unlock()

	if lastTournament != nil {
		client.writeCh <- *lastTournament
	}
	if lastState != nil {
		client.writeCh <- *lastState
	}

	// Start the write goroutine, then register so hub.Run() starts fanning to it.
	go clientWriteLoop(client)

	h.mu.Lock()
	h.clients[ws] = client
	h.mu.Unlock()
	log.Printf("spectator connected (%d total)", len(h.clients))

	// Spectators are read-only; drain incoming frames until disconnect.
	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			break
		}
	}

	h.mu.Lock()
	delete(h.clients, ws)
	h.mu.Unlock()
	close(client.writeCh) // signals clientWriteLoop to exit
	log.Printf("spectator disconnected (%d remaining)", len(h.clients))
}
