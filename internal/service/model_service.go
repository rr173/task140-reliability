package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"task140-reliability/internal/domain"
)

// SaveFTATree validates and persists a fault tree, resolving component-derived
// probabilities lazily at solve time. The tree must have a top gate.
func (svc *Service) SaveFTATree(ctx context.Context, tree domain.FTATree) (*domain.FTATree, error) {
	tree.AnalysisID = strings.TrimSpace(tree.AnalysisID)
	if tree.AnalysisID == "" {
		return nil, fmt.Errorf("%w: analysis_id required", domain.ErrInvalidArgument)
	}
	if tree.TopGateID == "" {
		return nil, fmt.Errorf("%w: top_gate_id required", domain.ErrInvalidArgument)
	}
	// validate gate references
	gateIDs := make(map[string]bool, len(tree.Gates))
	for _, g := range tree.Gates {
		if g.ID == "" {
			return nil, fmt.Errorf("%w: gate id required", domain.ErrInvalidArgument)
		}
		if gateIDs[g.ID] {
			return nil, fmt.Errorf("%w: duplicate gate id %s", domain.ErrAlreadyExists, g.ID)
		}
		gateIDs[g.ID] = true
		switch g.Type {
		case domain.GateAND, domain.GateOR, domain.GateKOFN:
		default:
			return nil, fmt.Errorf("%w: gate type %s", domain.ErrInvalidArgument, g.Type)
		}
		if g.Type == domain.GateKOFN && g.K < 1 {
			return nil, fmt.Errorf("%w: kofn gate k must be >= 1", domain.ErrInvalidArgument)
		}
	}
	if !gateIDs[tree.TopGateID] {
		return nil, fmt.Errorf("%w: top gate %s not in gates", domain.ErrInvalidArgument, tree.TopGateID)
	}
	// validate inputs reference existing gates/events
	eventIDs := make(map[string]bool, len(tree.Events))
	for _, e := range tree.Events {
		if e.ID == "" {
			return nil, fmt.Errorf("%w: event id required", domain.ErrInvalidArgument)
		}
		if eventIDs[e.ID] {
			return nil, fmt.Errorf("%w: duplicate event id %s", domain.ErrAlreadyExists, e.ID)
		}
		eventIDs[e.ID] = true
		if e.ProbMicro < 0 || e.ProbMicro > 1_000_000 {
			return nil, fmt.Errorf("%w: event %s prob_micro out of [0,1000000]", domain.ErrInvalidArgument, e.ID)
		}
	}
	for _, g := range tree.Gates {
		for _, in := range g.Inputs {
			if in.GateID == "" && in.EventID == "" {
				return nil, fmt.Errorf("%w: gate %s input has neither gate_id nor event_id", domain.ErrInvalidArgument, g.ID)
			}
			if in.GateID != "" && !gateIDs[in.GateID] {
				return nil, fmt.Errorf("%w: gate %s references unknown gate %s", domain.ErrInvalidArgument, g.ID, in.GateID)
			}
			if in.EventID != "" && !eventIDs[in.EventID] {
				return nil, fmt.Errorf("%w: gate %s references unknown event %s", domain.ErrInvalidArgument, g.ID, in.EventID)
			}
		}
	}
	// refuse baselined
	a, err := svc.store.GetAnalysis(ctx, tree.AnalysisID)
	if err != nil {
		return nil, err
	}
	if a.State == domain.StateBaselined {
		return nil, fmt.Errorf("%w: analysis %s is baselined", domain.ErrStateConflict, tree.AnalysisID)
	}
	// mark top gate
	for i := range tree.Gates {
		tree.Gates[i].IsTop = tree.Gates[i].ID == tree.TopGateID
	}
	// sort events for deterministic storage
	sort.Slice(tree.Events, func(i, j int) bool { return tree.Events[i].ID < tree.Events[j].ID })
	if err := svc.store.SaveFTATree(ctx, tree); err != nil {
		return nil, err
	}
	return &tree, nil
}

func (svc *Service) LoadFTATree(ctx context.Context, analysisID string) (*domain.FTATree, error) {
	return svc.store.LoadFTATree(ctx, analysisID)
}

// SaveRBDDiagram validates and persists a block diagram.
func (svc *Service) SaveRBDDiagram(ctx context.Context, diag domain.RBDDiagram) (*domain.RBDDiagram, error) {
	diag.AnalysisID = strings.TrimSpace(diag.AnalysisID)
	if diag.AnalysisID == "" {
		return nil, fmt.Errorf("%w: analysis_id required", domain.ErrInvalidArgument)
	}
	if diag.RootID == "" {
		return nil, fmt.Errorf("%w: root_id required", domain.ErrInvalidArgument)
	}
	blockIDs := make(map[string]bool, len(diag.Blocks))
	for _, b := range diag.Blocks {
		if b.ID == "" {
			return nil, fmt.Errorf("%w: block id required", domain.ErrInvalidArgument)
		}
		if blockIDs[b.ID] {
			return nil, fmt.Errorf("%w: duplicate block id %s", domain.ErrAlreadyExists, b.ID)
		}
		blockIDs[b.ID] = true
		switch b.Type {
		case domain.BlockBasic, domain.BlockSeries, domain.BlockParallel, domain.BlockKOFN:
		default:
			return nil, fmt.Errorf("%w: block type %s", domain.ErrInvalidArgument, b.Type)
		}
		if b.Type == domain.BlockKOFN && b.K < 1 {
			return nil, fmt.Errorf("%w: kofn block k must be >= 1", domain.ErrInvalidArgument)
		}
		if b.FailureRatePPT < 0 || b.RepairRatePPT < 0 {
			return nil, fmt.Errorf("%w: negative rate", domain.ErrInvalidArgument)
		}
	}
	if !blockIDs[diag.RootID] {
		return nil, fmt.Errorf("%w: root block %s not in blocks", domain.ErrInvalidArgument, diag.RootID)
	}
	for _, b := range diag.Blocks {
		for _, cid := range b.Children {
			if !blockIDs[cid] {
				return nil, fmt.Errorf("%w: block %s references unknown child %s", domain.ErrInvalidArgument, b.ID, cid)
			}
		}
	}
	a, err := svc.store.GetAnalysis(ctx, diag.AnalysisID)
	if err != nil {
		return nil, err
	}
	if a.State == domain.StateBaselined {
		return nil, fmt.Errorf("%w: analysis %s is baselined", domain.ErrStateConflict, diag.AnalysisID)
	}
	if err := svc.store.SaveRBDDiagram(ctx, diag); err != nil {
		return nil, err
	}
	return &diag, nil
}

func (svc *Service) LoadRBDDiagram(ctx context.Context, analysisID string) (*domain.RBDDiagram, error) {
	return svc.store.LoadRBDDiagram(ctx, analysisID)
}

// CreateFailureEvent appends a failure event to the log.
func (svc *Service) CreateFailureEvent(ctx context.Context, e domain.FailureEvent) (*domain.FailureEvent, error) {
	if e.AnalysisID == "" {
		return nil, fmt.Errorf("%w: analysis_id required", domain.ErrInvalidArgument)
	}
	if e.TTFHours < 0 {
		return nil, fmt.Errorf("%w: ttf_hours must be >= 0", domain.ErrInvalidArgument)
	}
	if e.TTRHours < 0 {
		return nil, fmt.Errorf("%w: ttr_hours must be >= 0", domain.ErrInvalidArgument)
	}
	e.ID = newID()
	if e.OccurredAt.IsZero() {
		e.OccurredAt = now()
	}
	if err := svc.store.CreateFailureEvent(ctx, e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (svc *Service) LoadFailureEvents(ctx context.Context, analysisID string) ([]domain.FailureEvent, error) {
	return svc.store.LoadFailureEvents(ctx, analysisID)
}
