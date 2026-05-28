package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	multicamcp "github.com/multica-ai/multica/server/internal/mcp"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the Multica MCP server (stdio)",
	Long:  "Run an MCP server over stdio.\n\nAdd to Claude Code with:\n  claude mcp add multica -- multica mcp",
	RunE:  runMCP,
}

func runMCP(cmd *cobra.Command, _ []string) error {
	// Check token before resolveServerURL so the first error the user sees
	// is the relevant one (token missing vs no server configured).
	if token := resolveToken(cmd); token == "" {
		fmt.Fprintln(os.Stderr, "MULTICA_TOKEN is not set. Run 'multica setup' or set MULTICA_TOKEN.")
		os.Exit(2)
	}

	client, err := newAPIClient(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.HealthCheck(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Multica server is unreachable: %v\nCheck MULTICA_SERVER_URL or run 'multica setup'.\n", err)
		os.Exit(2)
	}

	s := multicamcp.New(client, version)
	return s.Serve(context.Background())
}
