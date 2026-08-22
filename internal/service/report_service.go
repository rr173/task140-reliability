package service

import (
	"context"
	"fmt"

	"task140-reliability/internal/domain"
	"task140-reliability/internal/fdist"
)

// FitFailureData computes the life-distribution fit for the analysis's event
// log without persisting (the result is part of the full solve). Useful as a
// standalone read endpoint.
func (svc *Service) FitFailureData(ctx context.Context, analysisID string) (*domain.FitResult, error) {
	a, err := svc.store.GetAnalysis(ctx, analysisID)
	if err != nil {
		return nil, err
	}
	events, err := svc.store.LoadFailureEvents(ctx, analysisID)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("%w: no failure events", domain.ErrInsufficientData)
	}
	fit, err := fdist.FitAuto(events)
	if err != nil {
		return nil, err
	}
	fit.MissionReliability = fdist.MissionReliability(fit, a.MissionHours)
	return fit, nil
}

// Report assembles a combined report for the analysis: the persisted result
// plus current model snapshots (tree, fmea, rbd, fit). Used by the frontend.
func (svc *Service) Report(ctx context.Context, id string) (map[string]any, error) {
	a, err := svc.store.GetAnalysis(ctx, id)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"analysis": a,
	}
	if res, err := svc.store.LoadResult(ctx, id, a.Version); err == nil {
		out["result"] = res
	} else {
		out["result"] = nil
	}
	if t, err := svc.store.LoadFTATree(ctx, id); err == nil {
		out["fta"] = t
	}
	if tb, err := svc.store.LoadFMEATable(ctx, id); err == nil {
		out["fmea"] = tb
	}
	if d, err := svc.store.LoadRBDDiagram(ctx, id); err == nil {
		out["rbd"] = d
	}
	if ev, err := svc.store.LoadFailureEvents(ctx, id); err == nil {
		out["failure_events"] = ev
	}
	if revs, err := svc.store.ListRevisions(ctx, id); err == nil {
		out["revisions"] = revs
	}
	return out, nil
}
