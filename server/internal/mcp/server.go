package mcp

import (
	"context"
	"io"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/multica-ai/multica/server/internal/cli"
)

// Server is the Multica MCP server. It wraps the mcp-go MCPServer and holds
// the API client used by tool handlers.
type Server struct {
	mcp    *mcpserver.MCPServer
	client *cli.APIClient
}

// New creates a new Server with the given API client and binary version string.
func New(client *cli.APIClient, version string) *Server {
	s := mcpserver.NewMCPServer("multica", version, mcpserver.WithToolCapabilities(false))
	srv := &Server{mcp: s, client: client}
	registerReadTools(srv)
	registerFeatureTools(srv)
	registerIssueTools(srv)
	return srv
}

// Serve runs the MCP server over os.Stdin / os.Stdout.
func (s *Server) Serve(ctx context.Context) error {
	return mcpserver.ServeStdio(s.mcp)
}

// ServeReadWriter runs the server over the provided reader/writer.
// Used by tests to exercise the protocol without spawning a subprocess.
func (s *Server) ServeReadWriter(ctx context.Context, r io.Reader, w io.Writer) error {
	ss := mcpserver.NewStdioServer(s.mcp)
	return ss.Listen(ctx, r, w)
}
