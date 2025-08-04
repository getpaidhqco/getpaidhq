package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

// OAuthController handles OAuth 2.1 discovery endpoints
type OAuthController struct {
	logger logger.Logger
	env    lib.Env
}

// NewOAuthController creates a new OAuth controller
func NewOAuthController(logger logger.Logger, env lib.Env) OAuthController {
	return OAuthController{
		logger: logger,
		env:    env,
	}
}

// OAuthDiscovery handles OAuth 2.1/OpenID Connect discovery
// It redirects to Clerk's well-known configuration endpoint
func (oc OAuthController) OAuthDiscovery(c *gin.Context) {
	oc.logger.Info("OAuth discovery endpoint called", "userAgent", c.GetHeader("User-Agent"))
	
	// Get Clerk domain from environment configuration
	clerkDomain := oc.env.ClerkDomain
	if clerkDomain == "" {
		oc.logger.Error("Clerk domain not configured")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":       "configuration_error",
			"description": "Clerk domain not configured. Set GPHQ_CLERK_DOMAIN environment variable.",
		})
		return
	}
	
	// Ensure the domain starts with https://
	if !strings.HasPrefix(clerkDomain, "http://") && !strings.HasPrefix(clerkDomain, "https://") {
		clerkDomain = "https://" + clerkDomain
	}
	
	// Construct the Clerk discovery URL
	clerkDiscoveryURL := clerkDomain + "/.well-known/openid_configuration"
	
	oc.logger.Debug("Redirecting to Clerk discovery endpoint", "url", clerkDiscoveryURL)
	
	// Return a 302 redirect to Clerk's discovery endpoint
	c.Redirect(http.StatusFound, clerkDiscoveryURL)
}