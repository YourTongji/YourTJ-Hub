package console

import (
	"bytes"
	"encoding/json"
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
}
