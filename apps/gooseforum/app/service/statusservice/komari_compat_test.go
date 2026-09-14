package statusservice

import (
	"context"
	"testing"
)

// The deployed Komari returns a UUID-keyed history map and zero ram_total,
// despite the newer documentation's single-node array example.
func TestKomariMappedHistoryUsesCurrentCapacity(t *testing.T) {
	s, _, _ := mockService(t, true)
	result := s.Get(context.Background(), "24h", "1h").Server
	if result.State != "ok" || !result.Data.HistoryAvailable || len(result.Data.History) != 1 || result.Data.History[0].MemoryPercent != 50 {
		t.Fatalf("mapped node history unavailable or mixed with another node: %+v", result.Data)
	}
}

func TestIncompleteProbeDoesNotBecomeZeroTraffic(t *testing.T) {
	s, _, _ := mockService(t, false, true)
	result := s.Get(context.Background(), "24h", "1h")
	if result.Server.State != "unavailable" || result.Server.Data != nil || result.Traffic.State != "ok" {
		t.Fatal("missing probe metrics must not be published as genuine zero values")
	}
}
