package signaling

import (
	"encoding/json"
	"errors"
	"sync"
)

type Message struct {
	Type string          `json:"type"`
	Room string          `json:"room,omitempty"`
	Peer string          `json:"peer,omitempty"`
	To   string          `json:"to,omitempty"`
	From string          `json:"from,omitempty"`
	Data json.RawMessage `json:"data,omitempty"`
}

type Client struct {
	Peer string
	Send chan []byte
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[string]*Client
}

func NewHub() *Hub { return &Hub{rooms: make(map[string]map[string]*Client)} }

func (h *Hub) Join(room string, c *Client) ([]string, error) {
	if room == "" || c == nil || c.Peer == "" {
		return nil, errors.New("room and peer are required")
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	peers := h.rooms[room]
	if peers == nil {
		peers = make(map[string]*Client)
		h.rooms[room] = peers
	}
	if _, exists := peers[c.Peer]; exists {
		return nil, errors.New("peer is already connected")
	}
	ids := make([]string, 0, len(peers))
	for id := range peers {
		ids = append(ids, id)
	}
	peers[c.Peer] = c
	return ids, nil
}

func (h *Hub) Leave(room, peer string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if peers := h.rooms[room]; peers != nil {
		delete(peers, peer)
		if len(peers) == 0 {
			delete(h.rooms, room)
		}
	}
}

func (h *Hub) Broadcast(room, except string, msg Message) {
	payload, _ := json.Marshal(msg)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for peer, c := range h.rooms[room] {
		if peer == except {
			continue
		}
		select {
		case c.Send <- payload:
		default:
		}
	}
}

func (h *Hub) SendTo(room, peer string, msg Message) bool {
	payload, _ := json.Marshal(msg)
	h.mu.RLock()
	c := h.rooms[room][peer]
	h.mu.RUnlock()
	if c == nil {
		return false
	}
	select {
	case c.Send <- payload:
		return true
	default:
		return false
	}
}

func (h *Hub) Count(room string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[room])
}
