package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

func RegisterAPIRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.GET("/get-podman-containers", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			podmanContainers, err := podmanapi.ListPodmanContainers(podmanContext)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting Podman Containers: %v", err)
			}
			c.JSON(http.StatusOK, podmanContainers)
		})
		api.POST("/stop/:id", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := c.Param("id")
			status, err := podmanapi.StopPodmanContainer(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error stopping Podman Containers: %v", err)
			}
			c.JSON(http.StatusOK, status)
		})
		api.POST("/start/:id", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := c.Param("id")
			status, err := podmanapi.StartPodmanContainer(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error starting Podman Containers: %v", err)
			}
			c.JSON(http.StatusOK, status)
		})
	}
}
