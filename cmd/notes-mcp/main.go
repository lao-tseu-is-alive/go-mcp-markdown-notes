// notes-mcp exposes the notes-server Connect API as Model Context Protocol
// tools over stdio, for use by MCP clients such as Claude Code or Claude
// Desktop.
//
// Configuration (environment variables):
//
//	NOTES_SERVER  base URL of the notes-server (default http://127.0.0.1:8080)
//	NOTES_TOKEN   bearer token: a personal access token "pat_..." created in
//	              the auth service UI, or the dev token when the notes-server
//	              runs with NOTES_AUTH_MODE=dev (required)
//
// stdout carries the MCP protocol; all logging goes to stderr.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/mcpnotes"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/version"
)

const defaultNotesServer = "http://127.0.0.1:8080"

func main() {
	notesServer := strings.TrimSpace(os.Getenv("NOTES_SERVER"))
	if notesServer == "" {
		notesServer = defaultNotesServer
	}
	token := strings.TrimSpace(os.Getenv("NOTES_TOKEN"))
	if token == "" {
		fmt.Fprintln(os.Stderr, "💥 NOTES_TOKEN is required (a pat_... personal access token or the notes-server dev token)")
		os.Exit(1)
	}

	client, err := mcpnotes.NewNotesClient(notesServer, token, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "💥 failed to create notes client: %v\n", err)
		os.Exit(1)
	}
	server, err := mcpnotes.NewServer(client, version.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "💥 failed to create MCP server: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "🚀 notes-mcp v%s ready (notes-server: %s), serving MCP over stdio\n", version.Version, notesServer)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "💥 MCP server stopped: %v\n", err)
		os.Exit(1)
	}
}
