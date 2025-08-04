package routes

import (
	"payloop/internal/api/controllers"
	"payloop/internal/lib"
)

// OAuthRoutes contains OAuth-related routes
type OAuthRoutes struct {
	handler         lib.RequestHandler
	oauthController controllers.OAuthController
}

// NewOAuthRoutes creates new OAuth routes
func NewOAuthRoutes(handler lib.RequestHandler, oauthController controllers.OAuthController) OAuthRoutes {
	return OAuthRoutes{
		handler:         handler,
		oauthController: oauthController,
	}
}

// Setup OAuth routes
func (r OAuthRoutes) Setup() {
	// OAuth 2.1 discovery endpoint
	// This proxies to Clerk's well-known configuration to simplify the flow
	r.handler.Gin.GET("/.well-known/oauth-authorization-server", r.oauthController.OAuthDiscovery)

	// Also support standard OAuth 2.0 discovery endpoint
	r.handler.Gin.GET("/.well-known/openid_configuration", r.oauthController.OAuthDiscovery)
}
