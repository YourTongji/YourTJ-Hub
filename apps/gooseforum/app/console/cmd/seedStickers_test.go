package cmd

import (
	"context"
	"github.com/spf13/cobra"
	"io"
	"testing"
)

func TestSeedStickersFailureReturnsError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	command := &cobra.Command{}
	command.SetContext(ctx)
	command.SetOut(io.Discard)
	if err := runSeedStickers(command, nil); err == nil {
		t.Fatal("failed imports must fail the command")
	}
}
