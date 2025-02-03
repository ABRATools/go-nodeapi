package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/dockerapi"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

// containers, err := ListContainers()
// if err != nil {
// 	log.Fatalf("Error listing containers: %v", err)
// }

// for _, c := range containers {
// 	fmt.Printf("Container ID: %s\nStatus: %s\nCPU Usage: %.2f%%\nMemory Usage: %d bytes\n\n",
// 		c.ID, c.Status, c.CPU, c.Memory)
// }

func RegisterAPIRoutes(router *gin.Engine) {
	api := router.Group("/api")
	{
		api.GET("/get-docker-containers", func(c *gin.Context) {
			dockerContainers, err := dockerapi.ListContainers()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error listing containers: %v", err)
			}
			c.JSON(http.StatusOK, dockerContainers)
		})
		api.GET("/get-podman-containers", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			podmanContainers := podmanapi.ListPodmanContainers(podmanContext)
			c.JSON(http.StatusOK, podmanContainers)
		})
	}
}
