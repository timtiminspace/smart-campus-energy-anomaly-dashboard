package integrations

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type WeatherClient struct {
	cache map[string]float64
	mu    sync.RWMutex
}

func NewWeatherClient() *WeatherClient {
	return &WeatherClient{cache: make(map[string]float64)}
}

// GetCurrentTemperature returns outdoor temperature for University of Surrey (Guildford) with daily caching.
func (c *WeatherClient) GetCurrentTemperature() (float64, error) {
	key := time.Now().Format("2006-01-02")

	c.mu.RLock()
	if t, ok := c.cache[key]; ok {
		c.mu.RUnlock()
		return t, nil
	}
	c.mu.RUnlock()

	// University of Surrey coordinates
	url := "https://api.open-meteo.com/v1/forecast?latitude=51.2365&longitude=-0.5917&current_weather=true"
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("weather API: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		CurrentWeather struct {
			Temperature float64 `json:"temperature"`
		} `json:"current_weather"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("weather decode: %w", err)
	}

	temp := result.CurrentWeather.Temperature
	c.mu.Lock()
	c.cache[key] = temp
	c.mu.Unlock()

	return temp, nil
}
