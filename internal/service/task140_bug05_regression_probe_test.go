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

func TestBug05_ConfiguredCutSetOrderIsHonoredEndToEnd(t *testing.T) {
    ctx, svc := task140ClassService(t)
    params := domain.Analysis{MaxOrder: 1}
    analysis, err := svc.CreateAnalysis(ctx, "compressor", "loss of pressure", &params)
    if err != nil { t.Fatal(err) }
    if analysis.MaxOrder != 1 { t.Fatalf("created max order=%d, want 1", analysis.MaxOrder) }
    tree := domain.FTATree{AnalysisID: analysis.ID, TopGateID: "TOP", Gates: []domain.Gate{{ID: "TOP", AnalysisID: analysis.ID, Type: domain.GateAND, Inputs: []domain.GateInput{{EventID: "A"}, {EventID: "B"}, {EventID: "C"}, {EventID: "D"}}}}, Events: []domain.BasicEvent{{ID: "A", AnalysisID: analysis.ID, ProbMicro: 100000}, {ID: "B", AnalysisID: analysis.ID, ProbMicro: 100000}, {ID: "C", AnalysisID: analysis.ID, ProbMicro: 100000}, {ID: "D", AnalysisID: analysis.ID, ProbMicro: 100000}}}
    if _, err = svc.SaveFTATree(ctx, tree); err != nil { t.Fatal(err) }
    persisted, err := svc.GetAnalysis(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if persisted.MaxOrder != 1 { t.Fatalf("persisted max order=%d, want 1", persisted.MaxOrder) }
    result, err := svc.Solve(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if result.FTA.TruncatedCount != 1 || len(result.FTA.CutSets) != 0 { t.Fatalf("truncated=%d cutsets=%+v, want 1 and 0", result.FTA.TruncatedCount, result.FTA.CutSets) }
}
