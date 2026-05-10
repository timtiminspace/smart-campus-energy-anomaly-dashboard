package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smart-campus-dashboard/internal/database"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to initialise test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func newTestHandlers(t *testing.T) (*Handlers, *sql.DB) {
	t.Helper()

	db := newTestDB(t)

	// nil ML client is okay for most handler tests.
	// Tests that touch /api/forecast should avoid forcing a real ML call unless using a mock.
	h := NewHandlers(db, nil)

	return h, db
}

func TestCreateReport_AcceptsValidAnonymousReport(t *testing.T) {
	h, db := newTestHandlers(t)

	body := []byte(`{
		"building_id": "library",
		"category": "lighting",
		"description": "Lights left on overnight"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/reports", bytes.NewReader(body))
	req.RemoteAddr = "192.168.1.55:12345"
	req.Header.Set("User-Agent", "Test Browser")

	rr := httptest.NewRecorder()

	h.CreateReport(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("response was not valid JSON: %v", err)
	}

	if response["status"] != "created" {
		t.Fatalf("expected status created, got %v", response["status"])
	}

	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM reports`).Scan(&count)
	if err != nil {
		t.Fatalf("failed to count reports: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 report in database, got %d", count)
	}
}

func TestCreateReport_DoesNotStoreIPOrUserAgent(t *testing.T) {
	h, db := newTestHandlers(t)

	body := []byte(`{
		"building_id": "library",
		"category": "lighting",
		"description": "Anonymous issue report"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/reports", bytes.NewReader(body))
	req.RemoteAddr = "203.0.113.42:5555"
	req.Header.Set("User-Agent", "Very Specific Test Browser")

	rr := httptest.NewRecorder()

	h.CreateReport(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	rows, err := db.Query(`PRAGMA table_info(reports)`)
	if err != nil {
		t.Fatalf("failed to inspect reports table: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue interface{}
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("failed to scan table info: %v", err)
		}

		switch name {
		case "ip", "ip_address", "remote_addr", "user_agent", "email", "name", "session":
			t.Fatalf("reports table contains PII/tracking column %q", name)
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("table info rows error: %v", err)
	}
}

func TestCreateReport_RejectsWrongMethod(t *testing.T) {
	h, _ := newTestHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
	rr := httptest.NewRecorder()

	h.CreateReport(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestCreateReport_RejectsInvalidJSON(t *testing.T) {
	h, _ := newTestHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/api/reports", bytes.NewReader([]byte(`{bad json`)))
	rr := httptest.NewRecorder()

	h.CreateReport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateReport_RejectsMissingCategory(t *testing.T) {
	h, _ := newTestHandlers(t)

	body := []byte(`{
		"building_id": "library",
		"description": "Lights left on"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/reports", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateReport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateReport_RejectsMissingDescription(t *testing.T) {
	h, _ := newTestHandlers(t)

	body := []byte(`{
		"building_id": "library",
		"category": "lighting"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/reports", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateReport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

func TestCreateReport_RejectsInvalidCategory(t *testing.T) {
	h, _ := newTestHandlers(t)

	body := []byte(`{
		"building_id": "library",
		"category": "definitely-not-a-real-category",
		"description": "Something is wrong"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/reports", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.CreateReport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid category to be rejected with %d, got %d. This exposes weak validation in CreateReport.", http.StatusBadRequest, rr.Code)
	}
}

func TestGetReadings_ReturnsInsertedReadings(t *testing.T) {
	h, db := newTestHandlers(t)

	_, err := db.Exec(`
		INSERT INTO readings (building_id, timestamp, kwh, temperature, co2_ppm)
		VALUES
		('library', ?, 100.5, 21.0, 500.0),
		('library', ?, 110.5, 22.0, 510.0),
		('engineering', ?, 90.0, 20.0, 480.0)
	`,
		time.Now().UTC().Add(-10*time.Minute).Format("2006-01-02 15:04:05"),
		time.Now().UTC().Add(-5*time.Minute).Format("2006-01-02 15:04:05"),
		time.Now().UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("failed to insert readings: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/readings?building=library&limit=2", nil)
	rr := httptest.NewRecorder()

	h.GetReadings(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var readings []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&readings); err != nil {
		t.Fatalf("response was not valid JSON: %v", err)
	}

	if len(readings) != 2 {
		t.Fatalf("expected 2 library readings, got %d", len(readings))
	}

	for _, reading := range readings {
		if reading["building_id"] != "library" {
			t.Fatalf("expected only library readings, got %v", reading["building_id"])
		}
	}
}

func TestGetReadings_RejectsWrongMethod(t *testing.T) {
	h, _ := newTestHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/api/readings", nil)
	rr := httptest.NewRecorder()

	h.GetReadings(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestGetAnomalies_ReturnsJoinedAnomalyData(t *testing.T) {
	h, db := newTestHandlers(t)

	result, err := db.Exec(`
		INSERT INTO readings (building_id, timestamp, kwh, temperature, co2_ppm)
		VALUES ('library', ?, 250.0, 23.0, 900.0)
	`, time.Now().UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		t.Fatalf("failed to insert reading: %v", err)
	}

	readingID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get reading id: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO anomalies (reading_id, detected_at, severity, tag)
		VALUES (?, datetime('now'), 'high', 'out-of-hours')
	`, readingID)
	if err != nil {
		t.Fatalf("failed to insert anomaly: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/anomalies?building=library&severity=high", nil)
	rr := httptest.NewRecorder()

	h.GetAnomalies(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var anomalies []map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&anomalies); err != nil {
		t.Fatalf("response was not valid JSON: %v", err)
	}

	if len(anomalies) != 1 {
		t.Fatalf("expected 1 anomaly, got %d", len(anomalies))
	}

	a := anomalies[0]

	if a["building_id"] != "library" {
		t.Fatalf("expected building_id library, got %v", a["building_id"])
	}

	if a["severity"] != "high" {
		t.Fatalf("expected severity high, got %v", a["severity"])
	}

	if a["tag"] != "out-of-hours" {
		t.Fatalf("expected tag out-of-hours, got %v", a["tag"])
	}
}

func TestGetAnomalies_RejectsWrongMethod(t *testing.T) {
	h, _ := newTestHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/api/anomalies", nil)
	rr := httptest.NewRecorder()

	h.GetAnomalies(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
	}
}

func TestGetForecast_RejectsWrongMethod(t *testing.T) {
	h, _ := newTestHandlers(t)

	req := httptest.NewRequest(http.MethodPost, "/api/forecast", nil)
	rr := httptest.NewRecorder()

	h.GetForecast(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected forecast endpoint to reject POST with %d, got %d. This exposes missing method validation in GetForecast.", http.StatusMethodNotAllowed, rr.Code)
	}
}
