package main

import (
	"log"
	"net/http"
	"os"
	"smart-campus-dashboard/internal/database"
	"smart-campus-dashboard/internal/handlers"
	"smart-campus-dashboard/internal/integrations"
	"smart-campus-dashboard/internal/workers"
	"smart-campus-dashboard/internal/ws"

	"github.com/robfig/cron/v3"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/campus.db"
	}

	if err := os.MkdirAll("./data", 0755); err != nil {
		log.Fatalf("failed to create data dir: %v", err)
	}

	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("failed to init database: %v", err)
	}
	defer db.Close()

	hub := ws.NewHub()
	go hub.Run()

	weather := integrations.NewWeatherClient()
	holidays := integrations.NewHolidayClient()
	ingestor := workers.NewIngestor(db, hub, weather, holidays)

	mlURL := os.Getenv("ML_URL")
	if mlURL == "" {
		mlURL = "http://127.0.0.1:8000"
	}
	mlClient := integrations.NewMLClient(mlURL)

	// Seed an initial batch of readings on startup
	go ingestor.Run()

	// Then generate fresh readings every 5 minutes
	c := cron.New()
	if _, err := c.AddFunc("@every 5m", ingestor.Run); err != nil {
		log.Fatalf("failed to schedule ingestor: %v", err)
	}
	c.Start()
	defer c.Stop()

	h := handlers.NewHandlers(db, mlClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/readings", h.GetReadings)
	mux.HandleFunc("/api/anomalies", h.GetAnomalies)
	mux.HandleFunc("/api/reports", h.CreateReport)
	mux.HandleFunc("/api/forecast", h.GetForecast)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWS(hub, w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, corsMiddleware(mux)))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
