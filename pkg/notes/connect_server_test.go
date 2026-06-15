package notes

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	connectvalidate "connectrpc.com/validate"
	"github.com/google/uuid"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/authadapter"
)

func TestConnectServerCreateUsesAuthenticatedOwner(t *testing.T) {
	repository := &recordingRepository{note: &Note{ID: uuid.New(), OwnerUserID: 71}}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := newTestService(t, repository)
	server, err := NewConnectServer(service, logger)
	if err != nil {
		t.Fatal(err)
	}
	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{
		AppUserID: 71, Scopes: []string{ScopeWrite},
	})

	response, err := server.CreateNote(ctx, connect.NewRequest(&notesv1.CreateNoteRequest{Title: "test"}))
	if err != nil {
		t.Fatal(err)
	}
	if repository.ownerUserID != 71 || response.Msg.Note.OwnerUserId != "71" {
		t.Fatalf("owner was not propagated: repository=%d response=%q", repository.ownerUserID, response.Msg.Note.OwnerUserId)
	}
}

func TestConnectServerRequiresAuthenticationAndScope(t *testing.T) {
	service := newTestService(t, &recordingRepository{})
	server, err := NewConnectServer(service, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.ListRecentNotes(context.Background(), connect.NewRequest(&notesv1.ListRecentNotesRequest{}))
	assertConnectCode(t, err, connect.CodeUnauthenticated)

	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 1, Scopes: []string{ScopeRead}})
	_, err = server.CreateNote(ctx, connect.NewRequest(&notesv1.CreateNoteRequest{Title: "test"}))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

func TestConnectServerDeleteNote(t *testing.T) {
	repository := &recordingRepository{}
	service := newTestService(t, repository)
	server, err := NewConnectServer(service, nil)
	if err != nil {
		t.Fatal(err)
	}
	noteID := uuid.New()
	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{
		AppUserID: 71, Scopes: []string{ScopeWrite},
	})

	if _, err := server.DeleteNote(ctx, connect.NewRequest(&notesv1.DeleteNoteRequest{NoteId: noteID.String()})); err != nil {
		t.Fatal(err)
	}
	if repository.ownerUserID != 71 || repository.deletedID != noteID {
		t.Fatalf("delete was not scoped: owner=%d noteID=%v", repository.ownerUserID, repository.deletedID)
	}

	// Read-only scope is rejected.
	readCtx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeRead}})
	_, err = server.DeleteNote(readCtx, connect.NewRequest(&notesv1.DeleteNoteRequest{NoteId: noteID.String()}))
	assertConnectCode(t, err, connect.CodePermissionDenied)

	// Deleting another user's note (repository miss) maps to NotFound.
	repository.err = ErrNoteNotFound
	_, err = server.DeleteNote(ctx, connect.NewRequest(&notesv1.DeleteNoteRequest{NoteId: noteID.String()}))
	assertConnectCode(t, err, connect.CodeNotFound)
}

func TestConnectHandlerValidatesProtoRequests(t *testing.T) {
	repository := &recordingRepository{note: &Note{
		ID:           uuid.New(),
		OwnerUserID:  71,
		Title:        "test",
		BodyMarkdown: "",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}}
	service := newTestService(t, repository)
	server, err := NewConnectServer(service, nil)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := authadapter.NewDevTokenVerifier("test-token", authadapter.AuthenticatedUser{
		AppUserID: 71,
		Scopes:    []string{ScopeWrite},
	})
	if err != nil {
		t.Fatal(err)
	}
	path, handler := notesv1connect.NewNotesServiceHandler(server, connect.WithInterceptors(
		authadapter.NewInterceptor(verifier, slog.New(slog.NewTextHandler(io.Discard, nil))),
		connectvalidate.NewInterceptor(connectvalidate.WithValidateResponses()),
	))
	wantPath := "/" + notesv1connect.NotesServiceName + "/"
	if path != wantPath {
		t.Fatalf("handler path = %q, want %q", path, wantPath)
	}

	noteID := uuid.NewString()
	req := httptest.NewRequest(
		http.MethodPost,
		notesv1connect.NotesServiceAddTagsProcedure,
		strings.NewReader(`{"noteId":"`+noteID+`","tags":["go","go"]}`),
	)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if repository.ownerUserID != 0 {
		t.Fatalf("request reached repository with owner %d, want validation to short-circuit", repository.ownerUserID)
	}
}

func assertConnectCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	connectErr, ok := err.(*connect.Error)
	if !ok || connectErr.Code() != want {
		t.Fatalf("error = %v, want Connect code %v", err, want)
	}
}
