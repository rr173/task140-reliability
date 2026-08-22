package service

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"task140-reliability/internal/domain"
	"task140-reliability/internal/store"
)

func numEqF(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func newSvc(t *testing.T) (*Service, func()) {
	t.Helper()
	s, err := store.Open(t.TempDir() + "/rel.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return New(s), func() { _ = s.Close() }
}

func mkAnalysis(id string) domain.Analysis {
	a := domain.DefaultAnalysisParams()
	a.ID = id
	a.Name = id
	a.TopEvent = "top"
	a.State = domain.StateDraft
	a.Version = 1
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	return a
}

// A single-event OR tree makes the top probability exactly the event's
// probability, so changing the event probability between solves is observable
// in the refreshed result.
func oneEventTree(aid string, probMicro int64) domain.FTATree {
	return domain.FTATree{
		AnalysisID: aid, TopGateID: "TOP",
		Gates: []domain.Gate{
			{ID: "TOP", AnalysisID: aid, Type: domain.GateOR, IsTop: true, Inputs: []domain.GateInput{{EventID: "A"}}},
		},
		Events: []domain.BasicEvent{{ID: "A", AnalysisID: aid, ProbMicro: domain.ProbMicro(probMicro)}},
	}
}

func svcSetupTree(t *testing.T, svc *Service, aid string, probMicro int64) {
	t.Helper()
	if err := svc.store.CreateAnalysis(context.Background(), mkAnalysis(aid)); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.SaveFTATree(context.Background(), oneEventTree(aid, probMicro)); err != nil {
		t.Fatalf("save tree: %v", err)
	}
}

// topProb extracts the FTA top probability from a solved result.
func topProb(t *testing.T, res *domain.AnalysisResult) float64 {
	t.Helper()
	if res == nil || res.FTA == nil {
		t.Fatalf("nil result/fta: %+v", res)
	}
	return res.FTA.TopProbability
}

// TestSolveTransitionToAnalyzed locks in that solving a draft analysis
// transitions it to "analyzed" (the core lifecycle invariant that was broken).
func TestSolveTransitionToAnalyzed(t *testing.T) {
	svc, cleanup := newSvc(t)
	defer cleanup()
	ctx := context.Background()
	svcSetupTree(t, svc, "a1", 100_000)

	res, err := svc.Solve(ctx, "a1")
	if err != nil {
		t.Fatalf("first solve: %v", err)
	}
	if !numEqF(topProb(t, res), 0.1) {
		t.Fatalf("top probability = %v, want 0.1", res.FTA.TopProbability)
	}
	a, err := svc.GetAnalysis(ctx, "a1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if a.State != domain.StateAnalyzed {
		t.Fatalf("after solve state = %s, want analyzed", a.State)
	}
	if a.Version != 1 {
		t.Fatalf("version = %d, want 1", a.Version)
	}
}

// TestReSolveFromAnalyzedRefreshes is the regression for the reported bug:
// once an analysis is in the analyzed state, solving again must be accepted
// (not rejected as a state conflict) and must refresh the persisted result when
// the inputs change — the state machine, service orchestration and state
// write must all cooperate so the second analysis is not wrongly refused.
func TestReSolveFromAnalyzedRefreshes(t *testing.T) {
	svc, cleanup := newSvc(t)
	defer cleanup()
	ctx := context.Background()
	svcSetupTree(t, svc, "a1", 100_000)

	if _, err := svc.Solve(ctx, "a1"); err != nil {
		t.Fatalf("first solve: %v", err)
	}

	// Second solve on the already-analyzed analysis must NOT be rejected.
	if _, err := svc.Solve(ctx, "a1"); err != nil {
		t.Fatalf("re-solve from analyzed: %v", err)
	}
	if a, _ := svc.GetAnalysis(ctx, "a1"); a.State != domain.StateAnalyzed {
		t.Fatalf("after re-solve state = %s, want analyzed", a.State)
	}

	// Change the input (still editable — analysis is analyzed, not baselined)
	// and re-solve: the persisted result must reflect the new probability.
	if _, err := svc.SaveFTATree(ctx, oneEventTree("a1", 200_000)); err != nil {
		t.Fatalf("update tree: %v", err)
	}
	res3, err := svc.Solve(ctx, "a1")
	if err != nil {
		t.Fatalf("re-solve after input change: %v", err)
	}
	if !numEqF(topProb(t, res3), 0.2) {
		t.Fatalf("after input change top probability = %v, want 0.2", res3.FTA.TopProbability)
	}
	// The persisted result (loaded fresh) must match the refreshed solve.
	persisted, err := svc.GetResult(ctx, "a1")
	if err != nil {
		t.Fatalf("load persisted result: %v", err)
	}
	if !numEqF(topProb(t, persisted), 0.2) {
		t.Fatalf("persisted result not refreshed = %v, want 0.2", persisted.FTA.TopProbability)
	}

	// Reviewed/baselined analyses are locked: re-solve is rejected there.
	a, _ := svc.GetAnalysis(ctx, "a1")
	a.State = domain.StateReviewed
	if err := svc.store.SetAnalysisState(ctx, "a1", domain.StateReviewed, a.Version, nil); err != nil {
		t.Fatalf("set reviewed: %v", err)
	}
	if _, err := svc.Solve(ctx, "a1"); !errors.Is(err, domain.ErrStateConflict) {
		t.Fatalf("re-solve from reviewed: err = %v, want ErrStateConflict", err)
	}
}

// TestReconcilePreservesAnalyzedState verifies the restart path re-solves an
// already-analyzed analysis without disturbing its state or version.
func TestReconcilePreservesAnalyzedState(t *testing.T) {
	svc, cleanup := newSvc(t)
	defer cleanup()
	ctx := context.Background()
	svcSetupTree(t, svc, "a1", 100_000)

	if _, err := svc.Solve(ctx, "a1"); err != nil {
		t.Fatalf("first solve: %v", err)
	}
	if a, _ := svc.GetAnalysis(ctx, "a1"); a.State != domain.StateAnalyzed {
		t.Fatalf("precondition: state = %s, want analyzed", a.State)
	}

	n, err := svc.ReconcileAll(ctx)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if n != 1 {
		t.Fatalf("reconciled = %d, want 1", n)
	}
	a, err := svc.GetAnalysis(ctx, "a1")
	if err != nil {
		t.Fatalf("get after reconcile: %v", err)
	}
	if a.State != domain.StateAnalyzed {
		t.Fatalf("after reconcile state = %s, want analyzed", a.State)
	}
	if a.Version != 1 {
		t.Fatalf("after reconcile version = %d, want 1", a.Version)
	}
	persisted, err := svc.GetResult(ctx, "a1")
	if err != nil {
		t.Fatalf("load persisted result: %v", err)
	}
	if !numEqF(topProb(t, persisted), 0.1) {
		t.Fatalf("reconciled result = %v, want 0.1", persisted.FTA.TopProbability)
	}
}
