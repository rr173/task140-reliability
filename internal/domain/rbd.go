package domain

// BlockType enumerates reliability-block-diagram node kinds.
type BlockType string

const (
	BlockBasic   BlockType = "block"     // a single component with a reliability
	BlockSeries  BlockType = "series"     // all children must work
	BlockParallel BlockType = "parallel"  // at least one child works (1-of-n)
	BlockKOFN    BlockType = "kofn"       // >= k of n children must work
)

// RBDBlock is a node in the reliability block diagram. Basic blocks carry a
// failure rate (failure_rate_ppt) and optional repair rate (repair_rate_ppt,
// 1e-12/h) for steady-state availability. Composite blocks have children.
type RBDBlock struct {
	ID             string       `json:"id"`
	AnalysisID     string       `json:"analysis_id"`
	Name           string       `json:"name"`
	Type           BlockType    `json:"type"`
	K              int          `json:"k,omitempty"` // for kofn
	FailureRatePPT FailureRatePPT `json:"failure_rate_ppt,omitempty"`
	RepairRatePPT  FailureRatePPT `json:"repair_rate_ppt,omitempty"`
	Children       []string     `json:"children,omitempty"` // child block IDs
}

// RBDDiagram is the block diagram of an analysis.
type RBDDiagram struct {
	AnalysisID string     `json:"analysis_id"`
	RootID     string     `json:"root_id"`
	Blocks     []RBDBlock `json:"blocks"`
}

// RBDResult is the quantification of a reliability block diagram.
type RBDResult struct {
	RootReliability float64 `json:"root_reliability"`   // mission reliability
	Availability    float64 `json:"availability"`      // steady-state (repairable); 0 if not repairable
	Repairable      bool    `json:"repairable"`
	Method          string  `json:"method"`             // "series"/"parallel"/"kofn"/"block"
}
