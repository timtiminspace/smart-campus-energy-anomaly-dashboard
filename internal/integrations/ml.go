package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MLClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewMLClient(baseURL string) *MLClient {
	return &MLClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Second * 5,
		},
	}
}

// GetForecast now accepts all three historical streams to enable multivariate prediction
func (c *MLClient) GetForecast(kwh, temp, co2 []float64, steps int, startTs string) (map[string][]float64, error) {
	payload := map[string]interface{}{
		"history_kwh":     kwh,
		"history_temp":    temp,
		"history_co2":     co2,
		"steps":           steps,
		"start_timestamp": startTs,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/api/ml/forecast", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("ml service network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ml service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// The Python brain now returns a JSON object with three distinct arrays
	var result map[string][]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode ml response: %w", err)
	}

	return result, nil
}
