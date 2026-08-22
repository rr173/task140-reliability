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

func TestBug07_MissionDurationFlowsIntoRBDReliability(t *testing.T) {
    ctx, svc := task140ClassService(t)
    params := domain.Analysis{MissionHours: 1000}
    analysis, err := svc.CreateAnalysis(ctx, "pump train", "loss of flow", &params)
    if err != nil { t.Fatal(err) }
    diagram := domain.RBDDiagram{AnalysisID: analysis.ID, RootID: "P", Blocks: []domain.RBDBlock{{ID: "P", AnalysisID: analysis.ID, Type: domain.BlockBasic, FailureRatePPT: 1000000000}}}
    if _, err = svc.SaveRBDDiagram(ctx, diagram); err != nil { t.Fatal(err) }
    result, err := svc.Solve(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if result.RBD.RootReliability < 0.3677 || result.RBD.RootReliability > 0.3680 { t.Fatalf("mission reliability=%0.9f, want exp(-1)", result.RBD.RootReliability) }
}
