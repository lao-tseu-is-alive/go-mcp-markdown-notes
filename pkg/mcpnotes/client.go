package mcpnotes

import (
	"net/http"

	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notesclient"
)

// NewNotesClient builds a Connect client for the notes-server that sends the
// given bearer token (a personal access token "pat_..." or a dev token) on
// every RPC call.
//
// This is a thin wrapper around the shared implementation in pkg/notesclient.
func NewNotesClient(baseURL, token string, httpClient *http.Client) (notesv1connect.NotesServiceClient, error) {
	opts := []notesclient.Option{}
	if httpClient != nil {
		opts = append(opts, notesclient.WithHTTPClient(httpClient))
	}
	return notesclient.New(baseURL, token, opts...)
}
