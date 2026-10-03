package console

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/YourTongji/YourTJ-Hub/apps/gooseforum/app/bundles/buildinfo"
	"github.com/spf13/cobra"
)

func TestVersionReportsCompiledIdentityWithoutStartingRuntime(t *testing.T) {
	command, _, err := rootCmd.Find([]string{"version"})
	if err != nil || command.Name() != "version" {
		t.Fatal("version command must be registered")
	}
	original := rootCmd.PersistentPreRun
	rootCmd.PersistentPreRun = func(*cobra.Command, []string) {
		t.Fatal("version must not run migrations or start the event bus")
	}
	var output bytes.Buffer
	rootCmd.SetOut(&output)
	rootCmd.SetArgs([]string{"version"})
	t.Cleanup(func() {
		rootCmd.PersistentPreRun = original
		rootCmd.SetOut(nil)
		rootCmd.SetArgs(nil)
		_ = command.Flags().Set("output", "")
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var actual buildinfo.Info
	if err := json.Unmarshal(output.Bytes(), &actual); err != nil {
		t.Fatalf("version must emit only JSON: %v; %s", err, output.Bytes())
	}
	if expected := buildinfo.Get(); actual != expected {
		t.Fatalf("got %#v, want %#v", actual, expected)
	}
	path := filepath.Join(t.TempDir(), "identity.json")
	output.Reset()
	rootCmd.SetArgs([]string{"version", "--output", path})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &actual); err != nil || actual != buildinfo.Get() {
		t.Fatalf("file must contain the compiled identity: %s; %v", data, err)
	}
	if output.Len() != 0 {
		t.Fatalf("file output must not duplicate metadata on stdout: %s", output.Bytes())
	}
}
