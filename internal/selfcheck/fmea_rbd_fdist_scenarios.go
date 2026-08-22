package selfcheck

import (
	"fmt"
	"math"
	"net/http/httptest"
)

// smokeFMEARPN: add two rows, one with severity 9 (floor => high), one with
// severity 5 RPN 250 (=> high). Solve and verify high_count=2.
func smokeFMEARPN(srv *httptest.Server, dbPath string) error {
	aid, err := createAnalysis(srv, "fmea", "fmea")
	if err != nil {
		return err
	}
	// row1: S=9 => high (floor), tiny RPN
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/fmea/rows", map[string]any{
		"function": "pump", "failure_mode": "seal-leak", "effect": "leak",
		"severity": 9, "occurrence": 1, "detection": 1,
	}, nil); err != nil {
		return err
	}
	// row2: S=5, RPN 250 => high
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/fmea/rows", map[string]any{
		"function": "motor", "failure_mode": "overheat", "effect": "stop",
		"severity": 5, "occurrence": 10, "detection": 5,
	}, nil); err != nil {
		return err
	}
	// row3: S=5, RPN 50 => low
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/fmea/rows", map[string]any{
		"function": "bearing", "failure_mode": "wear", "effect": "noise",
		"severity": 5, "occurrence": 2, "detection": 5,
	}, nil); err != nil {
		return err
	}
	var res map[string]any
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/solve", nil, &res); err != nil {
		return err
	}
	fmea, ok := res["fmea"].(map[string]any)
	if !ok {
		return fmt.Errorf("no fmea in result: %v", res)
	}
	if !numEq(fmea["high_count"], 2) {
		return fmt.Errorf("high_count = %v, want 2", fmea["high_count"])
	}
	if !numEq(fmea["low_count"], 1) {
		return fmt.Errorf("low_count = %v, want 1", fmea["low_count"])
	}
	if !numEq(fmea["max_rpn"], 250) {
		return fmt.Errorf("max_rpn = %v, want 250", fmea["max_rpn"])
	}
	// compliant=false (has high rows)
	if res["compliant"] != false {
		return fmt.Errorf("compliant = %v, want false", res["compliant"])
	}
	// invalid rating rejected
	if err := expectCode(srv, "POST", "/analyses/"+aid+"/fmea/rows",
		map[string]any{"function": "x", "failure_mode": "y", "severity": 0, "occurrence": 1, "detection": 1}, 422); err != nil {
		return err
	}
	return nil
}

// smokeRBD: series(A,B) reliability uses the analysis default 8760-hour
// mission, and k>n is rejected.
func smokeRBD(srv *httptest.Server, dbPath string) error {
	aid, err := createAnalysis(srv, "rbd", "rbd")
	if err != nil {
		return err
	}
	diag := map[string]any{
		"analysis_id": aid, "root_id": "ROOT",
		"blocks": []map[string]any{
			{"id": "ROOT", "analysis_id": aid, "type": "series", "children": []string{"A", "B"}},
			{"id": "A", "analysis_id": aid, "type": "block", "failure_rate_ppt": 10_000_000},
			{"id": "B", "analysis_id": aid, "type": "block", "failure_rate_ppt": 10_000_000},
		},
	}
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/rbd", diag, nil); err != nil {
		return err
	}
	var res map[string]any
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/solve", nil, &res); err != nil {
		return err
	}
	rbd, ok := res["rbd"].(map[string]any)
	if !ok {
		return fmt.Errorf("no rbd in result: %v", res)
	}
	want := math.Exp(-2 * 1e-5 * 8760)
	if !numEqTol(rbd["root_reliability"], want, 1e-5) {
		return fmt.Errorf("series reliability = %v, want %v", rbd["root_reliability"], want)
	}
	if rbd["repairable"] != false {
		return fmt.Errorf("non-repairable expected")
	}

	// k>n rejected at solve
	aid2, _ := createAnalysis(srv, "rbd-bad", "rbd-bad")
	bad := map[string]any{
		"analysis_id": aid2, "root_id": "ROOT",
		"blocks": []map[string]any{
			{"id": "ROOT", "analysis_id": aid2, "type": "kofn", "k": 5, "children": []string{"A", "B"}},
			{"id": "A", "analysis_id": aid2, "type": "block", "failure_rate_ppt": 10_000_000},
			{"id": "B", "analysis_id": aid2, "type": "block", "failure_rate_ppt": 10_000_000},
		},
	}
	if err := mustDo(srv, "POST", "/analyses/"+aid2+"/rbd", bad, nil); err != nil {
		return err
	}
	if err := expectCode(srv, "POST", "/analyses/"+aid2+"/solve", nil, 422); err != nil {
		return err
	}
	return nil
}

// smokeFailureDataFit: three distinct events select the documented Weibull
// median-rank fit, while availability remains the direct event-log ratio.
func smokeFailureDataFit(srv *httptest.Server, dbPath string) error {
	aid, err := createAnalysis(srv, "fdist", "fdist")
	if err != nil {
		return err
	}
	for _, tt := range []int{100, 200, 300} {
		if err := mustDo(srv, "POST", "/analyses/"+aid+"/failure-events", map[string]any{
			"ttf_hours": tt, "ttr_hours": 5,
		}, nil); err != nil {
			return err
		}
	}
	var fit map[string]any
	if err := mustDo(srv, "GET", "/analyses/"+aid+"/failure-events/fit", nil, &fit); err != nil {
		return err
	}
	if fit["distribution"] != "weibull" {
		return fmt.Errorf("distribution = %v, want weibull", fit["distribution"])
	}
	if beta, ok := fit["beta"].(float64); !ok || beta <= 0 {
		return fmt.Errorf("weibull beta = %v, want positive", fit["beta"])
	}
	// availability: uptime=600, downtime=15 => 600/615
	want := 600.0 / 615.0
	if !numEqTol(fit["availability"], want, 1e-6) {
		return fmt.Errorf("availability = %v, want %v", fit["availability"], want)
	}
	return nil
}

// smokeLifecycle: solve -> review -> baseline -> revise(v2).
func smokeLifecycle(srv *httptest.Server, dbPath string) error {
	aid, err := createAnalysis(srv, "life", "life")
	if err != nil {
		return err
	}
	// need something to solve: add an FMEA row
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/fmea/rows", map[string]any{
		"function": "f", "failure_mode": "m", "severity": 5, "occurrence": 5, "detection": 5,
	}, nil); err != nil {
		return err
	}
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/solve", nil, nil); err != nil {
		return err
	}
	var a map[string]any
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/review", nil, &a); err != nil {
		return err
	}
	if a["state"] != "reviewed" {
		return fmt.Errorf("review: state = %v, want reviewed", a["state"])
	}
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/baseline", nil, &a); err != nil {
		return err
	}
	if a["state"] != "baselined" {
		return fmt.Errorf("baseline: state = %v, want baselined", a["state"])
	}
	// editing a baselined analysis should fail
	if err := expectCode(srv, "POST", "/analyses/"+aid+"/fmea/rows",
		map[string]any{"function": "x", "failure_mode": "y", "severity": 5, "occurrence": 5, "detection": 5}, 409); err != nil {
		return err
	}
	// revise => v2 draft
	if err := mustDo(srv, "POST", "/analyses/"+aid+"/revise", nil, &a); err != nil {
		return err
	}
	if a["state"] != "draft" {
		return fmt.Errorf("revise: state = %v, want draft", a["state"])
	}
	if !numEq(a["version"], 2) {
		return fmt.Errorf("revise: version = %v, want 2", a["version"])
	}
	return nil
}
