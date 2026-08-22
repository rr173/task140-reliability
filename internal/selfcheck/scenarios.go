package selfcheck

import (
	"fmt"
	"net/http/httptest"
)

// smokeComponentLibrary: create a component, add a failure mode, verify list.
func smokeComponentLibrary(srv *httptest.Server, dbPath string) error {
	var c map[string]any
	if err := mustDo(srv, "POST", "/components", map[string]any{
		"name": "Centrifugal Pump", "category": "rotating", "note": "main feed",
		"failure_rate_ppt": 1_000_000,
	}, &c); err != nil {
		return err
	}
	id, _ := requireField(srv, c, "id")
	if got := fieldOf(c, "failure_rate_ppt"); !numEq(got, 1000000) {
		return fmt.Errorf("failure_rate_ppt = %v, want 1000000", got)
	}
	var fm map[string]any
	if err := mustDo(srv, "POST", "/components/"+fmt.Sprint(id)+"/failure-modes", map[string]any{
		"name": "seal-leak", "effect": "loss of containment", "severity": 8, "occurrence": 5, "detection": 3,
	}, &fm); err != nil {
		return err
	}
	// invalid rating rejected
	if err := expectCode(srv, "POST", "/components/"+fmt.Sprint(id)+"/failure-modes",
		map[string]any{"name": "bad", "severity": 11, "occurrence": 1, "detection": 1}, 422); err != nil {
		return err
	}
	var modes []map[string]any
	if err := mustDo(srv, "GET", "/components/"+fmt.Sprint(id)+"/failure-modes", nil, &modes); err != nil {
		return err
	}
	if len(modes) != 1 {
		return fmt.Errorf("want 1 failure mode, got %d", len(modes))
	}
	return nil
}

// createAnalysis creates an analysis and returns its id.
func createAnalysis(srv *httptest.Server, name, top string) (string, error) {
	var a map[string]any
	if err := mustDo(srv, "POST", "/analyses", map[string]any{"name": name, "top_event": top}, &a); err != nil {
		return "", err
	}
	id, _ := requireField(srv, a, "id")
	return fmt.Sprint(id), nil
}

// saveFTA posts a fault tree to an analysis.
func saveFTA(srv *httptest.Server, aid string, tree map[string]any) error {
	return mustDo(srv, "POST", "/analyses/"+aid+"/fta", tree, nil)
}

var _ = httptest.NewServer
