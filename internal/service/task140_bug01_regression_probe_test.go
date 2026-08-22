package service

import (
    "context"
    "testing"

    "task140-reliability/internal/domain"
    "task140-reliability/internal/store"
)

func TestBug01_FMEARPNStaysConsistentAcrossCreateUpdateAndSolve(t *testing.T) {
    ctx := context.Background()
    st, err := store.Open(t.TempDir() + "/reliability.db")
    if err != nil { t.Fatal(err) }
    t.Cleanup(func() { _ = st.Close() })
    svc := New(st)
    analysis, err := svc.CreateAnalysis(ctx, "pump", "loss of flow", nil)
    if err != nil { t.Fatal(err) }
    row, err := svc.AddFMEARow(ctx, domain.FMEARow{AnalysisID: analysis.ID, Function: "pump", FailureMode: "seal leak", Severity: 8, Occurrence: 5, Detection: 3})
    if err != nil { t.Fatal(err) }
    if row.RPN != 120 { t.Fatalf("new row RPN=%d, want 120", row.RPN) }
    severity := 10
    row, err = svc.UpdateFMEARow(ctx, row.ID, nil, nil, nil, &severity, nil, nil, nil, nil)
    if err != nil { t.Fatal(err) }
    if row.RPN != 150 { t.Fatalf("updated row RPN=%d, want 150", row.RPN) }
    result, err := svc.Solve(ctx, analysis.ID)
    if err != nil { t.Fatal(err) }
    if len(result.FMEA.Rows) != 1 || result.FMEA.Rows[0].RPN != 150 { t.Fatalf("solved RPN=%d, want 150", result.FMEA.Rows[0].RPN) }
}
