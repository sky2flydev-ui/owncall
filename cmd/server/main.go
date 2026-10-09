package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourname/owncall/internal/signaling"
)

const (
	readLimit   = 64 << 10
	idleTimeout = 2 * time.Minute
)

var hub = signaling.NewHub()

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     allowedOrigin,
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthz)
	mux.HandleFunc("/config.js", configJS)
	mux.HandleFunc("/ws", websocketHandler)
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	addr := envOr("ADDR", ":8080")
	server := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(logging(mux)),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := contextWithTimeout(10 * time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	log.Printf("owncall listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func configJS(w http.ResponseWriter, _ *http.Request) {
	ice := []map[string]any{}
	if raw := strings.TrimSpace(os.Getenv("ICE_SERVERS_JSON")); raw != "" {
		if err := json.Unmarshal([]byte(raw), &ice); err != nil {
			http.Error(w, "invalid ICE_SERVERS_JSON", http.StatusInternalServerError)
			return
		}
	}
	payload, _ := json.Marshal(ice)
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("window.OWNCALL_ICE_SERVERS = " + string(payload) + ";"))
}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	room, peer := strings.TrimSpace(r.URL.Query().Get("room")), strings.TrimSpace(r.URL.Query().Get("peer"))
	if !validID(room) || !validID(peer) {
		http.Error(w, "room and peer must contain 1-128 safe characters", http.StatusBadRequest)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(readLimit)
	client := &signaling.Client{Peer: peer, Send: make(chan []byte, 32)}
	peers, err := hub.Join(room, client)
	if err != nil {
		_ = conn.WriteJSON(signaling.Message{Type: "error", Data: json.RawMessage(`{"message":"peer is already connected"}`)})
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
	go writeLoop(conn, client.Send)

	_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(idleTimeout)) })
	for {
		var msg signaling.Message
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(idleTimeout))
		if msg.Type == "ping" {
			continue
		}
		if !validSignalType(msg.Type) || msg.To == "" || len(msg.Data) > readLimit {
			continue
		}
		msg.From, msg.Room = peer, ""
		hub.SendTo(room, msg.To, msg)
	}
}

func writeLoop(conn *websocket.Conn, out <-chan []byte) {
	for payload := range out {
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			return
		}
	}
}

func validSignalType(kind string) bool {
	return kind == "offer" || kind == "answer" || kind == "candidate"
}

func validID(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !(r == '-' || r == '_' || r == '.' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func allowedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range strings.Split(envOr("ALLOWED_ORIGINS", ""), ",") {
		allowed = strings.TrimSpace(allowed)
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	parsed, err := url.Parse(origin)
	return err == nil && parsed.Host == r.Host
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func contextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

func mustJSON(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
