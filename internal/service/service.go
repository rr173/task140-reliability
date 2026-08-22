// Package service is the business use-case layer: it validates parameters,
// coordinates the store with the analysis engines, enforces the lifecycle
// state machine, baselining and revision, and reconciles derived state on
// restart. A sync.Mutex guards compound operations (solve/baseline/revise).
package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"task140-reliability/internal/store"

	"github.com/google/uuid"
)

// Service is the reliability-engine use-case facade. Safe for concurrent use.
type Service struct {
	store *store.Store
	mu    sync.Mutex
}

// New builds a service over a store.
func New(st *store.Store) *Service {
	return &Service{store: st}
}

// now returns a UTC timestamp.
func now() time.Time { return time.Now().UTC() }

// newID returns a uuid v4 string.
func newID() string { return uuid.NewString() }

// ReconcileAll re-Solves every analysis from its persisted authoritative inputs
// (trees, diagrams, event logs) and overwrites the derived result rows. Used on
// startup so a crashed process converges to the same state. Returns the number
// of analyses re-solved. This is idempotent and deterministic.
func (svc *Service) ReconcileAll(ctx context.Context) (int, error) {
	ids, err := svc.store.ListAnalysisIDs(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, id := range ids {
		a, err := svc.store.GetAnalysis(ctx, id)
		if err != nil {
			return n, err
		}
		if _, err := svc.solve(ctx, a); err != nil {
			// an analysis missing all inputs has nothing to reconcile; skip
			// but continue. Real errors propagate.
			if !isNothingToSolve(err) {
				return n, fmt.Errorf("reconcile %s: %w", id, err)
			}
			continue
		}
		n++
	}
	return n, nil
}

// isNothingToSolve reports whether err means the analysis has no inputs yet.
func isNothingToSolve(err error) bool {
	return err == ErrNothingToSolve
}

// ErrNothingToSolve is returned when an analysis has no tree/diagram/events to solve.
var ErrNothingToSolve = fmt.Errorf("nothing to solve")
