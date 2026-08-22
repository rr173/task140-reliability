package fta

import (
	"testing"

	"task140-reliability/internal/domain"
)

func makeTree() *domain.FTATree {
	// top OR (A, AND(B,C))
	// A q=0.1, B q=0.2, C q=0.3
	return &domain.FTATree{
		TopGateID: "TOP",
		Gates: []domain.Gate{
			{ID: "TOP", AnalysisID: "a1", Type: domain.GateOR, IsTop: true, Inputs: []domain.GateInput{
				{EventID: "A"},
				{GateID: "AND1"},
			}},
			{ID: "AND1", AnalysisID: "a1", Type: domain.GateAND, Inputs: []domain.GateInput{
				{EventID: "B"},
				{EventID: "C"},
			}},
		},
		Events: []domain.BasicEvent{
			{ID: "A", AnalysisID: "a1", Label: "A", ProbMicro: 100000}, // 0.1
			{ID: "B", AnalysisID: "a1", Label: "B", ProbMicro: 200000}, // 0.2
			{ID: "C", AnalysisID: "a1", Label: "C", ProbMicro: 300000}, // 0.3
		},
	}
}

func TestCutSetsAndProbability(t *testing.T) {
	s, err := NewSolver(makeTree(), 4, 0)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Solve(8)
	if err != nil {
		t.Fatal(err)
	}
	// cut sets: {A}, {B,C}
	if len(res.CutSets) != 2 {
		t.Fatalf("want 2 cut sets, got %d: %+v", len(res.CutSets), res.CutSets)
	}
	// exact inclusion-exclusion: P(A∪BC) = P(A)+P(BC)-P(A∩BC)
	//   = 0.1 + 0.06 - 0.1*0.06 = 0.16 - 0.006 = 0.154
	want := 0.1 + 0.06 - 0.1*0.06
	if !approx(res.TopProbability, want, 1e-9) {
		t.Fatalf("top prob = %v, want %v", res.TopProbability, want)
	}
	if res.Method != "exact" {
		t.Fatalf("method = %s, want exact", res.Method)
	}
	if !res.Coherence.Coherent {
		t.Fatalf("tree should be coherent")
	}
}

func TestAbsorption(t *testing.T) {
	// AND(A, OR(A, B)) => cut sets {A}, {A,B}; {A,B} superset of {A} => absorbed.
	tree := &domain.FTATree{
		TopGateID: "TOP",
		Gates: []domain.Gate{
			{ID: "TOP", Type: domain.GateAND, IsTop: true, Inputs: []domain.GateInput{
				{EventID: "A"}, {GateID: "OR1"},
			}},
			{ID: "OR1", Type: domain.GateOR, Inputs: []domain.GateInput{
				{EventID: "A"}, {EventID: "B"},
			}},
		},
		Events: []domain.BasicEvent{
			{ID: "A", ProbMicro: 100000},
			{ID: "B", ProbMicro: 200000},
		},
	}
	s, _ := NewSolver(tree, 4, 0)
	res, _ := s.Solve(8)
	if len(res.CutSets) != 1 {
		t.Fatalf("want 1 minimal cut set after absorption, got %d", len(res.CutSets))
	}
	if res.CutSets[0].Order != 1 || res.CutSets[0].Events[0] != "A" {
		t.Fatalf("unexpected cut set: %+v", res.CutSets[0])
	}
}

func TestOrderTruncation(t *testing.T) {
	// AND(A,B,C,D) with maxOrder=2 => cut set {A,B,C,D} truncated.
	tree := &domain.FTATree{
		TopGateID: "TOP",
		Gates: []domain.Gate{
			{ID: "TOP", Type: domain.GateAND, IsTop: true, Inputs: []domain.GateInput{
				{EventID: "A"}, {EventID: "B"}, {EventID: "C"}, {EventID: "D"},
			}},
		},
		Events: []domain.BasicEvent{
			{ID: "A", ProbMicro: 50000},
			{ID: "B", ProbMicro: 50000},
			{ID: "C", ProbMicro: 50000},
			{ID: "D", ProbMicro: 50000},
		},
	}
	s, _ := NewSolver(tree, 2, 0)
	res, _ := s.Solve(8)
	if len(res.CutSets) != 0 {
		t.Fatalf("want 0 cut sets after truncation, got %d", len(res.CutSets))
	}
	if res.TruncatedCount != 1 {
		t.Fatalf("truncated count = %d, want 1", res.TruncatedCount)
	}
}

func TestRareEventApprox(t *testing.T) {
	s, _ := NewSolver(makeTree(), 4, 0)
	res, _ := s.Solve(0) // exactLimit=0 forces rare_event
	if res.Method != "rare_event" {
		t.Fatalf("method = %s, want rare_event", res.Method)
	}
	// rare: 1-(1-0.1)(1-0.2*0.3) = 1 - 0.9*0.94 = 1 - 0.846 = 0.154
	want := 1 - (1-0.1)*(1-0.2*0.3)
	if !approx(res.TopProbability, want, 1e-9) {
		t.Fatalf("rare event = %v, want %v", res.TopProbability, want)
	}
}

func TestNonCoherent(t *testing.T) {
	tree := &domain.FTATree{
		TopGateID: "TOP",
		Gates: []domain.Gate{
			{ID: "TOP", Type: domain.GateOR, IsTop: true, Inputs: []domain.GateInput{
				{EventID: "A"},
			}},
		},
		Events: []domain.BasicEvent{
			{ID: "A", ProbMicro: 100000, Negated: true},
		},
	}
	s, _ := NewSolver(tree, 4, 0)
	rep := s.CheckCoherence()
	if rep.Coherent {
		t.Fatalf("expected non-coherent")
	}
}

func approx(a, b, eps float64) bool {
	if a-b > eps || b-a > eps {
		return false
	}
	return true
}
