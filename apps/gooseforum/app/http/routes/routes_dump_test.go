package routes

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/preferences"
	"github.com/gin-gonic/gin"
)

// routeSnapshotEntry is one method+path pair as reported by gin's Routes().
type routeSnapshotEntry struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// routesSnapshotPath locates the committed route snapshot under
// packages/api-contract/fixtures via the test file's own location, the same
// trick contractFixture uses.
func routesSnapshotPath(t *testing.T) string {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve routes snapshot path")
	}
	root := filepath.Join(filepath.Dir(testFile), "..", "..", "..", "..", "..")
	return filepath.Join(root, "packages", "api-contract", "fixtures", "routes-snapshot.json")
}

// collectRouteSnapshot assembles the production router the same way main does
// (RegisterByGin on a fresh engine) and normalizes the route list: gin expands
// Any() and StaticFS() into one entry per method, so we dedupe method+path
// pairs and sort them for a stable, diff-friendly snapshot.
//
// OIDC /api/oauth/* routes are intentionally absent: they are only registered
// when oidc.enabled=true, and this snapshot uses the default config. The OIDC
// surface is tracked in its own contract slice.
func collectRouteSnapshot(t *testing.T) []routeSnapshotEntry {
	t.Helper()
	setupMcpRouteTestDB(t) // mcpRoute needs the page_config table migrated
	// The committed snapshot describes the production single-binary route
	// surface. Isolate this assembly from tests that temporarily switch the
	// process-wide environment to local development (which enables Vite proxy
	// routes under /assets/*path).
	previousEnv := preferences.Get("app.env", "production")
	preferences.Set("app.env", "production")
	t.Cleanup(func() { preferences.Set("app.env", previousEnv) })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterByGin(router)

	seen := map[routeSnapshotEntry]bool{}
	var entries []routeSnapshotEntry
	for _, route := range router.Routes() {
		entry := routeSnapshotEntry{Method: route.Method, Path: route.Path}
		if seen[entry] {
			continue
		}
		seen[entry] = true
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(a, b routeSnapshotEntry) int {
		return cmp.Or(cmp.Compare(a.Method, b.Method), cmp.Compare(a.Path, b.Path))
	})
	return entries
}

// TestRoutesSnapshot guards the route snapshot consumed by the contract
// coverage gate (packages/api-contract/scripts/check-route-coverage.mjs).
// With YOURTJ_UPDATE_ROUTES_SNAPSHOT=1 it rewrites the snapshot; otherwise it
// fails on any drift and reports which routes were added or removed, so
// ci-backend's `go test ./...` becomes the "routes changed but snapshot not
// regenerated" gate.
func TestRoutesSnapshot(t *testing.T) {
	entries := collectRouteSnapshot(t)
	encoded, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("marshal routes snapshot: %v", err)
	}
	encoded = append(encoded, '\n')

	snapshotPath := routesSnapshotPath(t)
	if os.Getenv("YOURTJ_UPDATE_ROUTES_SNAPSHOT") == "1" {
		if err := os.WriteFile(snapshotPath, encoded, 0o644); err != nil {
			t.Fatalf("write routes snapshot: %v", err)
		}
		t.Logf("wrote %d routes to %s", len(entries), snapshotPath)
		return
	}

	contents, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatalf("read routes snapshot: %v (regenerate with YOURTJ_UPDATE_ROUTES_SNAPSHOT=1)", err)
	}
	var snapshot []routeSnapshotEntry
	if err := json.Unmarshal(contents, &snapshot); err != nil {
		t.Fatalf("decode routes snapshot: %v", err)
	}

	key := func(e routeSnapshotEntry) string { return e.Method + " " + e.Path }
	current := map[string]bool{}
	for _, e := range entries {
		current[key(e)] = true
	}
	expected := map[string]bool{}
	for _, e := range snapshot {
		expected[key(e)] = true
	}

	var added, removed []string
	for k := range current {
		if !expected[k] {
			added = append(added, k)
		}
	}
	for k := range expected {
		if !current[k] {
			removed = append(removed, k)
		}
	}
	if len(added) > 0 || len(removed) > 0 {
		sort.Strings(added)
		sort.Strings(removed)
		msg := "route snapshot drift detected; regenerate with YOURTJ_UPDATE_ROUTES_SNAPSHOT=1 go test ./app/http/routes/ -run TestRoutesSnapshot"
		if len(added) > 0 {
			msg += fmt.Sprintf("\nadded routes (in code, not in snapshot):\n  + %s", strings.Join(added, "\n  + "))
		}
		if len(removed) > 0 {
			msg += fmt.Sprintf("\nremoved routes (in snapshot, not in code):\n  - %s", strings.Join(removed, "\n  - "))
		}
		t.Fatal(msg)
	}
}
