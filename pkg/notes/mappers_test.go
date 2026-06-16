package notes

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
)

func TestDomainNotesToProto(t *testing.T) {
	notes := []*Note{
		{ID: uuid.New(), OwnerUserID: 1, Title: "a"},
		{ID: uuid.New(), OwnerUserID: 1, Title: "b"},
	}
	protos := DomainNotesToProto(notes)
	if len(protos) != 2 || protos[0].Title != "a" || protos[1].Title != "b" {
		t.Fatalf("DomainNotesToProto = %v", protos)
	}
	if got := DomainNotesToProto(nil); len(got) != 0 {
		t.Fatalf("nil input: len = %d, want 0", len(got))
	}
}

func TestDomainNoteToProtoNil(t *testing.T) {
	if got := DomainNoteToProto(nil); got != nil {
		t.Fatalf("DomainNoteToProto(nil) = %v, want nil", got)
	}
}

func TestProtoNoteToDomainErrors(t *testing.T) {
	if got, err := ProtoNoteToDomain(nil); got != nil || err != nil {
		t.Fatalf("nil note: got=%v err=%v", got, err)
	}
	if _, err := ProtoNoteToDomain(&notesv1.Note{Id: "bad-uuid"}); err == nil {
		t.Error("bad UUID should return an error")
	}
}

func TestNoteProtoRoundTrip(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 30, 0, 123, time.UTC)
	want := &Note{
		ID: uuid.New(), OwnerUserID: 123, Title: "title", BodyMarkdown: "body",
		Category: "architecture", Tags: []string{"go", "mcp"}, CreatedAt: now, UpdatedAt: now,
	}
	got, err := ProtoNoteToDomain(DomainNoteToProto(want))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}
