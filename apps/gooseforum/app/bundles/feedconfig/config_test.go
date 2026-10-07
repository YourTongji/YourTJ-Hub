package feedconfig

import (
	"math"
	"testing"
)

func TestParametersRejectInvalidWholeSnapshot(t *testing.T) {
	for _, value := range []any{31, "oops", 0} {
		_, err := Decode(map[string]any{"feed": map[string]any{"metrics": map[string]any{"raw_retention_days": value}}})
		if err == nil {
			t.Fatalf("accepted retention %v", value)
		}
	}
	for _, value := range []any{math.NaN(), math.Inf(1), "oops", -1, 11} {
		_, err := Decode(map[string]any{"feed": map[string]any{"weights": map[string]any{"follow": value}}})
		if err == nil {
			t.Fatalf("accepted weight %v", value)
		}
	}
	c, err := Decode(map[string]any{})
	if err != nil || c.Rollout != 20 || c.RetentionDays != 30 || c.Enabled || c.Ranking || c.Metrics || c.Hash == "" {
		t.Fatalf("unsafe defaults %+v %v", c, err)
	}
}
