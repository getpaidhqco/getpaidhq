package http

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"net/http"
)

func Server(lc fx.Lifecycle) *gin.Engine {

	router := gin.Default()
	addGroups(router) // define rules for router

	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	lc.Append(fx.Hook{

		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return router
}

func addGroups(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/orders", createOrder)

		orders := api.Group("/orders")
		{
			orders.GET("/:id", createOrder)
		}
		users := api.Group("/users")
		{
			users.GET("/", usersFunction)
		}
	}
}

func usersFunction(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, gin.H{"usersFunction": "usersFunction content"})
}
