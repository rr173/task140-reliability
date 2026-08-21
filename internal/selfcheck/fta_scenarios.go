package selfcheck

import (
	"fmt"
	"net/http/httptest"
)

// smokeFTACutSets: top OR(A, AND(B,C)). Cut sets {A},{B,C}. Exact probability
// = P(A)+P(B*C) = 0.1 + 0.2*0.3 = 0.16. Method "exact".
func smokeFTACutSets(srv *httptest.Server, dbPath string) error {
	aid, err := createAnalysis(srv, "fta-basic", "top-pump-fails")
	if err != nil {
		return err
	}
	tree := map[string]any{
		"analysis_id": aid,
		"top_gate_id": "TOP",
		"gates": []map[string]any{
			{"id": "TOP", "analysis_id": aid, "type": "OR", "is_top": true, "inputs": []map[string]any{
				{"event_id": "A"}, {"gate_id": "AND1"},
			}},
			{"id": "AND1", "analysis_id": aid, "type": "AND", "inputs": []map[string]any{
				{"event_id": "B"}, {"event_id": "C"},
			}},
		},
		"events": []map[string]any{
			{"id": "A", "analysis_id": aid, "label": "A", "prob_micro": 100000},
			{"id": "B", "analysis_id": aid, "label": "B", "prob_micro": 200000},
			{"id": "C", "analysis_id": aid, "label": "C", "prob_micro": 300000},
		},
	}
	if err := saveFTA(srv, aid, tree); err != nil {
		return err
	}
	var res map[string]any
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/solve", nil, &res); err != nil {
		return err
	}
	fta, _ := requireField(srv, res, "fta")
	fm := fta.(map[string]any)
	if got := fm["method"]; got != "exact" {
		return fmt.Errorf("method = %v, want exact", got)
	}
	// exact inclusion-exclusion: P(A∪BC)=P(A)+P(BC)-P(A)P(BC)=0.1+0.06-0.006=0.154
	if !numEqTol(fm["top_probability"], 0.154, 1e-9) {
		return fmt.Errorf("top_probability = %v, want 0.154", fm["top_probability"])
	}
	cs := fm["cut_sets"].([]any)
	if len(cs) != 2 {
		return fmt.Errorf("want 2 cut sets, got %d", len(cs))
	}
	return nil
}

// smokeAbsorptionTruncation: AND(A, OR(A,B)) => {A} absorbs {A,B}; max_order=1
// truncates nothing here but verifies absorption. A separate truncation check.
func smokeAbsorptionTruncation(srv *httptest.Server, dbPath string) error {
	aid, err := createAnalysis(srv, "fta-absorb", "absorb")
	if err != nil {
		return err
	}
	// AND(A, OR(A,B)) with default max_order=4: minimal cut set {A}
	tree := map[string]any{
		"analysis_id": aid, "top_gate_id": "TOP",
		"gates": []map[string]any{
			{"id": "TOP", "analysis_id": aid, "type": "AND", "is_top": true, "inputs": []map[string]any{
				{"event_id": "A"}, {"gate_id": "OR1"},
			}},
			{"id": "OR1", "analysis_id": aid, "type": "OR", "inputs": []map[string]any{
				{"event_id": "A"}, {"event_id": "B"},
			}},
		},
		"events": []map[string]any{
			{"id": "A", "analysis_id": aid, "prob_micro": 100000},
			{"id": "B", "analysis_id": aid, "prob_micro": 200000},
		},
	}
	if err := saveFTA(srv, aid, tree); err != nil {
		return err
	}
	var res map[string]any
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/solve", nil, &res); err != nil {
		return err
	}
	fta := res["fta"].(map[string]any)
	cs := fta["cut_sets"].([]any)
	if len(cs) != 1 {
		return fmt.Errorf("absorption: want 1 minimal cut set, got %d", len(cs))
	}

	// truncation: 4-event AND with max_order=2
	aid2, err := createAnalysis(srv, "fta-trunc", "trunc")
	if err != nil {
		return err
	}
	// set max_order=2 via PATCH
	if err := mustDo(srv, "PATCH", "/analyses/"+aid2, map[string]any{
		"name": "fta-trunc", "top_event": "trunc", "max_order": 2, "s_crit": 8, "rpn_high": 200, "rpn_medium": 100, "exact_limit": 8, "ccf_beta": 0.1, "mission_hours": 8760,
	}, nil); err != nil {
		return err
	}
	tree2 := map[string]any{
		"analysis_id": aid2, "top_gate_id": "TOP",
		"gates": []map[string]any{
			{"id": "TOP", "analysis_id": aid2, "type": "AND", "is_top": true, "inputs": []map[string]any{
				{"event_id": "A"}, {"event_id": "B"}, {"event_id": "C"}, {"event_id": "D"},
			}},
		},
		"events": []map[string]any{
			{"id": "A", "analysis_id": aid2, "prob_micro": 50000},
			{"id": "B", "analysis_id": aid2, "prob_micro": 50000},
			{"id": "C", "analysis_id": aid2, "prob_micro": 50000},
			{"id": "D", "analysis_id": aid2, "prob_micro": 50000},
		},
	}
	if err := saveFTA(srv, aid2, tree2); err != nil {
		return err
	}
	var res2 map[string]any
	if err := mustDo(srv, "POST", "/analyses/"+aid2+"/solve", nil, &res2); err != nil {
		return err
	}
	fta2 := res2["fta"].(map[string]any)
	cs2 := fta2["cut_sets"].([]any)
	if len(cs2) != 0 {
		return fmt.Errorf("truncation: want 0 cut sets, got %d", len(cs2))
	}
	if !numEq(fta2["truncated_count"], 1) {
		return fmt.Errorf("truncated_count = %v, want 1", fta2["truncated_count"])
	}
	return nil
}

// smokeNonCoherent: a negated event flags non_coherent.
func smokeNonCoherent(srv *httptest.Server, dbPath string) error {
	aid, err := createAnalysis(srv, "fta-neg", "neg")
	if err != nil {
		return err
	}
	tree := map[string]any{
		"analysis_id": aid, "top_gate_id": "TOP",
		"gates": []map[string]any{
			{"id": "TOP", "analysis_id": aid, "type": "OR", "is_top": true, "inputs": []map[string]any{
				{"event_id": "A"},
			}},
		},
		"events": []map[string]any{
			{"id": "A", "analysis_id": aid, "prob_micro": 100000, "negated": true},
		},
	}
	if err := saveFTA(srv, aid, tree); err != nil {
		return err
	}
	var res map[string]any
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/solve", nil, &res); err != nil {
		return err
	}
	fta := res["fta"].(map[string]any)
	coh := fta["coherence"].(map[string]any)
	if coh["coherent"] != false {
		return fmt.Errorf("expected non-coherent, got coherent=%v", coh["coherent"])
	}
	return nil
}
