// Package ml provides streaming anomaly detection for time-series metrics.
//
// Detector implements rolling Z-score + EWMA. It tracks a window of
// recent values and reports how many standard deviations the latest
// value is from the moving mean. Values beyond |z| >= Threshold are
// flagged as anomalies.
//
// Usage:
//
//	d := ml.NewDetector(20, 3.0)  // window=20, anomaly threshold=3.0
//	for v := range stream {
//	    score, isAnomaly := d.Update(v)
//	    if isAnomaly { alert(score) }
//	}
package ml

import (
	"math"
	"sync"
)

// Detector is goroutine-safe; one instance per (tenant, server, metric).
type Detector struct {
	mu        sync.Mutex
	Window    int     // number of recent samples to keep (default 20)
	Threshold float64 // |z| >= Threshold means anomaly (default 3.0)
	Alpha     float64 // EWMA smoothing factor (default 0.3 = heavy recent weight)

	values []float64 // circular buffer of last Window values
	idx    int       // next write position
	full   bool      // true once we've seen >= Window samples

	// Running statistics (Welford-style for stability).
	mean  float64
	m2    float64 // sum of (x - mean)^2
	ewma  float64 // exponentially-weighted moving average
	count int     // total samples seen
}

// NewDetector returns a Detector with the given window and threshold.
// Window must be > 1; threshold defaults to 3.0 if <= 0.
func NewDetector(window int, threshold float64) *Detector {
	if window < 2 {
		window = 20
	}
	if threshold <= 0 {
		threshold = 3.0
	}
	return &Detector{
		Window:    window,
		Threshold: threshold,
		Alpha:     0.3,
		values:    make([]float64, window),
	}
}

// Update feeds a new value into the detector. Returns:
//   - zscore: standard deviations from the running mean (NaN until enough samples)
//   - isAnomaly: true if |zscore| >= threshold
//
// The detector tracks the running mean/variance using Welford's algorithm,
// which is numerically stable for streaming data. The EWMA is a smoother
// version that weights recent samples more heavily.
func (d *Detector) Update(v float64) (zscore float64, isAnomaly bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Update running stats via Welford
	d.count++
	delta := v - d.mean
	d.mean += delta / float64(d.count)
	delta2 := v - d.mean
	d.m2 += delta * delta2

	// Update EWMA
	if d.count == 1 {
		d.ewma = v
	} else {
		d.ewma = d.Alpha*v + (1-d.Alpha)*d.ewma
	}

	// Push into circular buffer
	d.values[d.idx] = v
	d.idx = (d.idx + 1) % d.Window
	if d.idx == 0 {
		d.full = true
	}

	// Need at least 2 samples for a variance
	if d.count < 2 {
		return math.NaN(), false
	}
	variance := d.m2 / float64(d.count-1)
	if variance <= 0 {
		// All samples identical — only anomaly if this one is wildly different
		if math.Abs(v-d.mean) > 0 {
			return math.Inf(int(d.Threshold) + 1), true
		}
		return 0, false
	}
	stddev := math.Sqrt(variance)
	zscore = (v - d.mean) / stddev
	return zscore, math.Abs(zscore) >= d.Threshold
}

// Mean returns the current running mean.
func (d *Detector) Mean() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.mean
}

// Stddev returns the current running stddev (0 if <2 samples).
func (d *Detector) Stddev() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.count < 2 {
		return 0
	}
	return math.Sqrt(d.m2 / float64(d.count-1))
}

// EWMA returns the exponentially-weighted moving average.
func (d *Detector) EWMA() float64 {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.ewma
}

// Count returns the total number of samples seen so far.
func (d *Detector) Count() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.count
}

// IsWarmedUp returns true once we've seen >= Window samples (so the
// running mean/stddev are stable).
func (d *Detector) IsWarmedUp() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.full
}

// IsAnomalyValue is a stateless helper for one-off checks: given a
// slice of values (oldest first) and a new value, return the z-score
// and whether it's an anomaly.
//
// Useful for testing or simple synchronous detection without a Detector.
func IsAnomalyValue(values []float64, newVal float64, threshold float64) (float64, bool) {
	if len(values) < 2 {
		return math.NaN(), false
	}
	var sum, m2 float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	for _, v := range values {
		d := v - mean
		m2 += d * d
	}
	variance := m2 / float64(len(values)-1)
	if variance <= 0 {
		return math.Inf(1), true
	}
	stddev := math.Sqrt(variance)
	z := (newVal - mean) / stddev
	return z, math.Abs(z) >= threshold
}
