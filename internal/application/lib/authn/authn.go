package authn

import (
	"context"
	"payloop/internal/api/authn"
)

var PublicPaths = []string{"/api/health", "/api/notify", "/api/notify/cdc"}

type Authenticator interface {
	Setup()
	Authenticate(ctx context.Context, token string) (authn.User, error)
}
