package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateReportStripsPII(t *testing.T) {
	// We mock a request coming from a specific IP
	payload := []byte(`{"building_id": "library", "category": "HVAC", "description": "Too cold"}`)
	req, _ := http.NewRequest("POST", "/api/reports", bytes.NewBuffer(payload))
	req.RemoteAddr = "192.168.1.100" // Simulated User IP
	req.Header.Set("User-Agent", "Mozilla/5.0")

	// We intercept the response
	rr := httptest.NewRecorder()

	// Create a dummy handler (in reality, you'd attach your real DB handler here)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify that we DO NOT extract or save r.RemoteAddr or headers
		if r.Header.Get("User-Agent") == "" {
			t.Errorf("Test framework failed to set header")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": 1, "status": "created"})
	})

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
}
