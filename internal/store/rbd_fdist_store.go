package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"task140-reliability/internal/domain"
)

// SaveRBDDiagram replaces the block diagram of an analysis atomically.
func (s *Store) SaveRBDDiagram(ctx context.Context, diag domain.RBDDiagram) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM rbd_children WHERE analysis_id=?`, diag.AnalysisID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM rbd_blocks WHERE analysis_id=?`, diag.AnalysisID); err != nil {
			return err
		}
		for _, b := range diag.Blocks {
			isRoot := 0
			if b.ID == diag.RootID {
				isRoot = 1
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO rbd_blocks(analysis_id,id,name,type,k,failure_rate_ppt,repair_rate_ppt,is_root) VALUES(?,?,?,?,?,?,?,?)`,
				diag.AnalysisID, b.ID, b.Name, string(b.Type), b.K, int64(b.FailureRatePPT), int64(b.RepairRatePPT), isRoot); err != nil {
				return err
			}
			for seq, cid := range b.Children {
				if _, err := tx.ExecContext(ctx, `INSERT INTO rbd_children(analysis_id,block_id,seq,child_id) VALUES(?,?,?,?)`,
					diag.AnalysisID, b.ID, seq, cid); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// LoadRBDDiagram reconstructs the block diagram from the database.
func (s *Store) LoadRBDDiagram(ctx context.Context, analysisID string) (*domain.RBDDiagram, error) {
	diag := &domain.RBDDiagram{AnalysisID: analysisID}
	brows, err := s.db.QueryContext(ctx, `SELECT id,analysis_id,name,type,k,failure_rate_ppt,repair_rate_ppt,is_root FROM rbd_blocks WHERE analysis_id=? ORDER BY id`, analysisID)
	if err != nil {
		return nil, err
	}
	defer brows.Close()
	for brows.Next() {
		var b domain.RBDBlock
		var btype string
		var isRoot, k int
		var fr, rr int64
		if err := brows.Scan(&b.ID, &b.AnalysisID, &b.Name, &btype, &k, &fr, &rr, &isRoot); err != nil {
			return nil, err
		}
		b.Type = domain.BlockType(btype)
		b.K = k
		b.FailureRatePPT = domain.FailureRatePPT(fr)
		b.RepairRatePPT = domain.FailureRatePPT(rr)
		if isRoot == 1 {
			diag.RootID = b.ID
		}
		diag.Blocks = append(diag.Blocks, b)
	}
	if err := brows.Err(); err != nil {
		return nil, err
	}
	if len(diag.Blocks) == 0 {
		return diag, nil
	}
	for i := range diag.Blocks {
		b := &diag.Blocks[i]
		crows, err := s.db.QueryContext(ctx, `SELECT child_id FROM rbd_children WHERE analysis_id=? AND block_id=? ORDER BY seq`, diag.AnalysisID, b.ID)
		if err != nil {
			return nil, err
		}
		for crows.Next() {
			var cid string
			if err := crows.Scan(&cid); err != nil {
				crows.Close()
				return nil, err
			}
			b.Children = append(b.Children, cid)
		}
		crows.Close()
	}
	return diag, nil
}

// CreateFailureEvent appends a failure event to the event log of an analysis.
func (s *Store) CreateFailureEvent(ctx context.Context, e domain.FailureEvent) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		cur, err := s.getAnalysisForTx(ctx, tx, e.AnalysisID)
		if err != nil {
			return err
		}
		if cur.State == domain.StateBaselined {
			return fmt.Errorf("%w: analysis %s is baselined", domain.ErrStateConflict, e.AnalysisID)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO failure_events(id,analysis_id,component_id,ttf_hours,ttr_hours,occurred_at) VALUES(?,?,?,?,?,?)`,
			e.ID, e.AnalysisID, nullableStr(e.ComponentID), e.TTFHours, e.TTRHours, e.OccurredAt.UTC().Format(time.RFC3339))
		return err
	})
}

// LoadFailureEvents reconstructs the event log of an analysis.
func (s *Store) LoadFailureEvents(ctx context.Context, analysisID string) ([]domain.FailureEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,analysis_id,component_id,ttf_hours,ttr_hours,occurred_at FROM failure_events WHERE analysis_id=? ORDER BY occurred_at`, analysisID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FailureEvent
	for rows.Next() {
		var e domain.FailureEvent
		var compID sql.NullString
		var occ string
		if err := rows.Scan(&e.ID, &e.AnalysisID, &compID, &e.TTFHours, &e.TTRHours, &occ); err != nil {
			return nil, err
		}
		if compID.Valid {
			e.ComponentID = compID.String
		}
		e.OccurredAt = parseTime(occ)
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullableStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
