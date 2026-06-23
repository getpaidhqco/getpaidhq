package middleware

import (
	"context"
	"net/http"

	"github.com/eben-vranken/idempo"

	"getpaidhq/internal/core/port"
)

// idempoStore adapts our port.IdempotencyStore to idempo.Store and scopes every
// key by an org prefix, so two orgs that send the same Idempotency-Key can never
// collide.
//
// The org is resolved from the request context once, at request time, and baked
// into the shim — NOT read from the per-call ctx. idempo calls Complete and
// Abandon with a fresh context.Background() (so persistence survives client
// disconnects), which carries no AuthUser; reading the org off that ctx would
// scope Complete/Abandon under a different key than Claim and silently break
// replay. Binding the org per request keeps Claim/Complete/Abandon on one key.
type idempoStore struct {
	store port.IdempotencyStore
	org   string
}

var _ idempo.Store = idempoStore{}

func (a idempoStore) Claim(ctx context.Context, key, requestHash, token string) (idempo.ClaimResult, error) {
	c, err := a.store.Claim(ctx, a.scopeKey(key), requestHash, token)
	return idempo.ClaimResult{
		Status:  idempo.ClaimStatus(c.Status),
		Code:    c.Code,
		Headers: c.Headers,
		Body:    c.Body,
	}, err
}

func (a idempoStore) Complete(ctx context.Context, key, token string, statusCode int, headers, body []byte) error {
	return a.store.Complete(ctx, a.scopeKey(key), token, statusCode, headers, body)
}

func (a idempoStore) Abandon(ctx context.Context, key, token string) error {
	return a.store.Abandon(ctx, a.scopeKey(key), token)
}

func (a idempoStore) scopeKey(key string) string {
	return a.org + ":" + key
}

// NewIdempotencyMiddleware builds the idempo middleware over our store. It
// org-scopes keys: the order group's middleware runs AFTER authn, so AuthUser is
// on the request ctx here. A fresh idempo instance is built per request, bound
// to that request's org, because idempo's own Complete/Abandon contexts do not
// carry the AuthUser.
func NewIdempotencyMiddleware(store port.IdempotencyStore, opts idempo.Options) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			org := ""
			if u, ok := AuthUserFrom(r.Context()); ok {
				org = u.OrgId
			}
			scoped := idempo.New(idempoStore{store: store, org: org}, opts)
			scoped.Handler(next).ServeHTTP(w, r)
		})
	}
}
