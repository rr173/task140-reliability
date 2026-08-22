package domain

import "time"

// FailureEvent is one observed failure in the failure-event log used to fit a
// life distribution. TTFHours is time-to-failure (operating hours from start
// or last repair to this failure); TTRHours is time-to-repair.
type FailureEvent struct {
	ID          string    `json:"id"`
	AnalysisID  string    `json:"analysis_id"`
	ComponentID string    `json:"component_id,omitempty"`
	TTFHours    int64     `json:"ttf_hours"` // time-to-failure, operating hours
	TTRHours    int64     `json:"ttr_hours"` // time-to-repair, hours
	OccurredAt  time.Time `json:"occurred_at"`
}

// Distribution is the fitted life-distribution kind.
type Distribution string

const (
	DistExponential Distribution = "exponential"
	DistWeibull     Distribution = "weibull"
)

// FitResult is the failure-data fit output.
type FitResult struct {
	Distribution Distribution `json:"distribution"`
	// Exponential: lambda = 1/MTTF
	Lambda float64 `json:"lambda"` // per hour
	MTTF   float64 `json:"mttf"`   // hours
	// Weibull: shape beta, scale eta
	Beta  float64 `json:"beta,omitempty"`
	Eta   float64 `json:"eta,omitempty"`
	// Availability from event log: uptime/(uptime+downtime)
	Availability float64 `json:"availability"`
	// Reliability at mission time t: R(t). For exponential exp(-lambda*t);
	// for Weibull exp(-(t/eta)^beta).
	MissionReliability float64 `json:"mission_reliability"`
	SampleSize         int     `json:"sample_size"`
	Warnings           []string `json:"warnings,omitempty"`
	Method             string  `json:"method"` // "exponential_mle" / "weibull_median_rank"
}
