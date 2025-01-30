package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.GET("/ping/:pong", func(c *gin.Context) {
			response_pong := c.Param("pong")
			c.String(http.StatusOK, "Pong: %s\n", response_pong)
		})
		api.GET("/testdb", func(c *gin.Context) {
			c.String(http.StatusOK, "Test DB\n")
		})
	}
}
