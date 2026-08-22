package rbd

import (
	"testing"

	"task140-reliability/internal/domain"
)

// helper to make a basic block with failure rate ppt.
func basic(id string, rate int64) domain.RBDBlock {
	return domain.RBDBlock{ID: id, AnalysisID: "a", Type: domain.BlockBasic, FailureRatePPT: domain.FailureRatePPT(rate)}
}

func TestSeriesParallel(t *testing.T) {
	// series(A,B) under root, mission=10000h
	// A: lambda=1e-5/h => rate=1e7 ppt? 1e-5 * 1e12 = 1e7
	// R_A = exp(-1e-5*10000)=exp(-0.1)=0.904837
	diag := &domain.RBDDiagram{
		AnalysisID: "a", RootID: "ROOT",
		Blocks: []domain.RBDBlock{
			{ID: "ROOT", AnalysisID: "a", Type: domain.BlockSeries, Children: []string{"A", "B"}},
			basic("A", 10_000_000), // 1e7 ppt = 1e-5/h
			basic("B", 10_000_000),
		},
	}
	s, err := NewSolver(diag)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Solve(10000)
	if err != nil {
		t.Fatal(err)
	}
	ra := 0.9048374180
	want := ra * ra
	if !approx(res.RootReliability, want, 1e-6) {
		t.Fatalf("series reliability = %v, want %v", res.RootReliability, want)
	}
	if res.Repairable {
		t.Fatalf("non-repairable blocks should not be repairable")
	}
}

func TestParallel(t *testing.T) {
	diag := &domain.RBDDiagram{
		AnalysisID: "a", RootID: "ROOT",
		Blocks: []domain.RBDBlock{
			{ID: "ROOT", AnalysisID: "a", Type: domain.BlockParallel, Children: []string{"A", "B"}},
			basic("A", 10_000_000),
			basic("B", 10_000_000),
		},
	}
	s, _ := NewSolver(diag)
	res, _ := s.Solve(10000)
	ra := 0.9048374180
	want := 1 - (1-ra)*(1-ra)
	if !approx(res.RootReliability, want, 1e-6) {
		t.Fatalf("parallel reliability = %v, want %v", res.RootReliability, want)
	}
}

func TestKOFN(t *testing.T) {
	// 2-of-3 with identical R
	diag := &domain.RBDDiagram{
		AnalysisID: "a", RootID: "ROOT",
		Blocks: []domain.RBDBlock{
			{ID: "ROOT", AnalysisID: "a", Type: domain.BlockKOFN, K: 2, Children: []string{"A", "B", "C"}},
			basic("A", 10_000_000),
			basic("B", 10_000_000),
			basic("C", 10_000_000),
		},
	}
	s, _ := NewSolver(diag)
	res, _ := s.Solve(10000)
	ra := 0.9048374180
	// exact heterogeneous (all equal): sum_{j=2}^{3} C(3,j) r^j (1-r)^(3-j)
	want := 3*ra*ra*(1-ra) + ra*ra*ra
	if !approx(res.RootReliability, want, 1e-6) {
		t.Fatalf("2of3 reliability = %v, want %v", res.RootReliability, want)
	}
}

func TestKOFNInvalid(t *testing.T) {
	diag := &domain.RBDDiagram{
		AnalysisID: "a", RootID: "ROOT",
		Blocks: []domain.RBDBlock{
			{ID: "ROOT", AnalysisID: "a", Type: domain.BlockKOFN, K: 5, Children: []string{"A", "B"}},
			basic("A", 10_000_000),
			basic("B", 10_000_000),
		},
	}
	s, _ := NewSolver(diag)
	if _, err := s.Solve(10000); err == nil {
		t.Fatalf("k>n should error")
	}
}

func TestAvailability(t *testing.T) {
	// basic block with repair rate: A = mu/(lambda+mu)
	// lambda=1e-5 (1e7 ppt), mu=1e-3 (1e9 ppt) => A = 1e-3/(1e-5+1e-3) = 0.99009901
	diag := &domain.RBDDiagram{
		AnalysisID: "a", RootID: "ROOT",
		Blocks: []domain.RBDBlock{
			{ID: "ROOT", AnalysisID: "a", Type: domain.BlockBasic, FailureRatePPT: 10_000_000, RepairRatePPT: 1_000_000_000},
		},
	}
	s, _ := NewSolver(diag)
	res, _ := s.Solve(10000)
	want := 1e-3 / (1e-5 + 1e-3)
	if !approx(res.Availability, want, 1e-6) {
		t.Fatalf("availability = %v, want %v", res.Availability, want)
	}
	if !res.Repairable {
		t.Fatalf("should be repairable")
	}
}

func approx(a, b, eps float64) bool {
	if a-b > eps || b-a > eps {
		return false
	}
	return true
}
