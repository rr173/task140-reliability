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

func TestBug06_ExactProbabilityLimitIsPreservedAcrossAnalysisLayers(t *testing.T) {
    ctx, svc := task140ClassService(t)
    params := domain.Analysis{ExactLimit: 1}
    analysis, err := svc.CreateAnalysis(ctx, "sensor", "false shutdown", &params)
    if err != nil { t.Fatal(err) }
    tree := domain.FTATree{AnalysisID: analysis.ID, TopGateID: "TOP", Gates: []domain.Gate{{ID: "TOP", AnalysisID: analysis.ID, Type: domain.GateOR, Inputs: []domain.GateInput{{EventID: "A"}}}}, Events: []domain.BasicEvent{{ID: "A", AnalysisID: analysis.ID, ProbMicro: 100000}}}
    if _, err = svc.SaveFTATree(ctx, tree); err != nil { t.Fatal(err) }
    result, err := svc.Solve(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if result.FTA.Method != "exact" { t.Fatalf("method=%s, want exact", result.FTA.Method) }
}
