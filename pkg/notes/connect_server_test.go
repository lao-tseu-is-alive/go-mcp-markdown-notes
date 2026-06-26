package notes

import (
	"context"
	"errors"
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

func TestConnectServerGetNote(t *testing.T) {
	noteID := uuid.New()
	repository := &recordingRepository{note: &Note{ID: noteID, OwnerUserID: 71, Title: "hello"}}
	service := newTestService(t, repository)
	server, _ := NewConnectServer(service, nil)

	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeRead}})
	resp, err := server.GetNote(ctx, connect.NewRequest(&notesv1.GetNoteRequest{Id: noteID.String()}))
	if err != nil || resp.Msg.Note.Id != noteID.String() {
		t.Fatalf("GetNote() note=%v err=%v", resp, err)
	}

	// unauthenticated
	_, err = server.GetNote(context.Background(), connect.NewRequest(&notesv1.GetNoteRequest{Id: noteID.String()}))
	assertConnectCode(t, err, connect.CodeUnauthenticated)

	// repository miss → CodeNotFound
	repository.err = ErrNoteNotFound
	_, err = server.GetNote(ctx, connect.NewRequest(&notesv1.GetNoteRequest{Id: noteID.String()}))
	assertConnectCode(t, err, connect.CodeNotFound)

	// invalid UUID → CodeInvalidArgument
	repository.err = nil
	_, err = server.GetNote(ctx, connect.NewRequest(&notesv1.GetNoteRequest{Id: "not-a-uuid"}))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestConnectServerSearchNotes(t *testing.T) {
	repository := &recordingRepository{note: &Note{ID: uuid.New(), OwnerUserID: 71, Title: "result"}}
	service := newTestService(t, repository)
	server, _ := NewConnectServer(service, nil)

	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeRead}})
	resp, err := server.SearchNotes(ctx, connect.NewRequest(&notesv1.SearchNotesRequest{Query: "result"}))
	if err != nil || len(resp.Msg.Notes) == 0 {
		t.Fatalf("SearchNotes() notes=%v err=%v", resp, err)
	}

	// write-only scope is not sufficient for reads
	writeCtx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeWrite}})
	_, err = server.SearchNotes(writeCtx, connect.NewRequest(&notesv1.SearchNotesRequest{}))
	assertConnectCode(t, err, connect.CodePermissionDenied)
}

func TestConnectServerSearchNotesPageToken(t *testing.T) {
	repository := &searchPageRepository{
		recordingRepository: recordingRepository{note: &Note{ID: uuid.New(), OwnerUserID: 71}},
		result: SearchResult{
			Notes: []*Note{
				{ID: uuid.New(), OwnerUserID: 71},
				{ID: uuid.New(), OwnerUserID: 71},
			},
			TotalSize: 5,
		},
	}
	service := newTestService(t, repository)
	server, _ := NewConnectServer(service, nil)
	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeRead}})

	pageToken := "2"
	limit := int32(2)
	resp, err := server.SearchNotes(ctx, connect.NewRequest(&notesv1.SearchNotesRequest{
		Query: "x", Limit: &limit, PageToken: &pageToken,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if repository.search.Offset != 2 {
		t.Fatalf("search offset = %d, want 2", repository.search.Offset)
	}
	if resp.Msg.PageResponse.NextPageToken != "4" {
		t.Fatalf("next_page_token = %q, want 4", resp.Msg.PageResponse.NextPageToken)
	}
	if resp.Msg.PageResponse.TotalSize != 5 {
		t.Fatalf("total_size = %d, want 5", resp.Msg.PageResponse.TotalSize)
	}
}

func TestConnectServerSearchNotesRejectsInvalidPageToken(t *testing.T) {
	service := newTestService(t, &recordingRepository{})
	server, _ := NewConnectServer(service, nil)
	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeRead}})

	badToken := "not-a-number"
	_, err := server.SearchNotes(ctx, connect.NewRequest(&notesv1.SearchNotesRequest{PageToken: &badToken}))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

type searchPageRepository struct {
	recordingRepository
	result SearchResult
}

func (r *searchPageRepository) SearchNotes(_ context.Context, ownerUserID int64, filter SearchFilter) (SearchResult, error) {
	r.recordingRepository.SearchNotes(context.Background(), ownerUserID, filter)
	return r.result, r.err
}

func TestConnectServerAddTags(t *testing.T) {
	noteID := uuid.New()
	repository := &recordingRepository{note: &Note{ID: noteID, OwnerUserID: 71, Tags: []string{"go"}}}
	service := newTestService(t, repository)
	server, _ := NewConnectServer(service, nil)

	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeWrite}})
	resp, err := server.AddTags(ctx, connect.NewRequest(&notesv1.AddTagsRequest{NoteId: noteID.String(), Tags: []string{"mcp"}}))
	if err != nil || resp.Msg.Note == nil {
		t.Fatalf("AddTags() note=%v err=%v", resp, err)
	}

	// read-only scope rejected
	readCtx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeRead}})
	_, err = server.AddTags(readCtx, connect.NewRequest(&notesv1.AddTagsRequest{NoteId: noteID.String(), Tags: []string{"x"}}))
	assertConnectCode(t, err, connect.CodePermissionDenied)

	// invalid UUID
	_, err = server.AddTags(ctx, connect.NewRequest(&notesv1.AddTagsRequest{NoteId: "bad", Tags: []string{"x"}}))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestConnectServerUpdateNote(t *testing.T) {
	noteID := uuid.New()
	repository := &recordingRepository{note: &Note{ID: noteID, OwnerUserID: 71, Title: "updated"}}
	service := newTestService(t, repository)
	server, _ := NewConnectServer(service, nil)

	ctx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeWrite}})
	resp, err := server.UpdateNote(ctx, connect.NewRequest(&notesv1.UpdateNoteRequest{
		NoteId: noteID.String(), Title: "updated",
	}))
	if err != nil || resp.Msg.Note.Id != noteID.String() {
		t.Fatalf("UpdateNote() note=%v err=%v", resp, err)
	}

	// read scope is not enough
	readCtx := authadapter.ContextWithUser(context.Background(), &authadapter.AuthenticatedUser{AppUserID: 71, Scopes: []string{ScopeRead}})
	_, err = server.UpdateNote(readCtx, connect.NewRequest(&notesv1.UpdateNoteRequest{NoteId: noteID.String(), Title: "x"}))
	assertConnectCode(t, err, connect.CodePermissionDenied)

	// invalid note UUID
	_, err = server.UpdateNote(ctx, connect.NewRequest(&notesv1.UpdateNoteRequest{NoteId: "bad", Title: "x"}))
	assertConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestNewTimeoutInterceptor(t *testing.T) {
	interceptor := NewTimeoutInterceptor(50 * time.Millisecond)
	var deadlineSet bool
	handler := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		_, deadlineSet = ctx.Deadline()
		return connect.NewResponse(&notesv1.ListRecentNotesResponse{}), nil
	})
	req := connect.NewRequest(&notesv1.ListRecentNotesRequest{})
	if _, err := interceptor.WrapUnary(handler)(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if !deadlineSet {
		t.Fatal("timeout interceptor did not set a deadline on the context")
	}
}

func TestConnectServerMapError(t *testing.T) {
	service := newTestService(t, &recordingRepository{})
	server, _ := NewConnectServer(service, slog.New(slog.NewTextHandler(io.Discard, nil)))

	cases := []struct {
		err  error
		code connect.Code
	}{
		{ErrInvalidInput, connect.CodeInvalidArgument},
		{ErrNoteNotFound, connect.CodeNotFound},
		{ErrUnauthenticated, connect.CodeUnauthenticated},
		{errors.New("unexpected"), connect.CodeInternal},
	}
	for _, tc := range cases {
		got := server.mapError(tc.err)
		if got.Code() != tc.code {
			t.Errorf("mapError(%v).Code() = %v, want %v", tc.err, got.Code(), tc.code)
		}
	}
}

func assertConnectCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	connectErr, ok := err.(*connect.Error)
	if !ok || connectErr.Code() != want {
		t.Fatalf("error = %v, want Connect code %v", err, want)
	}
}
