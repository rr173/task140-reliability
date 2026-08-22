// Package fdist fits life distributions to failure-event logs: the exponential
// distribution (lambda = 1/MTTF, MLE) and the 2-parameter Weibull (shape beta,
// scale eta) via median-rank regression on the Benard plotting position. It also
// computes steady-state availability from the event log (uptime/total).
package fdist

import (
	"fmt"
	"math"
	"sort"

	"task140-reliability/internal/domain"
)

// FitExponential fits lambda = n/sum(ttf) (MLE of the exponential rate), MTTF =
// mean of ttf. Returns the FitResult (Availability filled from event log).
func FitExponential(events []domain.FailureEvent) (*domain.FitResult, error) {
	ttfs := extractTTFs(events)
	if len(ttfs) < 1 {
		return nil, fmt.Errorf("%w: need >=1 failure event", domain.ErrInsufficientData)
	}
	sum := 0.0
	for _, t := range ttfs {
		sum += float64(t)
	}
	mttf := sum / float64(len(ttfs))
	lambda := 1.0 / mttf
	avail := availabilityFromEvents(events)
	return &domain.FitResult{
		Distribution:       domain.DistExponential,
		Lambda:             lambda,
		MTTF:               mttf,
		Availability:       avail,
		SampleSize:         len(ttfs),
		Method:             "exponential_mle",
		MissionReliability: math.Exp(-lambda * 0), // filled by caller with mission time
	}, nil
}

// FitWeibull fits the 2-parameter Weibull by median-rank (Benard) regression:
// F_i = (i - 0.3)/(n + 0.4); regress y = ln(-ln(1-F)) on x = ln(t). Slope = beta;
// intercept = -beta*ln(eta) => eta = exp(-intercept/beta). MTTF = eta*Gamma(1+1/beta).
// Requires >=2 distinct failure times.
func FitWeibull(events []domain.FailureEvent) (*domain.FitResult, error) {
	ttfs := extractTTFs(events)
	if len(ttfs) < 2 {
		return nil, fmt.Errorf("%w: weibull fit needs >=2 failure events", domain.ErrInsufficientData)
	}
	sorted := append([]float64{}, ttfs...)
	sort.Float64s(sorted)
	n := len(sorted)
	// dedup? Weibull rank regression tolerates duplicates; keep all.
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i, t := range sorted {
		F := (float64(i+1) - 0.3) / (float64(n) + 0.4)
		if F >= 1 {
			F = 0.999999
		}
		if F <= 0 {
			F = 1e-9
		}
		xs[i] = math.Log(t)
		ys[i] = math.Log(-math.Log(1 - F))
	}
	beta, intercept := linearRegression(xs, ys)
	if beta <= 0 {
		// degenerate; fall back gracefully
		return nil, fmt.Errorf("%w: weibull beta non-positive", domain.ErrInsufficientData)
	}
	eta := math.Exp(-intercept / beta)
	mttf := eta * math.Gamma(1+1/beta)
	avail := availabilityFromEvents(events)
	res := &domain.FitResult{
		Distribution: domain.DistWeibull,
		Beta:         beta,
		Eta:          eta,
		MTTF:         mttf,
		Availability: avail,
		SampleSize:   n,
		Method:       "weibull_median_rank",
	}
	if beta < 1 {
		res.Warnings = append(res.Warnings, "decreasing_failure_rate: weibull beta < 1")
	}
	return res, nil
}

// FitAuto chooses Weibull when >=2 events, else exponential.
func FitAuto(events []domain.FailureEvent) (*domain.FitResult, error) {
	if len(events) < 2 {
		return FitExponential(events)
	}
	r, err := FitWeibull(events)
	if err != nil {
		// fall back to exponential if weibull fails
		return FitExponential(events)
	}
	return r, nil
}

// MissionReliability computes R(t) for the given mission time under a fit.
// Exponential: R = exp(-lambda*t). Weibull: R = exp(-(t/eta)^beta).
func MissionReliability(f *domain.FitResult, hours int64) float64 {
	t := float64(hours)
	if t < 0 {
		t = 0
	}
	switch f.Distribution {
	case domain.DistWeibull:
		return math.Exp(-math.Pow(t/f.Eta, f.Beta))
	default:
		return math.Exp(-f.Lambda * t)
	}
}

// availabilityFromEvents computes A = uptime/(uptime+downtime) from the event
// log. uptime = sum of ttf; downtime = sum of ttr.
func availabilityFromEvents(events []domain.FailureEvent) float64 {
	up := 0.0
	down := 0.0
	for _, e := range events {
		up += float64(e.TTFHours)
		down += float64(e.TTRHours)
	}
	total := up + down
	if total <= 0 {
		return 0
	}
	return up / total
}

// extractTTFs returns the non-negative ttf values as floats.
func extractTTFs(events []domain.FailureEvent) []float64 {
	out := make([]float64, 0, len(events))
	for _, e := range events {
		if e.TTFHours < 0 {
			continue
		}
		out = append(out, float64(e.TTFHours))
	}
	return out
}

// linearRegression returns (slope, intercept) of y on x (ordinary least
// squares). If all x equal, returns slope 0.
func linearRegression(xs, ys []float64) (float64, float64) {
	n := float64(len(xs))
	if n == 0 {
		return 0, 0
	}
	sx, sy, sxx, sxy := 0.0, 0.0, 0.0, 0.0
	for i := range xs {
		sx += xs[i]
		sy += ys[i]
		sxx += xs[i] * xs[i]
		sxy += xs[i] * ys[i]
	}
	denom := n*sxx - sx*sx
	if denom == 0 {
		return 0, sy / n
	}
	slope := (n*sxy - sx*sy) / denom
	intercept := (sy - slope*sx) / n
	return slope, intercept
}

// QFromLambdaT returns the failure probability q = 1 - exp(-lambda*t) for a
// given hourly failure rate and mission time. Used by FTA when an event
// derives its probability from a component's fitted rate.
func QFromLambdaT(lambda float64, hours int64) float64 {
	if lambda <= 0 || hours <= 0 {
		return 0
	}
	return 1 - math.Exp(-lambda*float64(hours))
}
