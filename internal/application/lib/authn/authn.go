package authn

import (
	"context"
	"payloop/internal/api/authn"
)

var PublicPaths = []string{"/api/health", "/api/notify", "/api/notify/cdc"}

func IsPublicPath(path string) bool {
	for _, publicPath := range PublicPaths {
		if path == publicPath {
			return true
		}
	}
	return false
}

type Authenticator interface {
	Setup()
	Authenticate(ctx context.Context, token string) (authn.User, error)
}
