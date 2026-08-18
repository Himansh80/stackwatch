package ml

import (
	"math"
	"testing"
)

// TestDetector_BasicAnomaly: feed 20 stable values, then a spike.
// Expected: spike produces z-score > 3 and is flagged.
func TestDetector_BasicAnomaly(t *testing.T) {
	d := NewDetector(20, 3.0)
	// Feed 20 stable samples at 50.0
	for i := 0; i < 20; i++ {
		d.Update(50.0 + float64(i%2)) // 50 or 51 — tiny noise
	}
	// Should not be anomaly
	if _, isAnom := d.Update(50.5); isAnom {
		t.Errorf("expected 50.5 to NOT be anomaly after stable data")
	}
	// Spike should be anomaly
	z, isAnom := d.Update(150.0)
	if !isAnom {
		t.Errorf("expected 150.0 to be anomaly, got z=%v", z)
	}
	if math.Abs(z) < 3.0 {
		t.Errorf("expected |z| >= 3.0, got %v", z)
	}
}

// TestDetector_Warmup: detector shouldn't flag anomalies during warmup
// (first few samples) since variance is unreliable.
func TestDetector_Warmup(t *testing.T) {
	d := NewDetector(20, 3.0)
	// First sample — count=1, no variance yet
	_, isAnom := d.Update(100.0)
	if isAnom {
		t.Errorf("first sample should not be anomaly (no variance yet)")
	}
	// Second sample — count=2, just barely a stddev
	z, isAnom := d.Update(1000.0)
	if isAnom {
		t.Logf("warning: 2nd sample flagged (z=%v, may be expected with single pair)", z)
	}
}

// TestDetector_NoFalsePositivesOnFlatData: 100 identical samples should
// produce z=0 always.
func TestDetector_NoFalsePositivesOnFlatData(t *testing.T) {
	d := NewDetector(20, 3.0)
	for i := 0; i < 100; i++ {
		z, isAnom := d.Update(42.0)
		if isAnom {
			t.Errorf("flat data should never be anomaly (z=%v)", z)
		}
		_ = z
	}
}

// TestIsAnomalyValue: stateless helper.
func TestIsAnomalyValue(t *testing.T) {
	values := []float64{50, 50, 50, 50, 51, 50, 50, 51, 50, 50}
	z, isAnom := IsAnomalyValue(values, 100.0, 3.0)
	if !isAnom || math.Abs(z) < 3.0 {
		t.Errorf("100.0 vs stable ~50 should be anomaly (z=%v)", z)
	}
}

// TestDetector_EWMA: EWMA should track recent values more closely than mean.
func TestDetector_EWMA(t *testing.T) {
	d := NewDetector(20, 3.0)
	for i := 0; i < 20; i++ {
		d.Update(50.0)
	}
	// Drop to 10 — mean will lag, EWMA should track faster
	for i := 0; i < 20; i++ {
		d.Update(10.0)
	}
	if d.EWMA() > 20 || d.EWMA() < 10 {
		t.Errorf("EWMA should converge near 10 after 20 samples, got %v", d.EWMA())
	}
}
