package workers

import (
	"database/sql"
	"testing"
	"time"

	"smart-campus-dashboard/internal/database"
	"smart-campus-dashboard/internal/models"
	"smart-campus-dashboard/internal/ws"
)

func newDetectorTestIngestor(t *testing.T) (*Ingestor, *sql.DB) {
	t.Helper()

	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to initialise test database: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	hub := ws.NewHub()

	ingestor := &Ingestor{
		db:  db,
		hub: hub,
	}

	return ingestor, db
}

func insertReading(t *testing.T, db *sql.DB, buildingID string, timestamp time.Time, kwh float64) models.Reading {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO readings (building_id, timestamp, kwh, temperature, co2_ppm)
		VALUES (?, ?, ?, ?, ?)
	`,
		buildingID,
		timestamp.UTC().Format("2006-01-02 15:04:05"),
		kwh,
		21.0,
		500.0,
	)
	if err != nil {
		t.Fatalf("failed to insert reading: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("failed to get reading id: %v", err)
	}

	return models.Reading{
		ID:          id,
		BuildingID:  buildingID,
		Timestamp:   timestamp,
		KWh:         kwh,
		Temperature: 21.0,
		CO2PPM:      500.0,
	}
}

func countAnomalies(t *testing.T, db *sql.DB) int {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM anomalies`).Scan(&count); err != nil {
		t.Fatalf("failed to count anomalies: %v", err)
	}

	return count
}

func seedStableHistory(t *testing.T, db *sql.DB, buildingID string, start time.Time, values []float64) {
	t.Helper()

	for index, value := range values {
		insertReading(t, db, buildingID, start.Add(time.Duration(index)*5*time.Minute), value)
	}
}

func TestDetectAnomaly_DoesNotFlagNormalReading(t *testing.T) {
	ingestor, db := newDetectorTestIngestor(t)

	start := time.Now().UTC().Add(-2 * time.Hour)

	seedStableHistory(t, db, "library", start, []float64{
		98, 101, 100, 99, 102, 100,
		101, 99, 100, 98, 102, 101,
	})

	normal := insertReading(t, db, "library", time.Now().UTC(), 101)

	if err := ingestor.detectAnomaly(normal, false, false); err != nil {
		t.Fatalf("detectAnomaly returned error: %v", err)
	}

	if got := countAnomalies(t, db); got != 0 {
		t.Fatalf("expected normal reading not to create anomaly, got %d anomalies", got)
	}
}

func TestDetectAnomaly_FlagsLargeSpike(t *testing.T) {
	ingestor, db := newDetectorTestIngestor(t)

	start := time.Now().UTC().Add(-2 * time.Hour)

	seedStableHistory(t, db, "library", start, []float64{
		98, 101, 100, 99, 102, 100,
		101, 99, 100, 98, 102, 101,
	})

	spike := insertReading(t, db, "library", time.Now().UTC(), 250)

	if err := ingestor.detectAnomaly(spike, false, false); err != nil {
		t.Fatalf("detectAnomaly returned error: %v", err)
	}

	if got := countAnomalies(t, db); got != 1 {
		t.Fatalf("expected spike to create 1 anomaly, got %d", got)
	}

	var severity string
	var tag string

	err := db.QueryRow(`SELECT severity, tag FROM anomalies LIMIT 1`).Scan(&severity, &tag)
	if err != nil {
		t.Fatalf("failed to read anomaly: %v", err)
	}

	if severity == "" {
		t.Fatalf("expected anomaly severity to be populated")
	}

	if tag != "kwh-spike" {
		t.Fatalf("expected kwh-spike tag during normal hours/non-holiday, got %q", tag)
	}
}

func TestDetectAnomaly_AddsOutOfHoursTag(t *testing.T) {
	ingestor, db := newDetectorTestIngestor(t)

	start := time.Now().UTC().Add(-2 * time.Hour)

	seedStableHistory(t, db, "library", start, []float64{
		98, 101, 100, 99, 102, 100,
		101, 99, 100, 98, 102, 101,
	})

	spike := insertReading(t, db, "library", time.Now().UTC(), 250)

	if err := ingestor.detectAnomaly(spike, true, false); err != nil {
		t.Fatalf("detectAnomaly returned error: %v", err)
	}

	var tag string
	if err := db.QueryRow(`SELECT tag FROM anomalies LIMIT 1`).Scan(&tag); err != nil {
		t.Fatalf("failed to read anomaly tag: %v", err)
	}

	if tag != "kwh-spike out-of-hours" {
		t.Fatalf("expected 'kwh-spike out-of-hours' tag, got %q", tag)
	}
}

func TestDetectAnomaly_AddsHolidayTag(t *testing.T) {
	ingestor, db := newDetectorTestIngestor(t)

	start := time.Now().UTC().Add(-2 * time.Hour)

	seedStableHistory(t, db, "library", start, []float64{
		98, 101, 100, 99, 102, 100,
		101, 99, 100, 98, 102, 101,
	})

	spike := insertReading(t, db, "library", time.Now().UTC(), 250)

	if err := ingestor.detectAnomaly(spike, false, true); err != nil {
		t.Fatalf("detectAnomaly returned error: %v", err)
	}

	var tag string
	if err := db.QueryRow(`SELECT tag FROM anomalies LIMIT 1`).Scan(&tag); err != nil {
		t.Fatalf("failed to read anomaly tag: %v", err)
	}

	if tag != "kwh-spike holiday-spike" {
		t.Fatalf("expected 'kwh-spike holiday-spike' tag, got %q", tag)
	}
}

func TestDetectAnomaly_RequiresEnoughPriorHistory(t *testing.T) {
	ingestor, db := newDetectorTestIngestor(t)

	start := time.Now().UTC().Add(-20 * time.Minute)

	// Only two historical points is not enough to form a reliable statistical baseline.
	seedStableHistory(t, db, "library", start, []float64{
		100, 101,
	})

	spike := insertReading(t, db, "library", time.Now().UTC(), 250)

	if err := ingestor.detectAnomaly(spike, false, false); err != nil {
		t.Fatalf("detectAnomaly returned error: %v", err)
	}

	if got := countAnomalies(t, db); got != 0 {
		t.Fatalf("expected no anomaly with insufficient prior history, got %d. This exposes that the detector allows too little history or includes the current reading in its baseline.", got)
	}
}

func TestDetectAnomaly_IgnoresOtherBuildingsHistory(t *testing.T) {
	ingestor, db := newDetectorTestIngestor(t)

	start := time.Now().UTC().Add(-2 * time.Hour)

	seedStableHistory(t, db, "engineering", start, []float64{
		10, 11, 9, 10, 11, 9,
		10, 11, 9, 10, 11, 9,
	})

	seedStableHistory(t, db, "library", start, []float64{
		98, 101, 100, 99, 102, 100,
		101, 99, 100, 98, 102, 101,
	})

	spike := insertReading(t, db, "library", time.Now().UTC(), 250)

	if err := ingestor.detectAnomaly(spike, false, false); err != nil {
		t.Fatalf("detectAnomaly returned error: %v", err)
	}

	if got := countAnomalies(t, db); got != 1 {
		t.Fatalf("expected library spike to create exactly 1 anomaly, got %d", got)
	}
}
