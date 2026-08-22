package service

import (
    "context"
    "testing"

    "task140-reliability/internal/domain"
    "task140-reliability/internal/store"
)

func task140ClassService(t *testing.T) (context.Context, *Service) {
    t.Helper()
    st, err := store.Open(t.TempDir() + "/reliability.db")
    if err != nil { t.Fatal(err) }
    t.Cleanup(func() { _ = st.Close() })
    return context.Background(), New(st)
}

func TestBug09_ResolvedAnalysisRemainsEligibleForRecalculation(t *testing.T) {
    ctx, svc := task140ClassService(t)
    analysis, err := svc.CreateAnalysis(ctx, "actuator", "loss of control", nil)
    if err != nil { t.Fatal(err) }
    if _, err = svc.AddFMEARow(ctx, domain.FMEARow{AnalysisID: analysis.ID, Function: "actuator", FailureMode: "jam", Severity: 7, Occurrence: 2, Detection: 3}); err != nil { t.Fatal(err) }
    if _, err = svc.Solve(ctx, analysis.ID); err != nil { t.Fatal(err) }
    current, err := svc.GetAnalysis(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if current.State != domain.StateAnalyzed { t.Fatalf("state after solve=%s, want analyzed", current.State) }
    if _, err = svc.Solve(ctx, analysis.ID); err != nil { t.Fatalf("second solve rejected: %v", err) }
}
