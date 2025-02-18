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
			errInfoChan := make(chan error)
			getHostInfoCtx, getInfoCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer getInfoCancel()
			retContainersChan := make(chan []podmanapi.PodmanContainer)
			errContainersChan := make(chan error)
			getContainersCtx, getContainersCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer getContainersCancel()
			go func() {
				hostInfo, err := hostdata.GetHostInfo()
				if err != nil {
					errInfoChan <- fmt.Errorf("Error getting host info: %v", err)
					return
				}
				retInfoChan <- hostInfo
			}()
			go func() {
				podmanContext, err := podmanapi.InitPodmanConnection()
				if err != nil {
					errContainersChan <- fmt.Errorf("Error connecting to Podman Socket: %v", err)
					return
				}
				podmanContainers, err := podmanapi.ListPodmanContainers(podmanContext)
				if err != nil {
					errContainersChan <- fmt.Errorf("Error getting Podman Containers: %v", err)
					return
				}
				retContainersChan <- podmanContainers
			}()
			var hostInfo *hostdata.HostInfo
			var podmanContainers []podmanapi.PodmanContainer
			select {
			case hostInfo = <-retInfoChan:
				break
			case err := <-errInfoChan:
				c.String(http.StatusInternalServerError, "Error retrieving host info: %v", err)
				return
			case <-getHostInfoCtx.Done():
				c.String(http.StatusInternalServerError, "Timeout retrieving host info")
				return
			}
			select {
			case podmanContainers = <-retContainersChan:
				break
			case err := <-errContainersChan:
				c.String(http.StatusInternalServerError, "Error retrieving containers: %v", err)
				return
			case <-getContainersCtx.Done():
				c.String(http.StatusInternalServerError, "Timeout retrieving host info")
				return
			}
			hostInfo.NumContainers = len(podmanContainers)
			c.JSON(http.StatusOK, gin.H{"host": hostInfo, "containers": podmanContainers})
		})
	}
}
