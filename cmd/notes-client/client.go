package main

import (
	"fmt"
	"os"

	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notesclient"
)

// newNotesClient creates a ConnectRPC client for the NotesService, automatically
// attaching the authorization token (if provided) to all requests via an interceptor.
//
// It uses the shared implementation from pkg/notesclient.
func newNotesClient(serverURL, token string) notesv1connect.NotesServiceClient {
	client, err := notesclient.New(serverURL, token)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to create notes client:", err)
		os.Exit(1)
	}
	return client
}
