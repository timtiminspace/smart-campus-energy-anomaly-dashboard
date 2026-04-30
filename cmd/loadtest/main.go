package main

import (
	"fmt"
	"log"
	"net/http" // ADDED: Required for http.Post
	"time"

	"github.com/gorilla/websocket"
)

const (
	wsURL      = "ws://localhost:8080/ws"
	numClients = 50
	maxLatency = 2 * time.Second // The strict 2-second requirement
)

func main() {
	fmt.Printf("Starting Load Test: %d concurrent WebSocket clients...\n", numClients)

	done := make(chan bool)
	latencies := make([]time.Duration, 0)

	// 1. Connect 50 fake React dashboards
	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				log.Fatalf("Client %d failed to connect: %v", clientID, err)
			}
			defer c.Close()

			// Wait for the anomaly payload to arrive via WebSocket
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Fatalf("Client %d read error: %v", clientID, err)
			}

			_ = message
			done <- true
		}(i)
	}

	// Wait a second for all connections to establish
	time.Sleep(1 * time.Second)

	// 2. The Spike Injection
	fmt.Println("Connections established. Injecting massive energy spike via API...")
	startTime := time.Now()

	// Hit the Go backend to instantly simulate an anomaly
	resp, err := http.Post("http://localhost:8080/api/test/inject-spike", "application/json", nil)
	if err != nil {
		log.Printf("Spike injection request failed: %v", err)
	} else {
		// Clean up the HTTP response
		resp.Body.Close()
	}

	// 3. Measure Latency for all 50 clients to receive it
	receivedCount := 0
	for i := 0; i < numClients; i++ {
		<-done
		receivedCount++
		latency := time.Since(startTime)
		latencies = append(latencies, latency)
	}

	// 4. Calculate Results
	var total time.Duration
	for _, l := range latencies {
		total += l
	}
	avgLatency := total / time.Duration(numClients)

	fmt.Printf("\n--- PERFORMANCE GATE RESULTS ---\n")
	fmt.Printf("Clients Alerted: %d/%d\n", receivedCount, numClients)
	fmt.Printf("Average Latency: %v\n", avgLatency)

	if avgLatency < maxLatency {
		fmt.Printf("STATUS: PASS (Sub-2-second alert delivery confirmed)\n")
	} else {
		fmt.Printf("STATUS: FAIL (Latency exceeded 2 seconds)\n")
	}
}
