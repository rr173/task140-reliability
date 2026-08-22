package domain

import "time"

// FMEARow is one line of a Failure Modes and Effects Analysis worksheet.
// RPN = Severity × Occurrence × Detection. Risk classification uses the
// severity floor rule: Severity >= analysis.s_crit => "high" regardless of RPN.
type FMEARow struct {
	ID            string    `json:"id"`
	AnalysisID    string    `json:"analysis_id"`
	Function      string    `json:"function"`
	FailureMode   string    `json:"failure_mode"`
	Effect        string    `json:"effect"`
	Severity      int       `json:"severity"`      // 1-10
	Occurrence    int       `json:"occurrence"`    // 1-10
	Detection     int       `json:"detection"`     // 1-10
	RPN           int       `json:"rpn"`           // computed: S*O*D
	RiskClass     RiskClass `json:"risk_class"`    // computed
	Action        string    `json:"action,omitempty"`
	ActionState   string    `json:"action_state,omitempty"` // open/in_progress/closed
	CreatedAt     time.Time `json:"created_at"`
}

// FMEATable is the FMEA of an analysis: a list of rows + the computed summary.
type FMEATable struct {
	AnalysisID string     `json:"analysis_id"`
	Rows       []FMEARow  `json:"rows"`
	MaxRPN     int        `json:"max_rpn"`
	HighCount  int        `json:"high_count"`
	MediumCount int       `json:"medium_count"`
	LowCount   int        `json:"low_count"`
}
