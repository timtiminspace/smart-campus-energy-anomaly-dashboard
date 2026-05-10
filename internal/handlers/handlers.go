package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"smart-campus-dashboard/internal/integrations"
	"smart-campus-dashboard/internal/models"
	"strconv"
	"strings"
	"time"
	"log"
	"smart-campus-dashboard/internal/ws"
)

type Handlers struct {
    db       *sql.DB
    mlClient *integrations.MLClient
    hub      *ws.Hub
}

func NewHandlers(db *sql.DB, mlClient *integrations.MLClient, hub *ws.Hub) *Handlers {
	return &Handlers{db: db, mlClient: mlClient, hub: hub}
}

func jsonResponse(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// Replaces http.Error() calls.
func jsonError(w http.ResponseWriter, msg string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func parseTimestamp(s string) time.Time {
	for _, layout := range []string{"2006-01-02 15:04:05", time.RFC3339, "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// GET /api/readings?building=X&limit=N
func (h *Handlers) GetReadings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	building := r.URL.Query().Get("building")
	limit := 60
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 3000 {
			limit = n
		}
	}

	var (
		rows *sql.Rows
		err  error
	)
	if building != "" {
		rows, err = h.db.Query(
			`SELECT id, building_id, timestamp, kwh, temperature, co2_ppm FROM readings WHERE building_id = ? ORDER BY timestamp DESC LIMIT ?`,
			building, limit,
		)
	} else {
		rows, err = h.db.Query(
			`SELECT id, building_id, timestamp, kwh, temperature, co2_ppm FROM readings ORDER BY timestamp DESC LIMIT ?`,
			limit,
		)
	}
	if err != nil {
		log.Printf("db error in GetReadings: %v", err)
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	readings := []models.Reading{}
	for rows.Next() {
		var rd models.Reading
		var ts string
		if err := rows.Scan(&rd.ID, &rd.BuildingID, &ts, &rd.KWh, &rd.Temperature, &rd.CO2PPM); err != nil {
			continue
		}
		rd.Timestamp = parseTimestamp(ts)
		readings = append(readings, rd)
	}

	jsonResponse(w, http.StatusOK, readings)
}

// GET /api/anomalies?building=X&severity=Y  |  PATCH /api/anomalies?id=X
func (h *Handlers) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listAnomalies(w, r)
	case http.MethodPatch:
		h.updateAnomalyTag(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) listAnomalies(w http.ResponseWriter, r *http.Request) {
	building := r.URL.Query().Get("building")
	severity := r.URL.Query().Get("severity")

	query := `SELECT a.id, a.reading_id, a.detected_at, a.severity, COALESCE(a.tag,''),
        r.building_id, r.kwh, r.temperature, r.co2_ppm
        FROM anomalies a JOIN readings r ON a.reading_id = r.id`

	var conds []string
	var args []interface{}
	if building != "" {
		conds = append(conds, "r.building_id = ?")
		args = append(args, building)
	}
	if severity != "" {
		conds = append(conds, "a.severity = ?")
		args = append(args, severity)
	}
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += " ORDER BY a.detected_at DESC LIMIT 100"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		log.Printf("db error in listAnomalies: %v", err)
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	anomalies := []models.Anomaly{}
	for rows.Next() {
		var a models.Anomaly
		var ts string
		if err := rows.Scan(&a.ID, &a.ReadingID, &ts, &a.Severity, &a.Tag,
			&a.BuildingID, &a.KWh, &a.Temperature, &a.CO2PPM); err != nil {
			continue
		}
		a.DetectedAt = parseTimestamp(ts)
		anomalies = append(anomalies, a)
	}

	jsonResponse(w, http.StatusOK, anomalies)
}

func (h *Handlers) updateAnomalyTag(w http.ResponseWriter, r *http.Request) {
	// Adds body size limit.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	
	id := r.URL.Query().Get("id")
	if id == "" {
		jsonError(w, "id required", http.StatusBadRequest)
		return
	}

	// Ensures id is numeric.
	if _, err := strconv.ParseInt(id, 10, 64); err != nil {
    	jsonError(w, "id must be a valid integer", http.StatusBadRequest)
    	return
	}

	var body struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}

	if len(body.Tag) > 100 {
		jsonError(w, "tag exceeds maximum length", http.StatusBadRequest)
		return
	}

	if _, err := h.db.Exec("UPDATE anomalies SET tag = ? WHERE id = ?", body.Tag, id); err != nil {
		log.Printf("db error in updateAnomalies: %v", err)
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// POST /api/reports  — no IP or identity data logged
func (h *Handlers) CreateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Adds body size limit.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) 

	var body struct {
		BuildingID  string `json:"building_id"`
		Category    string `json:"category"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Category == "" || body.Description == "" {
		jsonError(w, "category and description required", http.StatusBadRequest)
		return
	}

	validCategories := map[string]bool{
		"lighting": true, "hvac": true, "equipment": true,
		"water": true, "security": true, "other": true,
	}
	if !validCategories[body.Category] {
		jsonError(w, "invalid category", http.StatusBadRequest)
		return
	}

	if len(body.Category) > 100 || len(body.Description) > 2000 {
		jsonError(w, "category or description exceeds maximum length", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec(
		`INSERT INTO reports (building_id, created_at, category, description) VALUES (?, datetime('now'), ?, ?)`,
		body.BuildingID, body.Category, body.Description,
	)
	if err != nil {
		log.Printf("db error in CreateReport: %v", err)
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	jsonResponse(w, http.StatusCreated, map[string]interface{}{"id": id, "status": "created"})
}

// GET /api/forecast?building=X&steps=Y
func (h *Handlers) GetForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
    	jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
    	return
	}
	
	building := r.URL.Query().Get("building")
	if building == "" {
		building = "library"
	}

	// --- THE FIX: Dynamically read 'steps' from the React frontend ---
	steps := 12 // Default to 1 hour
	if s := r.URL.Query().Get("steps"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 288 {
			steps = n
		}
	}
	// -----------------------------------------------------------------

	// 1. Pull all metrics: kWh, Temperature, and CO2
	rows, err := h.db.Query(
		`SELECT kwh, temperature, co2_ppm, timestamp 
         FROM readings 
         WHERE building_id = ? 
         ORDER BY timestamp DESC LIMIT 144`,
		building,
	)
	if err != nil {
		log.Printf("db error in GetForecast: %v", err)
		jsonError(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var historyKWh []float64
	var historyTemp []float64
	var historyCO2 []float64
	var oldestTs string
	var latestTs string

	for rows.Next() {
		var kwh, temp, co2 float64
		var ts string
		if err := rows.Scan(&kwh, &temp, &co2, &ts); err == nil {
			historyKWh = append(historyKWh, kwh)
			historyTemp = append(historyTemp, temp)
			historyCO2 = append(historyCO2, co2)
			if latestTs == "" {
				latestTs = ts
			}
			oldestTs = ts
		}
	}

	if len(historyKWh) < 24 {
		// Return empty if we don't have enough data for a PhD-tier LSTM window
		jsonResponse(w, http.StatusOK, map[string]interface{}{})
		return
	}

	// 2. Reverse all slices (chronological order for Python)
	reverse := func(s []float64) {
		for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
			s[i], s[j] = s[j], s[i]
		}
	}
	reverse(historyKWh)
	reverse(historyTemp)
	reverse(historyCO2)

	// 3. Call ML Service with the multivariate data and the DYNAMIC steps
	startTime := parseTimestamp(oldestTs).Format(time.RFC3339)
	forecasts, err := h.mlClient.GetForecast(historyKWh, historyTemp, historyCO2, steps, startTime)
	if err != nil {
		log.Printf("ml error in GetForecast: %v", err)
		jsonError(w, "forecast service unavailable", http.StatusBadGateway)
		return
	}

	// 4. Format the response for React
	type predPoint struct {
		Timestamp string  `json:"timestamp"`
		Predicted float64 `json:"predicted"`
	}

	lastTime := parseTimestamp(latestTs)
	response := make(map[string][]predPoint)

	// Helper to map the raw float slices from Python to timestamped points for Recharts
	mapToTimeline := func(key string, vals []float64) {
		var points []predPoint
		for i, val := range vals {
			points = append(points, predPoint{
				Timestamp: lastTime.Add(time.Duration(i+1) * 5 * time.Minute).Format(time.RFC3339),
				Predicted: val,
			})
		}
		response[key] = points
	}

	mapToTimeline("kwh", forecasts["kwh"])
	mapToTimeline("temp", forecasts["temp"])
	mapToTimeline("co2", forecasts["co2"])

	jsonResponse(w, http.StatusOK, response)
}

// POST /api/test/inject-spike  — forces a high-kWh anomaly for load/latency testing
func (h *Handlers) InjectSpike(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    building := r.URL.Query().Get("building")
    if building == "" {
        building = "library"
    }

    now := time.Now()
    result, err := h.db.Exec(
        `INSERT INTO readings (building_id, timestamp, kwh, temperature, co2_ppm)
         VALUES (?, ?, ?, ?, ?)`,
        building, now.Format("2006-01-02 15:04:05"), 9999.0, 22.0, 400.0,
    )
    if err != nil {
        log.Printf("db error in InjectSpike: %v", err)
        jsonError(w, "internal server error", http.StatusInternalServerError)
        return
    }
    readingID, _ := result.LastInsertId()

    anomalyResult, err := h.db.Exec(
        `INSERT INTO anomalies (reading_id, detected_at, severity, tag) VALUES (?, datetime('now'), ?, ?)`,
        readingID, "high", "test-spike",
    )
    if err != nil {
        log.Printf("db error in InjectSpike (anomaly): %v", err)
		jsonError(w, "internal server error", http.StatusInternalServerError)
        return
    }
    anomalyID, _ := anomalyResult.LastInsertId()

    reading := models.Reading{
        ID: readingID, BuildingID: building,
        Timestamp: now, KWh: 9999.0, Temperature: 22.0, CO2PPM: 400.0,
    }
    anomaly := models.Anomaly{
        ID: anomalyID, ReadingID: readingID,
        DetectedAt: now, Severity: "high", Tag: "test-spike",
        BuildingID: building, KWh: 9999.0, Temperature: 22.0, CO2PPM: 400.0,
    }
    h.hub.Broadcast(models.AnomalyAlert{Type: "anomaly", Anomaly: anomaly, Reading: reading})

    jsonResponse(w, http.StatusOK, map[string]string{"status": "spike injected"})
}