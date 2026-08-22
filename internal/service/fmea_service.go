package service

import (
	"context"
	"fmt"

	"task140-reliability/internal/domain"
	"task140-reliability/internal/fmea"
)

// AddFMEARow validates and adds an FMEA row. RPN and risk class are computed
// by the analysis engine at solve time, but we also store a provisional RPN
// so the table is queryable before solve.
func (svc *Service) AddFMEARow(ctx context.Context, row domain.FMEARow) (*domain.FMEARow, error) {
	if row.AnalysisID == "" {
		return nil, fmt.Errorf("%w: analysis_id required", domain.ErrInvalidArgument)
	}
	if err := fmea.ValidateRow(row); err != nil {
		return nil, err
	}
	row.ID = newID()
	row.CreatedAt = now()
	row.RPN = fmea.ComputeRPN(row)
	a, err := svc.store.GetAnalysis(ctx, row.AnalysisID)
	if err != nil {
		return nil, err
	}
	row.RiskClass = domain.RiskLow
	if err := svc.store.AddFMEARow(ctx, row); err != nil {
		return nil, err
	}
	return &row, nil
}

func (svc *Service) UpdateFMEARow(ctx context.Context, id string, fn, mode, effect *string, sev, occ, det *int, action, actionState *string) (*domain.FMEARow, error) {
	if sev != nil {
		if err := domain.ValidateRating("severity", *sev); err != nil {
			return nil, err
		}
	}
	if occ != nil {
		if err := domain.ValidateRating("occurrence", *occ); err != nil {
			return nil, err
		}
	}
	if det != nil {
		if err := domain.ValidateRating("detection", *det); err != nil {
			return nil, err
		}
	}
	if err := svc.store.UpdateFMEARow(ctx, id, fn, mode, effect, sev, occ, det, action, actionState); err != nil {
		return nil, err
	}
	return svc.getFMEARowByID(ctx, id)
}

func (svc *Service) DeleteFMEARow(ctx context.Context, id string) error {
	return svc.store.DeleteFMEARow(ctx, id)
}

func (svc *Service) LoadFMEATable(ctx context.Context, analysisID string) (*domain.FMEATable, error) {
	return svc.store.LoadFMEATable(ctx, analysisID)
}

func (svc *Service) getFMEARowByID(ctx context.Context, id string) (*domain.FMEARow, error) {
	return svc.store.GetFMEARow(ctx, id)
}
