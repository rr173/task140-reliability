package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"task140-reliability/internal/domain"
)

// SaveResult persists a solved result bundle for an analysis version. It is an
// upsert on (analysis_id, version).
func (s *Store) SaveResult(ctx context.Context, res *domain.AnalysisResult) error {
	if res == nil {
		return fmt.Errorf("nil result")
	}
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return s.InTx(ctx, func(tx *sql.Tx) error {
		compliant := 0
		if res.Compliant {
			compliant = 1
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO analysis_results(analysis_id,version,solved_at,result_json,compliant) VALUES(?,?,?,?,?)
ON CONFLICT(analysis_id,version) DO UPDATE SET solved_at=excluded.solved_at, result_json=excluded.result_json, compliant=excluded.compliant`,
			res.AnalysisID, res.Version, res.SolvedAt.UTC().Format(time.RFC3339), string(b), compliant)
		return err
	})
}

// LoadResult reads a persisted result bundle.
func (s *Store) LoadResult(ctx context.Context, analysisID string, version int) (*domain.AnalysisResult, error) {
	var b string
	var solved string
	var compl int
	err := s.db.QueryRowContext(ctx, `SELECT solved_at,result_json,compliant FROM analysis_results WHERE analysis_id=? AND version=?`, analysisID, version).
		Scan(&solved, &b, &compl)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("%w: result for %s v%d", domain.ErrNotFound, analysisID, version)
	}
	if err != nil {
		return nil, err
	}
	var res domain.AnalysisResult
	if err := json.Unmarshal([]byte(b), &res); err != nil {
		return nil, err
	}
	res.SolvedAt = parseTime(solved)
	res.Compliant = compl == 1
	return &res, nil
}

// ListAnalysisIDs returns all analysis IDs (for ReconcileAll).
func (s *Store) ListAnalysisIDs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM analyses ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
