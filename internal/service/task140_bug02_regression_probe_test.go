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

func TestBug02_SeverityFloorPersistsAcrossCreateLoadAndSolve(t *testing.T) {
    ctx, svc := task140ClassService(t)
    params := domain.Analysis{SCrit: 8, RPNHigh: 200, RPNMedium: 100}
    analysis, err := svc.CreateAnalysis(ctx, "valve", "loss of containment", &params)
    if err != nil { t.Fatal(err) }
    row, err := svc.AddFMEARow(ctx, domain.FMEARow{AnalysisID: analysis.ID, Function: "valve", FailureMode: "stuck", Severity: 8, Occurrence: 1, Detection: 1})
    if err != nil { t.Fatal(err) }
    if row.RiskClass != domain.RiskHigh { t.Fatalf("created class=%s, want high", row.RiskClass) }
    table, err := svc.LoadFMEATable(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if len(table.Rows) != 1 || table.Rows[0].RiskClass != domain.RiskHigh { t.Fatalf("loaded class=%s, want high", table.Rows[0].RiskClass) }
    result, err := svc.Solve(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if result.FMEA.HighCount != 1 { t.Fatalf("solved high count=%d, want 1", result.FMEA.HighCount) }
}
