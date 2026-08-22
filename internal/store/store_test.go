package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"task140-reliability/internal/domain"
)

func TestSchemaApplyAndReopen(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/rel.db"
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// schema_version present?
	n, err := s.Count(context.Background(), `SELECT COUNT(*) FROM schema_version`)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("schema_version count = %d, want 1", n)
	}
}

func TestAnalysisLifecycle(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()
	ctx := context.Background()
	a := domain.DefaultAnalysisParams()
	a.ID = "a1"
	a.Name = "pump-reliability"
	a.TopEvent = "top-pump-fails"
	a.State = domain.StateDraft
	a.Version = 1
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	if err := s.CreateAnalysis(ctx, a); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetAnalysis(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.State != domain.StateDraft || got.Version != 1 {
		t.Fatalf("state=%s version=%d", got.State, got.Version)
	}
	// state transitions
	if err := s.SetAnalysisState(ctx, "a1", domain.StateAnalyzed, 1, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAnalysisState(ctx, "a1", domain.StateReviewed, 1, nil); err != nil {
		t.Fatal(err)
	}
	// cannot baseline from reviewed? can
	if err := s.SetAnalysisState(ctx, "a1", domain.StateBaselined, 1, nil); err != nil {
		t.Fatal(err)
	}
	// baselined is immutable: any further state change (except revise->draft) errors
	if err := s.SetAnalysisState(ctx, "a1", domain.StateAnalyzed, 1, nil); err == nil {
		t.Fatalf("editing baselined analysis should error")
	}
}

func TestComponentAndFailureMode(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()
	ctx := context.Background()
	c := domain.Component{ID: "c1", Name: "Pump", Category: "rotating", FailureRatePPT: 1_000_000}
	if err := s.CreateComponent(ctx, c); err != nil {
		t.Fatal(err)
	}
	fm := domain.FailureMode{ID: "fm1", ComponentID: "c1", Name: "seal-leak", Severity: 8, Occurrence: 5, Detection: 3}
	if err := s.CreateFailureMode(ctx, fm); err != nil {
		t.Fatal(err)
	}
	// missing component
	if err := s.CreateFailureMode(ctx, domain.FailureMode{ID: "fm2", ComponentID: "nope"}); err == nil {
		t.Fatalf("missing component should error")
	}
	modes, err := s.ListFailureModes(ctx, "c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(modes) != 1 {
		t.Fatalf("want 1 mode, got %d", len(modes))
	}
}

func TestFTATreeRoundTrip(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()
	ctx := context.Background()
	a := domain.DefaultAnalysisParams()
	a.ID = "a1"
	a.State = domain.StateDraft
	a.Version = 1
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	if err := s.CreateAnalysis(ctx, a); err != nil {
		t.Fatal(err)
	}
	tree := domain.FTATree{
		AnalysisID: "a1", TopGateID: "TOP",
		Gates: []domain.Gate{
			{ID: "TOP", AnalysisID: "a1", Type: domain.GateOR, IsTop: true, Inputs: []domain.GateInput{
				{EventID: "A"}, {GateID: "AND1"},
			}},
			{ID: "AND1", AnalysisID: "a1", Type: domain.GateAND, Inputs: []domain.GateInput{
				{EventID: "B"}, {EventID: "C"},
			}},
		},
		Events: []domain.BasicEvent{
			{ID: "A", AnalysisID: "a1", Label: "A", ProbMicro: 100000},
			{ID: "B", AnalysisID: "a1", Label: "B", ProbMicro: 200000},
			{ID: "C", AnalysisID: "a1", Label: "C", ProbMicro: 300000},
		},
	}
	if err := s.SaveFTATree(ctx, tree); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.LoadFTATree(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.TopGateID != "TOP" {
		t.Fatalf("top gate = %s", loaded.TopGateID)
	}
	if len(loaded.Gates) != 2 || len(loaded.Events) != 3 {
		t.Fatalf("gates=%d events=%d", len(loaded.Gates), len(loaded.Events))
	}
}

func TestRBDDiagramRoundTrip(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()
	ctx := context.Background()
	a := domain.DefaultAnalysisParams()
	a.ID = "a1"
	a.State = domain.StateDraft
	a.Version = 1
	a.CreatedAt = time.Now().UTC()
	a.UpdatedAt = a.CreatedAt
	if err := s.CreateAnalysis(ctx, a); err != nil {
		t.Fatal(err)
	}
	diag := domain.RBDDiagram{
		AnalysisID: "a1", RootID: "ROOT",
		Blocks: []domain.RBDBlock{
			{ID: "ROOT", AnalysisID: "a1", Type: domain.BlockSeries, Children: []string{"A", "B"}},
			{ID: "A", AnalysisID: "a1", Type: domain.BlockBasic, FailureRatePPT: 10_000_000},
			{ID: "B", AnalysisID: "a1", Type: domain.BlockBasic, FailureRatePPT: 10_000_000},
		},
	}
	if err := s.SaveRBDDiagram(ctx, diag); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.LoadRBDDiagram(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.RootID != "ROOT" || len(loaded.Blocks) != 3 {
		t.Fatalf("root=%s blocks=%d", loaded.RootID, len(loaded.Blocks))
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := t.TempDir() + "/rel.db"
	s, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	return s
}

var _ = fmt.Errorf
