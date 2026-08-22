// Package fmea computes the Failure Modes and Effects Analysis worksheet:
// RPN (Risk Priority Number = Severity × Occurrence × Detection) and the
// risk classification under the severity-floor rule.
package fmea

import (
	"fmt"

	"task140-reliability/internal/domain"
)

// Classify computes the risk class of an FMEA row under the given analysis
// parameters. The severity-floor rule: Severity >= sCrit => "high" regardless
// of RPN. Otherwise RPN >= rpnHigh => high, >= rpnMedium => medium, else low.
func Classify(row domain.FMEARow, sCrit, rpnHigh, rpnMedium int) domain.RiskClass {
	if row.Severity >= sCrit {
		return domain.RiskHigh
	}
	switch {
	case row.RPN > rpnHigh:
		return domain.RiskHigh
	case row.RPN >= rpnMedium:
		return domain.RiskMedium
	default:
		return domain.RiskLow
	}
}

// ValidateRow validates the ratings of an FMEA row.
func ValidateRow(row domain.FMEARow) error {
	if err := domain.ValidateRating("severity", row.Severity); err != nil {
		return err
	}
	if err := domain.ValidateRating("occurrence", row.Occurrence); err != nil {
		return err
	}
	if err := domain.ValidateRating("detection", row.Detection); err != nil {
		return err
	}
	if row.Function == "" {
		return fmt.Errorf("%w: function required", domain.ErrInvalidArgument)
	}
	if row.FailureMode == "" {
		return fmt.Errorf("%w: failure_mode required", domain.ErrInvalidArgument)
	}
	return nil
}

// ComputeRPN returns Severity * Occurrence * Detection.
func ComputeRPN(row domain.FMEARow) int {
	return row.Severity * row.Occurrence * row.Detection
}

// Summarize computes the per-row RPN and risk class, then the table summary
// (max RPN and counts by class).
func Summarize(table *domain.FMEATable, a domain.Analysis) *domain.FMEATable {
	out := &domain.FMEATable{AnalysisID: table.AnalysisID}
	maxRPN := 0
	for _, row := range table.Rows {
		r := row
		r.RPN = ComputeRPN(r)
		r.RiskClass = Classify(r, a.SCrit, a.RPNHigh, a.RPNMedium)
		if r.RPN > maxRPN {
			maxRPN = r.RPN
		}
		switch r.RiskClass {
		case domain.RiskHigh:
			out.HighCount++
		case domain.RiskMedium:
			out.MediumCount++
		case domain.RiskLow:
			out.LowCount++
		}
		out.Rows = append(out.Rows, r)
	}
	out.MaxRPN = maxRPN
	return out
}
