package integrations

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type weatherCache struct {
	temp      float64
	expiresAt time.Time
}

type WeatherClient struct {
	cache  *weatherCache
	mu     sync.RWMutex
	client *http.Client
}

func NewWeatherClient() *WeatherClient {
	return &WeatherClient{
		client: &http.Client{Timeout: 5 * time.Second},
	}
}


// GetCurrentTemperature returns outdoor temperature for University of Surrey (Guildford) with daily caching.
func (c *WeatherClient) GetCurrentTemperature() (float64, error) {
    c.mu.RLock()
    if c.cache != nil && time.Now().Before(c.cache.expiresAt) {
        t := c.cache.temp
        c.mu.RUnlock()
        return t, nil
    }
    c.mu.RUnlock()

    url := "https://api.open-meteo.com/v1/forecast?latitude=51.2365&longitude=-0.5917&current_weather=true"
    resp, err := c.client.Get(url)
    if err != nil {
        return 0, fmt.Errorf("weather API: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return 0, fmt.Errorf("weather API: unexpected status %d", resp.StatusCode)
    }

    var result struct {
        CurrentWeather struct {
            Temperature float64 `json:"temperature"`
        } `json:"current_weather"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, fmt.Errorf("weather decode: %w", err)
    }

    c.mu.Lock()
    c.cache = &weatherCache{
        temp:      result.CurrentWeather.Temperature,
        expiresAt: time.Now().Add(1 * time.Hour),
    }
    c.mu.Unlock()

    return result.CurrentWeather.Temperature, nil
}