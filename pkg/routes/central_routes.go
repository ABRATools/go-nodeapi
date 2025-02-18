package routes

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/hostdata"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

func RegisterCentralRoutes(router *gin.Engine) {
	central := router.Group("/")
	{
		central.GET("node-info", func(c *gin.Context) {
			retInfoChan := make(chan *hostdata.HostInfo)
			getHostInfoCtx, getInfoCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer getInfoCancel()
			retContainersChan := make(chan []podmanapi.PodmanContainer)
			getContainersCtx, getContainersCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer getContainersCancel()
			go func() {
				hostInfo, err := hostdata.GetHostInfo()
				if err != nil {
					fmt.Printf("Error retrieving host info: %v\n", err)
					c.String(http.StatusInternalServerError, "Error retrieving host info: %v", err)
					return
				}
				retInfoChan <- hostInfo
			}()
			go func() {
				podmanContext, err := podmanapi.InitPodmanConnection()
				if err != nil {
					c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
				}
				podmanContainers, err := podmanapi.ListPodmanContainers(podmanContext)
				if err != nil {
					c.String(http.StatusInternalServerError, "Error getting Podman Containers: %v", err)
					return
				}
				retContainersChan <- podmanContainers
			}()
			select {
			case <-retInfoChan:
				break
			case <-getHostInfoCtx.Done():
				c.String(http.StatusInternalServerError, "Timeout retrieving host info")
				return
			}
			select {
			case <-retContainersChan:
				break
			case <-getContainersCtx.Done():
				c.String(http.StatusInternalServerError, "Timeout retrieving host info")
				return
			}
			hostInfo := <-retInfoChan
			podmanContainers := <-retContainersChan
			hostInfo.NumContainers = len(podmanContainers)
			c.JSON(http.StatusOK, gin.H{"host": hostInfo, "containers": podmanContainers})
		})
	}
}
