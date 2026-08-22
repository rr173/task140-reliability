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

func TestBug04_MediumRPNBoundarySurvivesEveryFMEAPath(t *testing.T) {
    ctx, svc := task140ClassService(t)
    analysis, err := svc.CreateAnalysis(ctx, "motor", "loss of torque", nil)
    if err != nil { t.Fatal(err) }
    row, err := svc.AddFMEARow(ctx, domain.FMEARow{AnalysisID: analysis.ID, Function: "motor", FailureMode: "overheat", Severity: 5, Occurrence: 5, Detection: 4})
    if err != nil { t.Fatal(err) }
    if row.RiskClass != domain.RiskMedium { t.Fatalf("created class=%s, want medium", row.RiskClass) }
    table, err := svc.LoadFMEATable(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if len(table.Rows) != 1 || table.Rows[0].RiskClass != domain.RiskMedium { t.Fatalf("loaded class=%s, want medium", table.Rows[0].RiskClass) }
    result, err := svc.Solve(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if result.FMEA.MediumCount != 1 { t.Fatalf("solved medium count=%d, want 1", result.FMEA.MediumCount) }
}
