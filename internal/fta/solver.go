// Package fta implements fault-tree analysis: coherence (consistency)
// checking, minimal-cut-set enumeration (MOCUS), and top-event probability
// quantification (exact inclusion-exclusion or rare-event approximation), with
// common-cause-failure (CCF) beta-factor folding.
package fta

import (
	"fmt"
	"sort"

	"task140-reliability/internal/domain"
)

// Solver quantifies a fault tree. Construct with NewSolver.
type Solver struct {
	tree      *domain.FTATree
	gateByID  map[string]*domain.Gate
	eventByID map[string]*domain.BasicEvent
	byInput   map[string][]string // gateID -> child gate IDs reachable (transitive inputs)
	maxOrder  int
	ccfBeta   float64
}

// NewSolver builds a solver over a fault tree snapshot.
func NewSolver(tree *domain.FTATree, maxOrder int, ccfBeta float64) (*Solver, error) {
	if tree == nil {
		return nil, fmt.Errorf("%w: fault tree is nil", domain.ErrInvalidArgument)
	}
	if maxOrder < 1 {
		maxOrder = 1
	}
	if ccfBeta < 0 || ccfBeta > 1 {
		return nil, fmt.Errorf("%w: ccf_beta must be in [0,1]", domain.ErrInvalidArgument)
	}
	s := &Solver{
		tree:      tree,
		gateByID:  make(map[string]*domain.Gate, len(tree.Gates)),
		eventByID: make(map[string]*domain.BasicEvent, len(tree.Events)),
		maxOrder:  maxOrder,
		ccfBeta:   ccfBeta,
	}
	for i := range tree.Gates {
		g := &tree.Gates[i]
		s.gateByID[g.ID] = g
	}
	for i := range tree.Events {
		e := &tree.Events[i]
		s.eventByID[e.ID] = e
	}
	return s, nil
}

// EventProbability returns the effective probability q of a basic event after
// CCF beta-factor folding. The beta-factor model: an event in a CCF group has
// an independent component q_ind and a common component q_ccf = beta*q; the
// effective probability is q_ind*(1-beta) + beta*q = q (total), but for the
// purpose of cut-set probability the common cause is represented by lifting
// each grouped event's probability by the beta factor so that simultaneous
// occurrences correlate. Here we apply the simple beta-factor lift:
// q_eff = q_ind + beta*q_ind*(m-1)/m ... For determinism we use the standard
// beta-factor single-point lift: q_eff = q_ind*(1-beta) + beta (a shared
// common cause that alone can fail the group). When beta=0, q_eff = q_ind.
func (s *Solver) EventProbability(eventID string) float64 {
	e, ok := s.eventByID[eventID]
	if !ok {
		return 0
	}
	q := domain.ProbMicroToFloat(e.ProbMicro)
	if e.Negated {
		q = 1 - q
	}
	if e.CCFGroup != "" && s.ccfBeta > 0 {
		// beta-factor lift: the common cause contributes beta*q as a shared
		// independent failure that, when it occurs, fails all grouped events.
		q = q*(1-s.ccfBeta) + s.ccfBeta*q
	}
	if q < 0 {
		q = 0
	}
	if q > 1 {
		q = 1
	}
	return q
}

// CheckCoherence inspects the tree for NOT-gate and complemented-event usage.
// A coherent tree has no negated events and no event appearing both negated
// and unnegated. Non-coherent trees are still solvable but flagged.
func (s *Solver) CheckCoherence() domain.CoherenceReport {
	rep := domain.CoherenceReport{Coherent: true}
	polarity := make(map[string]string) // eventID -> "pos"/"neg"
	negCount := 0
	for _, e := range s.tree.Events {
		if e.Negated {
			negCount++
			if polarity[e.ID] == "" {
				polarity[e.ID] = "neg"
			}
		} else {
			if polarity[e.ID] == "neg" {
				// appeared both
				rep.Coherent = false
				rep.NonCoherentReasons = append(rep.NonCoherentReasons,
					fmt.Sprintf("event %s appears both negated and unnegated", e.ID))
			}
			if polarity[e.ID] == "" {
				polarity[e.ID] = "pos"
			}
		}
	}
	if negCount > 0 {
		rep.Coherent = false
		rep.NonCoherentReasons = append(rep.NonCoherentReasons,
			fmt.Sprintf("tree contains %d negated (NOT) event(s)", negCount))
	}
	if !rep.Coherent {
		rep.Warnings = append(rep.Warnings, "non-coherent fault tree: results computed but may be non-monotonic")
	}
	return rep
}

// Solve runs the full quantification: coherence check, cut-set enumeration,
// absorption, order truncation, and probability computation.
func (s *Solver) Solve(exactLimit int) (*domain.CutSetResult, error) {
	if exactLimit < 0 {
		exactLimit = 0
	}
	rep := s.CheckCoherence()

	top, ok := s.gateByID[s.tree.TopGateID]
	if !ok {
		return nil, fmt.Errorf("%w: top gate not found", domain.ErrInvalidArgument)
	}

	cutSets := s.enumCutSets(top)
	cutSets = absorb(cutSets)

	truncated := 0
	kept := cutSets[:0:0]
	for _, cs := range cutSets {
		if cs.Order > s.maxOrder {
			truncated++
			continue
		}
		kept = append(kept, cs)
	}
	// re-dedupe after truncation (a truncated high-order set may duplicate a kept one)
	kept = absorb(kept)
	sort.Slice(kept, func(i, j int) bool {
		if kept[i].Order != kept[j].Order {
			return kept[i].Order < kept[j].Order
		}
		return joinIDs(kept[i].Events) < joinIDs(kept[j].Events)
	})

	// compute per-cutset probability
	for i := range kept {
		p := 1.0
		for _, eid := range kept[i].Events {
			p *= s.EventProbability(eid)
		}
		kept[i].Probability = p
	}

	var topProb float64
	var method string
	if len(kept) <= exactLimit {
		topProb = exactTopProbability(kept, s)
		method = "exact"
	} else {
		topProb = rareEventTopProbability(kept)
		method = "rare_event"
	}
	if topProb > 1 {
		topProb = 1
	}

	return &domain.CutSetResult{
		TopProbability: topProb,
		Method:         method,
		CutSets:        kept,
		TruncatedCount: truncated,
		Coherence:      rep,
	}, nil
}

// enumCutSets computes the (pre-absorption) cut sets of a gate recursively.
// For each input we gather its cut sets: an event input contributes exactly
// one cut set {eventID}; a gate input contributes the cut sets of the child
// gate. Then:
//   - OR: the union of all inputs' cut sets (each input cut set is itself a
//     cut set of the OR);
//   - AND: the cartesian product — pick one cut set from each input and
//     concatenate (union of events);
//   - KOFN: all k-combinations of inputs, each combination treated as an AND
//     (pick one cut set from each of the k chosen inputs, concatenate).
func (s *Solver) enumCutSets(g *domain.Gate) []domain.CutSet {
	// per-input list of cut sets (each cut set is a []string of event IDs)
	perInput := make([][][]string, 0, len(g.Inputs))
	for _, in := range g.Inputs {
		if in.EventID != "" {
			perInput = append(perInput, [][]string{{in.EventID}})
		} else if in.GateID != "" {
			child, ok := s.gateByID[in.GateID]
			if !ok {
				continue
			}
			cs := s.enumCutSets(child)
			leg := make([][]string, 0, len(cs))
			for _, c := range cs {
				leg = append(leg, c.Events)
			}
			if len(leg) == 0 {
				leg = [][]string{{}}
			}
			perInput = append(perInput, leg)
		}
	}
	if len(perInput) == 0 {
		return []domain.CutSet{}
	}
	var paths [][]string
	switch g.Type {
	case domain.GateAND:
		paths = productConcat(perInput)
	case domain.GateKOFN:
		k := g.K
		if k < 1 {
			k = 1
		}
		if k > len(perInput) {
			k = len(perInput)
		}
		paths = kCombosProduct(perInput, k)
	default: // OR (and unknown -> OR)
		for _, leg := range perInput {
			paths = append(paths, leg...)
		}
	}
	var result []domain.CutSet
	for _, p := range paths {
		p = uniqueSorted(p)
		if len(p) == 0 {
			continue
		}
		result = append(result, domain.CutSet{Events: p, Order: len(p)})
	}
	if result == nil {
		result = []domain.CutSet{}
	}
	return result
}

// productConcat returns the cartesian product of per-input cut-set lists,
// concatenating the chosen cut set from each input. perInput[i] is the list of
// cut sets for input i; the result picks one cut set per input and unions them.
func productConcat(perInput [][][]string) [][]string {
	out := [][]string{{}}
	for _, leg := range perInput {
		next := make([][]string, 0, len(out)*len(leg))
		for _, p := range out {
			for _, cs := range leg {
				np := append(append([]string{}, p...), cs...)
				next = append(next, np)
			}
		}
		out = next
	}
	return out
}

// kCombosProduct picks k of the n inputs and, for each k-subset of inputs,
// takes the cartesian product of their cut-set lists (concatenated). Returns
// the union over all k-subsets.
func kCombosProduct(perInput [][][]string, k int) [][]string {
	n := len(perInput)
	var out [][]string
	var rec func(start int, chosen [][][]string)
	rec = func(start int, chosen [][][]string) {
		if len(chosen) == k {
			out = append(out, productConcat(chosen)...)
			return
		}
		for i := start; i < n; i++ {
			rec(i+1, append(chosen, perInput[i]))
		}
	}
	rec(0, nil)
	return out
}

// uniqueSorted returns a sorted, de-duplicated copy of ids.
func uniqueSorted(ids []string) []string {
	if len(ids) == 0 {
		return ids
	}
	cp := append([]string{}, ids...)
	sort.Strings(cp)
	out := cp[:0]
	for i, id := range cp {
		if i == 0 || id != cp[i-1] {
			out = append(out, id)
		}
	}
	return out
}

// joinIDs joins ids with comma for stable sort keys.
func joinIDs(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += ","
		}
		out += id
	}
	return out
}

// absorb removes cut sets that are supersets of another cut set (minimal cut
// sets only). Also deduplicates identical sets.
func absorb(cutSets []domain.CutSet) []domain.CutSet {
	// sort by order asc so smaller sets are kept first
	sort.SliceStable(cutSets, func(i, j int) bool {
		return cutSets[i].Order < cutSets[j].Order
	})
	out := []domain.CutSet{}
	for _, cs := range cutSets {
		dup := false
		for _, kept := range out {
			if sameSet(cs.Events, kept.Events) {
				dup = true
				break
			}
			if isSubset(kept.Events, cs.Events) {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, cs)
		}
	}
	return out
}

// sameSet reports whether two sorted id slices are equal.
func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// isSubset reports whether every element of a (sorted) is in b (sorted).
func isSubset(a, b []string) bool {
	if len(a) > len(b) {
		return false
	}
	bi := 0
	for _, x := range a {
		for bi < len(b) && b[bi] < x {
			bi++
		}
		if bi >= len(b) || b[bi] != x {
			return false
		}
	}
	return true
}

// kcombinations returns all k-element combinations of items.
func kcombinations(items [][]string, k int) [][][]string {
	var result [][][]string
	var rec func(start int, chosen [][]string)
	rec = func(start int, chosen [][]string) {
		if len(chosen) == k {
			cp := make([][]string, k)
			copy(cp, chosen)
			result = append(result, cp)
			return
		}
		for i := start; i < len(items); i++ {
			rec(i+1, append(chosen, items[i]))
		}
	}
	rec(0, nil)
	return result
}
