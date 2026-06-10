package mcpnotes

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
)

// NewNotesClient builds a Connect client for the notes-server that sends the
// given bearer token (a personal access token "pat_..." or a dev token) on
// every RPC call.
func NewNotesClient(baseURL, token string, httpClient *http.Client) (notesv1connect.NotesServiceClient, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil, errors.New("notes server URL is required")
	}
	if token == "" {
		return nil, errors.New("bearer token is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return notesv1connect.NewNotesServiceClient(
		httpClient,
		baseURL,
		connect.WithInterceptors(bearerTokenInterceptor(token)),
	), nil
}

// bearerTokenInterceptor attaches the Authorization header to every request.
func bearerTokenInterceptor(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	}
}
