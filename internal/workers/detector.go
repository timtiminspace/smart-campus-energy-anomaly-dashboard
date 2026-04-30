package workers

import (
	"log"
	"math"
	"smart-campus-dashboard/internal/models"
)

// detectAnomaly runs z-score analysis on the latest reading against recent history.
// If anomalous, it persists an anomaly record and broadcasts it over WebSocket.
func (i *Ingestor) detectAnomaly(reading models.Reading, outOfHours, isHoliday bool) error {
	rows, err := i.db.Query(
		`SELECT kwh FROM readings WHERE building_id = ? ORDER BY timestamp DESC LIMIT 12`,
		reading.BuildingID,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	var vals []float64
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			continue
		}
		vals = append(vals, v)
	}

	if len(vals) < 3 {
		return nil
	}

	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))

	variance := 0.0
	for _, v := range vals {
		d := v - mean
		variance += d * d
	}
	stdDev := math.Sqrt(variance / float64(len(vals)))

	if stdDev == 0 {
		return nil
	}

	zScore := math.Abs((reading.KWh - mean) / stdDev)

	// Lower the z-score threshold outside business hours or on holidays
	threshold := 2.5
	if outOfHours || isHoliday {
		threshold = 2.0
	}

	rollingThreshold := reading.KWh > mean*1.5

	if zScore < threshold && !rollingThreshold {
		return nil
	}

	severity := "low"
	switch {
	case zScore >= 4:
		severity = "high"
	case zScore >= 3:
		severity = "medium"
	}

	tag := ""
	switch {
	case isHoliday:
		tag = "holiday-spike"
	case outOfHours:
		tag = "out-of-hours"
	}

	result, err := i.db.Exec(
		`INSERT INTO anomalies (reading_id, detected_at, severity, tag) VALUES (?, datetime('now'), ?, ?)`,
		reading.ID, severity, tag,
	)
	if err != nil {
		return err
	}

	anomalyID, _ := result.LastInsertId()
	anomaly := models.Anomaly{
		ID:          anomalyID,
		ReadingID:   reading.ID,
		DetectedAt:  reading.Timestamp,
		Severity:    severity,
		Tag:         tag,
		BuildingID:  reading.BuildingID,
		KWh:         reading.KWh,
		Temperature: reading.Temperature,
		CO2PPM:      reading.CO2PPM,
	}

	i.hub.Broadcast(models.AnomalyAlert{
		Type:    "anomaly",
		Anomaly: anomaly,
		Reading: reading,
	})

	log.Printf("anomaly: building=%s severity=%s z=%.2f kwh=%.1f mean=%.1f",
		reading.BuildingID, severity, zScore, reading.KWh, mean)
	return nil
}
