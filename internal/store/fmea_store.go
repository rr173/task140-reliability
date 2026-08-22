package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"task140-reliability/internal/domain"
)

// SaveFMEATable replaces the FMEA rows of an analysis atomically.
func (s *Store) SaveFMEATable(ctx context.Context, analysisID string, rows []domain.FMEARow) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM fmea_rows WHERE analysis_id=?`, analysisID); err != nil {
			return err
		}
		for _, r := range rows {
			if _, err := tx.ExecContext(ctx, `INSERT INTO fmea_rows(id,analysis_id,function,failure_mode,effect,severity,occurrence,detection,rpn,risk_class,action,action_state,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				r.ID, analysisID, r.Function, r.FailureMode, r.Effect, r.Severity, r.Occurrence, r.Detection, r.RPN, string(r.RiskClass), r.Action, r.ActionState, r.CreatedAt.UTC().Format(time.RFC3339)); err != nil {
				return err
			}
		}
		return nil
	})
}

// AddFMEARow inserts one FMEA row.
func (s *Store) AddFMEARow(ctx context.Context, r domain.FMEARow) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		// analysis must exist & not be baselined
		cur, err := s.getAnalysisForTx(ctx, tx, r.AnalysisID)
		if err != nil {
			return err
		}
		if cur.State == domain.StateBaselined {
			return fmt.Errorf("%w: analysis %s is baselined", domain.ErrStateConflict, r.AnalysisID)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO fmea_rows(id,analysis_id,function,failure_mode,effect,severity,occurrence,detection,rpn,risk_class,action,action_state,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			r.ID, r.AnalysisID, r.Function, r.FailureMode, r.Effect, r.Severity, r.Occurrence, r.Detection, r.RPN, string(r.RiskClass), r.Action, r.ActionState, r.CreatedAt.UTC().Format(time.RFC3339))
		return err
	})
}

// UpdateFMEARow patches an existing FMEA row (function/mode/effect/ratings/action/state).
func (s *Store) UpdateFMEARow(ctx context.Context, id string, fn, mode, effect *string, sev, occ, det *int, action, actionState *string) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		row, err := s.getFMEARowForTx(ctx, tx, id)
		if err != nil {
			return err
		}
		// analysis must not be baselined
		cur, err := s.getAnalysisForTx(ctx, tx, row.AnalysisID)
		if err != nil {
			return err
		}
		if cur.State == domain.StateBaselined {
			return fmt.Errorf("%w: analysis %s is baselined", domain.ErrStateConflict, row.AnalysisID)
		}
		if fn != nil {
			row.Function = *fn
		}
		if mode != nil {
			row.FailureMode = *mode
		}
		if effect != nil {
			row.Effect = *effect
		}
		if sev != nil {
			row.Severity = *sev
		}
		if occ != nil {
			row.Occurrence = *occ
		}
		if det != nil {
			row.Detection = *det
		}
		if action != nil {
			row.Action = *action
		}
		if actionState != nil {
			row.ActionState = *actionState
		}
		row.RPN = row.Severity * row.Occurrence * row.Detection
		_, err = tx.ExecContext(ctx, `UPDATE fmea_rows SET function=?,failure_mode=?,effect=?,severity=?,occurrence=?,detection=?,rpn=?,action=?,action_state=? WHERE id=?`,
			row.Function, row.FailureMode, row.Effect, row.Severity, row.Occurrence, row.Detection, row.RPN, row.Action, row.ActionState, id)
		return err
	})
}

// DeleteFMEARow removes an FMEA row.
func (s *Store) DeleteFMEARow(ctx context.Context, id string) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		row, err := s.getFMEARowForTx(ctx, tx, id)
		if err != nil {
			return err
		}
		cur, err := s.getAnalysisForTx(ctx, tx, row.AnalysisID)
		if err != nil {
			return err
		}
		if cur.State == domain.StateBaselined {
			return fmt.Errorf("%w: analysis %s is baselined", domain.ErrStateConflict, row.AnalysisID)
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM fmea_rows WHERE id=?`, id)
		return err
	})
}

func (s *Store) getFMEARowForTx(ctx context.Context, tx *sql.Tx, id string) (*domain.FMEARow, error) {
	var r domain.FMEARow
	var rclass, created string
	err := tx.QueryRowContext(ctx, `SELECT id,analysis_id,function,failure_mode,effect,severity,occurrence,detection,rpn,risk_class,action,action_state,created_at FROM fmea_rows WHERE id=?`, id).
		Scan(&r.ID, &r.AnalysisID, &r.Function, &r.FailureMode, &r.Effect, &r.Severity, &r.Occurrence, &r.Detection, &r.RPN, &rclass, &r.Action, &r.ActionState, &created)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: fmea row %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	r.RiskClass = domain.RiskClass(rclass)
	r.CreatedAt = parseTime(created)
	return &r, nil
}

// GetFMEARow reads a single FMEA row by ID (outside-tx).
func (s *Store) GetFMEARow(ctx context.Context, id string) (*domain.FMEARow, error) {
	var r domain.FMEARow
	var rclass, created string
	err := s.db.QueryRowContext(ctx, `SELECT id,analysis_id,function,failure_mode,effect,severity,occurrence,detection,rpn,risk_class,action,action_state,created_at FROM fmea_rows WHERE id=?`, id).
		Scan(&r.ID, &r.AnalysisID, &r.Function, &r.FailureMode, &r.Effect, &r.Severity, &r.Occurrence, &r.Detection, &r.RPN, &rclass, &r.Action, &r.ActionState, &created)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: fmea row %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	r.RiskClass = domain.RiskClass(rclass)
	r.CreatedAt = parseTime(created)
	return &r, nil
}

// LoadFMEATable reconstructs the FMEA table from the database.
func (s *Store) LoadFMEATable(ctx context.Context, analysisID string) (*domain.FMEATable, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,analysis_id,function,failure_mode,effect,severity,occurrence,detection,rpn,risk_class,action,action_state,created_at FROM fmea_rows WHERE analysis_id=? ORDER BY created_at`, analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	table := &domain.FMEATable{AnalysisID: analysisID}
	for rows.Next() {
		var r domain.FMEARow
		var rclass, created string
		if err := rows.Scan(&r.ID, &r.AnalysisID, &r.Function, &r.FailureMode, &r.Effect, &r.Severity, &r.Occurrence, &r.Detection, &r.RPN, &rclass, &r.Action, &r.ActionState, &created); err != nil {
			return nil, err
		}
		r.RiskClass = domain.RiskClass(rclass)
		r.CreatedAt = parseTime(created)
		r.RiskClass = domain.RiskLow
		table.Rows = append(table.Rows, r)
	}
	return table, rows.Err()
}
