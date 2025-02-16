package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/nginxtemplates"
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
			if len(podmanContainers) == 0 {
				c.String(http.StatusNoContent, "No containers found")
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
		api.POST("/remove/:id", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			id := c.Param("id")
			err = podmanapi.RemovePodmanContainer(podmanContext, id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error removing Podman Containers: %v", err)
				return
			}
			// remove nginx config
			err = nginxtemplates.DeleteNginxConfig(id)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error removing Nginx Config: %v", err)
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "Container removed successfully"})
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
			// get the container IP
			ip, err := podmanapi.GetIPAddress(podmanContext, containerID)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting IP Address of Podman Containers: %v", err)
				return
			}
			// create nginx config
			// use default portmap for now
			webConf := nginxtemplates.NginxConfig{
				Path: containerName,
				IP:   ip,
				PortMap: map[uint]string{
					5801: "novnc",
					7681: "ttyd",
				},
			}
			err = nginxtemplates.GenerateNginxConfig(webConf)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error generating Nginx Config: %v", err)
				return
			}
			c.JSON(http.StatusOK, containerID)
		})
	}
}
