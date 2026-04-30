package models

import "time"

type Reading struct {
	ID          int64     `json:"id"`
	BuildingID  string    `json:"building_id"`
	Timestamp   time.Time `json:"timestamp"`
	KWh         float64   `json:"kwh"`
	Temperature float64   `json:"temperature"`
	CO2PPM      float64   `json:"co2_ppm"`
}

type Anomaly struct {
	ID         int64     `json:"id"`
	ReadingID  int64     `json:"reading_id"`
	DetectedAt time.Time `json:"detected_at"`
	Severity   string    `json:"severity"`
	Tag        string    `json:"tag"`
	// Populated via JOIN
	BuildingID  string  `json:"building_id,omitempty"`
	KWh         float64 `json:"kwh,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	CO2PPM      float64 `json:"co2_ppm,omitempty"`
}

type Report struct {
	ID          int64     `json:"id"`
	BuildingID  string    `json:"building_id"`
	CreatedAt   time.Time `json:"created_at"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
}

type AnomalyAlert struct {
	Type    string  `json:"type"`
	Anomaly Anomaly `json:"anomaly"`
	Reading Reading `json:"reading"`
}
