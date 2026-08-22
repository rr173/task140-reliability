package domain

import "time"

// AnalysisResult is the persisted, recomputable result bundle of one analysis
// version. It is a pure recompute from the stored tree/diagram/event log.
type AnalysisResult struct {
	AnalysisID string         `json:"analysis_id"`
	Version    int            `json:"version"`
	SolvedAt   time.Time      `json:"solved_at"`
	FTA        *CutSetResult  `json:"fta,omitempty"`
	FMEA       *FMEATable     `json:"fmea,omitempty"`
	RBD        *RBDResult     `json:"rbd,omitempty"`
	Fit        *FitResult     `json:"fit,omitempty"`
	Warnings   []string       `json:"warnings,omitempty"`
	Compliant  bool           `json:"compliant"` // true if no "high" FMEA row and top prob within target
}

// Revision is one entry in the analysis version chain.
type Revision struct {
	AnalysisID string        `json:"analysis_id"`
	Version    int           `json:"version"`
	State      AnalysisState `json:"state"`
	CreatedAt  time.Time     `json:"created_at"`
	Note       string        `json:"note,omitempty"`
}
