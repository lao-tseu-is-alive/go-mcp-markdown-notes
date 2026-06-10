package mcpnotes

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
)

// fakeNotesClient implements notesv1connect.NotesServiceClient in memory.
type fakeNotesClient struct {
	note      *notesv1.Note
	err       error
	deletedID string
}

var _ notesv1connect.NotesServiceClient = (*fakeNotesClient)(nil)

func (f *fakeNotesClient) CreateNote(_ context.Context, req *connect.Request[notesv1.CreateNoteRequest]) (*connect.Response[notesv1.CreateNoteResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&notesv1.CreateNoteResponse{Note: f.note}), nil
}

func (f *fakeNotesClient) GetNote(_ context.Context, _ *connect.Request[notesv1.GetNoteRequest]) (*connect.Response[notesv1.GetNoteResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&notesv1.GetNoteResponse{Note: f.note}), nil
}

func (f *fakeNotesClient) ListRecentNotes(_ context.Context, _ *connect.Request[notesv1.ListRecentNotesRequest]) (*connect.Response[notesv1.ListRecentNotesResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&notesv1.ListRecentNotesResponse{Notes: []*notesv1.Note{f.note}}), nil
}

func (f *fakeNotesClient) SearchNotes(_ context.Context, _ *connect.Request[notesv1.SearchNotesRequest]) (*connect.Response[notesv1.SearchNotesResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&notesv1.SearchNotesResponse{Notes: []*notesv1.Note{f.note}}), nil
}

func (f *fakeNotesClient) AddTags(_ context.Context, _ *connect.Request[notesv1.AddTagsRequest]) (*connect.Response[notesv1.AddTagsResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&notesv1.AddTagsResponse{Note: f.note}), nil
}

func (f *fakeNotesClient) UpdateNote(_ context.Context, _ *connect.Request[notesv1.UpdateNoteRequest]) (*connect.Response[notesv1.UpdateNoteResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	return connect.NewResponse(&notesv1.UpdateNoteResponse{Note: f.note}), nil
}

func (f *fakeNotesClient) DeleteNote(_ context.Context, req *connect.Request[notesv1.DeleteNoteRequest]) (*connect.Response[notesv1.DeleteNoteResponse], error) {
	if f.err != nil {
		return nil, f.err
	}
	f.deletedID = req.Msg.NoteId
	return connect.NewResponse(&notesv1.DeleteNoteResponse{}), nil
}

// newTestSession wires the MCP server to an in-memory client session.
func newTestSession(t *testing.T, fake *fakeNotesClient) *mcp.ClientSession {
	t.Helper()
	server, err := NewServer(fake, "test")
	if err != nil {
		t.Fatal(err)
	}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	if _, err := server.Connect(context.Background(), serverTransport, nil); err != nil {
		t.Fatal(err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestServerExposesAllTools(t *testing.T) {
	session := newTestSession(t, &fakeNotesClient{})
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"create_note": false, "get_note": false, "list_recent_notes": false,
		"search_notes": false, "add_tags": false, "update_note": false, "delete_note": false,
	}
	for _, tool := range tools.Tools {
		if _, ok := want[tool.Name]; ok {
			want[tool.Name] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("tool %q is not exposed", name)
		}
	}
}

func TestCreateAndDeleteNoteTools(t *testing.T) {
	fake := &fakeNotesClient{note: &notesv1.Note{Id: "11111111-2222-3333-4444-555555555555", Title: "hello", BodyMarkdown: "# hi"}}
	session := newTestSession(t, fake)
	ctx := context.Background()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "create_note",
		Arguments: map[string]any{"title": "hello", "body_markdown": "# hi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("create_note returned a tool error: %+v", res.Content)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var created NoteResult
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}
	if created.Note.Title != "hello" {
		t.Fatalf("created = %+v", created)
	}

	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "delete_note",
		Arguments: map[string]any{"note_id": created.Note.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("delete_note returned a tool error: %+v", res.Content)
	}
	if fake.deletedID != created.Note.ID {
		t.Fatalf("deletedID = %q, want %q", fake.deletedID, created.Note.ID)
	}
}

func TestUnauthenticatedErrorMentionsToken(t *testing.T) {
	fake := &fakeNotesClient{err: connect.NewError(connect.CodeUnauthenticated, errors.New("invalid bearer token"))}
	session := newTestSession(t, fake)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "list_recent_notes",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected a tool error for an unauthenticated call")
	}
	text := ""
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			text += tc.Text
		}
	}
	if !strings.Contains(text, "NOTES_TOKEN") {
		t.Fatalf("error message should point at NOTES_TOKEN, got: %s", text)
	}
}
