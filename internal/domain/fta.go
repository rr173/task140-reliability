package domain

// GateType enumerates the fault-tree gate kinds. NOT gates are accepted but
// flagged non-coherent (a coherent tree has no NOT and no event appearing
// both complemented and uncomplemented).
type GateType string

const (
	GateAND  GateType = "AND"
	GateOR   GateType = "OR"
	GateKOFN GateType = "KOFN" // voting: true if >= k of n inputs
)

// BasicEvent is a leaf of the fault tree. Either ComponentID is set (probability
// derived from the component failure rate × mission hours) or an explicit
// ProbMicro is given. Negated marks a complemented event (NOT of the event).
type BasicEvent struct {
	ID          string    `json:"id"`
	AnalysisID  string    `json:"analysis_id"`
	Label       string    `json:"label"`
	ComponentID string    `json:"component_id,omitempty"` // optional: derive q from component
	ProbMicro   ProbMicro `json:"prob_micro"`             // explicit q (1e-6) if no component
	CCFGroup    string    `json:"ccf_group,omitempty"`    // common-cause group tag
	Negated     bool      `json:"negated,omitempty"`
}

// Gate is an internal or top node of the fault tree. Inputs reference either a
// child Gate (by GateID) or a BasicEvent (by EventID); exactly one of the two
// is set per input.
type Gate struct {
	ID          string    `json:"id"`
	AnalysisID  string    `json:"analysis_id"`
	Type        GateType  `json:"type"`
	K           int       `json:"k,omitempty"` // for KOFN
	IsTop       bool      `json:"is_top"`
	Inputs      []GateInput `json:"inputs,omitempty"`
}

// GateInput is one input to a gate. Either GateID or EventID is set.
type GateInput struct {
	GateID  string `json:"gate_id,omitempty"`
	EventID string `json:"event_id,omitempty"`
}

// FTATree is the fault tree of an analysis: a set of gates + basic events,
// rooted at the top gate.
type FTATree struct {
	AnalysisID string        `json:"analysis_id"`
	TopGateID  string        `json:"top_gate_id"`
	Gates      []Gate        `json:"gates"`
	Events     []BasicEvent `json:"events"`
}

// CCFGroup defines a common-cause-failure group: events sharing a tag share a
// common-cause probability component governed by the analysis CCFBeta.
type CCFGroup struct {
	AnalysisID string `json:"analysis_id"`
	Tag        string `json:"tag"`
	EventIDs   []string `json:"event_ids"`
}

// CoherenceReport is the result of the coherence (consistency) check on a
// fault tree.
type CoherenceReport struct {
	Coherent     bool     `json:"coherent"`     // true if no NOT and no event both polarities
	Warnings     []string `json:"warnings,omitempty"`
	NonCoherentReasons []string `json:"non_coherent_reasons,omitempty"`
}

// CutSet is a minimal cut set: an unordered set of basic-event IDs whose
// simultaneous occurrence causes the top event.
type CutSet struct {
	Events      []string `json:"events"`            // basic event IDs
	Order       int      `json:"order"`
	Probability float64  `json:"probability"`       // product of event q (post-CCF)
}

// CutSetResult is the complete fault-tree quantification output.
type CutSetResult struct {
	TopProbability   float64   `json:"top_probability"`
	Method          string    `json:"method"`        // "exact" or "rare_event"
	CutSets         []CutSet  `json:"cut_sets"`
	TruncatedCount  int       `json:"truncated_count"` // cut sets dropped by max_order
	Coherence       CoherenceReport `json:"coherence"`
}
