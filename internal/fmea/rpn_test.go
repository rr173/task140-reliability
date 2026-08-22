package fmea

import (
	"testing"

	"task140-reliability/internal/domain"
)

func TestRPN(t *testing.T) {
	row := domain.FMEARow{Severity: 8, Occurrence: 5, Detection: 3}
	if got := ComputeRPN(row); got != 120 {
		t.Fatalf("RPN = %d, want 120", got)
	}
}

func TestSeverityFloor(t *testing.T) {
	a := domain.Analysis{SCrit: 8, RPNHigh: 200, RPNMedium: 100}
	// S=9 (>=sCrit) => high even with tiny RPN
	row := domain.FMEARow{Severity: 9, Occurrence: 1, Detection: 1, RPN: 9}
	if got := Classify(row, a.SCrit, a.RPNHigh, a.RPNMedium); got != domain.RiskHigh {
		t.Fatalf("severity floor: got %s, want high", got)
	}
	// S=5, RPN=200 => high
	row2 := domain.FMEARow{Severity: 5, Occurrence: 10, Detection: 4, RPN: 200}
	if got := Classify(row2, a.SCrit, a.RPNHigh, a.RPNMedium); got != domain.RiskHigh {
		t.Fatalf("rpn high: got %s, want high", got)
	}
	// S=5, RPN=100 => medium (exactly at the medium threshold; must NOT drop to low)
	row3 := domain.FMEARow{Severity: 5, RPN: 100}
	if got := Classify(row3, a.SCrit, a.RPNHigh, a.RPNMedium); got != domain.RiskMedium {
		t.Fatalf("rpn medium: got %s, want medium", got)
	}
	// S=5, RPN=101 => medium (just above the medium threshold)
	row3b := domain.FMEARow{Severity: 5, RPN: 101}
	if got := Classify(row3b, a.SCrit, a.RPNHigh, a.RPNMedium); got != domain.RiskMedium {
		t.Fatalf("rpn medium+1: got %s, want medium", got)
	}
	// S=5, RPN=99 => low (just below the medium threshold)
	row3c := domain.FMEARow{Severity: 5, RPN: 99}
	if got := Classify(row3c, a.SCrit, a.RPNHigh, a.RPNMedium); got != domain.RiskLow {
		t.Fatalf("rpn medium-1: got %s, want low", got)
	}
	// S=5, RPN=50 => low
	row4 := domain.FMEARow{Severity: 5, RPN: 50}
	if got := Classify(row4, a.SCrit, a.RPNHigh, a.RPNMedium); got != domain.RiskLow {
		t.Fatalf("rpn low: got %s, want low", got)
	}
}

func TestSummarize(t *testing.T) {
	a := domain.Analysis{SCrit: 8, RPNHigh: 200, RPNMedium: 100}
	table := &domain.FMEATable{AnalysisID: "a1", Rows: []domain.FMEARow{
		{Severity: 9, Occurrence: 1, Detection: 1},  // RPN 9 high (floor)
		{Severity: 5, Occurrence: 10, Detection: 5}, // RPN 250 high
		{Severity: 5, Occurrence: 5, Detection: 5},  // RPN 125 medium
		{Severity: 5, Occurrence: 2, Detection: 5},  // RPN 50 low
		{Severity: 5, Occurrence: 5, Detection: 4},  // RPN 100 medium (exactly at threshold)
	}}
	out := Summarize(table, a)
	if out.HighCount != 2 || out.MediumCount != 2 || out.LowCount != 1 {
		t.Fatalf("counts H/M/L = %d/%d/%d, want 2/2/1", out.HighCount, out.MediumCount, out.LowCount)
	}
	if out.MaxRPN != 250 {
		t.Fatalf("max rpn = %d, want 250", out.MaxRPN)
	}
}

func TestValidateRow(t *testing.T) {
	if err := ValidateRow(domain.FMEARow{Function: "f", FailureMode: "m", Severity: 5, Occurrence: 5, Detection: 5}); err != nil {
		t.Fatalf("valid row rejected: %v", err)
	}
	if err := ValidateRow(domain.FMEARow{Severity: 11, Occurrence: 5, Detection: 5}); err == nil {
		t.Fatalf("severity 11 should be rejected")
	}
}
