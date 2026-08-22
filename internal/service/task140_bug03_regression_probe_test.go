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

func TestBug03_HighRPNBoundarySurvivesEveryFMEAPath(t *testing.T) {
    ctx, svc := task140ClassService(t)
    analysis, err := svc.CreateAnalysis(ctx, "pump", "loss of flow", nil)
    if err != nil { t.Fatal(err) }
    row, err := svc.AddFMEARow(ctx, domain.FMEARow{AnalysisID: analysis.ID, Function: "pump", FailureMode: "seal leak", Severity: 5, Occurrence: 10, Detection: 4})
    if err != nil { t.Fatal(err) }
    if row.RiskClass != domain.RiskHigh { t.Fatalf("created class=%s, want high", row.RiskClass) }
    table, err := svc.LoadFMEATable(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if len(table.Rows) != 1 || table.Rows[0].RiskClass != domain.RiskHigh { t.Fatalf("loaded class=%s, want high", table.Rows[0].RiskClass) }
    result, err := svc.Solve(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if result.FMEA.HighCount != 1 { t.Fatalf("solved high count=%d, want 1", result.FMEA.HighCount) }
}
