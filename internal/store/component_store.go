package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"task140-reliability/internal/domain"
)

// ComponentStore methods for the component library.

func (s *Store) CreateComponent(ctx context.Context, c domain.Component) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO components(id,name,category,failure_rate_ppt,note,created_at) VALUES(?,?,?,?,?,?)`,
			c.ID, c.Name, c.Category, int64(c.FailureRatePPT), c.Note, c.CreatedAt.UTC().Format(time.RFC3339))
		return err
	})
}

func (s *Store) ListComponents(ctx context.Context) ([]domain.Component, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,category,failure_rate_ppt,note,created_at FROM components ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Component
	for rows.Next() {
		var c domain.Component
		var created string
		if err := rows.Scan(&c.ID, &c.Name, &c.Category, &c.FailureRatePPT, &c.Note, &created); err != nil {
			return nil, err
		}
		c.CreatedAt = parseTime(created)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetComponent(ctx context.Context, id string) (*domain.Component, error) {
	var c domain.Component
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id,name,category,failure_rate_ppt,note,created_at FROM components WHERE id=?`, id).
		Scan(&c.ID, &c.Name, &c.Category, &c.FailureRatePPT, &c.Note, &created)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: component %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	c.CreatedAt = parseTime(created)
	return &c, nil
}

func (s *Store) UpdateComponent(ctx context.Context, id string, name, category, note *string, rate *domain.FailureRatePPT) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		cur, err := s.getComponentForTx(ctx, tx, id)
		if err != nil {
			return err
		}
		if name != nil {
			cur.Name = *name
		}
		if category != nil {
			cur.Category = *category
		}
		if note != nil {
			cur.Note = *note
		}
		if rate != nil {
			cur.FailureRatePPT = *rate
		}
		_, err = tx.ExecContext(ctx, `UPDATE components SET name=?,category=?,failure_rate_ppt=?,note=? WHERE id=?`,
			cur.Name, cur.Category, int64(cur.FailureRatePPT), cur.Note, id)
		return err
	})
}

func (s *Store) getComponentForTx(ctx context.Context, tx *sql.Tx, id string) (*domain.Component, error) {
	var c domain.Component
	var created string
	err := tx.QueryRowContext(ctx, `SELECT id,name,category,failure_rate_ppt,note,created_at FROM components WHERE id=?`, id).
		Scan(&c.ID, &c.Name, &c.Category, &c.FailureRatePPT, &c.Note, &created)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: component %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	c.CreatedAt = parseTime(created)
	return &c, nil
}

func (s *Store) CreateFailureMode(ctx context.Context, fm domain.FailureMode) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		// ensure component exists (tx-scoped read to avoid deadlock)
		var cnt int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM components WHERE id=?`, fm.ComponentID).Scan(&cnt); err != nil {
			return err
		}
		if cnt == 0 {
			return fmt.Errorf("%w: component %s", domain.ErrDependentMissing, fm.ComponentID)
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO failure_modes(id,component_id,name,effect,severity,occurrence,detection) VALUES(?,?,?,?,?,?,?)`,
			fm.ID, fm.ComponentID, fm.Name, fm.Effect, fm.Severity, fm.Occurrence, fm.Detection)
		return err
	})
}

func (s *Store) ListFailureModes(ctx context.Context, componentID string) ([]domain.FailureMode, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,component_id,name,effect,severity,occurrence,detection FROM failure_modes WHERE component_id=? ORDER BY id`, componentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FailureMode
	for rows.Next() {
		var fm domain.FailureMode
		if err := rows.Scan(&fm.ID, &fm.ComponentID, &fm.Name, &fm.Effect, &fm.Severity, &fm.Occurrence, &fm.Detection); err != nil {
			return nil, err
		}
		out = append(out, fm)
	}
	return out, rows.Err()
}

// parseTime parses an RFC3339 string stored in the DB.
func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
