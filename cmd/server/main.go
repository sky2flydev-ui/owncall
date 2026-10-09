package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourname/owncall/internal/signaling"
)

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var hub = signaling.NewHub()

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/ws", websocketHandler)
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{Addr: addr, Handler: logging(mux), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("owncall listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	room, peer := strings.TrimSpace(r.URL.Query().Get("room")), strings.TrimSpace(r.URL.Query().Get("peer"))
	if room == "" || peer == "" || len(room) > 128 || len(peer) > 128 {
		http.Error(w, "room and peer are required", http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &signaling.Client{Peer: peer, Send: make(chan []byte, 32)}
	peers, err := hub.Join(room, client)
	if err != nil {
		_ = conn.WriteJSON(signaling.Message{Type: "error", Data: json.RawMessage(`{"message":"peer already connected"}`)})
		_ = conn.Close()
		return
	}
	defer func() {
		hub.Leave(room, peer)
		hub.Broadcast(room, peer, signaling.Message{Type: "peer-left", Peer: peer})
		close(client.Send)
		_ = conn.Close()
	}()
	_ = conn.WriteJSON(signaling.Message{Type: "peers", Data: mustJSON(peers)})
	hub.Broadcast(room, peer, signaling.Message{Type: "peer-joined", Peer: peer})
	go func() {
		for payload := range client.Send {
			_ = conn.WriteMessage(websocket.TextMessage, payload)
		}
	}()
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	for {
		var msg signaling.Message
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if msg.Type == "ping" {
			continue
		}
		if msg.To != "" {
			msg.From = peer
			msg.Room = ""
			hub.SendTo(room, msg.To, msg)
			continue
		}
	}
}

func mustJSON(v any) json.RawMessage { b, _ := json.Marshal(v); return b }
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
