// Package rbd implements reliability block-diagram quantification: structural
// reduction of series / parallel / k-of-n blocks to a single equivalent, and
// mission reliability + steady-state availability (for repairable blocks).
package rbd

import (
	"fmt"

	"task140-reliability/internal/domain"
)

// Solver quantifies a reliability block diagram. Build with NewSolver.
type Solver struct {
	diagram   *domain.RBDDiagram
	blockByID map[string]*domain.RBDBlock
}

// NewSolver builds a solver over a block diagram snapshot.
func NewSolver(diagram *domain.RBDDiagram) (*Solver, error) {
	if diagram == nil {
		return nil, fmt.Errorf("%w: rbd diagram is nil", domain.ErrInvalidArgument)
	}
	s := &Solver{diagram: diagram, blockByID: make(map[string]*domain.RBDBlock, len(diagram.Blocks))}
	for i := range diagram.Blocks {
		b := &diagram.Blocks[i]
		s.blockByID[b.ID] = b
	}
	return s, nil
}

// Solve reduces the diagram root to a single equivalent and returns mission
// reliability and (if repairable) steady-state availability.
//
// Mission reliability of a basic block: R = exp(-lambda*t), lambda in 1/h.
// Steady-state availability of a basic repairable block: A = mu/(lambda+mu).
// Series: R = prod R_i ; A = prod A_i.
// Parallel (1-of-n): R = 1 - prod(1-R_i) ; A = 1 - prod(1-A_i).
// K-of-N: R = sum_{j=k}^{n} C(n,j) R^j (1-R)^(n-j) (equal reliabilities assumed
// when children are identical; for heterogeneous children we compute exactly by
// enumerating subsets).
func (s *Solver) Solve(missionHours int64) (*domain.RBDResult, error) {
	if missionHours <= 0 {
		return nil, fmt.Errorf("%w: mission_hours must be positive", domain.ErrInvalidArgument)
	}
	root, ok := s.blockByID[s.diagram.RootID]
	if !ok {
		return nil, fmt.Errorf("%w: root block not found", domain.ErrInvalidArgument)
	}
	rel, avail, repairable, method, err := s.reduce(root, missionHours)
	if err != nil {
		return nil, err
	}
	if !repairable {
		avail = 0
	}
	return &domain.RBDResult{
		RootReliability: rel,
		Availability:    avail,
		Repairable:      repairable,
		Method:          method,
	}, nil
}

// reduce returns (reliability, availability, repairable, method, err) for a
// block by recursively reducing its children.
func (s *Solver) reduce(b *domain.RBDBlock, hours int64) (float64, float64, bool, string, error) {
	switch b.Type {
	case domain.BlockBasic:
		if b.FailureRatePPT <= 0 {
			// non-failing block
			return 1.0, 1.0, false, "block", nil
		}
		lambda := float64(b.FailureRatePPT) * 1e-12
		rel := domain.Exp(-lambda * float64(hours))
		if b.RepairRatePPT > 0 {
			mu := float64(b.RepairRatePPT) * 1e-12
			avail := mu / (lambda + mu)
			return rel, avail, true, "block", nil
		}
		return rel, 0, false, "block", nil

	case domain.BlockSeries:
		rel := 1.0
		avail := 1.0
		repairable := false
		for _, cid := range b.Children {
			c, ok := s.blockByID[cid]
			if !ok {
				return 0, 0, false, "", fmt.Errorf("%w: missing child %s", domain.ErrInvalidArgument, cid)
			}
			r, a, rep, _, err := s.reduce(c, hours)
			if err != nil {
				return 0, 0, false, "", err
			}
			rel *= r
			avail *= a
			if rep {
				repairable = true
			}
		}
		return rel, avail, repairable, "series", nil

	case domain.BlockParallel:
		rel := 1.0
		avail := 1.0
		repairable := false
		for _, cid := range b.Children {
			c, ok := s.blockByID[cid]
			if !ok {
				return 0, 0, false, "", fmt.Errorf("%w: missing child %s", domain.ErrInvalidArgument, cid)
			}
			r, a, rep, _, err := s.reduce(c, hours)
			if err != nil {
				return 0, 0, false, "", err
			}
			rel *= (1 - r)
			avail *= (1 - a)
			if rep {
				repairable = true
			}
		}
		return 1 - rel, 1 - avail, repairable, "parallel", nil

	case domain.BlockKOFN:
		k := b.K
		n := len(b.Children)
		if k < 1 || k > n {
			return 0, 0, false, "", fmt.Errorf("%w: k=%d out of range [1,%d]", domain.ErrInvalidArgument, k, n)
		}
		// collect child (r,a,rep)
		cls := make([]child, 0, n)
		repairable := false
		for _, cid := range b.Children {
			c, ok := s.blockByID[cid]
			if !ok {
				return 0, 0, false, "", fmt.Errorf("%w: missing child %s", domain.ErrInvalidArgument, cid)
			}
			r, a, rep, _, err := s.reduce(c, hours)
			if err != nil {
				return 0, 0, false, "", err
			}
			cls = append(cls, child{r, a, rep})
			if rep {
				repairable = true
			}
		}
		// exact heterogeneous k-of-n by subset enumeration: reliability is the
		// probability that at least k children succeed.
		rel := kofnExact(cls, k, true)
		avail := kofnExact(cls, k, false)
		return rel, avail, repairable, "kofn", nil
	}
	return 0, 0, false, "", fmt.Errorf("%w: unknown block type %s", domain.ErrRBDCannotReduce, b.Type)
}

// child is a reduced block's reliability, availability and repairable flag.
type child struct {
	r, a float64
	rep  bool
}

// kofnExact computes the probability that at least k of n children succeed.
// field selects reliability (true) or availability (false) per child.
func kofnExact(cls []child, k int, field bool) float64 {
	n := len(cls)
	var prob float64
	for mask := 0; mask < (1 << n); mask++ {
		bits := 0
		pSuccess := 1.0
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				bits++
				// this child succeeds
				if field {
					pSuccess *= cls[i].r
				} else {
					pSuccess *= cls[i].a
				}
			} else {
				// this child fails
				if field {
					pSuccess *= (1 - cls[i].r)
				} else {
					pSuccess *= (1 - cls[i].a)
				}
			}
		}
		if bits >= k {
			prob += pSuccess
		}
	}
	return prob
}

// Validate checks the diagram for structural legality (children exist, k valid).
func (s *Solver) Validate() error {
	for i := range s.diagram.Blocks {
		b := &s.diagram.Blocks[i]
		if b.Type == domain.BlockKOFN {
			if b.K < 1 || b.K > len(b.Children) {
				return fmt.Errorf("%w: block %s k=%d invalid for %d children", domain.ErrInvalidArgument, b.ID, b.K, len(b.Children))
			}
		}
		for _, cid := range b.Children {
			if _, ok := s.blockByID[cid]; !ok {
				return fmt.Errorf("%w: block %s references missing child %s", domain.ErrInvalidArgument, b.ID, cid)
			}
		}
	}
	return nil
}
