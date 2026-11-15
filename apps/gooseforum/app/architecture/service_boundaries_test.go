package architecture

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Service entry points must remain usable by HTTP, MCP and background workers
// without importing a transport adapter back into the business dependency graph.
func TestServicesDoNotImportHTTP(t *testing.T) {
	err := filepath.WalkDir("../service", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			name, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return err
			}
			if strings.HasPrefix(name, "github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/http/") {
				t.Errorf("%s imports transport package %s; move shared results or business behavior to its owner", path, name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
