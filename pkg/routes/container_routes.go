package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

func RegisterContainerRoutes(router *gin.Engine) {
	api := router.Group("/containers")
	{
		api.GET("/list", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			podmanContainers, err := podmanapi.ListPodmanContainers(podmanContext)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting Podman Containers: %v", err)
				return
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
				return
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
				return
			}
			c.JSON(http.StatusOK, status)
		})
		api.POST("/create", func(c *gin.Context) {
			// expects data in form-data in the format:
			// image: <image name>
			// name: <container name>
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			imageName := c.PostForm("image")
			containerName := c.PostForm("name")
			if imageName == "" || containerName == "" {
				c.String(http.StatusBadRequest, "Image and Name are required")
				return
			}
			// create the container
			containerID, err := podmanapi.CreateFromImage(podmanContext, imageName, containerName)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error creating Podman Containers: %v", err)
				return
			}
			// start the container
			_, err = podmanapi.StartPodmanContainer(podmanContext, containerID)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error starting Podman Containers: %v", err)
				return
			}
			c.JSON(http.StatusOK, containerID)
		})
	}
}
