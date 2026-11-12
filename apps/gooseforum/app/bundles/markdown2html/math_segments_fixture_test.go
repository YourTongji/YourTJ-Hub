package markdown2html

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestExtractMathSegmentsSharedFixture keeps the server delimiter rules in step
// with the Web KaTeX enhancer and the app renderer, which read the same file.
func TestExtractMathSegmentsSharedFixture(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate math segment test file")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "..", "packages", "api-contract", "fixtures", "markdown-math-segments.json"))
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Cases []struct {
			Source   string `json:"source"`
			Segments []struct {
				Text    string `json:"text"`
				Display bool   `json:"display"`
			} `json:"segments"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, item := range fixture.Cases {
		t.Run(item.Source, func(t *testing.T) {
			got := extractMathSegments(item.Source)
			if len(got) != len(item.Segments) {
				t.Fatalf("extractMathSegments(%q) = %+v, want %+v", item.Source, got, item.Segments)
			}
			for i, want := range item.Segments {
				if got[i].text != want.Text || got[i].display != want.Display {
					t.Fatalf("segment %d of %q = %+v, want %+v", i, item.Source, got[i], want)
				}
			}
		})
	}
}
