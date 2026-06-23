package middleware

import (
	"context"
	"net/http"

	"github.com/eben-vranken/idempo"

	"getpaidhq/internal/core/port"
)

// idempoStore adapts our port.IdempotencyStore to idempo.Store and scopes every
// key by the authenticated org, so two orgs that send the same Idempotency-Key
// can never collide. The order group's middleware runs AFTER authn, so AuthUser
// is on ctx here.
type idempoStore struct{ store port.IdempotencyStore }

var _ idempo.Store = idempoStore{}

func (a idempoStore) Claim(ctx context.Context, key, requestHash, token string) (idempo.ClaimResult, error) {
	c, err := a.store.Claim(ctx, scopeKey(ctx, key), requestHash, token)
	return idempo.ClaimResult{
		Status:  idempo.ClaimStatus(c.Status),
		Code:    c.Code,
		Headers: c.Headers,
		Body:    c.Body,
	}, err
}

func (a idempoStore) Complete(ctx context.Context, key, token string, statusCode int, headers, body []byte) error {
	return a.store.Complete(ctx, scopeKey(ctx, key), token, statusCode, headers, body)
}

func (a idempoStore) Abandon(ctx context.Context, key, token string) error {
	return a.store.Abandon(ctx, scopeKey(ctx, key), token)
}

func scopeKey(ctx context.Context, key string) string {
	if u, ok := AuthUserFrom(ctx); ok {
		return u.OrgId + ":" + key
	}
	return ":" + key
}

// NewIdempotencyMiddleware builds the idempo middleware over our store.
func NewIdempotencyMiddleware(store port.IdempotencyStore, opts idempo.Options) func(http.Handler) http.Handler {
	return idempo.New(idempoStore{store: store}, opts).Handler
}
