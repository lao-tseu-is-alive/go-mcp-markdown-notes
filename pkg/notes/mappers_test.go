package notes

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
)

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
