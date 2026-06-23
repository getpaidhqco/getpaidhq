package cli

import (
	"context"
	"errors"

	"github.com/ogen-go/ogen/ogenerrors"

	"github.com/getpaidhqco/getpaidhq/cli/internal/apigen"
)

// apiKeySource supplies the x-api-key header to the generated client. The
// server accepts either a Clerk bearer token OR an x-api-key (see the authn
// wrapper in the server's middleware); the CLI is headless and uses the API
// key, so BearerAuth is skipped, letting ogen fall through to the api-key
// requirement.
type apiKeySource struct{ key string }

func (s apiKeySource) ApiKeyAuth(_ context.Context, _ apigen.OperationName) (apigen.ApiKeyAuth, error) {
	if s.key == "" {
		return apigen.ApiKeyAuth{}, errors.New("no API key configured: set GPHQ_API_KEY, pass --api-key, or add api_key to ~/.config/gphq/config.toml")
	}
	return apigen.ApiKeyAuth{APIKey: s.key}, nil
}

func (s apiKeySource) BearerAuth(_ context.Context, _ apigen.OperationName) (apigen.BearerAuth, error) {
	// Not used by the CLI. Returning the skip sentinel makes ogen try the
	// next (api-key) security requirement instead of failing the request.
	return apigen.BearerAuth{}, ogenerrors.ErrSkipClientSecurity
}
