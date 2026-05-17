package api_handlers

import (
	"github.com/Vadman97/GolangChessAI/pkg/api"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

// SpectatorHub relays tournament game state to any number of read-only WebSocket clients.
type SpectatorHub struct {
	mu              sync.Mutex
	clients         map[*websocket.Conn]bool
	broadcastCh     chan api.ChessMessage
	lastState       *api.ChessMessage
	lastTournament  *api.ChessMessage
}

func NewSpectatorHub() *SpectatorHub {
	return &SpectatorHub{
		clients:     make(map[*websocket.Conn]bool),
		broadcastCh: make(chan api.ChessMessage, 256),
	}
}

// Run drains broadcastCh and fans out to all connected spectators. Call in a goroutine.
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
		for conn := range h.clients {
			if err := conn.WriteJSON(msg); err != nil {
				log.Printf("spectator send error: %v", err)
				conn.Close()
				delete(h.clients, conn)
			}
		}
		h.mu.Unlock()
	}
}

// BroadcastCh returns the channel that the tournament writes messages to.
func (h *SpectatorHub) BroadcastCh() chan api.ChessMessage {
	return h.broadcastCh
}

// Broadcast enqueues a message for all spectators. Non-blocking; drops if the buffer is full.
func (h *SpectatorHub) Broadcast(msg api.ChessMessage) {
	select {
	case h.broadcastCh <- msg:
	default:
		log.Printf("spectator broadcast buffer full, dropping %s message", msg.Type)
	}
}

// HandleSpectatorConnection upgrades an HTTP request to a spectator WebSocket.
func (h *SpectatorHub) HandleSpectatorConnection(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("spectator upgrade error: %v", err)
		return
	}

	// Snapshot catch-up state before registering so hub.Run() doesn't write
	// to this connection concurrently while we send the initial messages.
	h.mu.Lock()
	lastTournament := h.lastTournament
	lastState := h.lastState
	h.mu.Unlock()

	if lastTournament != nil {
		ws.WriteJSON(*lastTournament) //nolint:errcheck
	}
	if lastState != nil {
		ws.WriteJSON(*lastState) //nolint:errcheck
	}

	// Register only after initial writes are done.
	h.mu.Lock()
	h.clients[ws] = true
	h.mu.Unlock()
	log.Printf("spectator connected (%d total)", len(h.clients))

	// Spectators are read-only; drain any incoming frames until disconnect.
	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			break
		}
	}

	h.mu.Lock()
	delete(h.clients, ws)
	h.mu.Unlock()
	ws.Close()
	log.Printf("spectator disconnected (%d remaining)", len(h.clients))
}
