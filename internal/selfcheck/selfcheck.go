// Package selfcheck runs the --smoke-test: an end-to-end, deterministic
// exercise of the reliability engine contract (component library, fault-tree
// cut sets & probability, FMEA RPN & severity floor, RBD reduction, failure-
// data fit, analysis lifecycle, restart recovery, and the frontend page).
package selfcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"task140-reliability/internal/httpapi"
	"task140-reliability/internal/service"
	"task140-reliability/internal/store"
	"task140-reliability/internal/webfs"
)

// Run executes all smoke scenarios. Returns nil on success.
func Run() error {
	dir, err := os.MkdirTemp("", "reliability-smoke-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)

	cases := []struct {
		name string
		fn   func(srv *httptest.Server, dbPath string) error
	}{
		{"component-library", smokeComponentLibrary},
		{"fta-cutsets-probability", smokeFTACutSets},
		{"fta-absorption-truncation", smokeAbsorptionTruncation},
		{"fta-noncoherent", smokeNonCoherent},
		{"fmea-rpn-severity-floor", smokeFMEARPN},
		{"rbd-series-parallel-kofn", smokeRBD},
		{"failure-data-fit", smokeFailureDataFit},
		{"lifecycle-baseline-revise", smokeLifecycle},
		{"restart-recovery", smokeRestartRecovery},
		{"frontend-page-served", smokeFrontend},
	}
	for i, c := range cases {
		dbPath := filepath.Join(dir, fmt.Sprintf("smoke-%02d.db", i))
		srv, err := newServer(dbPath)
		if err != nil {
			return fmt.Errorf("%s: new server: %w", c.name, err)
		}
		if err := c.fn(srv, dbPath); err != nil {
			srv.Close()
			return fmt.Errorf("%s: %w", c.name, err)
		}
		srv.Close()
	}
	return nil
}

// newServer opens a fresh SQLite file, builds the service + mux and returns an
// httptest server over the real HTTP handler tree.
func newServer(dbPath string) (*httptest.Server, error) {
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	svc := service.New(st)
	webFS := http.FS(webfs.FS())
	mux := httpapi.NewMux(httpapi.Services{Svc: svc}, webFS)
	srv := httptest.NewServer(mux)
	srv.Config.RegisterOnShutdown(func() { _ = st.Close() })
	return srv, nil
}

// doJSON performs an HTTP call and returns status + body.
func doJSON(srv *httptest.Server, method, path string, body any) (int, []byte, error) {
	var r *http.Request
	if body == nil {
		r = httptest.NewRequest(method, path, nil)
	} else {
		buf, _ := json.Marshal(body)
		r = httptest.NewRequest(method, path, bytes.NewReader(buf))
		r.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	srv.Config.Handler.ServeHTTP(rec, r)
	return rec.Code, rec.Body.Bytes(), nil
}

// mustDo performs an HTTP call and returns the decoded body; fails the scenario
// immediately on a non-2xx status (success-expecting helpers must surface
// non-200s, not swallow them).
func mustDo(srv *httptest.Server, method, path string, body any, out any) error {
	code, respBody, err := doJSON(srv, method, path, body)
	if err != nil {
		return err
	}
	if code < 200 || code >= 300 {
		return fmt.Errorf("%s %s: status %d: %s", method, path, code, string(respBody))
	}
	if out != nil && len(respBody) > 0 {
		if e := json.Unmarshal(respBody, out); e != nil {
			return fmt.Errorf("%s %s: decode: %w", method, path, e)
		}
	}
	return nil
}

// expectCode asserts the HTTP status without decoding the body.
func expectCode(srv *httptest.Server, method, path string, body any, wantCode int) error {
	code, respBody, err := doJSON(srv, method, path, body)
	if err != nil {
		return err
	}
	if code != wantCode {
		return fmt.Errorf("%s %s: status %d, want %d: %s", method, path, code, wantCode, string(respBody))
	}
	return nil
}

// newCtx returns a background context.
func newCtx() context.Context { return context.Background() }

// numEq compares two JSON-decoded numeric values for near-equality (float).
// Reliability probabilities are float64 and may have tiny FP drift; compare to
// 1e-9. Use numEqInt for integer fields.
func numEq(a, b any) bool {
	af := toFloat(a)
	bf := toFloat(b)
	return math.Abs(af-bf) < 1e-9
}

func numEqTol(a, b any, tol float64) bool {
	return math.Abs(toFloat(a)-toFloat(b)) < tol
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	case int:
		return float64(n)
	}
	return 0
}

// fieldOf extracts a top-level field from a decoded JSON map.
func fieldOf(m map[string]any, key string) any { return m[key] }

func requireField(srv *httptest.Server, m map[string]any, key string) (any, error) {
	v, ok := m[key]
	if !ok {
		return nil, fmt.Errorf("response missing field %q: %v", key, m)
	}
	return v, nil
}

func contains(s, sub string) bool {
	return len(sub) == 0 || bytesContains([]byte(s), []byte(sub))
}

func bytesContains(s, sub []byte) bool {
	return bytes.Contains(s, sub)
}

var _ = io.Discard
