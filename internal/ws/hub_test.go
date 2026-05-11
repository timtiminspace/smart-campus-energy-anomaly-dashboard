package ws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func waitForClientRegistration(t *testing.T, hub *Hub) {
	t.Helper()

	deadline := time.Now().Add(1 * time.Second)

	for time.Now().Before(deadline) {
		hub.mu.Lock()
		clientCount := len(hub.clients)
		hub.mu.Unlock()

		if clientCount == 1 {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("expected WebSocket client to register within 1 second")
}

func TestWebSocketBroadcastDeliversWithinTwoSeconds(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWS(hub, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect WebSocket client: %v", err)
	}
	defer conn.Close()

	waitForClientRegistration(t, hub)

	type testAlert struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}

	expected := testAlert{
		Type:    "anomaly",
		Message: "latency-test",
	}

	received := make(chan []byte, 1)
	readErr := make(chan error, 1)

	go func() {
		_, message, err := conn.ReadMessage()
		if err != nil {
			readErr <- err
			return
		}
		received <- message
	}()

	start := time.Now()
	hub.Broadcast(expected)

	select {
	case err := <-readErr:
		t.Fatalf("failed to read WebSocket message: %v", err)

	case message := <-received:
		elapsed := time.Since(start)

		if elapsed > 2*time.Second {
			t.Fatalf("expected WebSocket broadcast within 2 seconds, got %v", elapsed)
		}

		var actual testAlert
		if err := json.Unmarshal(message, &actual); err != nil {
			t.Fatalf("message was not valid JSON: %v", err)
		}

		if actual.Type != expected.Type || actual.Message != expected.Message {
			t.Fatalf("unexpected WebSocket message: got %+v, want %+v", actual, expected)
		}

	case <-time.After(2 * time.Second):
		t.Fatalf("WebSocket message was not received within 2 seconds")
	}
}
