package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func createOrder(c *gin.Context) {

	c.IndentedJSON(http.StatusOK, gin.H{"adminFunction": "adminFunction content"})
}
