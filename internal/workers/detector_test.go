package workers

import (
	"math"
	"testing"
)

// Helper function that mimics your detector's z-score logic
func calculateZScore(value, mean, stdDev float64) float64 {
	if stdDev == 0 {
		return 0
	}
	return math.Abs(value-mean) / stdDev
}

func TestZScoreMath(t *testing.T) {
	// Test standard deviation calculation
	val := 150.0
	mean := 100.0
	stdDev := 20.0

	z := calculateZScore(val, mean, stdDev)
	expected := 2.5

	if z != expected {
		t.Errorf("Expected z-score %.2f, got %.2f", expected, z)
	}
}

func TestThresholdLogic(t *testing.T) {
	// Standard hours threshold is 2.5, Out-of-hours is 2.0
	isOutOfHours := true
	zScore := 2.2

	threshold := 2.5
	if isOutOfHours {
		threshold = 2.0
	}

	isAnomaly := zScore > threshold
	if !isAnomaly {
		t.Errorf("Expected out-of-hours z-score of 2.2 to trigger anomaly (threshold 2.0)")
	}
}
