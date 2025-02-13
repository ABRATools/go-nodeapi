package routes

import (
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sonarping/go-nodeapi/pkg/podmanapi"
)

func RegisterNetworkingRoutes(router *gin.Engine) {
	api := router.Group("/networks")
	{
		api.GET("/list", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			podmanNetworks, err := podmanapi.ListNetworks(podmanContext)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error getting Podman Containers: %v", err)
				return
			}
			c.JSON(http.StatusOK, podmanNetworks)
		})
		api.POST("/create", func(c *gin.Context) {
			// expects data in form-data in the format:
			// name: <network name>
			// network-ip: <network ip>
			// netmask: <network netmask>
			// gateway: <network gateway>

			// generates net.IPNet from network-ip, netmask and net.IP from gateway

			netName := c.PostForm("name")
			netIPNet := net.IPNet{
				IP:   net.ParseIP(c.PostForm("network-ip")),
				Mask: net.IPMask(net.ParseIP(c.PostForm("netmask"))),
			}
			gateway := net.ParseIP(c.PostForm("gateway"))

			if netName == "" || netIPNet.IP == nil || netIPNet.Mask == nil || gateway == nil {
				c.String(http.StatusBadRequest, "Name, Network IP, Netmask and Gateway are required. Please provide valid values.")
				return
			}
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			err = podmanapi.InitNewNetwork(podmanContext, netName, netIPNet, gateway)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error creating Podman Containers: %v", err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "Network created successfully"})
		})
		api.POST("/remove/:name", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			name := c.Param("name")
			err = podmanapi.RemoveNetwork(podmanContext, name)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error removing Podman Containers: %v", err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "Network removed successfully"})
		})
		api.POST("/attach/:containerID/:networkName", func(c *gin.Context) {
			podmanContext, err := podmanapi.InitPodmanConnection()
			if err != nil {
				c.String(http.StatusInternalServerError, "Error connecting to Podman Socket: %v", err)
			}
			containerID := c.Param("containerID")
			networkName := c.Param("networkName")
			ip, err := podmanapi.AttachContainerToNetwork(podmanContext, containerID, networkName)
			if err != nil {
				c.String(http.StatusInternalServerError, "Error attaching container to network: %v", err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "Container attached to network successfully", "ip": ip})
		})
	}
}
