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
	if os.Getenv("GO_WANT_GIN_SERVE_WEAK") == "1" {
		ginServe()
		os.Exit(0) // 守卫被绕过才到达这里：弱密钥下服务不应启动
	}

	tmp := t.TempDir()
	weak := []byte("[app]\nsigningKey = \"REPLACE_SIGNING_KEY\"\n")
	if err := os.WriteFile(filepath.Join(tmp, "config.toml"), weak, 0o600); err != nil {
		t.Fatalf("write weak config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestGinServeExitsNonZeroOnWeakSigningKey$")
	cmd.Dir = tmp
	cmd.Env = append(os.Environ(), "GO_WANT_GIN_SERVE_WEAK=1")
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("ginServe under weak signing key must exit non-zero, got err=%v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("ginServe exit code = %d, want 1 (os.Exit(1) from the startup guard)", exitErr.ExitCode())
	}
}
