package signaling

import "testing"

func TestHubJoinAndRoute(t *testing.T) {
	h := NewHub()
	a := &Client{Peer: "a", Send: make(chan []byte, 1)}
	b := &Client{Peer: "b", Send: make(chan []byte, 1)}
	if ids, err := h.Join("room", a); err != nil || len(ids) != 0 {
		t.Fatalf("first join: ids=%v err=%v", ids, err)
	}
	if ids, err := h.Join("room", b); err != nil || len(ids) != 1 || ids[0] != "a" {
		t.Fatalf("second join: ids=%v err=%v", ids, err)
	}
	h.Broadcast("room", "b", Message{Type: "peer-joined", Peer: "a"})
	if got := string(<-a.Send); got == "" {
		t.Fatal("broadcast was not delivered")
	}
	if !h.SendTo("room", "a", Message{Type: "answer"}) {
		t.Fatal("direct message was not delivered")
	}
	h.Leave("room", "a")
	if h.Count("room") != 1 {
		t.Fatal("peer was not removed")
	}
}
