package oauth

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RefreshingTokenSource caches an OAuth token and refreshes it shortly before
// it expires. It is safe for concurrent use and implements the API client's
// consumer-defined token source through AccessToken.
type RefreshingTokenSource struct {
	// Auth performs the refresh grant. It is required.
	Auth *Client
	// OnRefresh is called after a successful refresh. Use it to persist the
	// new access and refresh tokens. Returning an error fails the request; the
	// refreshed token stays cached.
	OnRefresh func(ctx context.Context, token *Token) error

	mu     sync.Mutex
	token  *Token
	now    func() time.Time
	leeway time.Duration
}

// NewRefreshingTokenSource creates a token source for a previously issued token.
func NewRefreshingTokenSource(auth *Client, token *Token) (*RefreshingTokenSource, error) {
	if auth == nil {
		return nil, errors.New("etsy: oauth client is nil")
	}
	if token == nil || token.AccessToken == "" {
		return nil, ErrNoToken
	}
	s := &RefreshingTokenSource{
		Auth:   auth,
		now:    time.Now,
		leeway: time.Minute,
	}
	s.setTokenLocked(token)
	return s, nil
}

// AccessToken returns a valid bearer token, refreshing it when necessary.
func (s *RefreshingTokenSource) AccessToken(ctx context.Context) (string, error) {
	token, err := s.Token(ctx)
	if err != nil {
		return "", err
	}
	if token == nil || token.AccessToken == "" {
		return "", ErrNoToken
	}
	return token.AccessToken, nil
}

// Token returns a valid token, refreshing it when it is close to expiry.
func (s *RefreshingTokenSource) Token(ctx context.Context) (*Token, error) {
	if s == nil {
		return nil, ErrNoToken
	}

	s.mu.Lock()
	if s.token == nil {
		s.mu.Unlock()
		return nil, ErrNoToken
	}
	if !s.expiring(s.token) {
		token := *s.token
		s.mu.Unlock()
		return &token, nil
	}
	if s.token.RefreshToken == "" {
		s.mu.Unlock()
		return nil, ErrNoRefreshToken
	}
	if s.Auth == nil {
		s.mu.Unlock()
		return nil, errors.New("etsy: oauth client is nil")
	}

	refreshed, err := s.Auth.Refresh(ctx, s.token.RefreshToken)
	if err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.setTokenLocked(refreshed)
	token := *s.token
	onRefresh := s.OnRefresh
	s.mu.Unlock()

	if onRefresh != nil {
		if err := onRefresh(ctx, &token); err != nil {
			return nil, fmt.Errorf("etsy: persist refreshed token: %w", err)
		}
	}
	return &token, nil
}

// SetToken replaces the cached token.
func (s *RefreshingTokenSource) SetToken(token *Token) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setTokenLocked(token)
}

func (s *RefreshingTokenSource) setTokenLocked(token *Token) {
	if token == nil {
		s.token = nil
		return
	}
	copyOfToken := *token
	if copyOfToken.TokenType == "" {
		copyOfToken.TokenType = "Bearer"
	}
	if copyOfToken.ExpiresAt.IsZero() && copyOfToken.ExpiresIn > 0 {
		now := s.now
		if now == nil {
			now = time.Now
		}
		copyOfToken.ExpiresAt = now().UTC().Add(time.Duration(copyOfToken.ExpiresIn) * time.Second)
	}
	s.token = &copyOfToken
}

func (s *RefreshingTokenSource) expiring(token *Token) bool {
	if token.ExpiresAt.IsZero() {
		return false
	}
	now := s.now
	if now == nil {
		now = time.Now
	}
	leeway := s.leeway
	if leeway <= 0 {
		leeway = time.Minute
	}
	return !now().Add(leeway).Before(token.ExpiresAt)
}
