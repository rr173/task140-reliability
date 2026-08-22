package analysis

import (
	"context"
	"math"
	"testing"

	"task140-reliability/internal/domain"
)

func TestNextState(t *testing.T) {
	cases := []struct {
		cur, want domain.AnalysisState
		ok        bool
	}{
		{domain.StateDraft, domain.StateAnalyzed, true},
		{domain.StateAnalyzed, domain.StateAnalyzed, true}, // re-solve allowed
		{domain.StateReviewed, domain.StateAnalyzed, false},
		{domain.StateAnalyzed, domain.StateReviewed, true},
		{domain.StateDraft, domain.StateReviewed, false},
		{domain.StateReviewed, domain.StateBaselined, true},
		{domain.StateDraft, domain.StateBaselined, false},
		{domain.StateBaselined, domain.StateDraft, true}, // revise
		{domain.StateAnalyzed, domain.StateDraft, false},
	}
	for _, c := range cases {
		_, err := NextState(c.cur, c.want)
		if c.ok && err != nil {
			t.Fatalf("%s->%s expected ok, got %v", c.cur, c.want, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("%s->%s expected error", c.cur, c.want)
		}
	}
}

// TestSolveFitMissionReliabilityUsesAnalysisMissionHours guards against the
// fault-data-fit mission reliability being evaluated at a fixed mission time
// instead of the analysis's configured MissionHours. With a single event
// ttf=200 => FitAuto uses the exponential fit, lambda=1/200=0.005; R(t) =
// exp(-lambda*t). We pick MissionHours=200 so R is exp(-1)~=0.368 (well away
// from the saturation regions where R(h) and R(h+1) both collapse to 0 or 1).
// R(MissionHours) must equal the distribution's R at exactly MissionHours,
// not at 1 or MissionHours+1.
func TestSolveFitMissionReliabilityUsesAnalysisMissionHours(t *testing.T) {
	events := []domain.FailureEvent{
		{TTFHours: 200, TTRHours: 5},
	}
	const missionHours int64 = 200
	params := domain.DefaultAnalysisParams()
	params.ID = "test"
	params.Version = 1
	params.MissionHours = missionHours

	eng := NewEngine(params, nil, nil, nil, events)
	res, err := eng.Solve(newCtx(t))
	if err != nil {
		t.Fatalf("Solve: %v", err)
	}
	if res.Fit == nil {
		t.Fatal("expected fit result")
	}
	if res.Fit.Distribution != domain.DistExponential {
		t.Fatalf("expected exponential fit, got %s", res.Fit.Distribution)
	}
	// Expected R at the configured mission time, computed with the SAME
	// (unshifted) mission time as the engine must use.
	want := reliabilityAt(res.Fit, missionHours)
	if !approxReliability(res.Fit.MissionReliability, want, 1e-12) {
		t.Fatalf("mission_reliability = %v, want %v (R at MissionHours=%d, not a fixed/shifted time)",
			res.Fit.MissionReliability, want, missionHours)
	}
	// Sanity: it must NOT equal R(1) or R(missionHours+1) — those are the
	// failure modes this test exists to catch.
	if approxReliability(res.Fit.MissionReliability, reliabilityAt(res.Fit, 1), 1e-9) {
		t.Fatalf("mission_reliability equals R(1); engine used a fixed mission time of 1")
	}
	if approxReliability(res.Fit.MissionReliability, reliabilityAt(res.Fit, missionHours+1), 1e-9) {
		t.Fatalf("mission_reliability equals R(missionHours+1); distribution shifted the mission time by one")
	}
}

// reliabilityAt mirrors the fdist R(t) formula without importing fdist, so the
// test independently checks the engine used the right time.
func reliabilityAt(f *domain.FitResult, hours int64) float64 {
	t := float64(hours)
	switch f.Distribution {
	case domain.DistWeibull:
		return math.Exp(-math.Pow(t/f.Eta, f.Beta))
	default:
		return math.Exp(-f.Lambda * t)
	}
}

func approxReliability(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}

// newCtx returns a background context for Solve.
func newCtx(t *testing.T) context.Context {
	t.Helper()
	return context.Background()
}

