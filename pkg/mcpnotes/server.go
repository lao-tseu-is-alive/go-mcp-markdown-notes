package mcpnotes

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NoteOutput is the JSON shape of a note returned by every tool.
type NoteOutput struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	BodyMarkdown string   `json:"body_markdown"`
	Category     string   `json:"category,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Status       string   `json:"status,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

func noteStatusToString(s notesv1.NoteStatus) string {
	switch s {
	case notesv1.NoteStatus_NOTE_STATUS_DRAFT:
		return "draft"
	case notesv1.NoteStatus_NOTE_STATUS_ACTIVE:
		return "active"
	case notesv1.NoteStatus_NOTE_STATUS_FINAL:
		return "final"
	case notesv1.NoteStatus_NOTE_STATUS_ARCHIVED:
		return "archived"
	default:
		return ""
	}
}

func noteStatusFromString(s string) *notesv1.NoteStatus {
	var v notesv1.NoteStatus
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "draft":
		v = notesv1.NoteStatus_NOTE_STATUS_DRAFT
	case "active":
		v = notesv1.NoteStatus_NOTE_STATUS_ACTIVE
	case "final":
		v = notesv1.NoteStatus_NOTE_STATUS_FINAL
	case "archived":
		v = notesv1.NoteStatus_NOTE_STATUS_ARCHIVED
	default:
		return nil
	}
	return &v
}

func protoNoteToOutput(n *notesv1.Note) NoteOutput {
	if n == nil {
		return NoteOutput{}
	}
	return NoteOutput{
		ID:           n.Id,
		Title:        n.Title,
		BodyMarkdown: n.BodyMarkdown,
		Category:     n.Category,
		Tags:         n.Tags,
		Status:       noteStatusToString(n.Status),
		CreatedAt:    protoTimestampToString(n.CreatedAt),
		UpdatedAt:    protoTimestampToString(n.UpdatedAt),
	}
}

func protoTimestampToString(ts *timestamppb.Timestamp) string {
	if ts == nil || ts.CheckValid() != nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339Nano)
}

func protoNotesToOutput(notes []*notesv1.Note) []NoteOutput {
	result := make([]NoteOutput, len(notes))
	for i, n := range notes {
		result[i] = protoNoteToOutput(n)
	}
	return result
}

// mapRPCError converts Connect errors to actionable tool errors. The token is
// configured out-of-band (env var), so authentication problems need a hint.
func mapRPCError(operation string, err error) error {
	switch connect.CodeOf(err) {
	case connect.CodeUnauthenticated:
		return fmt.Errorf("%s: the notes server rejected the token; check the NOTES_TOKEN environment variable (expired or revoked personal access token?)", operation)
	case connect.CodePermissionDenied:
		return fmt.Errorf("%s: the token lacks the required scope (notes:read / notes:write)", operation)
	case connect.CodeNotFound:
		return fmt.Errorf("%s: note not found (or it belongs to another user)", operation)
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}

// --- Tool inputs & outputs ---

type CreateNoteInput struct {
	Title        string   `json:"title" jsonschema:"title of the note (required, max 200 characters)"`
	BodyMarkdown string   `json:"body_markdown" jsonschema:"note content in Markdown format"`
	Category     string   `json:"category,omitempty" jsonschema:"optional category, e.g. devops"`
	Tags         []string `json:"tags,omitempty" jsonschema:"optional lowercase tags"`
	Status       string   `json:"status,omitempty" jsonschema:"lifecycle status: draft, active, final, or archived (default: active)"`
}

type GetNoteInput struct {
	ID string `json:"id" jsonschema:"UUID of the note"`
}

type ListRecentNotesInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"maximum number of notes to return (default 10, max 100)"`
}

type SearchNotesInput struct {
	Query     string   `json:"query,omitempty" jsonschema:"text searched in title and body"`
	Tags      []string `json:"tags,omitempty" jsonschema:"notes must carry all of these tags"`
	Category  string   `json:"category,omitempty" jsonschema:"exact category filter"`
	Limit     int      `json:"limit,omitempty" jsonschema:"maximum number of notes to return (default 10, max 100)"`
	PageToken string   `json:"page_token,omitempty" jsonschema:"pagination cursor from a prior search_notes response"`
}

type AddTagsInput struct {
	NoteID string   `json:"note_id" jsonschema:"UUID of the note"`
	Tags   []string `json:"tags" jsonschema:"tags to add to the note"`
}

type UpdateNoteInput struct {
	NoteID       string   `json:"note_id" jsonschema:"UUID of the note to update"`
	Title        string   `json:"title" jsonschema:"new title (required, the full note is replaced)"`
	BodyMarkdown string   `json:"body_markdown" jsonschema:"new note content in Markdown format"`
	Category     string   `json:"category,omitempty" jsonschema:"new category"`
	Tags         []string `json:"tags,omitempty" jsonschema:"new full set of tags (replaces existing tags)"`
	Status       string   `json:"status,omitempty" jsonschema:"lifecycle status: draft, active, final, or archived"`
}

type DeleteNoteInput struct {
	NoteID string `json:"note_id" jsonschema:"UUID of the note to delete"`
}

type NoteResult struct {
	Note NoteOutput `json:"note"`
}

type NotesResult struct {
	Notes []NoteOutput `json:"notes"`
	Count int          `json:"count"`
}

type DeleteNoteResult struct {
	Deleted bool   `json:"deleted"`
	NoteID  string `json:"note_id"`
}

// NewServer builds the MCP server exposing the notes tools over the given
// Connect client.
func NewServer(client notesv1connect.NotesServiceClient, version string) (*mcp.Server, error) {
	if client == nil {
		return nil, errors.New("notes client is required")
	}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "notes-mcp",
		Title:   "Personal Markdown Notes",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "create_note",
		Description: "Create a new Markdown note with optional category and tags.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input CreateNoteInput) (*mcp.CallToolResult, NoteResult, error) {
		req := &notesv1.CreateNoteRequest{
			Title: input.Title, BodyMarkdown: input.BodyMarkdown, Category: input.Category, Tags: input.Tags,
			Status: noteStatusFromString(input.Status),
		}
		res, err := client.CreateNote(ctx, connect.NewRequest(req))
		if err != nil {
			return nil, NoteResult{}, mapRPCError("create_note", err)
		}
		return nil, NoteResult{Note: protoNoteToOutput(res.Msg.Note)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_note",
		Description: "Get a single note by its UUID, including its full Markdown body.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input GetNoteInput) (*mcp.CallToolResult, NoteResult, error) {
		res, err := client.GetNote(ctx, connect.NewRequest(&notesv1.GetNoteRequest{Id: input.ID}))
		if err != nil {
			return nil, NoteResult{}, mapRPCError("get_note", err)
		}
		return nil, NoteResult{Note: protoNoteToOutput(res.Msg.Note)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_recent_notes",
		Description: "List the most recently updated notes of the authenticated user.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input ListRecentNotesInput) (*mcp.CallToolResult, NotesResult, error) {
		res, err := client.ListRecentNotes(ctx, connect.NewRequest(&notesv1.ListRecentNotesRequest{Limit: int32(input.Limit)}))
		if err != nil {
			return nil, NotesResult{}, mapRPCError("list_recent_notes", err)
		}
		notes := protoNotesToOutput(res.Msg.Notes)
		return nil, NotesResult{Notes: notes, Count: len(notes)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_notes",
		Description: "Search notes by free text (title and body), tags and category.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input SearchNotesInput) (*mcp.CallToolResult, NotesResult, error) {
		var limitPtr *int32
		if input.Limit > 0 {
			l := int32(input.Limit)
			limitPtr = &l
		}
		var pageTokenPtr *string
		if input.PageToken != "" {
			pageTokenPtr = &input.PageToken
		}
		res, err := client.SearchNotes(ctx, connect.NewRequest(&notesv1.SearchNotesRequest{
			Query: input.Query, Tags: input.Tags, Category: input.Category, Limit: limitPtr, PageToken: pageTokenPtr,
		}))
		if err != nil {
			return nil, NotesResult{}, mapRPCError("search_notes", err)
		}
		notes := protoNotesToOutput(res.Msg.Notes)
		return nil, NotesResult{Notes: notes, Count: len(notes)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_tags",
		Description: "Add tags to an existing note (existing tags are kept).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input AddTagsInput) (*mcp.CallToolResult, NoteResult, error) {
		res, err := client.AddTags(ctx, connect.NewRequest(&notesv1.AddTagsRequest{
			NoteId: input.NoteID, Tags: input.Tags,
		}))
		if err != nil {
			return nil, NoteResult{}, mapRPCError("add_tags", err)
		}
		return nil, NoteResult{Note: protoNoteToOutput(res.Msg.Note)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "update_note",
		Description: "Replace the title, body, category and tags of an existing note.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input UpdateNoteInput) (*mcp.CallToolResult, NoteResult, error) {
		res, err := client.UpdateNote(ctx, connect.NewRequest(&notesv1.UpdateNoteRequest{
			NoteId: input.NoteID, Title: input.Title, BodyMarkdown: input.BodyMarkdown,
			Category: input.Category, Tags: input.Tags,
			Status: noteStatusFromString(input.Status),
		}))
		if err != nil {
			return nil, NoteResult{}, mapRPCError("update_note", err)
		}
		return nil, NoteResult{Note: protoNoteToOutput(res.Msg.Note)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "delete_note",
		Description: "Permanently delete a note by its UUID. This cannot be undone.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input DeleteNoteInput) (*mcp.CallToolResult, DeleteNoteResult, error) {
		_, err := client.DeleteNote(ctx, connect.NewRequest(&notesv1.DeleteNoteRequest{NoteId: input.NoteID}))
		if err != nil {
			return nil, DeleteNoteResult{}, mapRPCError("delete_note", err)
		}
		return nil, DeleteNoteResult{Deleted: true, NoteID: input.NoteID}, nil
	})

	return server, nil
}
