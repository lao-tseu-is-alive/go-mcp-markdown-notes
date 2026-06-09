package main

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
)

// newNotesClient creates a ConnectRPC client for the NotesService, automatically
// attaching the authorization token (if provided) to all requests via an interceptor.
func newNotesClient(serverURL, token string) notesv1connect.NotesServiceClient {
	serverURL = strings.TrimRight(serverURL, "/")
	httpClient := http.DefaultClient

	if token != "" {
		authInterceptor := connect.WithInterceptors(connect.UnaryInterceptorFunc(
			func(next connect.UnaryFunc) connect.UnaryFunc {
				return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
					// Attach Bearer token header
					req.Header().Set("Authorization", "Bearer "+token)
					return next(ctx, req)
				}
			},
		))
		return notesv1connect.NewNotesServiceClient(httpClient, serverURL, authInterceptor)
	}

	return notesv1connect.NewNotesServiceClient(httpClient, serverURL)
}
