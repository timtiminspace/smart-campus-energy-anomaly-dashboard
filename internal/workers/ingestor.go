package workers

import (
	"database/sql"
	"log"
	"math"
	"math/rand"
	"smart-campus-dashboard/internal/integrations"
	"smart-campus-dashboard/internal/models"
	"smart-campus-dashboard/internal/ws"
	"time"
)

var buildings = []struct {
	id   string
	name string
}{
	{id: "library", name: "Main Library"},
	{id: "sports-hall", name: "Sports Hall"},
	{id: "main-hall", name: "Main Hall"},
	{id: "engineering", name: "Engineering Block"},
	{id: "admin", name: "Admin Building"},
}

type Ingestor struct {
	db       *sql.DB
	hub      *ws.Hub
	weather  *integrations.WeatherClient
	holidays *integrations.HolidayClient
}

func NewIngestor(db *sql.DB, hub *ws.Hub, weather *integrations.WeatherClient, holidays *integrations.HolidayClient) *Ingestor {
	return &Ingestor{db: db, hub: hub, weather: weather, holidays: holidays}
}

// Run generates a batch of readings for all buildings and runs anomaly detection.
func (i *Ingestor) Run() {
	now := time.Now()
	isHoliday := i.holidays.IsHoliday(now)
	isOutOfHours := now.Hour() < 8 || now.Hour() >= 18

	for _, b := range buildings {
		reading := i.generateReading(b.id, now, isOutOfHours, isHoliday)

		result, err := i.db.Exec(
			`INSERT INTO readings (building_id, timestamp, kwh, temperature, co2_ppm) VALUES (?, ?, ?, ?, ?)`,
			reading.BuildingID, reading.Timestamp.UTC().Format("2006-01-02 15:04:05"),
			reading.KWh, reading.Temperature, reading.CO2PPM,
		)
		if err != nil {
			log.Printf("ingestor: insert reading for %s: %v", b.id, err)
			continue
		}

		id, _ := result.LastInsertId()
		reading.ID = id

		if err := i.detectAnomaly(reading, isOutOfHours, isHoliday); err != nil {
			log.Printf("ingestor: anomaly detection for %s: %v", b.id, err)
		}
	}

	log.Printf("ingestor: generated readings for %d buildings (holiday=%v, out-of-hours=%v)", len(buildings), isHoliday, isOutOfHours)
}

func (i *Ingestor) generateReading(buildingID string, ts time.Time, outOfHours, isHoliday bool) models.Reading {
	occupancyFactor := 1.0
	if outOfHours || isHoliday {
		occupancyFactor = 0.25
	}

	baseKWh := (10 + rand.Float64()*190) * occupancyFactor
	baseTemp := 18 + rand.Float64()*10
	baseCO2 := 400 + rand.Float64()*800*occupancyFactor

	// Inject a kWh spike ~15% of the time to exercise anomaly detection
	if rand.Float64() < 0.15 {
		baseKWh *= 1.5 + rand.Float64()*0.5
	}

	return models.Reading{
		BuildingID:  buildingID,
		Timestamp:   ts,
		KWh:         math.Round(baseKWh*100) / 100,
		Temperature: math.Round(baseTemp*100) / 100,
		CO2PPM:      math.Round(baseCO2*100) / 100,
	}
}
