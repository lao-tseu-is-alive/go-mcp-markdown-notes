package notes

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
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

func assertConnectCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	connectErr, ok := err.(*connect.Error)
	if !ok || connectErr.Code() != want {
		t.Fatalf("error = %v, want Connect code %v", err, want)
	}
}
