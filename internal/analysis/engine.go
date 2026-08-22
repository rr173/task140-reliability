// Package analysis orchestrates a reliability analysis project: it drives the
// lifecycle (draft -> analyzed -> reviewed -> baselined -> revised), runs the
// full quantification by calling the fta/fmea/rbd/fdist engines, baselines
// results, creates revisions, and reconciles all derived state on restart.
package analysis

import (
	"context"
	"fmt"
	"time"

	"task140-reliability/internal/domain"
	"task140-reliability/internal/fdist"
	"task140-reliability/internal/fmea"
	"task140-reliability/internal/fta"
	"task140-reliability/internal/rbd"
)

// Engine is the analysis orchestrator. It is constructed with the loaded
// domain snapshot (tree/diagram/events/fmea rows) and the analysis params,
// then Solve produces the recomputable result bundle. It holds no state that
// isn't derivable from the snapshot, so ReconcileAll simply re-Solves.
type Engine struct {
	tree    *domain.FTATree
	table   *domain.FMEATable
	diagram *domain.RBDDiagram
	events  []domain.FailureEvent
	params  domain.Analysis
}

// NewEngine builds an engine over a snapshot.
func NewEngine(params domain.Analysis, tree *domain.FTATree, table *domain.FMEATable, diagram *domain.RBDDiagram, events []domain.FailureEvent) *Engine {
	return &Engine{tree: tree, table: table, diagram: diagram, events: events, params: params}
}

// Solve runs the full quantification across the four analysis types. Missing
// pieces (nil tree/diagram/empty events) are skipped without error; an analysis
// may legitimately contain only an FMEA, for example. Returns a result bundle
// plus any warnings (non-fatal).
func (e *Engine) Solve(ctx context.Context) (*domain.AnalysisResult, error) {
	res := &domain.AnalysisResult{
		AnalysisID: e.params.ID,
		Version:    e.params.Version,
		SolvedAt:   time.Now().UTC(),
	}
	compliant := true

	if e.tree != nil && e.tree.TopGateID != "" {
		s, err := fta.NewSolver(e.tree, e.params.MaxOrder, e.params.CCFBeta)
		if err != nil {
			return nil, err
		}
		cs, err := s.Solve(e.params.ExactLimit)
		if err != nil {
			return nil, err
		}
		res.FTA = cs
		if !cs.Coherence.Coherent {
			res.Warnings = append(res.Warnings, "non_coherent_fault_tree")
		}
		if cs.TruncatedCount > 0 {
			res.Warnings = append(res.Warnings, fmt.Sprintf("cutsets_truncated:%d", cs.TruncatedCount))
		}
	}

	if e.table != nil && len(e.table.Rows) > 0 {
		out := fmea.Summarize(e.table, e.params)
		res.FMEA = out
		if out.HighCount > 0 {
			compliant = false
			res.Warnings = append(res.Warnings, fmt.Sprintf("fmea_high_rows:%d", out.HighCount))
		}
	}

	if e.diagram != nil && e.diagram.RootID != "" {
		s, err := rbd.NewSolver(e.diagram)
		if err != nil {
			return nil, err
		}
		if err := s.Validate(); err != nil {
			return nil, err
		}
		rr, err := s.Solve(e.params.MissionHours)
		if err != nil {
			return nil, err
		}
		res.RBD = rr
	}

	if len(e.events) > 0 {
		fit, err := fdist.FitAuto(e.events)
		if err != nil {
			return nil, err
		}
		fit.MissionReliability = fdist.MissionReliability(fit, e.params.MissionHours)
		res.Fit = fit
		if fit.Warnings != nil {
			res.Warnings = append(res.Warnings, fit.Warnings...)
		}
	}

	res.Compliant = compliant
	return res, nil
}

// NextState computes the next legal state given a requested transition.
func NextState(cur domain.AnalysisState, want domain.AnalysisState) (domain.AnalysisState, error) {
	switch want {
	case domain.StateAnalyzed:
		if cur != domain.StateDraft {
			return cur, fmt.Errorf("%w: can only solve from draft (cur=%s)", domain.ErrStateConflict, cur)
		}
		return domain.StateAnalyzed, nil
	case domain.StateReviewed:
		if cur != domain.StateAnalyzed {
			return cur, fmt.Errorf("%w: review requires analyzed (cur=%s)", domain.ErrStateConflict, cur)
		}
		return domain.StateReviewed, nil
	case domain.StateBaselined:
		if cur != domain.StateReviewed {
			return cur, fmt.Errorf("%w: baseline requires reviewed (cur=%s)", domain.ErrStateConflict, cur)
		}
		return domain.StateBaselined, nil
	case domain.StateDraft:
		// revise: from baselined back to draft (new version)
		if cur != domain.StateBaselined {
			return cur, fmt.Errorf("%w: revise requires baselined (cur=%s)", domain.ErrStateConflict, cur)
		}
		return domain.StateDraft, nil
	}
	return cur, fmt.Errorf("%w: unknown target state %s", domain.ErrInvalidArgument, want)
}
