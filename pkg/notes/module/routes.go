package module

// Route registration for the notes module.
//
// This project uses net/http + Connect RPC directly, without Vanguard REST transcoding.
// The analogue of VanguardServices() is ConnectHandlers(), which returns the Connect
// (path, http.Handler) pairs that the caller mounts on a shared http.ServeMux in bundle mode.
//
// Standalone mode   → call RegisterRoutes(mux).
// Bundle mode       → iterate ConnectHandlers() and mount each handler on the shared mux.

import (
	"net/http"

	"connectrpc.com/connect"
	connectvalidate "connectrpc.com/validate"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/authadapter"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB

// RoutePattern describes a URL pattern owned by this module, compatible with http.ServeMux.
type RoutePattern struct {
	// Pattern is an http.ServeMux-compatible pattern, e.g. "/notes.v1.NotesService/".
	Pattern string
}

// ConnectHandler pairs a URL pattern with its ready-to-use Connect HTTP handler.
// In bundle mode the caller mounts these on a shared http.ServeMux.
// This is the equivalent of VanguardServices() in projects that use Vanguard transcoding.
type ConnectHandler struct {
	Pattern string
	Handler http.Handler
}

// connectOption builds the standard Connect interceptor chain for this module:
// per-RPC timeout → bearer-token auth → proto validation.
func (m *Module) connectOption() connect.Option {
	return connect.WithInterceptors(
		notes.NewTimeoutInterceptor(m.cfg.requestTimeout()),
		authadapter.NewInterceptor(m.deps.Verifier, m.deps.Logger),
		connectvalidate.NewInterceptor(connectvalidate.WithValidateResponses()),
	)
}

// ConnectHandlers returns the Connect HTTP handlers for this module with the standard
// interceptor chain already applied. In bundle mode, callers mount these on a shared mux.
func (m *Module) ConnectHandlers() []ConnectHandler {
	path, handler := notesv1connect.NewNotesServiceHandler(m.connect, m.connectOption())
	return []ConnectHandler{
		{Pattern: path, Handler: http.MaxBytesHandler(handler, maxRequestBodyBytes)},
	}
}

// RoutePatterns returns the URL patterns this module serves. Useful for bundle
// callers that need an explicit list of owned patterns to avoid route conflicts.
func (m *Module) RoutePatterns() []RoutePattern {
	return []RoutePattern{
		{Pattern: "/" + notesv1connect.NotesServiceName + "/"},
	}
}

// ConnectPatterns returns the Connect/gRPC path prefixes handled by this module.
func (m *Module) ConnectPatterns() []string {
	return []string{"/" + notesv1connect.NotesServiceName + "/"}
}

// RegisterRoutes mounts all module handlers on mux for standalone mode.
//
// In bundle mode, callers should use ConnectHandlers() + RoutePatterns() instead
// so that a single shared mux can be used across all modules.
func (m *Module) RegisterRoutes(mux *http.ServeMux) error {
	for _, ch := range m.ConnectHandlers() {
		mux.Handle(ch.Pattern, ch.Handler)
	}
	m.deps.Logger.Info("notes module routes registered",
		"pattern", "/"+notesv1connect.NotesServiceName+"/")
	return nil
}
