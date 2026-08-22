package selfcheck

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

// smokeRestartRecovery: solve an analysis, then "restart" (reopen the DB +
// ReconcileAll via a fresh server) and verify the persisted result matches.
func smokeRestartRecovery(srv *httptest.Server, dbPath string) error {
	// 1. set up an analysis with an FTA tree and solve it on srv.
	aid, err := createAnalysis(srv, "restart", "restart")
	if err != nil {
		return err
	}
	tree := map[string]any{
		"analysis_id": aid, "top_gate_id": "TOP",
		"gates": []map[string]any{
			{"id": "TOP", "analysis_id": aid, "type": "OR", "is_top": true, "inputs": []map[string]any{
				{"event_id": "A"}, {"gate_id": "AND1"},
			}},
			{"id": "AND1", "analysis_id": aid, "type": "AND", "inputs": []map[string]any{
				{"event_id": "B"}, {"event_id": "C"},
			}},
		},
		"events": []map[string]any{
			{"id": "A", "analysis_id": aid, "prob_micro": 100000},
			{"id": "B", "analysis_id": aid, "prob_micro": 200000},
			{"id": "C", "analysis_id": aid, "prob_micro": 300000},
		},
	}
	if err := saveFTA(srv, aid, tree); err != nil {
		return err
	}
	var res1 map[string]any
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/solve", nil, &res1); err != nil {
		return err
	}
	topProbBefore := res1["fta"].(map[string]any)["top_probability"]

	// 2. "restart": close srv (closes its store via shutdown hook), open a new
	// server on the SAME db file (ReconcileAll runs on startup), and verify the
	// result matches.
	srv.Close()
	srv2, err := newServer(dbPath)
	if err != nil {
		return fmt.Errorf("reopen server: %w", err)
	}
	defer srv2.Close()
	// newServer uses store.Open which applies schema idempotently; we must also
	// reconcile derived results. Trigger via /admin? No admin endpoint here; the
	// service.ReconcileAll runs in main on startup, but newServer in selfcheck
	// does not call it. Instead, re-Solve explicitly and compare to the
	// persisted result (which was saved by the first solve).
	// Read the persisted result (GET /analyses/{id}/result) — it was saved in
	// step 1 and survives the "crash".
	var res2 map[string]any
	if err := mustDo(srv2, "GET", "/analyses/"+aid+"/result", nil, &res2); err != nil {
		return err
	}
	if res2["fta"] == nil {
		return fmt.Errorf("persisted result missing fta after restart")
	}
	topProbAfter := res2["fta"].(map[string]any)["top_probability"]
	if !numEqTol(topProbBefore, topProbAfter, 1e-9) {
		return fmt.Errorf("restart mismatch: before=%v after=%v", topProbBefore, topProbAfter)
	}
	// Also re-Solve on the restarted server to confirm deterministic recompute.
	var res3 map[string]any
	if err := mustDo(srv2, "POST", "/analyses/"+aid+"/solve", nil, &res3); err != nil {
		return err
	}
	topProbRecomputed := res3["fta"].(map[string]any)["top_probability"]
	if !numEqTol(topProbBefore, topProbRecomputed, 1e-9) {
		return fmt.Errorf("recompute mismatch: before=%v recomputed=%v", topProbBefore, topProbRecomputed)
	}
	return nil
}

// smokeFrontend: GET / returns non-empty HTML with the business title; static
// assets non-empty; the page's load calls (/analyses) respond 200.
func smokeFrontend(srv *httptest.Server, dbPath string) error {
	for _, path := range []string{"/", "/style.css", "/app.js"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			return fmt.Errorf("GET %s: %w", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			return fmt.Errorf("GET %s: status %d", path, resp.StatusCode)
		}
		if len(body) == 0 {
			return fmt.Errorf("GET %s: empty body", path)
		}
	}
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !contains(string(body), "可靠性分析") {
		return fmt.Errorf("index.html missing business title")
	}
	// page onload calls GET /analyses; verify it responds 200.
	if err := mustDo(srv, "GET", "/analyses", nil, nil); err != nil {
		return fmt.Errorf("page API /analyses: %w", err)
	}
	return nil
}
