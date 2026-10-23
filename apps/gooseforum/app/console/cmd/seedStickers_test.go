package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"strings"
	"testing"
)

func TestSeedStickersFailureReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := &cobra.Command{}
	command.SetContext(ctx)
	out := &bytes.Buffer{}
	command.SetOut(out)
	if err := runSeedStickers(command, nil); err == nil {
		t.Fatal("failed imports must fail the command")
	}
	// 每条失败都要打印原因分类, 否则 223 个预设里 6 个失败时无法定位(issue #1284)。
	if !strings.Contains(out.String(), "failed (") {
		t.Fatalf("per-sticker failure reason missing from output: %q", out.String())
	}
}

func TestFailureReason(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errors.New("tooLarge"), "tooLarge"},
		{errors.New("invalidImage"), "invalidImage"},
		{errors.New("unusableName"), "unusableName"},
		{fmt.Errorf("saveFailed: %w", errors.New("disk full")), "saveFailed"},
		// 无导入前缀的错误(如取消的 context)不得原样当作导入原因。
		{context.Canceled, "saveFailed"},
	}
	for _, c := range cases {
		if got := failureReason(c.err); got != c.want {
			t.Errorf("failureReason(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}
