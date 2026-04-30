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
)

type Handlers struct {
	db       *sql.DB
	mlClient *integrations.MLClient
}

func NewHandlers(db *sql.DB, mlClient *integrations.MLClient) *Handlers {
	return &Handlers{db: db, mlClient: mlClient}
}

func jsonResponse(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	var body struct {
		Tag string `json:"tag"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if _, err := h.db.Exec("UPDATE anomalies SET tag = ? WHERE id = ?", body.Tag, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, http.StatusOK, map[string]string{"status": "updated"})
}

// POST /api/reports  — no IP or identity data logged
func (h *Handlers) CreateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		BuildingID  string `json:"building_id"`
		Category    string `json:"category"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Category == "" || body.Description == "" {
		http.Error(w, "category and description required", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec(
		`INSERT INTO reports (building_id, created_at, category, description) VALUES (?, datetime('now'), ?, ?)`,
		body.BuildingID, body.Category, body.Description,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	jsonResponse(w, http.StatusCreated, map[string]interface{}{"id": id, "status": "created"})
}

// GET /api/forecast?building=X&steps=Y
func (h *Handlers) GetForecast(w http.ResponseWriter, r *http.Request) {
	building := r.URL.Query().Get("building")
	if building == "" {
		building = "library"
	}

	// --- THE FIX: Dynamically read 'steps' from the React frontend ---
	steps := 12 // Default to 1 hour
	if s := r.URL.Query().Get("steps"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		jsonResponse(w, http.StatusOK, map[string]interface{}{})
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
