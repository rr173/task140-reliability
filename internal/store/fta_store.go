package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"task140-reliability/internal/domain"
)

// SaveFTATree replaces the entire fault tree of an analysis atomically.
func (s *Store) SaveFTATree(ctx context.Context, tree domain.FTATree) error {
	return s.InTx(ctx, func(tx *sql.Tx) error {
		// clear existing
		if _, err := tx.ExecContext(ctx, `DELETE FROM fta_gate_inputs WHERE analysis_id=?`, tree.AnalysisID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM fta_gates WHERE analysis_id=?`, tree.AnalysisID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM fta_basic_events WHERE analysis_id=?`, tree.AnalysisID); err != nil {
			return err
		}
		for seq, g := range tree.Gates {
			isTop := 0
			if g.IsTop || g.ID == tree.TopGateID {
				isTop = 1
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO fta_gates(analysis_id,id,type,k,is_top,seq) VALUES(?,?,?,?,?,?)`,
				tree.AnalysisID, g.ID, string(g.Type), g.K, isTop, seq); err != nil {
				return err
			}
			for iseq, in := range g.Inputs {
				var gid, eid any
				if in.GateID != "" {
					gid = in.GateID
				}
				if in.EventID != "" {
					eid = in.EventID
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO fta_gate_inputs(analysis_id,gate_id,seq,child_gate_id,event_id) VALUES(?,?,?,?,?)`,
					tree.AnalysisID, g.ID, iseq, gid, eid); err != nil {
					return err
				}
			}
		}
		for _, e := range tree.Events {
			var compID any
			if e.ComponentID != "" {
				compID = e.ComponentID
			}
			neg := 0
			if e.Negated {
				neg = 1
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO fta_basic_events(analysis_id,id,label,component_id,prob_micro,ccf_group,negated) VALUES(?,?,?,?,?,?,?)`,
				tree.AnalysisID, e.ID, e.Label, compID, int64(e.ProbMicro), e.CCFGroup, neg); err != nil {
				return err
			}
		}
		return nil
	})
}

// LoadFTATree reconstructs the fault tree from the database. This is the
// authoritative input used by ReconcileAll to recompute cut sets.
func (s *Store) LoadFTATree(ctx context.Context, analysisID string) (*domain.FTATree, error) {
	tree := &domain.FTATree{AnalysisID: analysisID}
	grows, err := s.db.QueryContext(ctx, `SELECT id,analysis_id,type,k,is_top FROM fta_gates WHERE analysis_id=? ORDER BY seq`, analysisID)
	if err != nil {
		return nil, err
	}
	defer grows.Close()
	for grows.Next() {
		var g domain.Gate
		var gtype string
		var isTop, k int
		if err := grows.Scan(&g.ID, &g.AnalysisID, &gtype, &k, &isTop); err != nil {
			return nil, err
		}
		g.Type = domain.GateType(gtype)
		g.K = k
		g.IsTop = isTop == 1
		tree.Gates = append(tree.Gates, g)
	}
	if err := grows.Err(); err != nil {
		return nil, err
	}
	if len(tree.Gates) == 0 {
		return tree, nil
	}
	// load inputs per gate
	for i := range tree.Gates {
		g := &tree.Gates[i]
		irows, err := s.db.QueryContext(ctx, `SELECT child_gate_id,event_id FROM fta_gate_inputs WHERE analysis_id=? AND gate_id=? ORDER BY seq`, tree.AnalysisID, g.ID)
		if err != nil {
			return nil, err
		}
		for irows.Next() {
			var in domain.GateInput
			var gid, eid sql.NullString
			if err := irows.Scan(&gid, &eid); err != nil {
				irows.Close()
				return nil, err
			}
			if gid.Valid {
				in.GateID = gid.String
			}
			if eid.Valid {
				in.EventID = eid.String
			}
			g.Inputs = append(g.Inputs, in)
		}
		irows.Close()
	}
	// top gate
	for _, g := range tree.Gates {
		if g.IsTop {
			tree.TopGateID = g.ID
			break
		}
	}
	// events
	erows, err := s.db.QueryContext(ctx, `SELECT id,analysis_id,label,component_id,prob_micro,ccf_group,negated FROM fta_basic_events WHERE analysis_id=?`, analysisID)
	if err != nil {
		return nil, err
	}
	defer erows.Close()
	for erows.Next() {
		var e domain.BasicEvent
		var compID sql.NullString
		var label, ccf string
		var probMicro, neg int
		if err := erows.Scan(&e.ID, &e.AnalysisID, &label, &compID, &probMicro, &ccf, &neg); err != nil {
			return nil, err
		}
		e.Label = label
		if compID.Valid {
			e.ComponentID = compID.String
		}
		e.ProbMicro = domain.ProbMicro(probMicro)
		e.CCFGroup = ccf
		e.Negated = neg == 1
		tree.Events = append(tree.Events, e)
	}
	// sort events by id for deterministic output
	sort.Slice(tree.Events, func(i, j int) bool { return tree.Events[i].ID < tree.Events[j].ID })
	return tree, erows.Err()
}

// ResolveEventProbabilities replaces any basic event whose ComponentID is set
// (and ProbMicro is 0) with the component's failure-rate-derived probability.
// This is the bridge between the component library and FTA quantification.
func (s *Store) ResolveEventProbabilities(ctx context.Context, tree *domain.FTATree, missionHours int64) error {
	if tree == nil {
		return fmt.Errorf("nil tree")
	}
	compCache := make(map[string]*domain.Component)
	for i := range tree.Events {
		e := &tree.Events[i]
		if e.ComponentID == "" || e.ProbMicro != 0 {
			continue
		}
		comp, ok := compCache[e.ComponentID]
		if !ok {
			c, err := s.GetComponent(ctx, e.ComponentID)
			if err != nil {
				return err
			}
			comp = c
			compCache[e.ComponentID] = c
		}
		e.ProbMicro = domain.FailureRateToProb(comp.FailureRatePPT, missionHours)
	}
	return nil
}
