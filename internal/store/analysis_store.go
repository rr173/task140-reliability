package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"task140-reliability/internal/domain"
)

// CreateAnalysis inserts a new analysis project. It also seeds the first
// revision row.
func (s *Store) CreateAnalysis(ctx context.Context, a domain.Analysis) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO analyses
(id,name,top_event,state,version,max_order,s_crit,rpn_high,rpn_medium,exact_limit,ccf_beta,mission_hours,created_at,updated_at,baselined_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			a.ID, a.Name, a.TopEvent, string(a.State), a.Version, a.MaxOrder, a.SCrit,
			a.RPNHigh, a.RPNMedium, a.ExactLimit, a.CCFBeta, a.MissionHours,
			a.CreatedAt.UTC().Format(time.RFC3339), a.UpdatedAt.UTC().Format(time.RFC3339), nil)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO revisions(analysis_id,version,state,created_at,note) VALUES(?,?,?,?,?)`,
			a.ID, a.Version, string(a.State), time.Now().UTC().Format(time.RFC3339), "created")
		return err
	})
}

func (s *Store) ListAnalyses(ctx context.Context) ([]domain.Analysis, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,top_event,state,version,max_order,s_crit,rpn_high,rpn_medium,exact_limit,ccf_beta,mission_hours,created_at,updated_at,baselined_at FROM analyses ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Analysis
	for rows.Next() {
		a, err := scanAnalysis(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAnalysis(ctx context.Context, id string) (*domain.Analysis, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id,name,top_event,state,version,max_order,s_crit,rpn_high,rpn_medium,exact_limit,ccf_beta,mission_hours,created_at,updated_at,baselined_at FROM analyses WHERE id=?`, id)
	a, err := scanAnalysisRow(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: analysis %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) getAnalysisForTx(ctx context.Context, tx *sql.Tx, id string) (*domain.Analysis, error) {
	row := tx.QueryRowContext(ctx, `SELECT id,name,top_event,state,version,max_order,s_crit,rpn_high,rpn_medium,exact_limit,ccf_beta,mission_hours,created_at,updated_at,baselined_at FROM analyses WHERE id=?`, id)
	a, err := scanAnalysisRow(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: analysis %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func scanAnalysis(rows *sql.Rows) (domain.Analysis, error) {
	var a domain.Analysis
	var state, created, updated string
	var baselined sql.NullString
	err := rows.Scan(&a.ID, &a.Name, &a.TopEvent, &state, &a.Version, &a.MaxOrder, &a.SCrit,
		&a.RPNHigh, &a.RPNMedium, &a.ExactLimit, &a.CCFBeta, &a.MissionHours, &created, &updated, &baselined)
	if err != nil {
		return a, err
	}
	a.State = domain.AnalysisState(state)
	a.CreatedAt = parseTime(created)
	a.UpdatedAt = parseTime(updated)
	if baselined.Valid {
		t := parseTime(baselined.String)
		a.BaselinedAt = &t
	}
	return a, nil
}

func scanAnalysisRow(row *sql.Row) (domain.Analysis, error) {
	var a domain.Analysis
	var state, created, updated string
	var baselined sql.NullString
	err := row.Scan(&a.ID, &a.Name, &a.TopEvent, &state, &a.Version, &a.MaxOrder, &a.SCrit,
		&a.RPNHigh, &a.RPNMedium, &a.ExactLimit, &a.CCFBeta, &a.MissionHours, &created, &updated, &baselined)
	if err != nil {
		return a, err
	}
	a.State = domain.AnalysisState(state)
	a.CreatedAt = parseTime(created)
	a.UpdatedAt = parseTime(updated)
	if baselined.Valid {
		t := parseTime(baselined.String)
		a.BaselinedAt = &t
	}
	return a, nil
}

// SetAnalysisState updates state/version and the updated_at timestamp. It does
// not validate transitions (the service layer does).
func (s *Store) SetAnalysisState(ctx context.Context, id string, state domain.AnalysisState, version int, baselinedAt *time.Time) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		cur, err := s.getAnalysisForTx(ctx, tx, id)
		if err != nil {
			return err
		}
		// state transition guard: baselined is immutable; cannot edit a
		// baselined analysis except via revise (handled separately).
		if cur.State == domain.StateBaselined && state != domain.StateDraft {
			return fmt.Errorf("%w: analysis %s is baselined", domain.ErrStateConflict, id)
		}
		var bsql any
		if baselinedAt != nil {
			bsql = baselinedAt.UTC().Format(time.RFC3339)
		}
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = tx.ExecContext(ctx, `UPDATE analyses SET state=?,version=?,updated_at=?,baselined_at=COALESCE(?, baselined_at) WHERE id=?`,
			string(state), version, now, bsql, id)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO revisions(analysis_id,version,state,created_at,note) VALUES(?,?,?,?,?)`,
			id, version, string(state), now, string(state))
		if err != nil {
			// revision insert may conflict if same version; ignore unique violation
			return nil
		}
		return nil
	})
}

// UpdateAnalysisParams updates the project-level parameters of an analysis.
// Refuses a baselined analysis.
func (s *Store) UpdateAnalysisParams(ctx context.Context, id string, params domain.Analysis) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		cur, err := s.getAnalysisForTx(ctx, tx, id)
		if err != nil {
			return err
		}
		if cur.State == domain.StateBaselined {
			return fmt.Errorf("%w: analysis %s is baselined", domain.ErrStateConflict, id)
		}
		_, err = tx.ExecContext(ctx, `UPDATE analyses SET name=?,top_event=?,max_order=?,s_crit=?,rpn_high=?,rpn_medium=?,exact_limit=?,ccf_beta=?,mission_hours=?,updated_at=? WHERE id=?`,
			params.Name, params.TopEvent, params.MaxOrder, params.SCrit, params.RPNHigh, params.RPNMedium, params.ExactLimit, params.CCFBeta, params.MissionHours, time.Now().UTC().Format(time.RFC3339), id)
		return err
	})
}

func (s *Store) ListRevisions(ctx context.Context, analysisID string) ([]domain.Revision, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT analysis_id,version,state,created_at,note FROM revisions WHERE analysis_id=? ORDER BY version`, analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Revision
	for rows.Next() {
		var r domain.Revision
		var state, created string
		if err := rows.Scan(&r.AnalysisID, &r.Version, &state, &created, &r.Note); err != nil {
			return nil, err
		}
		r.State = domain.AnalysisState(state)
		r.CreatedAt = parseTime(created)
		out = append(out, r)
	}
	return out, rows.Err()
}
