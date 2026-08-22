package service

import (
	"context"
	"fmt"
	"strings"

	"task140-reliability/internal/analysis"
	"task140-reliability/internal/domain"
)

// CreateAnalysis creates a new analysis project with default parameters.
func (svc *Service) CreateAnalysis(ctx context.Context, name, topEvent string, params *domain.Analysis) (*domain.Analysis, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name required", domain.ErrInvalidArgument)
	}
	a := domain.DefaultAnalysisParams()
	a.ID = newID()
	a.Name = name
	a.TopEvent = topEvent
	a.State = domain.StateDraft
	a.Version = 1
	a.CreatedAt = now()
	a.UpdatedAt = a.CreatedAt
	if params != nil {
		mergeParams(&a, *params)
	}
	if err := validateParams(a); err != nil {
		return nil, err
	}
	if err := svc.store.CreateAnalysis(ctx, a); err != nil {
		return nil, err
	}
	return &a, nil
}

func mergeParams(dst *domain.Analysis, src domain.Analysis) {
	if src.MaxOrder > 0 {
		dst.MaxOrder = src.MaxOrder
	}
	if src.SCrit > 0 {
		dst.SCrit = src.SCrit
	}
	if src.RPNHigh > 0 {
		dst.RPNHigh = src.RPNHigh
	}
	if src.RPNMedium > 0 {
		dst.RPNMedium = src.RPNMedium
	}
	if src.ExactLimit >= 0 {
		dst.ExactLimit = src.ExactLimit
	}
	if src.CCFBeta >= 0 {
		dst.CCFBeta = src.CCFBeta
	}
	if src.MissionHours > 0 {
		dst.MissionHours = src.MissionHours
	}
}

func validateParams(a domain.Analysis) error {
	if a.MaxOrder < 1 {
		return fmt.Errorf("%w: max_order must be >= 1", domain.ErrInvalidArgument)
	}
	if a.SCrit < 1 || a.SCrit > 10 {
		return fmt.Errorf("%w: s_crit must be 1..10", domain.ErrInvalidArgument)
	}
	if a.RPNMedium > a.RPNHigh {
		return fmt.Errorf("%w: rpn_medium must be <= rpn_high", domain.ErrInvalidArgument)
	}
	if a.CCFBeta < 0 || a.CCFBeta > 1 {
		return fmt.Errorf("%w: ccf_beta must be in [0,1]", domain.ErrInvalidArgument)
	}
	if a.MissionHours <= 0 {
		return fmt.Errorf("%w: mission_hours must be > 0", domain.ErrInvalidArgument)
	}
	return nil
}

func (svc *Service) ListAnalyses(ctx context.Context) ([]domain.Analysis, error) {
	return svc.store.ListAnalyses(ctx)
}

func (svc *Service) GetAnalysis(ctx context.Context, id string) (*domain.Analysis, error) {
	return svc.store.GetAnalysis(ctx, id)
}

// UpdateAnalysisParams updates the project-level parameters (refuses baselined).
func (svc *Service) UpdateAnalysisParams(ctx context.Context, id string, params domain.Analysis) (*domain.Analysis, error) {
	if err := validateParams(params); err != nil {
		return nil, err
	}
	if err := svc.store.UpdateAnalysisParams(ctx, id, params); err != nil {
		return nil, err
	}
	return svc.store.GetAnalysis(ctx, id)
}

// Solve runs the full quantification and persists the result. Requires the
// analysis to be in draft or analyzed state; transitions to analyzed.
func (svc *Service) Solve(ctx context.Context, id string) (*domain.AnalysisResult, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	a, err := svc.store.GetAnalysis(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := analysis.NextState(a.State, domain.StateAnalyzed); err != nil {
		return nil, err
	}
	res, err := svc.solve(ctx, a)
	if err != nil {
		return nil, err
	}
	// advance state to analyzed
	if err := svc.store.SetAnalysisState(ctx, id, domain.StateAnalyzed, a.Version, nil); err != nil {
		return nil, err
	}
	return res, nil
}

// solve is the shared recompute path used by Solve and ReconcileAll. It loads
// authoritative inputs, resolves component-derived event probabilities, runs
// the analysis engine, and persists the result (without state transition).
func (svc *Service) solve(ctx context.Context, a *domain.Analysis) (*domain.AnalysisResult, error) {
	tree, err := svc.store.LoadFTATree(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	table, err := svc.store.LoadFMEATable(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	diag, err := svc.store.LoadRBDDiagram(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	events, err := svc.store.LoadFailureEvents(ctx, a.ID)
	if err != nil {
		return nil, err
	}
	hasInputs := (tree != nil && tree.TopGateID != "") ||
		(table != nil && len(table.Rows) > 0) ||
		(diag != nil && diag.RootID != "") ||
		len(events) > 0
	if !hasInputs {
		return nil, ErrNothingToSolve
	}
	// resolve component-derived event probabilities
	if tree != nil && tree.TopGateID != "" {
		if err := svc.store.ResolveEventProbabilities(ctx, tree, a.MissionHours); err != nil {
			return nil, err
		}
	}
	eng := analysis.NewEngine(*a, tree, table, diag, events)
	res, err := eng.Solve(ctx)
	if err != nil {
		return nil, err
	}
	if err := svc.store.SaveResult(ctx, res); err != nil {
		return nil, err
	}
	return res, nil
}

// GetResult returns the persisted result of the current version.
func (svc *Service) GetResult(ctx context.Context, id string) (*domain.AnalysisResult, error) {
	a, err := svc.store.GetAnalysis(ctx, id)
	if err != nil {
		return nil, err
	}
	return svc.store.LoadResult(ctx, id, a.Version)
}

// Review marks an analyzed analysis as reviewed.
func (svc *Service) Review(ctx context.Context, id string) (*domain.Analysis, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	a, err := svc.store.GetAnalysis(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := analysis.NextState(a.State, domain.StateReviewed); err != nil {
		return nil, err
	}
	if err := svc.store.SetAnalysisState(ctx, id, domain.StateReviewed, a.Version, nil); err != nil {
		return nil, err
	}
	return svc.store.GetAnalysis(ctx, id)
}

// Baseline locks a reviewed analysis into an immutable snapshot.
func (svc *Service) Baseline(ctx context.Context, id string) (*domain.Analysis, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	a, err := svc.store.GetAnalysis(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := analysis.NextState(a.State, domain.StateBaselined); err != nil {
		return nil, err
	}
	t := now()
	if err := svc.store.SetAnalysisState(ctx, id, domain.StateBaselined, a.Version, &t); err != nil {
		return nil, err
	}
	return svc.store.GetAnalysis(ctx, id)
}

// Revise creates a new version from a baselined analysis (back to draft).
func (svc *Service) Revise(ctx context.Context, id string) (*domain.Analysis, error) {
	svc.mu.Lock()
	defer svc.mu.Unlock()
	a, err := svc.store.GetAnalysis(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := analysis.NextState(a.State, domain.StateDraft); err != nil {
		return nil, err
	}
	newVer := a.Version + 1
	if err := svc.store.SetAnalysisState(ctx, id, domain.StateDraft, newVer, nil); err != nil {
		return nil, err
	}
	return svc.store.GetAnalysis(ctx, id)
}

func (svc *Service) ListRevisions(ctx context.Context, id string) ([]domain.Revision, error) {
	return svc.store.ListRevisions(ctx, id)
}
