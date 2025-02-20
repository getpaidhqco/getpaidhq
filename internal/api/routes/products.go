package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"payloop/internal/api/authn"
	"payloop/internal/api/controllers"
	"payloop/internal/application/lib/authz"
	"payloop/internal/application/lib/logger"
	"payloop/internal/lib"
)

type ProductRoutes struct {
	logger            logger.Logger
	handler           lib.RequestHandler
	productController controllers.ProductController
	authz             authz.Authz
}

// Setup user routes
func (s ProductRoutes) Setup() {
	s.logger.Info("Setting up Product")
	api := s.handler.Gin.Group("/api")
	{
		api.GET("/products", s.checkAuthz(authz.ListProducts), s.productController.List)
		api.GET("/products/:id", s.checkAuthz(authz.GetProduct), s.productController.Get)
		api.POST("/products", s.checkAuthz(authz.CreateProduct), s.productController.Create)

		// TODO move to a separate controller
		api.POST("/prices", s.checkAuthz(authz.CreatePrice), s.productController.CreatePrice)
	}
}

func (s ProductRoutes) checkAuthz(action authz.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _ := c.Get("user")
		authUser := user.(authn.User)
		allowed := s.authz.Enforce(authUser, action, "")
		if !allowed {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// NewProductRoutes creates new user controller
func NewProductRoutes(
	logger logger.Logger,
	handler lib.RequestHandler,
	productController controllers.ProductController,
	authz authz.Authz,
) ProductRoutes {
	return ProductRoutes{
		handler:           handler,
		logger:            logger,
		authz:             authz,
		productController: productController,
	}
}
