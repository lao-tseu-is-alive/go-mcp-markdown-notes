package notesclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
)

func TestNew_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		token   string
		wantErr string
	}{
		{
			name:    "empty baseURL",
			baseURL: "",
			token:   "tok",
			wantErr: "notes server URL is required",
		},
		{
			name:    "empty token",
			baseURL: "http://localhost:8080",
			token:   "",
			wantErr: "bearer token is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.baseURL, tc.token)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %v, want to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestNew_Success(t *testing.T) {
	client, err := New("http://localhost:8080/", "mytoken")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client == nil {
		t.Fatal("New() returned nil client")
	}
}

type captureTransport struct {
	lastReq *http.Request
}

func (c *captureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	c.lastReq = req.Clone(context.Background())
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
	}, nil
}

func TestNew_WithHTTPClient(t *testing.T) {
	ct := &captureTransport{}
	httpClient := &http.Client{Transport: ct}

	client, err := New("http://example.com", "secret-token", WithHTTPClient(httpClient))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Make a call to trigger the interceptor + transport
	_, _ = client.ListRecentNotes(context.Background(), connect.NewRequest(&notesv1.ListRecentNotesRequest{}))

	if ct.lastReq == nil {
		t.Fatal("no request reached transport")
	}
	if got := ct.lastReq.Header.Get("Authorization"); got != "Bearer secret-token" {
		t.Errorf("Authorization header = %q, want %q", got, "Bearer secret-token")
	}
}

func TestNew_WithTimeout(t *testing.T) {
	// A transport that blocks longer than the timeout
	blocking := &blockingTransport{delay: 200 * time.Millisecond}

	client, err := New("http://example.com", "tok",
		WithHTTPClient(&http.Client{Transport: blocking}),
		WithTimeout(20*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	_, err = client.ListRecentNotes(ctx, connect.NewRequest(&notesv1.ListRecentNotesRequest{}))

	if err == nil {
		t.Fatal("expected error due to timeout, got nil")
	}

	// The error should indicate a deadline was exceeded (either from context or connect wrapping)
	if !errors.Is(err, context.DeadlineExceeded) && !strings.Contains(err.Error(), "deadline") {
		t.Errorf("error = %v, want context deadline or 'deadline' in message", err)
	}
}

type blockingTransport struct {
	delay time.Duration
}

func (b *blockingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	timer := time.NewTimer(b.delay)
	defer timer.Stop()

	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	case <-timer.C:
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	}
}
