package notes

import (
	"context"
	"errors"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/authadapter"
)

const (
	ScopeRead  = "notes:read"
	ScopeWrite = "notes:write"
)

// ConnectServer exposes Service through the generated NotesService contract.
type ConnectServer struct {
	service *Service
	log     *slog.Logger
	notesv1connect.UnimplementedNotesServiceHandler
}

func NewConnectServer(service *Service, log *slog.Logger) (*ConnectServer, error) {
	if service == nil {
		return nil, errors.New("notes service is required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &ConnectServer{service: service, log: log}, nil
}

func (s *ConnectServer) CreateNote(ctx context.Context, req *connect.Request[notesv1.CreateNoteRequest]) (*connect.Response[notesv1.CreateNoteResponse], error) {
	ownerUserID, err := requireOwner(ctx, ScopeWrite)
	if err != nil {
		return nil, err
	}
	note, err := s.service.CreateNote(ctx, ownerUserID, CreateNoteInput{
		Title: req.Msg.Title, BodyMarkdown: req.Msg.BodyMarkdown, Category: req.Msg.Category, Tags: req.Msg.Tags,
	})
	if err != nil {
		return nil, s.mapError(err)
	}
	return connect.NewResponse(&notesv1.CreateNoteResponse{Note: DomainNoteToProto(note)}), nil
}

func (s *ConnectServer) GetNote(ctx context.Context, req *connect.Request[notesv1.GetNoteRequest]) (*connect.Response[notesv1.GetNoteResponse], error) {
	ownerUserID, err := requireOwner(ctx, ScopeRead)
	if err != nil {
		return nil, err
	}
	noteID, err := parseNoteID(req.Msg.Id)
	if err != nil {
		return nil, err
	}
	note, err := s.service.GetNote(ctx, ownerUserID, noteID)
	if err != nil {
		return nil, s.mapError(err)
	}
	return connect.NewResponse(&notesv1.GetNoteResponse{Note: DomainNoteToProto(note)}), nil
}

func (s *ConnectServer) ListRecentNotes(ctx context.Context, req *connect.Request[notesv1.ListRecentNotesRequest]) (*connect.Response[notesv1.ListRecentNotesResponse], error) {
	ownerUserID, err := requireOwner(ctx, ScopeRead)
	if err != nil {
		return nil, err
	}
	items, err := s.service.ListRecentNotes(ctx, ownerUserID, int(req.Msg.Limit))
	if err != nil {
		return nil, s.mapError(err)
	}
	return connect.NewResponse(&notesv1.ListRecentNotesResponse{Notes: DomainNotesToProto(items)}), nil
}

func (s *ConnectServer) SearchNotes(ctx context.Context, req *connect.Request[notesv1.SearchNotesRequest]) (*connect.Response[notesv1.SearchNotesResponse], error) {
	ownerUserID, err := requireOwner(ctx, ScopeRead)
	if err != nil {
		return nil, err
	}
	items, err := s.service.SearchNotes(ctx, ownerUserID, SearchFilter{
		Query: req.Msg.Query, Tags: req.Msg.Tags, Category: req.Msg.Category, Limit: int(req.Msg.Limit),
	})
	if err != nil {
		return nil, s.mapError(err)
	}
	return connect.NewResponse(&notesv1.SearchNotesResponse{Notes: DomainNotesToProto(items)}), nil
}

func (s *ConnectServer) AddTags(ctx context.Context, req *connect.Request[notesv1.AddTagsRequest]) (*connect.Response[notesv1.AddTagsResponse], error) {
	ownerUserID, err := requireOwner(ctx, ScopeWrite)
	if err != nil {
		return nil, err
	}
	noteID, err := parseNoteID(req.Msg.NoteId)
	if err != nil {
		return nil, err
	}
	note, err := s.service.AddTags(ctx, ownerUserID, noteID, req.Msg.Tags)
	if err != nil {
		return nil, s.mapError(err)
	}
	return connect.NewResponse(&notesv1.AddTagsResponse{Note: DomainNoteToProto(note)}), nil
}

func (s *ConnectServer) UpdateNote(ctx context.Context, req *connect.Request[notesv1.UpdateNoteRequest]) (*connect.Response[notesv1.UpdateNoteResponse], error) {
	ownerUserID, err := requireOwner(ctx, ScopeWrite)
	if err != nil {
		return nil, err
	}
	noteID, err := parseNoteID(req.Msg.NoteId)
	if err != nil {
		return nil, err
	}
	note, err := s.service.UpdateNote(ctx, ownerUserID, noteID, UpdateNoteInput{
		Title: req.Msg.Title, BodyMarkdown: req.Msg.BodyMarkdown, Category: req.Msg.Category, Tags: req.Msg.Tags,
	})
	if err != nil {
		return nil, s.mapError(err)
	}
	return connect.NewResponse(&notesv1.UpdateNoteResponse{Note: DomainNoteToProto(note)}), nil
}

func requireOwner(ctx context.Context, scope string) (int64, error) {
	user, err := authadapter.RequireUser(ctx)
	if err != nil {
		return 0, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if !user.HasScope(scope) {
		return 0, connect.NewError(connect.CodePermissionDenied, errors.New("required scope is missing"))
	}
	return user.AppUserID, nil
}

func parseNoteID(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid note ID"))
	}
	return id, nil
}

func (s *ConnectServer) mapError(err error) *connect.Error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrNoteNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrUnauthenticated):
		return connect.NewError(connect.CodeUnauthenticated, err)
	default:
		s.log.Error("notes request failed", "error", err)
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
}
