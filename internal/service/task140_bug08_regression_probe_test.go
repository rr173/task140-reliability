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

func TestBug08_MissionDurationFlowsIntoFailureDistribution(t *testing.T) {
    ctx, svc := task140ClassService(t)
    params := domain.Analysis{MissionHours: 1000}
    analysis, err := svc.CreateAnalysis(ctx, "bearing", "wearout", &params)
    if err != nil { t.Fatal(err) }
    if _, err = svc.CreateFailureEvent(ctx, domain.FailureEvent{AnalysisID: analysis.ID, TTFHours: 1000}); err != nil { t.Fatal(err) }
    result, err := svc.Solve(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if result.Fit.MissionReliability < 0.3677 || result.Fit.MissionReliability > 0.3680 { t.Fatalf("mission reliability=%0.9f, want exp(-1)", result.Fit.MissionReliability) }
}
