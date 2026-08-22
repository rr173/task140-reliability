package analysis

import (
	"testing"

	"task140-reliability/internal/domain"
)

func TestNextState(t *testing.T) {
	cases := []struct {
		cur, want domain.AnalysisState
		ok        bool
	}{
		{domain.StateDraft, domain.StateAnalyzed, true},
		{domain.StateAnalyzed, domain.StateAnalyzed, true}, // re-solve allowed
		{domain.StateReviewed, domain.StateAnalyzed, false},
		{domain.StateAnalyzed, domain.StateReviewed, true},
		{domain.StateDraft, domain.StateReviewed, false},
		{domain.StateReviewed, domain.StateBaselined, true},
		{domain.StateDraft, domain.StateBaselined, false},
		{domain.StateBaselined, domain.StateDraft, true}, // revise
		{domain.StateAnalyzed, domain.StateDraft, false},
	}
	for _, c := range cases {
		_, err := NextState(c.cur, c.want)
		if c.ok && err != nil {
			t.Fatalf("%s->%s expected ok, got %v", c.cur, c.want, err)
		}
		if !c.ok && err == nil {
			t.Fatalf("%s->%s expected error", c.cur, c.want)
		}
	}
}
