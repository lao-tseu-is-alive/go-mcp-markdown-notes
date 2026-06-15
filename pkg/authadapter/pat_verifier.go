package authadapter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// PatTokenPrefix marks a bearer token as a personal access token issued by
// go-cloud-k8s-auth; such tokens are verified by introspection instead of
// local JWT parsing.
const PatTokenPrefix = "pat_"

const (
	defaultIntrospectionTimeout = 5 * time.Second
	defaultPatCacheTTL          = 60 * time.Second
)

// introspectRequest / introspectResponse mirror the REST shape of the
// AuthService.IntrospectToken RPC exposed by go-cloud-k8s-auth via Vanguard.
// Calling the plain JSON route avoids a Go module dependency on the auth repo.
type introspectRequest struct {
	Token string `json:"token"`
}

type introspectResponse struct {
	Active bool     `json:"active"`
	UserID int64    `json:"userId,string"`
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Scopes []string `json:"scopes"`
}

type patCacheEntry struct {
	user      AuthenticatedUser
	expiresAt time.Time
}

// PatVerifier verifies personal access tokens by calling the auth service
// introspection endpoint, caching positive results briefly so each MCP tool
// call does not cost an extra HTTP round-trip. Negative results are never
// cached: a token created moments ago must work immediately.
type PatVerifier struct {
	introspectURL string
	client        *http.Client
	cacheTTL      time.Duration

	mu    sync.RWMutex
	cache map[[32]byte]patCacheEntry
}

// NewPatVerifier creates a PatVerifier pointing at the auth service base URL
// (e.g. http://localhost:9090).
func NewPatVerifier(authServerURL string) (*PatVerifier, error) {
	base := strings.TrimRight(authServerURL, "/")
	if base == "" {
		return nil, errors.New("auth server URL is required")
	}
	return &PatVerifier{
		introspectURL: base + "/goapi/v1/auth/introspect",
		client:        &http.Client{Timeout: defaultIntrospectionTimeout},
		cacheTTL:      defaultPatCacheTTL,
		cache:         make(map[[32]byte]patCacheEntry),
	}, nil
}

// VerifyBearerToken checks the in-memory cache first; on a miss it calls the introspection endpoint and caches a positive result.
func (v *PatVerifier) VerifyBearerToken(ctx context.Context, token string) (*AuthenticatedUser, error) {
	if !strings.HasPrefix(token, PatTokenPrefix) {
		return nil, ErrInvalidToken
	}

	key := sha256.Sum256([]byte(token))
	if user, ok := v.cachedUser(key); ok {
		return user, nil
	}

	result, err := v.introspect(ctx, token)
	if err != nil {
		return nil, err
	}
	if !result.Active || result.UserID <= 0 {
		return nil, ErrInvalidToken
	}

	user := AuthenticatedUser{
		AppUserID:   result.UserID,
		Email:       result.Email,
		DisplayName: result.Name,
		Scopes:      slices.Clone(result.Scopes),
	}
	v.mu.Lock()
	v.cache[key] = patCacheEntry{user: user, expiresAt: time.Now().Add(v.cacheTTL)}
	v.mu.Unlock()

	copied := user
	copied.Scopes = slices.Clone(user.Scopes)
	return &copied, nil
}

// cachedUser performs a read-lock lookup and returns a copy with an independent Scopes slice to prevent mutation of cached state.
func (v *PatVerifier) cachedUser(key [32]byte) (*AuthenticatedUser, bool) {
	v.mu.RLock()
	entry, ok := v.cache[key]
	v.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	user := entry.user
	user.Scopes = slices.Clone(entry.user.Scopes)
	return &user, true
}

// introspect posts the token to the auth service introspection endpoint and decodes the response.
func (v *PatVerifier) introspect(ctx context.Context, token string) (*introspectResponse, error) {
	body, err := json.Marshal(introspectRequest{Token: token})
	if err != nil {
		return nil, fmt.Errorf("marshal introspection request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.introspectURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build introspection request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call token introspection: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token introspection returned HTTP %d", res.StatusCode)
	}

	var result introspectResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode introspection response: %w", err)
	}
	return &result, nil
}
