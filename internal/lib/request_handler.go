package lib

import (
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/application/lib/logger"
)

// RequestHandler function
type RequestHandler struct {
	Gin *gin.Engine
}

// NewRequestHandler creates a new request handler
func NewRequestHandler(logger logger.Logger, reporter ErrorReporter) RequestHandler {
	engine := gin.Default()
	engine.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "not_found",
			"message": "Route not found",
		})
	})
	engine.Use(sentrygin.New(sentrygin.Options{
		Repanic: true,
	}))
	engine.Use(func(ctx *gin.Context) {
		user, _ := ctx.Get("user")
		authUser := user.(authn.User)
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.Scope().SetTags(map[string]string{
				"org_id":  authUser.OrgId,
				"user_id": authUser.Id,
				"role":    string(authUser.PrimaryRole),
			})
		}
		ctx.Next()
	})
	return RequestHandler{Gin: engine}
}
