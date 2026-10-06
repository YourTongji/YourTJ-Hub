package preferences

import (
	"sync"
	"testing"
)

func TestSnapshotsDoNotShareMutableValues(t *testing.T) {
	input := map[string]any{"items": []string{"original"}}
	Set("snapshot.test", input)
	input["items"].([]string)[0] = "changed"
	got := GetRaw("snapshot.test").(map[string]any)
	got["items"].([]string)[0] = "changed again"
	if GetRaw("snapshot.test").(map[string]any)["items"].([]string)[0] != "original" {
		t.Fatal("mutable configuration escaped the published snapshot")
	}
}

func TestConcurrentSnapshotReadersAndWriters(t *testing.T) {
	var workers sync.WaitGroup
	for n := 0; n < 8; n++ {
		workers.Go(func() {
			for i := 0; i < 100; i++ {
				Set("snapshot.concurrent", i)
				_ = GetInt("snapshot.concurrent")
				_ = All()
			}
		})
	}
	workers.Wait()
}
