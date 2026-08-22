package service

import (
	"context"
	"fmt"
	"strings"

	"task140-reliability/internal/domain"
)

// CreateComponent inserts a library component.
func (svc *Service) CreateComponent(ctx context.Context, name, category, note string, ratePPT int64) (*domain.Component, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("%w: name required", domain.ErrInvalidArgument)
	}
	if ratePPT < 0 {
		return nil, fmt.Errorf("%w: failure_rate_ppt must be >= 0", domain.ErrInvalidArgument)
	}
	c := domain.Component{
		ID: newID(), Name: name, Category: category, Note: note,
		FailureRatePPT: domain.FailureRatePPT(ratePPT), CreatedAt: now(),
	}
	if err := svc.store.CreateComponent(ctx, c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (svc *Service) ListComponents(ctx context.Context) ([]domain.Component, error) {
	return svc.store.ListComponents(ctx)
}

func (svc *Service) GetComponent(ctx context.Context, id string) (*domain.Component, error) {
	return svc.store.GetComponent(ctx, id)
}

func (svc *Service) UpdateComponent(ctx context.Context, id string, name, category, note *string, rate *int64) (*domain.Component, error) {
	var fr *domain.FailureRatePPT
	if rate != nil {
		if *rate < 0 {
			return nil, fmt.Errorf("%w: failure_rate_ppt must be >= 0", domain.ErrInvalidArgument)
		}
		v := domain.FailureRatePPT(*rate)
		fr = &v
	}
	if err := svc.store.UpdateComponent(ctx, id, name, category, note, fr); err != nil {
		return nil, err
	}
	return svc.store.GetComponent(ctx, id)
}

func (svc *Service) CreateFailureMode(ctx context.Context, componentID, name, effect string, sev, occ, det int) (*domain.FailureMode, error) {
	fm := domain.FailureMode{ID: newID(), ComponentID: componentID, Name: name, Effect: effect, Severity: sev, Occurrence: occ, Detection: det}
	if err := domain.ValidateRating("severity", sev); err != nil {
		return nil, err
	}
	if err := domain.ValidateRating("occurrence", occ); err != nil {
		return nil, err
	}
	if err := domain.ValidateRating("detection", det); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, fmt.Errorf("%w: name required", domain.ErrInvalidArgument)
	}
	if err := svc.store.CreateFailureMode(ctx, fm); err != nil {
		return nil, err
	}
	return &fm, nil
}

func (svc *Service) ListFailureModes(ctx context.Context, componentID string) ([]domain.FailureMode, error) {
	return svc.store.ListFailureModes(ctx, componentID)
}
