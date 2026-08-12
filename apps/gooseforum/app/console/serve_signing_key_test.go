package console

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// serveSigningKeyHelperEnv gates the child-process branch below. It is only
// consulted inside TestServeSigningKeyHelperProcess, which runs only when the
// parent test re-execs the test binary with `-test.run` pointing at it — an
// unrelated GO_WANT_* variable in the environment cannot make the main test or
// any other test take the child branch.
const serveSigningKeyHelperEnv = "YOURTJ_ISSUE106_SERVE_GUARD_HELPER"

// TestGinServeExitsNonZeroOnWeakSigningKey is the process-level guard test for
// issue #106 point 1: serve must refuse to boot (exit code 1) when
// app.signingKey is a weak/known-bad value.
//
// jwtopt captures the signing key at package init from the config.toml that
// preferences discovers (walking up from the working directory in test mode),
// so the child process is started from a temp directory that holds a weak-key
// config.toml of its own. This keeps the guard test end-to-end (weak config →
// serve guard → os.Exit(1)) without ever touching the shared config.toml, so
// it is safe under parallel `go test ./...`.
func TestGinServeExitsNonZeroOnWeakSigningKey(t *testing.T) {
	tmp := t.TempDir()
	weak := []byte("[app]\nsigningKey = \"REPLACE_SIGNING_KEY\"\n")
	if err := os.WriteFile(filepath.Join(tmp, "config.toml"), weak, 0o600); err != nil {
		t.Fatalf("write weak config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestServeSigningKeyHelperProcess$")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), serveSigningKeyHelperEnv+"=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("ginServe under weak signing key must exit non-zero, got err=%v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("ginServe exit code = %d, want 1 (os.Exit(1) from the startup guard)", exitErr.ExitCode())
	}
}

// TestServeSigningKeyHelperProcess is the child-process entry: with the helper
// env var set it calls ginServe, which must hit the startup guard and os.Exit(1)
// under the weak key. When run as part of the normal test suite (no env var) it
// is a silent no-op so it does not pollute `go test ./...` output.
func TestServeSigningKeyHelperProcess(t *testing.T) {
	if os.Getenv(serveSigningKeyHelperEnv) != "1" {
		return
	}
	ginServe()
	os.Exit(0) // 守卫被绕过才到达这里：弱密钥下服务不应启动
}
