package integrations

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type HolidayClient struct {
    cache  map[string]bool
    mu     sync.RWMutex
    client *http.Client
}

func NewHolidayClient() *HolidayClient {
    return &HolidayClient{
        cache:  make(map[string]bool),
        client: &http.Client{Timeout: 5 * time.Second},
    }
}

// IsHoliday returns true if the given time falls on a UK public holiday.
func (c *HolidayClient) IsHoliday(t time.Time) bool {
	dateStr := t.Format("2006-01-02")

	c.mu.RLock()
	if result, ok := c.cache[dateStr]; ok {
		c.mu.RUnlock()
		return result
	}
	c.mu.RUnlock()

	year := t.Year()
	url := fmt.Sprintf("https://date.nager.at/api/v3/PublicHolidays/%d/GB", year)
	resp, err := c.client.Get(url)
	
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}


	var holidays []struct {
		Date string `json:"date"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&holidays); err != nil {
		return false
	}

	// Cache all holidays for the year so subsequent calls are instant
	c.mu.Lock()
	isHoliday := false
	for _, h := range holidays {
		c.cache[h.Date] = true
		if h.Date == dateStr {
			isHoliday = true
		}
	}
	if !isHoliday {
		c.cache[dateStr] = false
	}
	c.mu.Unlock()

	return isHoliday
}
