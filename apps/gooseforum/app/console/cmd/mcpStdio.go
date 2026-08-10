package cmd

import (
	"log/slog"
	"os"

	db "github.com/leancodebox/GooseForum/app/bundles/connect/dbconnect"
	"github.com/leancodebox/GooseForum/app/service/agentservice"
	"github.com/leancodebox/GooseForum/app/service/mcpservice"
	"github.com/spf13/cobra"
)

// envAgentToken is the environment variable that carries the agt_ token for
// the stdio transport. Local CLI clients set it once; the MCP server resolves
// the agent identity from it at startup and binds the whole stdio session to
// that single bot persona.
const envAgentToken = "YOURTJ_AGENT_TOKEN"

func init() {
	cmd := &cobra.Command{
		Use:   "mcp-stdio",
		Short: "Serve the Agent Bot API over MCP stdio (JSON-RPC on stdin/stdout)",
		Args:  cobra.NoArgs,
		RunE:  runMcpStdio,
	}
	cmd.Flags().Bool("writes", false, "enable write tools (create_topic/create_post) for this session, independent of the global mcp.writes preference")
	appendCommand(cmd)
}

func runMcpStdio(cmd *cobra.Command, _ []string) error {
	// The token is read from the environment only, never from a command-line
	// flag: flags are visible to other local users via ps and are captured in
	// shell history, and this token is the long-lived bot credential.
	token := os.Getenv(envAgentToken)
	if token == "" {
		slog.Error("mcp-stdio requires an agt_ token", "hint", "set "+envAgentToken)
		os.Exit(1)
	}

	// Resolve the agent identity once; the whole stdio session acts as this bot.
	agent, _, err := agentservice.ResolveByToken(token)
	if err != nil || agent == nil {
		slog.Error("mcp-stdio token rejected", "error", err)
		os.Exit(1)
	}

	// Force a DB connection so test/startup wiring behaves like other commands.
	_ = db.Connect()

	writes, _ := cmd.Flags().GetBool("writes")
	svc := mcpservice.NewStdioService(agent.UserId, writes)
	if err := svc.RunStdio(cmd.Context()); err != nil {
		slog.Error("mcp-stdio exited", "error", err)
		os.Exit(1)
	}
	return nil
}
