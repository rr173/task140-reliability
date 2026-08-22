// Package domain defines the core entities and enums for the reliability
// analysis & safety assessment engine. It holds no business rules of its own;
// the fta/fmea/rbd/fdist/analysis packages compute over these types.
package domain

import "time"

// FailureRatePPT is the integer storage unit for a failure rate: parts per
// trillion per hour (1e-12 /hour). A rate of 1e-6/h is stored as 1_000_000.
// Using an integer unit avoids float drift in persistence.
type FailureRatePPT int64

// ProbMicro is the integer storage unit for a probability: parts per million
// (1e-6). A probability of 0.001 is stored as 1000. Range [0, 1_000_000].
type ProbMicro int64

// AnalysisState is the lifecycle state of an analysis project.
type AnalysisState string

const (
	StateDraft     AnalysisState = "draft"     // being assembled, not yet solved
	StateAnalyzed  AnalysisState = "analyzed"  // solved successfully, awaiting review
	StateReviewed  AnalysisState = "reviewed"  // reviewed and accepted, ready to baseline
	StateBaselined AnalysisState = "baselined" // locked snapshot, immutable
	StateRevised   AnalysisState = "revised"   // superseded by a newer version
)

// RiskClass is the FMEA risk classification.
type RiskClass string

const (
	RiskHigh   RiskClass = "high"
	RiskMedium RiskClass = "medium"
	RiskLow    RiskClass = "low"
)

// Component is a reusable equipment/item in the component library. It carries a
// base failure rate (failure_rate_ppt) used by FTA basic events and RBD blocks
// when no explicit probability is given.
type Component struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	Category       string        `json:"category"`
	FailureRatePPT FailureRatePPT `json:"failure_rate_ppt"`
	Note           string        `json:"note,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}

// FailureMode is a named mode under a component, carrying the FMEA severity /
// occurrence / detection ratings (1-10 each). It seeds FMEA rows.
type FailureMode struct {
	ID          string `json:"id"`
	ComponentID string `json:"component_id"`
	Name        string `json:"name"`
	Effect      string `json:"effect,omitempty"`
	Severity    int    `json:"severity"`    // 1-10
	Occurrence  int    `json:"occurrence"`  // 1-10
	Detection   int    `json:"detection"`   // 1-10
}

// Analysis is a reliability analysis project. It bundles one fault tree, one
// FMEA table, one RBD and one failure-event log, plus project-level parameters
// (max cut-set order, severity floor, RPN thresholds).
type Analysis struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	TopEvent      string        `json:"top_event"`
	State         AnalysisState `json:"state"`
	Version       int           `json:"version"`
	MaxOrder      int           `json:"max_order"`       // max cut-set order to keep
	SCrit         int           `json:"s_crit"`           // severity floor for "high"
	RPNHigh       int           `json:"rpn_high"`         // RPN >= this => high (when s<s_crit)
	RPNMedium     int           `json:"rpn_medium"`       // RPN >= this => medium
	ExactLimit    int           `json:"exact_limit"`      // #cutsets <= this => exact inclusion-exclusion
	CCFBeta       float64       `json:"ccf_beta"`         // common-cause beta factor [0,1]
	MissionHours  int64         `json:"mission_hours"`    // mission time for reliability calc
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
	BaselinedAt   *time.Time    `json:"baselined_at,omitempty"`
}

// DefaultAnalysisParams returns the conventional default parameters for a new
// analysis. They can be overridden via PATCH before solving.
func DefaultAnalysisParams() Analysis {
	return Analysis{
		MaxOrder:     4,
		SCrit:        8,
		RPNHigh:      200,
		RPNMedium:    100,
		ExactLimit:   8,
		CCFBeta:      0.1,
		MissionHours: 8760,
	}
}
