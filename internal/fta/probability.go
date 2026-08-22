package fta

import (
	"task140-reliability/internal/domain"
)

// exactTopProbability computes the top-event probability by inclusion-exclusion
// over the cut sets. Exact (no rare-event assumption) for small cut-set counts.
//
//	P(top) = sum P(CSi) - sum P(CSi∩CSj) + sum P(CSi∩CSj∩CSk) - ...
//
// where P(CS) = product of event q over the set, and intersection = union of
// event sets (event q taken once even if repeated across intersecting sets).
func exactTopProbability(cutSets []domain.CutSet, s *Solver) float64 {
	n := len(cutSets)
	if n == 0 {
		return 0
	}
	// Precompute each cut set's event ID list (dedup) and per-event q lookup.
	// Probability of a set of events = product of distinct event q in the set.
	pSet := func(ids []string) float64 {
		seen := make(map[string]bool, len(ids))
		p := 1.0
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			p *= s.EventProbability(id)
		}
		return p
	}
	// iterate over non-empty subfamilies by bitmask (n small here).
	total := 0.0
	for mask := 1; mask < (1 << n); mask++ {
		// union of event ids in this subfamily
		union := make(map[string]bool)
		bits := 0
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				bits++
				for _, id := range cutSets[i].Events {
					union[id] = true
				}
			}
		}
		ids := make([]string, 0, len(union))
		for id := range union {
			ids = append(ids, id)
		}
		term := pSet(ids)
		if bits%2 == 1 {
			total += term
		} else {
			total -= term
		}
	}
	if total < 0 {
		return 0
	}
	return total
}

// rareEventTopProbability is the rare-event approximation:
//
//	P(top) ≈ 1 - product(1 - P(CSi))
//
// Accurate when individual cut-set probabilities are small.
func rareEventTopProbability(cutSets []domain.CutSet) float64 {
	prod := 1.0
	for _, cs := range cutSets {
		prod *= (1 - cs.Probability)
	}
	return 1 - prod
}
