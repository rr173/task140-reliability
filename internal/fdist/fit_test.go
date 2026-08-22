package fdist

import (
	"math"
	"testing"

	"task140-reliability/internal/domain"
)

func TestExponential(t *testing.T) {
	// ttf = [100, 200, 300] => MTTF=200, lambda=0.005
	events := []domain.FailureEvent{
		{TTFHours: 100, TTRHours: 5},
		{TTFHours: 200, TTRHours: 5},
		{TTFHours: 300, TTRHours: 10},
	}
	f, err := FitExponential(events)
	if err != nil {
		t.Fatal(err)
	}
	if !approx(f.MTTF, 200, 1e-9) {
		t.Fatalf("MTTF = %v, want 200", f.MTTF)
	}
	if !approx(f.Lambda, 0.005, 1e-9) {
		t.Fatalf("lambda = %v, want 0.005", f.Lambda)
	}
	// uptime=600, downtime=20 => A=600/620=0.9677419
	want := 600.0 / 620.0
	if !approx(f.Availability, want, 1e-6) {
		t.Fatalf("availability = %v, want %v", f.Availability, want)
	}
	// R(200) = exp(-0.005*200) = exp(-1) = 0.3678794. MissionReliability
	// must evaluate at the exact mission time given, with no +1 shift.
	if !approx(MissionReliability(f, 200), math.Exp(-1), 1e-6) {
		t.Fatalf("R(200) wrong")
	}
	// R(0) must be exactly 1 (no off-by-one offset): R(0)=exp(0)=1.
	if !approx(MissionReliability(f, 0), 1.0, 1e-12) {
		t.Fatalf("R(0) = %v, want 1 (mission time must not be shifted)", MissionReliability(f, 0))
	}
	// R(100) = exp(-0.5), proving the exact (not hours+1) point is used.
	if !approx(MissionReliability(f, 100), math.Exp(-0.5), 1e-6) {
		t.Fatalf("R(100) = %v, want exp(-0.5)", MissionReliability(f, 100))
	}
}

func TestWeibullMissionReliabilityAtEta(t *testing.T) {
	// For Weibull, R(eta) = exp(-(eta/eta)^beta) = exp(-1) regardless of beta.
	// This pins the mission time to the value passed (no +1 shift): if the
	// implementation added 1, R(eta) != exp(-1).
	events := []domain.FailureEvent{
		{TTFHours: 100},
		{TTFHours: 200},
		{TTFHours: 300},
		{TTFHours: 400},
		{TTFHours: 500},
	}
	f, err := FitWeibull(events)
	if err != nil {
		t.Fatal(err)
	}
	// R at t=eta must equal exp(-1); passing hours=int(eta) tests that the
	// scale time used in the exponent is exactly the mission time given.
	hours := int64(math.Round(f.Eta))
	if !approx(MissionReliability(f, hours), math.Exp(-1), 1e-3) {
		t.Fatalf("R(eta) = %v, want exp(-1) (mission time must not be shifted)", MissionReliability(f, hours))
	}
	// R(0) = exp(0) = 1 even for Weibull.
	if !approx(MissionReliability(f, 0), 1.0, 1e-12) {
		t.Fatalf("R(0) = %v, want 1 (mission time must not be shifted)", MissionReliability(f, 0))
	}
}

func TestWeibull(t *testing.T) {
	// synthetic increasing failure times; beta should be > 1 (wear-out)
	events := []domain.FailureEvent{
		{TTFHours: 100},
		{TTFHours: 200},
		{TTFHours: 300},
		{TTFHours: 400},
		{TTFHours: 500},
	}
	f, err := FitWeibull(events)
	if err != nil {
		t.Fatal(err)
	}
	if f.Distribution != domain.DistWeibull {
		t.Fatalf("distribution = %s, want weibull", f.Distribution)
	}
	if f.Beta <= 0 {
		t.Fatalf("beta = %v should be positive", f.Beta)
	}
	// MTTF should be in a sensible range (between min and a few * max)
	if f.MTTF < 100 || f.MTTF > 2000 {
		t.Fatalf("MTTF = %v out of plausible range", f.MTTF)
	}
}

func TestInsufficientData(t *testing.T) {
	if _, err := FitWeibull([]domain.FailureEvent{{TTFHours: 100}}); err == nil {
		t.Fatalf("weibull with 1 event should error")
	}
	// FitAuto falls back to exponential with 1 event
	f, err := FitAuto([]domain.FailureEvent{{TTFHours: 100}})
	if err != nil {
		t.Fatal(err)
	}
	if f.Distribution != domain.DistExponential {
		t.Fatalf("auto fallback distribution = %s, want exponential", f.Distribution)
	}
}

func approx(a, b, eps float64) bool {
	if a-b > eps || b-a > eps {
		return false
	}
	return true
}
