// Tier 8 Phase 2 — Predictive Alerting (Tier 8.2). OLS linear
// regression math, lifted out of the HTTP handler so it can be
// tested independently and so handlers_predict_forecast.go stays
// under the 400-LOC cap.
//
// forecastSample is the bucketed input to olsFit — one hourly
// median of metric_points (oldest first).
//
// olsFit runs ordinary-least-squares linear regression over a
// slice of forecastSamples where x = hours since the first sample.
// Returns slope, intercept, residual standard deviation, and MAPE
// (Mean Absolute Percentage Error) over the training set.
//
// Pure stdlib math; intentionally kept simple. We deliberately do
// NOT include any iterative solver or matrix library because the
// spec bans new dependencies and 2-variable OLS is closed-form:
//
//	b = (Σy - m·Σx) / n
//	m = (n·Σxy - Σx·Σy) / (n·Σx² - (Σx)²)
//
// MAPE is computed against the held-out last point (we fit on the
// first n-1 and compute the percentage error on the n-th) which is
// what makes the metric a useful "how good was this forecast"
// signal rather than a tautological in-sample fit score.
package handler

import (
	"math"
	"sort"
	"time"
)

// forecastSample is the bucketed input to olsFit — one hourly
// median of metric_points (oldest first). Lifted to a named type
// (vs. anonymous struct) so olsFit can declare it in its signature
// and call sites can reuse it.
type forecastSample struct {
	ts  time.Time
	val float64
}

// olsFit runs ordinary-least-squares linear regression over the
// input samples. See the package-level doc above for the math
// rationale.
func olsFit(samples []forecastSample) (slope, intercept, residStddev, mape float64) {
	n := len(samples)
	if n < 2 {
		return 0, 0, 0, 100
	}
	// Stable sort by ts just in case the SQL ORDER BY somehow
	// delivered out-of-order rows (defense in depth).
	sort.SliceStable(samples, func(i, j int) bool {
		return samples[i].ts.Before(samples[j].ts)
	})
	start := samples[0].ts
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i, s := range samples {
		xs[i] = s.ts.Sub(start).Hours()
		ys[i] = s.val
	}
	var sumX, sumY, sumXY, sumXX float64
	for i := 0; i < n; i++ {
		sumX += xs[i]
		sumY += ys[i]
		sumXY += xs[i] * ys[i]
		sumXX += xs[i] * xs[i]
	}
	denom := float64(n)*sumXX - sumX*sumX
	if denom == 0 {
		// All samples share the same timestamp — flat line.
		// Predict the mean with zero slope.
		mean := sumY / float64(n)
		return 0, mean, 0, 0
	}
	slope = (float64(n)*sumXY - sumX*sumY) / denom
	intercept = (sumY - slope*sumX) / float64(n)

	// Residual standard deviation across the WHOLE series (in-sample
	// residual stddev; used to size the p10/p90 band).
	var sse float64
	for i := 0; i < n; i++ {
		pred := slope*xs[i] + intercept
		d := ys[i] - pred
		sse += d * d
	}
	if n > 2 {
		residStddev = math.Sqrt(sse / float64(n-2))
	} else {
		residStddev = math.Sqrt(sse)
	}

	// MAPE: held-out evaluation on the last point. Fit on the first
	// n-1 points, compute |actual - predicted| / |actual| on the n-th.
	if n >= 4 {
		// Re-fit on first n-1.
		var sumX2, sumY2, sumXY2, sumX2sq float64
		n1 := n - 1
		for i := 0; i < n1; i++ {
			sumX2 += xs[i]
			sumY2 += ys[i]
			sumXY2 += xs[i] * ys[i]
			sumX2sq += xs[i] * xs[i]
		}
		denom1 := float64(n1)*sumX2sq - sumX2*sumX2
		var m1, b1 float64
		if denom1 != 0 {
			m1 = (float64(n1)*sumXY2 - sumX2*sumY2) / denom1
			b1 = (sumY2 - m1*sumX2) / float64(n1)
		} else {
			m1 = 0
			b1 = sumY2 / float64(n1)
		}
		predLast := m1*xs[n-1] + b1
		actualLast := ys[n-1]
		if math.Abs(actualLast) > 0 {
			mape = math.Abs(predLast-actualLast) / math.Abs(actualLast) * 100.0
		} else {
			mape = 0
		}
	} else {
		// Not enough samples for a held-out evaluation — return
		// in-sample MAPE (residual / |y|) as a fallback.
		mape = 0
		for i := 0; i < n; i++ {
			pred := slope*xs[i] + intercept
			if math.Abs(ys[i]) > 0 {
				mape += math.Abs(pred-ys[i]) / math.Abs(ys[i])
			}
		}
		mape = mape / float64(n) * 100.0
	}
	if math.IsNaN(mape) || math.IsInf(mape, 0) {
		mape = 100
	}
	return slope, intercept, residStddev, mape
}
