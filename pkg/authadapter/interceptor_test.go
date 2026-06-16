package authadapter

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"connectrpc.com/connect"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
)

func TestNewInterceptor(t *testing.T) {
	verifier, err := NewDevTokenVerifier("test-token", AuthenticatedUser{AppUserID: 42, Scopes: []string{"notes:read"}})
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	var capturedUser *AuthenticatedUser
	passThrough := connect.UnaryFunc(func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		capturedUser, _ = UserFromContext(ctx)
		return connect.NewResponse(&notesv1.ListRecentNotesResponse{}), nil
	})

	interceptor := NewInterceptor(verifier, log)

	t.Run("valid token stores user in context", func(t *testing.T) {
		capturedUser = nil
		req := connect.NewRequest(&notesv1.ListRecentNotesRequest{})
		req.Header().Set("Authorization", "Bearer test-token")
		if _, err := interceptor.WrapUnary(passThrough)(context.Background(), req); err != nil {
			t.Fatal(err)
		}
		if capturedUser == nil || capturedUser.AppUserID != 42 {
			t.Fatalf("capturedUser = %#v, want AppUserID=42", capturedUser)
		}
	})

	t.Run("missing Authorization header returns Unauthenticated", func(t *testing.T) {
		req := connect.NewRequest(&notesv1.ListRecentNotesRequest{})
		_, err := interceptor.WrapUnary(passThrough)(context.Background(), req)
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("error = %v, want CodeUnauthenticated", err)
		}
	})

	t.Run("wrong token returns Unauthenticated", func(t *testing.T) {
		req := connect.NewRequest(&notesv1.ListRecentNotesRequest{})
		req.Header().Set("Authorization", "Bearer wrong-token")
		_, err := interceptor.WrapUnary(passThrough)(context.Background(), req)
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("error = %v, want CodeUnauthenticated", err)
		}
	})

	t.Run("malformed Authorization value returns Unauthenticated", func(t *testing.T) {
		req := connect.NewRequest(&notesv1.ListRecentNotesRequest{})
		req.Header().Set("Authorization", "NotBearer")
		_, err := interceptor.WrapUnary(passThrough)(context.Background(), req)
		if connect.CodeOf(err) != connect.CodeUnauthenticated {
			t.Fatalf("error = %v, want CodeUnauthenticated", err)
		}
	})

	t.Run("nil verifier returns Internal", func(t *testing.T) {
		nilInterceptor := NewInterceptor(nil, log)
		req := connect.NewRequest(&notesv1.ListRecentNotesRequest{})
		req.Header().Set("Authorization", "Bearer test-token")
		_, err := nilInterceptor.WrapUnary(passThrough)(context.Background(), req)
		if connect.CodeOf(err) != connect.CodeInternal {
			t.Fatalf("error = %v, want CodeInternal", err)
		}
	})

	t.Run("nil logger falls back to default without panic", func(t *testing.T) {
		noLogInterceptor := NewInterceptor(verifier, nil)
		req := connect.NewRequest(&notesv1.ListRecentNotesRequest{})
		req.Header().Set("Authorization", "Bearer test-token")
		if _, err := noLogInterceptor.WrapUnary(passThrough)(context.Background(), req); err != nil {
			t.Fatal(err)
		}
	})
}

func TestHasScopeAdminWildcard(t *testing.T) {
	admin := &AuthenticatedUser{AppUserID: 1, Scopes: []string{"notes:admin"}}
	for _, scope := range []string{"notes:read", "notes:write", "notes:mcp", "notes:anything"} {
		if !admin.HasScope(scope) {
			t.Errorf("admin should satisfy scope %q", scope)
		}
	}
	readOnly := &AuthenticatedUser{AppUserID: 1, Scopes: []string{"notes:read"}}
	if readOnly.HasScope("notes:write") {
		t.Error("read-only user should not have notes:write scope")
	}
	if readOnly.HasScope("notes:admin") {
		t.Error("read-only user should not have notes:admin scope")
	}
}

func TestCompositeVerifierNilRejected(t *testing.T) {
	valid, _ := NewDevTokenVerifier("x", AuthenticatedUser{AppUserID: 1})
	if _, err := NewCompositeVerifier(nil, valid); err == nil {
		t.Error("nil jwt verifier should be rejected")
	}
	if _, err := NewCompositeVerifier(valid, nil); err == nil {
		t.Error("nil pat verifier should be rejected")
	}
}
