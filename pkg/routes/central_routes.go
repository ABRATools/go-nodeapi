package routes

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/hostdata"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

func RegisterCentralRoutes(router *gin.Engine) {
	central := router.Group("/")
	{
		central.GET("node-info", func(c *gin.Context) {
			hostInfo, err := hostdata.GetHostInfo()
			if err != nil {
				fmt.Printf("Error retrieving host info: %v\n", err)
				c.String(http.StatusInternalServerError, "Error retrieving host info: %v", err)
				return
			}
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			podmanContainers, err := podmanapi.ListPodmanContainers(podmanContext)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting Podman Containers: %v", err)
				return
			}
			hostInfo.NumContainers = len(podmanContainers)
			c.JSON(http.StatusOK, gin.H{"host": hostInfo, "containers": podmanContainers})
		})
	}
}
