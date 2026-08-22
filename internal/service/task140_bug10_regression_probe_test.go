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

func TestBug10_BaselinedAnalysisCreatesTheNextRevision(t *testing.T) {
    ctx, svc := task140ClassService(t)
    analysis, err := svc.CreateAnalysis(ctx, "cooling loop", "thermal trip", nil)
    if err != nil { t.Fatal(err) }
    if _, err = svc.AddFMEARow(ctx, domain.FMEARow{AnalysisID: analysis.ID, Function: "cooler", FailureMode: "fan loss", Severity: 7, Occurrence: 2, Detection: 3}); err != nil { t.Fatal(err) }
    if _, err = svc.Solve(ctx, analysis.ID); err != nil { t.Fatal(err) }
    if _, err = svc.Review(ctx, analysis.ID); err != nil { t.Fatal(err) }
    if _, err = svc.Baseline(ctx, analysis.ID); err != nil { t.Fatal(err) }
    revised, err := svc.Revise(ctx, analysis.ID)
    if err != nil { t.Fatalf("revise failed: %v", err) }
    if revised.State != domain.StateDraft || revised.Version != 2 { t.Fatalf("revision state=%s version=%d, want draft/2", revised.State, revised.Version) }
}
