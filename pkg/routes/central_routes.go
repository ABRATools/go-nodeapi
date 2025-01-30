package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterCentralRoutes(router *gin.Engine) {
	central := router.Group("/")
	{
		central.GET("/ping/:pong", func(c *gin.Context) {
			response_pong := c.Param("pong")
			c.String(http.StatusOK, "Pong: %s\n", response_pong)
		})
	}
}
