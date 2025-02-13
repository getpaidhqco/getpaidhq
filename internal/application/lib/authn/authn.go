package authn

import (
	"context"
	"payloop/internal/api/authn"
)

type Authenticator interface {
	Setup()
	Authenticate(ctx context.Context, token string) (authn.User, error)
}
