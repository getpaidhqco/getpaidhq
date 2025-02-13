package clerk

import (
	"github.com/clerkinc/clerk-sdk-go/clerk"
	"payloop/internal/lib"
)

type ClerkClient struct {
	client clerk.Client
}

func NewClerkClient(env lib.Env, logger lib.Logger) ClerkClient {
	client, err := clerk.NewClient(env.ClerkSecretKey)
	if err != nil {
		logger.Error("Failed to create clerk client", "error", err)
		panic(err)
	}

	return ClerkClient{
		client: client,
	}
}
